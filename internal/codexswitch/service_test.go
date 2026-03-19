package codexswitch

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGetAppStateAutoImportsCurrentProfile(t *testing.T) {
	service := newTestService(t)
	codexHome := prepareCodexHome(t, "codex")
	if err := service.saveSettings(AppSettings{CodexHomePath: codexHome}); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}

	state, err := service.GetAppState()
	if err != nil {
		t.Fatalf("GetAppState returned error: %v", err)
	}

	if len(state.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(state.Profiles))
	}
	if !state.Current.Managed {
		t.Fatal("expected current profile to be auto-managed")
	}
	if !state.Profiles[0].IsActive {
		t.Fatal("expected imported profile to be active")
	}
}

func TestCreateAndSwitchAPIProfile(t *testing.T) {
	service := newTestService(t)
	codexHome := prepareCodexHome(t, "codex")
	if err := service.saveSettings(AppSettings{CodexHomePath: codexHome}); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}
	if _, err := service.GetAppState(); err != nil {
		t.Fatalf("GetAppState returned error: %v", err)
	}

	state, err := service.CreateAPIProfile(APIProfileInput{
		BaseURL:              "https://api.openai.com/v1",
		Model:                "gpt-5.4",
		ModelReasoningEffort: "xhigh",
		APIKey:               "sk-switch-1234567890",
	})
	if err != nil {
		t.Fatalf("CreateAPIProfile returned error: %v", err)
	}
	if len(state.Profiles) != 2 {
		t.Fatalf("expected 2 profiles after create, got %d", len(state.Profiles))
	}

	var apiProfile ProfileMeta
	for _, profile := range state.Profiles {
		if profile.Type == ProfileTypeAPI {
			apiProfile = profile
			break
		}
	}
	if apiProfile.ID == "" {
		t.Fatal("expected created api profile")
	}

	state, err = service.SwitchProfile(apiProfile.ID)
	if err != nil {
		t.Fatalf("SwitchProfile returned error: %v", err)
	}

	authRaw, err := os.ReadFile(filepath.Join(codexHome, "auth.json"))
	if err != nil {
		t.Fatalf("read switched auth.json failed: %v", err)
	}
	configRaw, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		t.Fatalf("read switched config.toml failed: %v", err)
	}

	if !strings.Contains(string(authRaw), "sk-switch-1234567890") {
		t.Fatalf("switched auth.json does not contain expected key: %s", string(authRaw))
	}
	if !strings.Contains(string(configRaw), "https://api.openai.com/v1") {
		t.Fatalf("switched config.toml does not contain expected base url: %s", string(configRaw))
	}
	if state.Current.Type != ProfileTypeAPI {
		t.Fatalf("expected current type api after switch, got %s", state.Current.Type)
	}
}

func TestUpdateSettingsRescansPath(t *testing.T) {
	service := newTestService(t)
	codexHomeA := prepareCodexHome(t, "codex")
	codexHomeB := prepareCodexHome(t, "api")

	state, err := service.UpdateSettings(UpdateSettingsInput{CodexHomePath: codexHomeA})
	if err != nil {
		t.Fatalf("UpdateSettings A returned error: %v", err)
	}
	if state.Current.Type != ProfileTypeOfficial {
		t.Fatalf("expected official current type, got %s", state.Current.Type)
	}

	state, err = service.UpdateSettings(UpdateSettingsInput{CodexHomePath: codexHomeB})
	if err != nil {
		t.Fatalf("UpdateSettings B returned error: %v", err)
	}
	if state.Current.Type != ProfileTypeAPI {
		t.Fatalf("expected api current type, got %s", state.Current.Type)
	}
}

func TestRefreshRateLimits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/backend-api/wham/usage" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		if !strings.HasPrefix(request.Header.Get("Authorization"), "Bearer ") {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
		  "plan_type": "pro",
		  "rate_limit": {
		    "primary_window": {
		      "used_percent": 42,
		      "limit_window_seconds": 300,
		      "reset_at": 1730947200
		    },
		    "secondary_window": {
		      "used_percent": 7,
		      "limit_window_seconds": 10080,
		      "reset_at": 1731547200
		    }
		  }
		}`))
	}))
	defer server.Close()

	service := newTestServiceWithHTTP(t, server.Client())

	authRaw := mustReadSample(t, "codex", "auth.json")
	configRaw := fmt.Sprintf(`model = "gpt-5.4"
model_reasoning_effort = "xhigh"
[model_providers.OpenAI]
base_url = "%s/backend-api"
`, server.URL)

	snapshot, err := buildProfileSnapshot(authRaw, configRaw, "imported_current", service.now())
	if err != nil {
		t.Fatalf("buildProfileSnapshot returned error: %v", err)
	}
	if err := service.saveProfileSnapshot(snapshot); err != nil {
		t.Fatalf("saveProfileSnapshot failed: %v", err)
	}

	state, err := service.RefreshRateLimits([]string{snapshot.Meta.ID})
	if err != nil {
		t.Fatalf("RefreshRateLimits returned error: %v", err)
	}

	var refreshed ProfileMeta
	for _, profile := range state.Profiles {
		if profile.ID == snapshot.Meta.ID {
			refreshed = profile
			break
		}
	}
	if refreshed.ID == "" {
		t.Fatal("expected refreshed profile in state")
	}
	if refreshed.RateLimits.Status != RateLimitStatusSuccess {
		t.Fatalf("expected success rate limit status, got %s", refreshed.RateLimits.Status)
	}
	if refreshed.RateLimits.Primary == nil || refreshed.RateLimits.Primary.UsedPercent != 42 {
		t.Fatalf("unexpected primary rate limit: %+v", refreshed.RateLimits.Primary)
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	return newTestServiceWithHTTP(t, &http.Client{Timeout: 5 * time.Second})
}

func newTestServiceWithHTTP(t *testing.T, client *http.Client) *Service {
	t.Helper()
	appConfigDir := t.TempDir()
	service, err := NewService(ServiceOptions{
		AppConfigDir: appConfigDir,
		HTTPClient:   client,
		Now: func() time.Time {
			return time.Date(2026, 3, 19, 15, 0, 0, 0, time.UTC)
		},
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	return service
}

func prepareCodexHome(t *testing.T, sampleDir string) string {
	t.Helper()
	root := t.TempDir()
	writeFileFromSample(t, filepath.Join(root, "auth.json"), sampleDir, "auth.json")
	writeFileFromSample(t, filepath.Join(root, "config.toml"), sampleDir, "config.toml")
	return root
}

func writeFileFromSample(t *testing.T, targetPath string, sampleParts ...string) {
	t.Helper()

	content := mustReadSample(t, sampleParts...)
	if err := os.WriteFile(targetPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write sample file failed: %v", err)
	}
}
