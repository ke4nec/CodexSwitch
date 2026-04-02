package codexswitch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
		ModelContextWindow:   "128000",
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
	if !strings.Contains(string(configRaw), "model_context_window = 128000") {
		t.Fatalf("switched config.toml does not contain expected context window: %s", string(configRaw))
	}
	if !strings.Contains(string(configRaw), "model_auto_compact_token_limit = 115200") {
		t.Fatalf("switched config.toml does not contain expected compact limit: %s", string(configRaw))
	}
	if state.Current.Type != ProfileTypeAPI {
		t.Fatalf("expected current type api after switch, got %s", state.Current.Type)
	}
}

func TestSetProfileDisabledPreventsSwitch(t *testing.T) {
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
		ModelContextWindow:   "128000",
		APIKey:               "sk-disabled-1234567890",
	})
	if err != nil {
		t.Fatalf("CreateAPIProfile returned error: %v", err)
	}

	apiProfile := findFirstProfileByType(state.Profiles, ProfileTypeAPI)
	if apiProfile.ID == "" {
		t.Fatal("expected created api profile")
	}

	state, err = service.SetProfileDisabled(apiProfile.ID, true)
	if err != nil {
		t.Fatalf("SetProfileDisabled returned error: %v", err)
	}

	disabledProfile := findProfileByID(state.Profiles, apiProfile.ID)
	if !disabledProfile.Disabled {
		t.Fatalf("expected api profile to be disabled, got %+v", disabledProfile)
	}

	if _, err := service.SwitchProfile(apiProfile.ID); err == nil || !strings.Contains(err.Error(), "已被禁用") {
		t.Fatalf("expected disabled profile switch to be rejected, got %v", err)
	}
}

func TestGetAPIProfileInputDefaultsMissingModelContextWindow(t *testing.T) {
	service := newTestService(t)

	authRaw := mustReadSample(t, "api", "auth.json")
	configRaw := strings.Replace(mustReadSample(t, "api", "config.toml"), "model_context_window = 1000000\n", "", 1)

	snapshot, err := buildProfileSnapshot(authRaw, configRaw, profileSourceCreatedAPIForm, service.now())
	if err != nil {
		t.Fatalf("buildProfileSnapshot returned error: %v", err)
	}
	if err := service.saveProfileSnapshot(snapshot); err != nil {
		t.Fatalf("saveProfileSnapshot failed: %v", err)
	}

	input, err := service.GetAPIProfileInput(snapshot.Meta.ID)
	if err != nil {
		t.Fatalf("GetAPIProfileInput returned error: %v", err)
	}
	if input.ModelContextWindow != defaultAPIModelContextWindow {
		t.Fatalf("expected default context window %s, got %s", defaultAPIModelContextWindow, input.ModelContextWindow)
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

func TestRefreshRateLimitsSkipsDisabledOfficialProfile(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
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
	snapshot.Meta.Disabled = true
	if err := service.saveProfileSnapshot(snapshot); err != nil {
		t.Fatalf("saveProfileSnapshot failed: %v", err)
	}

	state, err := service.RefreshRateLimits([]string{snapshot.Meta.ID})
	if err != nil {
		t.Fatalf("RefreshRateLimits returned error: %v", err)
	}

	refreshed := findProfileByID(state.Profiles, snapshot.Meta.ID)
	if !refreshed.Disabled {
		t.Fatalf("expected profile to remain disabled, got %+v", refreshed)
	}
	if requestCount != 0 {
		t.Fatalf("expected disabled official profile to skip rate limit refresh, got %d requests", requestCount)
	}
	if refreshed.RateLimits.Status != RateLimitStatusIdle {
		t.Fatalf("expected disabled profile rate limit status to remain idle, got %+v", refreshed.RateLimits)
	}
}

func TestRefreshAPILatencyTests(t *testing.T) {
	var (
		probeRequestMethod      string
		probeAuthorization      string
		availabilityMethod      string
		availabilityRequestBody map[string]any
	)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/":
			probeRequestMethod = request.Method
			probeAuthorization = request.Header.Get("Authorization")
			writer.WriteHeader(http.StatusNoContent)
		case "/v1/responses":
			availabilityMethod = request.Method
			if request.Header.Get("Authorization") != "Bearer sk-test-public-sample-key-0001" {
				writer.WriteHeader(http.StatusUnauthorized)
				return
			}
			time.Sleep(120 * time.Millisecond)
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("read request body failed: %v", err)
			}
			if err := json.Unmarshal(body, &availabilityRequestBody); err != nil {
				t.Fatalf("unmarshal request body failed: %v", err)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"id":"resp_123","status":"completed","output_text":"hello"}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
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
	if *refreshed.LatencyTest.LatencyMs >= 120 {
		t.Fatalf("expected latency ms to come from lightweight probe instead of responses request, got %+v", refreshed.LatencyTest.LatencyMs)
	}
	if len(refreshed.LatencyTest.History) != 1 {
		t.Fatalf("expected 1 latency history entry, got %+v", refreshed.LatencyTest.History)
	}
	if !refreshed.LatencyTest.History[0].Available {
		t.Fatalf("expected first history entry to be available, got %+v", refreshed.LatencyTest.History[0])
	}
	if refreshed.LatencyTest.History[0].CheckedAt != refreshed.LatencyTest.CheckedAt {
		t.Fatalf("expected history timestamp to match checkedAt, got %+v", refreshed.LatencyTest.History[0])
	}
	if probeRequestMethod != http.MethodGet {
		t.Fatalf("expected homepage probe GET request, got %s", probeRequestMethod)
	}
	if probeAuthorization != "" {
		t.Fatalf("expected homepage probe to skip authorization header, got %q", probeAuthorization)
	}
	if availabilityMethod != http.MethodPost {
		t.Fatalf("expected availability POST request, got %s", availabilityMethod)
	}
	if got := strings.TrimSpace(fmt.Sprintf("%v", availabilityRequestBody["model"])); got != "gpt-5.4" {
		t.Fatalf("expected model gpt-5.4, got %+v", availabilityRequestBody["model"])
	}
	if got := strings.TrimSpace(fmt.Sprintf("%v", availabilityRequestBody["input"])); got != latencyTestPrompt {
		t.Fatalf("expected input %q, got %+v", latencyTestPrompt, availabilityRequestBody["input"])
	}
}

func TestRefreshAPILatencyTestsPreservesUpdatedAt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/":
			writer.WriteHeader(http.StatusNoContent)
		case "/v1/responses":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"id":"resp_keep_updated_at","status":"completed","output_text":"ok"}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	nextNow := time.Date(2026, 3, 19, 15, 0, 0, 0, time.UTC)
	service := newTestServiceWithConfigDirHTTPAndNow(t, t.TempDir(), server.Client(), func() time.Time {
		current := nextNow
		nextNow = nextNow.Add(time.Minute)
		return current
	})
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
	originalUpdatedAt := snapshot.Meta.UpdatedAt
	if err := service.saveProfileSnapshot(snapshot); err != nil {
		t.Fatalf("saveProfileSnapshot failed: %v", err)
	}

	state, err := service.RefreshAPILatencyTests([]string{snapshot.Meta.ID})
	if err != nil {
		t.Fatalf("RefreshAPILatencyTests returned error: %v", err)
	}

	refreshed := findProfileByID(state.Profiles, snapshot.Meta.ID)
	if refreshed.ID == "" {
		t.Fatal("expected refreshed api profile in state")
	}
	if refreshed.UpdatedAt != originalUpdatedAt {
		t.Fatalf("expected latency refresh to preserve updatedAt %s, got %s", originalUpdatedAt, refreshed.UpdatedAt)
	}
	if refreshed.LatencyTest.CheckedAt == "" {
		t.Fatalf("expected latency refresh to set checkedAt, got %+v", refreshed.LatencyTest)
	}
	if refreshed.LatencyTest.CheckedAt == originalUpdatedAt {
		t.Fatalf("expected checkedAt to move forward independently from updatedAt, got %+v", refreshed.LatencyTest)
	}
}

func TestRefreshAPILatencyTestsSkipsDisabledProfile(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		writer.WriteHeader(http.StatusNoContent)
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
	snapshot.Meta.Disabled = true
	if err := service.saveProfileSnapshot(snapshot); err != nil {
		t.Fatalf("saveProfileSnapshot failed: %v", err)
	}

	state, err := service.RefreshAPILatencyTests([]string{snapshot.Meta.ID})
	if err != nil {
		t.Fatalf("RefreshAPILatencyTests returned error: %v", err)
	}

	refreshed := findProfileByID(state.Profiles, snapshot.Meta.ID)
	if !refreshed.Disabled {
		t.Fatalf("expected profile to remain disabled, got %+v", refreshed)
	}
	if requestCount != 0 {
		t.Fatalf("expected disabled api profile to skip latency refresh, got %d requests", requestCount)
	}
	if refreshed.LatencyTest.Status != LatencyTestStatusIdle {
		t.Fatalf("expected disabled profile latency status to remain idle, got %+v", refreshed.LatencyTest)
	}
}

func TestRefreshAPILatencyTestsUnauthorized(t *testing.T) {
	var (
		probeRequestMethod string
		probeAuthorization string
		requestMethod      string
		requestBody        map[string]any
	)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/":
			probeRequestMethod = request.Method
			probeAuthorization = request.Header.Get("Authorization")
			writer.WriteHeader(http.StatusNoContent)
		case "/v1/responses":
			requestMethod = request.Method
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("read request body failed: %v", err)
			}
			if err := json.Unmarshal(body, &requestBody); err != nil {
				t.Fatalf("unmarshal request body failed: %v", err)
			}
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusUnauthorized)
			_, _ = writer.Write([]byte(`{"error":{"message":"Incorrect API key provided","type":"invalid_request_error","code":"invalid_api_key"}}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
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
	if refreshed.LatencyTest.LatencyMs == nil || *refreshed.LatencyTest.LatencyMs <= 0 {
		t.Fatalf("expected lightweight latency ms to still be recorded, got %+v", refreshed.LatencyTest)
	}
	if refreshed.LatencyTest.StatusCode == nil || *refreshed.LatencyTest.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status code 401, got %+v", refreshed.LatencyTest.StatusCode)
	}
	if !strings.Contains(refreshed.LatencyTest.ErrorMessage, "Incorrect API key provided") {
		t.Fatalf("expected unauthorized message, got %s", refreshed.LatencyTest.ErrorMessage)
	}
	if refreshed.LatencyTest.ErrorType != "invalid_request_error" {
		t.Fatalf("expected error type invalid_request_error, got %s", refreshed.LatencyTest.ErrorType)
	}
	if refreshed.LatencyTest.ErrorCode != "invalid_api_key" {
		t.Fatalf("expected error code invalid_api_key, got %s", refreshed.LatencyTest.ErrorCode)
	}
	if len(refreshed.LatencyTest.History) != 1 {
		t.Fatalf("expected 1 latency history entry, got %+v", refreshed.LatencyTest.History)
	}
	if refreshed.LatencyTest.History[0].Available {
		t.Fatalf("expected unauthorized history entry to be unavailable, got %+v", refreshed.LatencyTest.History[0])
	}
	if refreshed.LatencyTest.History[0].StatusCode == nil || *refreshed.LatencyTest.History[0].StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected history status code 401, got %+v", refreshed.LatencyTest.History[0])
	}
	if refreshed.LatencyTest.History[0].ErrorType != "invalid_request_error" {
		t.Fatalf("expected history error type invalid_request_error, got %+v", refreshed.LatencyTest.History[0])
	}
	if refreshed.LatencyTest.History[0].ErrorCode != "invalid_api_key" {
		t.Fatalf("expected history error code invalid_api_key, got %+v", refreshed.LatencyTest.History[0])
	}
	if probeRequestMethod != http.MethodGet {
		t.Fatalf("expected homepage probe GET request, got %s", probeRequestMethod)
	}
	if probeAuthorization != "" {
		t.Fatalf("expected homepage probe to skip authorization header, got %q", probeAuthorization)
	}
	if requestMethod != http.MethodPost {
		t.Fatalf("expected POST request, got %s", requestMethod)
	}
	if got := strings.TrimSpace(fmt.Sprintf("%v", requestBody["input"])); got != latencyTestPrompt {
		t.Fatalf("expected input %q, got %+v", latencyTestPrompt, requestBody["input"])
	}
}

func TestRefreshOfficialLatencyTests(t *testing.T) {
	var requestMethod string

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/models" {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		requestMethod = request.Method
		if !strings.HasPrefix(request.Header.Get("Authorization"), "Bearer ") {
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
	if requestMethod != http.MethodGet {
		t.Fatalf("expected GET request, got %s", requestMethod)
	}
}

func TestRefreshAPILatencyTestsBuildErrorStillRecordsHistory(t *testing.T) {
	service := newTestService(t)
	if err := service.saveSettings(AppSettings{CodexHomePath: t.TempDir()}); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}

	authRaw := mustReadSample(t, "api", "auth.json")
	configRaw := `[model_providers.OpenAI]
base_url = "https://api.openai.com/v1"
`

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
	if refreshed.LatencyTest.Status != LatencyTestStatusError {
		t.Fatalf("expected error latency status, got %s", refreshed.LatencyTest.Status)
	}
	if len(refreshed.LatencyTest.History) != 1 {
		t.Fatalf("expected 1 latency history entry, got %+v", refreshed.LatencyTest.History)
	}
	if refreshed.LatencyTest.History[0].Status != LatencyTestStatusError {
		t.Fatalf("expected error history entry, got %+v", refreshed.LatencyTest.History[0])
	}
	if !strings.Contains(refreshed.LatencyTest.History[0].ErrorMessage, "缺少可用 model") {
		t.Fatalf("expected history error message to mention missing model, got %+v", refreshed.LatencyTest.History[0])
	}
}

func TestAutoRefreshAPILatencyTestsPreservesUpdatedAt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/":
			writer.WriteHeader(http.StatusNoContent)
		case "/v1/responses":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"id":"resp_auto_keep_updated_at","status":"completed","output_text":"ok"}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	nextNow := time.Date(2026, 3, 19, 16, 0, 0, 0, time.UTC)
	service := newTestServiceWithConfigDirHTTPAndNow(t, t.TempDir(), server.Client(), func() time.Time {
		current := nextNow
		nextNow = nextNow.Add(time.Minute)
		return current
	})
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
	originalUpdatedAt := snapshot.Meta.UpdatedAt
	if err := service.saveProfileSnapshot(snapshot); err != nil {
		t.Fatalf("saveProfileSnapshot failed: %v", err)
	}

	state, err := service.AutoRefreshAPILatencyTests([]string{snapshot.Meta.ID})
	if err != nil {
		t.Fatalf("AutoRefreshAPILatencyTests returned error: %v", err)
	}

	refreshed := findProfileByID(state.Profiles, snapshot.Meta.ID)
	if refreshed.ID == "" {
		t.Fatal("expected refreshed api profile in state")
	}
	if refreshed.UpdatedAt != originalUpdatedAt {
		t.Fatalf("expected auto latency refresh to preserve updatedAt %s, got %s", originalUpdatedAt, refreshed.UpdatedAt)
	}
	if refreshed.LatencyTest.CheckedAt == "" {
		t.Fatalf("expected auto latency refresh to set checkedAt, got %+v", refreshed.LatencyTest)
	}
	if refreshed.LatencyTest.CheckedAt == originalUpdatedAt {
		t.Fatalf("expected checkedAt to move forward independently from updatedAt, got %+v", refreshed.LatencyTest)
	}
}

func TestAutoRefreshAPILatencyTestsAppendRowsAndManualTestUpdatesLatestRow(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/":
			writer.WriteHeader(http.StatusNoContent)
		case "/v1/responses":
			callCount++
			writer.Header().Set("Content-Type", "application/json")
			switch callCount {
			case 1, 2:
				_, _ = writer.Write([]byte(fmt.Sprintf(`{"id":"resp_%d","status":"completed","output_text":"ok"}`, callCount)))
			default:
				writer.WriteHeader(http.StatusUnauthorized)
				_, _ = writer.Write([]byte(`{"error":{"message":"manual override","type":"invalid_request_error","code":"invalid_api_key"}}`))
			}
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
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

	state, err := service.AutoRefreshAPILatencyTests([]string{snapshot.Meta.ID})
	if err != nil {
		t.Fatalf("first AutoRefreshAPILatencyTests returned error: %v", err)
	}
	refreshed := findProfileByID(state.Profiles, snapshot.Meta.ID)
	if len(refreshed.LatencyTest.History) != 1 {
		t.Fatalf("expected 1 history entry after first auto refresh, got %+v", refreshed.LatencyTest.History)
	}

	state, err = service.AutoRefreshAPILatencyTests([]string{snapshot.Meta.ID})
	if err != nil {
		t.Fatalf("second AutoRefreshAPILatencyTests returned error: %v", err)
	}
	refreshed = findProfileByID(state.Profiles, snapshot.Meta.ID)
	if len(refreshed.LatencyTest.History) != 2 {
		t.Fatalf("expected 2 history entries after second auto refresh, got %+v", refreshed.LatencyTest.History)
	}
	if !refreshed.LatencyTest.History[0].Available || !refreshed.LatencyTest.History[1].Available {
		t.Fatalf("expected both auto history entries to be available, got %+v", refreshed.LatencyTest.History)
	}

	state, err = service.RefreshAPILatencyTests([]string{snapshot.Meta.ID})
	if err != nil {
		t.Fatalf("manual RefreshAPILatencyTests returned error: %v", err)
	}
	refreshed = findProfileByID(state.Profiles, snapshot.Meta.ID)
	if len(refreshed.LatencyTest.History) != 2 {
		t.Fatalf("expected manual refresh to update latest row instead of inserting, got %+v", refreshed.LatencyTest.History)
	}
	if !refreshed.LatencyTest.History[0].Available {
		t.Fatalf("expected first auto history entry to remain available, got %+v", refreshed.LatencyTest.History[0])
	}
	if refreshed.LatencyTest.History[1].Available {
		t.Fatalf("expected latest history entry to be updated to unavailable, got %+v", refreshed.LatencyTest.History[1])
	}
	if refreshed.LatencyTest.History[1].ErrorMessage != "manual override" {
		t.Fatalf("expected latest history entry to be updated by manual refresh, got %+v", refreshed.LatencyTest.History[1])
	}

	metaRaw, err := os.ReadFile(filepath.Join(service.profileDir(snapshot.Meta.ID), "meta.json"))
	if err != nil {
		t.Fatalf("read meta.json failed: %v", err)
	}
	if strings.Contains(string(metaRaw), "\"history\"") {
		t.Fatalf("expected history to be stored in sqlite instead of meta.json, got %s", string(metaRaw))
	}
}

func TestAPILatencyHistoryPersistsAcrossServiceRestart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/":
			writer.WriteHeader(http.StatusNoContent)
		case "/v1/responses":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"id":"resp_persist","status":"completed","output_text":"ok"}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	appConfigDir := t.TempDir()
	codexHome := t.TempDir()
	service := newTestServiceWithConfigDirAndHTTP(t, appConfigDir, server.Client())
	if err := service.saveSettings(AppSettings{CodexHomePath: codexHome}); err != nil {
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

	if _, err := service.AutoRefreshAPILatencyTests([]string{snapshot.Meta.ID}); err != nil {
		t.Fatalf("AutoRefreshAPILatencyTests returned error: %v", err)
	}
	if err := service.Close(); err != nil {
		t.Fatalf("service.Close returned error: %v", err)
	}

	restarted := newTestServiceWithConfigDirAndHTTP(t, appConfigDir, server.Client())
	state, err := restarted.GetAppState()
	if err != nil {
		t.Fatalf("restarted GetAppState returned error: %v", err)
	}

	refreshed := findProfileByID(state.Profiles, snapshot.Meta.ID)
	if len(refreshed.LatencyTest.History) != 1 {
		t.Fatalf("expected sqlite history to persist across restart, got %+v", refreshed.LatencyTest.History)
	}
	if !refreshed.LatencyTest.History[0].Available {
		t.Fatalf("expected persisted history entry to remain available, got %+v", refreshed.LatencyTest.History[0])
	}
}

func TestRefreshAPILatencyTestsRequestFailureKeepsHomepageLatency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/":
			writer.WriteHeader(http.StatusNoContent)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := server.Client()
	baseTransport := client.Transport
	client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path == "/v1/responses" {
			return nil, errors.New("session request failed")
		}
		return baseTransport.RoundTrip(request)
	})

	service := newTestServiceWithHTTP(t, client)
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

	refreshed := findProfileByID(state.Profiles, snapshot.Meta.ID)
	if refreshed.ID == "" {
		t.Fatal("expected refreshed api profile in state")
	}
	if refreshed.LatencyTest.Status != LatencyTestStatusError {
		t.Fatalf("expected error latency status, got %+v", refreshed.LatencyTest)
	}
	if refreshed.LatencyTest.LatencyMs == nil || *refreshed.LatencyTest.LatencyMs <= 0 {
		t.Fatalf("expected homepage latency ms to survive request failure, got %+v", refreshed.LatencyTest)
	}
	if !strings.Contains(refreshed.LatencyTest.ErrorMessage, "session request failed") {
		t.Fatalf("expected session request failure to be preserved, got %+v", refreshed.LatencyTest)
	}
	if len(refreshed.LatencyTest.History) != 1 {
		t.Fatalf("expected error result to be recorded into history, got %+v", refreshed.LatencyTest.History)
	}
	if refreshed.LatencyTest.History[0].LatencyMs == nil || *refreshed.LatencyTest.History[0].LatencyMs <= 0 {
		t.Fatalf("expected history entry to keep homepage latency, got %+v", refreshed.LatencyTest.History[0])
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
	return newTestServiceWithConfigDirAndHTTP(t, appConfigDir, client)
}

func newTestServiceWithConfigDirAndHTTP(t *testing.T, appConfigDir string, client *http.Client) *Service {
	t.Helper()
	return newTestServiceWithConfigDirHTTPAndNow(t, appConfigDir, client, func() time.Time {
		return time.Date(2026, 3, 19, 15, 0, 0, 0, time.UTC)
	})
}

func newTestServiceWithConfigDirHTTPAndNow(
	t *testing.T,
	appConfigDir string,
	client *http.Client,
	now func() time.Time,
) *Service {
	t.Helper()
	service, err := NewService(ServiceOptions{
		AppConfigDir: appConfigDir,
		HTTPClient:   client,
		Now:          now,
	})
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = service.Close()
	})
	return service
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func findProfileByID(profiles []ProfileMeta, id string) ProfileMeta {
	for _, profile := range profiles {
		if profile.ID == id {
			return profile
		}
	}
	return ProfileMeta{}
}

func findFirstProfileByType(profiles []ProfileMeta, profileType ProfileType) ProfileMeta {
	for _, profile := range profiles {
		if profile.Type == profileType {
			return profile
		}
	}
	return ProfileMeta{}
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
