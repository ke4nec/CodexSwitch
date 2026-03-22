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

func TestGetAppStateAutoImportsCurrentCLIProfile(t *testing.T) {
	service := newTestService(t)
	codexHome := prepareCodexHomeWithFiles(t, "codex", "cli-auth.json", "config.toml")
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
	if state.Current.Type != ProfileTypeOfficial {
		t.Fatalf("expected official current type, got %s", state.Current.Type)
	}
	if state.Profiles[0].Source != profileSourceImportedCurrent {
		t.Fatalf("expected imported_current source, got %s", state.Profiles[0].Source)
	}

	stored, err := service.loadProfile(state.Profiles[0].ID)
	if err != nil {
		t.Fatalf("loadProfile failed: %v", err)
	}
	if !strings.Contains(stored.AuthRaw, "\"tokens\"") {
		t.Fatalf("expected stored auth to be normalized official auth: %s", stored.AuthRaw)
	}
}

func TestGetAppStateRepairsCurrentOfficialConfigWhenBaseURLIsActive(t *testing.T) {
	service := newTestService(t)
	codexHome := prepareCodexHomeWithFiles(t, "codex", "auth.json", "config.toml")
	writeFileFromSample(t, filepath.Join(codexHome, "config.toml"), "api", "config.toml")

	backupConfigRaw := `model = "gpt-5.4"
model_reasoning_effort = "medium"
[windows]
sandbox = "elevated"
`
	if err := safeWriteText(service.sharedOfficialConfigPath(), backupConfigRaw); err != nil {
		t.Fatalf("safeWriteText shared official config failed: %v", err)
	}
	if err := service.saveSettings(AppSettings{CodexHomePath: codexHome}); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}

	state, err := service.GetAppState()
	if err != nil {
		t.Fatalf("GetAppState returned error: %v", err)
	}

	if state.Current.Type != ProfileTypeOfficial {
		t.Fatalf("expected official current type, got %s", state.Current.Type)
	}
	if len(state.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(state.Profiles))
	}

	currentConfigRaw, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		t.Fatalf("read repaired current config.toml failed: %v", err)
	}
	if strings.TrimSpace(string(currentConfigRaw)) != strings.TrimSpace(backupConfigRaw) {
		t.Fatalf("expected current config.toml to be replaced with backup official config, got %s", string(currentConfigRaw))
	}

	sharedConfigRaw, err := os.ReadFile(service.sharedOfficialConfigPath())
	if err != nil {
		t.Fatalf("read shared official config failed: %v", err)
	}
	if strings.TrimSpace(string(sharedConfigRaw)) != strings.TrimSpace(backupConfigRaw) {
		t.Fatalf("expected shared official config to remain backup config, got %s", string(sharedConfigRaw))
	}

	stored, err := service.loadProfile(state.Profiles[0].ID)
	if err != nil {
		t.Fatalf("loadProfile failed: %v", err)
	}
	if strings.TrimSpace(stored.ConfigRaw) != strings.TrimSpace(backupConfigRaw) {
		t.Fatalf("expected stored official config to use backup config, got %s", stored.ConfigRaw)
	}
	if stored.Meta.BaseURL != "" {
		t.Fatalf("expected repaired official profile to clear api base url, got %s", stored.Meta.BaseURL)
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

func TestImportOfficialProfileFileFromCLIAndSwitch(t *testing.T) {
	service := newTestService(t)
	codexHome := t.TempDir()
	if err := service.saveSettings(AppSettings{CodexHomePath: codexHome}); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}

	cliPath := samplePath("codex", "cli-auth.json")
	state, err := service.ImportOfficialProfileFile(cliPath)
	if err != nil {
		t.Fatalf("ImportOfficialProfileFile returned error: %v", err)
	}

	if len(state.Profiles) != 1 {
		t.Fatalf("expected 1 imported profile, got %d", len(state.Profiles))
	}

	imported := state.Profiles[0]
	if imported.Type != ProfileTypeOfficial {
		t.Fatalf("expected official profile, got %s", imported.Type)
	}
	if imported.Source != profileSourceImportedFileCLI {
		t.Fatalf("expected CLI import source, got %s", imported.Source)
	}

	stored, err := service.loadProfile(imported.ID)
	if err != nil {
		t.Fatalf("loadProfile failed: %v", err)
	}
	if strings.TrimSpace(stored.ConfigRaw) != strings.TrimSpace(officialConfigTemplate) {
		t.Fatalf("expected official config template, got %s", stored.ConfigRaw)
	}
	if strings.Contains(stored.AuthRaw, "\"type\": \"codex\"") {
		t.Fatalf("expected normalized auth, got raw CLI file: %s", stored.AuthRaw)
	}
	if !strings.Contains(stored.AuthRaw, "\"tokens\"") {
		t.Fatalf("expected normalized auth to contain tokens: %s", stored.AuthRaw)
	}

	state, err = service.ImportOfficialProfileFile(cliPath)
	if err != nil {
		t.Fatalf("second ImportOfficialProfileFile returned error: %v", err)
	}
	if len(state.Profiles) != 1 {
		t.Fatalf("expected import dedupe to keep 1 profile, got %d", len(state.Profiles))
	}

	state, err = service.SwitchProfile(imported.ID)
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
	if !strings.Contains(string(authRaw), "\"tokens\"") {
		t.Fatalf("expected switched auth.json to be standard official auth: %s", string(authRaw))
	}
	if strings.TrimSpace(string(configRaw)) != strings.TrimSpace(officialConfigTemplate) {
		t.Fatalf("unexpected switched config.toml: %s", string(configRaw))
	}
	if state.Current.Type != ProfileTypeOfficial {
		t.Fatalf("expected current type official after switch, got %s", state.Current.Type)
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

func TestRefreshAPILatencyTests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/models" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		if request.Header.Get("Authorization") != "Bearer sk-test-public-sample-key-0001" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[{"id":"gpt-5.4"}]}`))
	}))
	defer server.Close()

	service := newTestServiceWithHTTP(t, server.Client())
	if err := service.saveSettings(AppSettings{CodexHomePath: t.TempDir()}); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}

	authRaw := mustReadSample(t, "api", "auth.json")
	configRaw := fmt.Sprintf(`model = "gpt-5.4"
model_reasoning_effort = "xhigh"
[model_providers.OpenAI]
base_url = "%s/v1"
`, server.URL)

	snapshot, err := buildProfileSnapshot(authRaw, configRaw, profileSourceCreatedAPIForm, service.now())
	if err != nil {
		t.Fatalf("buildProfileSnapshot returned error: %v", err)
	}
	if err := service.saveProfileSnapshot(snapshot); err != nil {
		t.Fatalf("saveProfileSnapshot failed: %v", err)
	}

	state, err := service.RefreshAPILatencyTests([]string{snapshot.Meta.ID})
	if err != nil {
		t.Fatalf("RefreshAPILatencyTests returned error: %v", err)
	}

	var refreshed ProfileMeta
	for _, profile := range state.Profiles {
		if profile.ID == snapshot.Meta.ID {
			refreshed = profile
			break
		}
	}
	if refreshed.ID == "" {
		t.Fatal("expected refreshed api profile in state")
	}
	if refreshed.LatencyTest.Status != LatencyTestStatusSuccess {
		t.Fatalf("expected success latency status, got %s", refreshed.LatencyTest.Status)
	}
	if !refreshed.LatencyTest.Available {
		t.Fatal("expected api profile to be marked available")
	}
	if refreshed.LatencyTest.LatencyMs == nil || *refreshed.LatencyTest.LatencyMs <= 0 {
		t.Fatalf("expected latency ms to be recorded, got %+v", refreshed.LatencyTest)
	}
}

func TestRefreshAPILatencyTestsUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/models" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusUnauthorized)
		_, _ = writer.Write([]byte(`{"error":{"message":"Incorrect API key provided"}}`))
	}))
	defer server.Close()

	service := newTestServiceWithHTTP(t, server.Client())
	if err := service.saveSettings(AppSettings{CodexHomePath: t.TempDir()}); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}

	authRaw := mustReadSample(t, "api", "auth.json")
	configRaw := fmt.Sprintf(`model = "gpt-5.4"
model_reasoning_effort = "xhigh"
[model_providers.OpenAI]
base_url = "%s/v1"
`, server.URL)

	snapshot, err := buildProfileSnapshot(authRaw, configRaw, profileSourceCreatedAPIForm, service.now())
	if err != nil {
		t.Fatalf("buildProfileSnapshot returned error: %v", err)
	}
	if err := service.saveProfileSnapshot(snapshot); err != nil {
		t.Fatalf("saveProfileSnapshot failed: %v", err)
	}

	state, err := service.RefreshAPILatencyTests([]string{snapshot.Meta.ID})
	if err != nil {
		t.Fatalf("RefreshAPILatencyTests returned error: %v", err)
	}

	var refreshed ProfileMeta
	for _, profile := range state.Profiles {
		if profile.ID == snapshot.Meta.ID {
			refreshed = profile
			break
		}
	}
	if refreshed.ID == "" {
		t.Fatal("expected refreshed api profile in state")
	}
	if refreshed.LatencyTest.Status != LatencyTestStatusSuccess {
		t.Fatalf("expected finished latency status, got %s", refreshed.LatencyTest.Status)
	}
	if refreshed.LatencyTest.Available {
		t.Fatal("expected api profile to be marked unavailable")
	}
	if refreshed.LatencyTest.StatusCode == nil || *refreshed.LatencyTest.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status code 401, got %+v", refreshed.LatencyTest.StatusCode)
	}
	if !strings.Contains(refreshed.LatencyTest.ErrorMessage, "Incorrect API key provided") {
		t.Fatalf("expected unauthorized message, got %s", refreshed.LatencyTest.ErrorMessage)
	}
}

func TestRefreshOfficialLatencyTests(t *testing.T) {
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
		_, _ = writer.Write([]byte(`{"plan_type":"pro"}`))
	}))
	defer server.Close()

	service := newTestServiceWithHTTP(t, server.Client())
	if err := service.saveSettings(AppSettings{CodexHomePath: t.TempDir()}); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}

	authRaw := mustReadSample(t, "codex", "auth.json")
	configRaw := fmt.Sprintf(`model = "gpt-5.4"
model_reasoning_effort = "xhigh"
[model_providers.OpenAI]
base_url = "%s/backend-api"
`, server.URL)

	snapshot, err := buildProfileSnapshot(authRaw, configRaw, profileSourceImportedCurrent, service.now())
	if err != nil {
		t.Fatalf("buildProfileSnapshot returned error: %v", err)
	}
	if err := service.saveProfileSnapshot(snapshot); err != nil {
		t.Fatalf("saveProfileSnapshot failed: %v", err)
	}

	state, err := service.RefreshAPILatencyTests([]string{snapshot.Meta.ID})
	if err != nil {
		t.Fatalf("RefreshAPILatencyTests returned error: %v", err)
	}

	var refreshed ProfileMeta
	for _, profile := range state.Profiles {
		if profile.ID == snapshot.Meta.ID {
			refreshed = profile
			break
		}
	}
	if refreshed.ID == "" {
		t.Fatal("expected refreshed official profile in state")
	}
	if refreshed.LatencyTest.Status != LatencyTestStatusSuccess {
		t.Fatalf("expected success latency status, got %s", refreshed.LatencyTest.Status)
	}
	if !refreshed.LatencyTest.Available {
		t.Fatal("expected official profile to be marked available")
	}
	if refreshed.LatencyTest.LatencyMs == nil || *refreshed.LatencyTest.LatencyMs <= 0 {
		t.Fatalf("expected latency ms to be recorded, got %+v", refreshed.LatencyTest)
	}
}

func TestImportOfficialProfileFileRefreshesRateLimits(t *testing.T) {
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
		      "used_percent": 18,
		      "limit_window_seconds": 300,
		      "reset_at": 1730947200
		    },
		    "secondary_window": {
		      "used_percent": 3,
		      "limit_window_seconds": 10080,
		      "reset_at": 1731547200
		    }
		  }
		}`))
	}))
	defer server.Close()

	service := newTestServiceWithHTTP(t, server.Client())
	if err := service.saveSettings(AppSettings{CodexHomePath: t.TempDir()}); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}

	state, err := service.ImportOfficialProfileFile(samplePath("codex", "auth.json"))
	if err != nil {
		t.Fatalf("ImportOfficialProfileFile returned error: %v", err)
	}
	if len(state.Profiles) != 1 {
		t.Fatalf("expected 1 imported profile, got %d", len(state.Profiles))
	}

	stored, err := service.loadProfile(state.Profiles[0].ID)
	if err != nil {
		t.Fatalf("loadProfile failed: %v", err)
	}
	stored.ConfigRaw = fmt.Sprintf(`model = "gpt-5.4"
model_reasoning_effort = "xhigh"
[model_providers.OpenAI]
base_url = "%s/backend-api"
`, server.URL)
	if err := safeWriteText(service.sharedOfficialConfigPath(), stored.ConfigRaw); err != nil {
		t.Fatalf("safeWriteText shared official config failed: %v", err)
	}

	state, err = service.RefreshRateLimits([]string{state.Profiles[0].ID})
	if err != nil {
		t.Fatalf("RefreshRateLimits returned error: %v", err)
	}
	if state.Profiles[0].RateLimits.Primary == nil || state.Profiles[0].RateLimits.Primary.UsedPercent != 18 {
		t.Fatalf("unexpected rate limits after refresh: %+v", state.Profiles[0].RateLimits)
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

func prepareCodexHomeWithFiles(t *testing.T, sampleDir, authFileName, configFileName string) string {
	t.Helper()
	root := t.TempDir()
	writeFileFromSample(t, filepath.Join(root, "auth.json"), sampleDir, authFileName)
	writeFileFromSample(t, filepath.Join(root, "config.toml"), sampleDir, configFileName)
	return root
}

func writeFileFromSample(t *testing.T, targetPath string, sampleParts ...string) {
	t.Helper()

	content := mustReadSample(t, sampleParts...)
	if err := os.WriteFile(targetPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write sample file failed: %v", err)
	}
}

func samplePath(parts ...string) string {
	allParts := append([]string{"..", "..", "conf"}, parts...)
	return filepath.Join(allParts...)
}
