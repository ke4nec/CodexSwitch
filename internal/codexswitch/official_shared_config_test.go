package codexswitch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOfficialProfilesShareSingleConfigFile(t *testing.T) {
	service := newTestService(t)

	sharedConfigRaw := `model = "gpt-5.4"
model_reasoning_effort = "medium"
[windows]
sandbox = "elevated"
`

	firstSnapshot, err := buildProfileSnapshot(
		mustReadSample(t, "codex", "auth.json"),
		sharedConfigRaw,
		profileSourceImportedCurrent,
		service.now(),
	)
	if err != nil {
		t.Fatalf("buildProfileSnapshot for first official profile failed: %v", err)
	}
	if err := service.saveProfileSnapshot(firstSnapshot); err != nil {
		t.Fatalf("saveProfileSnapshot for first official profile failed: %v", err)
	}

	sharedStoredRaw, err := readTextFile(service.sharedOfficialConfigPath())
	if err != nil {
		t.Fatalf("read shared official config failed: %v", err)
	}
	if strings.TrimSpace(sharedStoredRaw) != strings.TrimSpace(sharedConfigRaw) {
		t.Fatalf("expected shared official config to be initialized from first profile, got %s", sharedStoredRaw)
	}
	if _, err := os.Stat(filepath.Join(service.profileDir(firstSnapshot.Meta.ID), "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("expected first official profile to stop storing per-profile config.toml, got err=%v", err)
	}

	secondIDToken, secondAccessToken := buildTestOfficialTokens(
		t,
		"account-two",
		"user-two",
		"two@example.com",
		"pro",
		"two",
	)
	secondSnapshot, err := buildProfileSnapshot(
		buildTestOfficialAuthRaw(t, secondIDToken, secondAccessToken, "refresh-two", "account-two"),
		`model = "gpt-5.4-mini"
model_reasoning_effort = "low"
[windows]
sandbox = "workspace-write"
`,
		profileSourceImportedFileStandard,
		service.now(),
	)
	if err != nil {
		t.Fatalf("buildProfileSnapshot for second official profile failed: %v", err)
	}
	if err := service.saveProfileSnapshot(secondSnapshot); err != nil {
		t.Fatalf("saveProfileSnapshot for second official profile failed: %v", err)
	}

	sharedStoredRaw, err = readTextFile(service.sharedOfficialConfigPath())
	if err != nil {
		t.Fatalf("read shared official config after second save failed: %v", err)
	}
	if strings.TrimSpace(sharedStoredRaw) != strings.TrimSpace(sharedConfigRaw) {
		t.Fatalf("expected later official saves not to overwrite shared config, got %s", sharedStoredRaw)
	}

	storedSecond, err := service.loadProfile(secondSnapshot.Meta.ID)
	if err != nil {
		t.Fatalf("loadProfile for second official profile failed: %v", err)
	}
	if strings.TrimSpace(storedSecond.ConfigRaw) != strings.TrimSpace(sharedConfigRaw) {
		t.Fatalf("expected second official profile to reuse shared config, got %s", storedSecond.ConfigRaw)
	}
	if storedSecond.Meta.Model != "gpt-5.4" {
		t.Fatalf("expected second official profile meta to reflect shared config model, got %s", storedSecond.Meta.Model)
	}
	if _, err := os.Stat(filepath.Join(service.profileDir(secondSnapshot.Meta.ID), "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("expected second official profile to stop storing per-profile config.toml, got err=%v", err)
	}
}

func TestSwitchProfileWritesSharedOfficialConfigBackToCodexHome(t *testing.T) {
	service := newTestService(t)
	codexHome := t.TempDir()
	if err := service.saveSettings(AppSettings{CodexHomePath: codexHome}); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}

	sharedConfigRaw := `model = "gpt-5.4"
model_reasoning_effort = "xhigh"
[windows]
sandbox = "elevated"
`
	officialSnapshot, err := buildProfileSnapshot(
		mustReadSample(t, "codex", "auth.json"),
		sharedConfigRaw,
		profileSourceImportedCurrent,
		service.now(),
	)
	if err != nil {
		t.Fatalf("buildProfileSnapshot for official profile failed: %v", err)
	}
	if err := service.saveProfileSnapshot(officialSnapshot); err != nil {
		t.Fatalf("saveProfileSnapshot for official profile failed: %v", err)
	}

	state, err := service.CreateAPIProfile(APIProfileInput{
		BaseURL:              "https://example.com/v1",
		Model:                "gpt-5.4",
		ModelReasoningEffort: "xhigh",
		ModelContextWindow:   "128000",
		APIKey:               "sk-switch-shared-config",
	})
	if err != nil {
		t.Fatalf("CreateAPIProfile returned error: %v", err)
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

	if _, err := service.SwitchProfile(apiProfile.ID); err != nil {
		t.Fatalf("SwitchProfile to api returned error: %v", err)
	}
	if _, err := service.SwitchProfile(officialSnapshot.Meta.ID); err != nil {
		t.Fatalf("SwitchProfile back to official returned error: %v", err)
	}

	configRaw, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		t.Fatalf("read switched config.toml failed: %v", err)
	}
	if strings.TrimSpace(string(configRaw)) != strings.TrimSpace(sharedConfigRaw) {
		t.Fatalf("expected switched official config.toml to use shared config, got %s", string(configRaw))
	}
}

func TestImportOfficialProfileFileUsesExistingSharedConfig(t *testing.T) {
	service := newTestService(t)
	if err := service.saveSettings(AppSettings{CodexHomePath: t.TempDir()}); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}

	sharedConfigRaw := `model = "gpt-5.4"
model_reasoning_effort = "medium"
[windows]
sandbox = "elevated"
`
	if err := safeWriteText(service.sharedOfficialConfigPath(), sharedConfigRaw); err != nil {
		t.Fatalf("write shared official config failed: %v", err)
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
	if strings.TrimSpace(stored.ConfigRaw) != strings.TrimSpace(sharedConfigRaw) {
		t.Fatalf("expected imported official profile to use existing shared config, got %s", stored.ConfigRaw)
	}
	if stored.Meta.Model != "gpt-5.4" {
		t.Fatalf("expected imported official profile meta model from shared config, got %s", stored.Meta.Model)
	}
	if stored.Meta.ModelReasoningEffort != "medium" {
		t.Fatalf("expected imported official profile reasoning effort from shared config, got %s", stored.Meta.ModelReasoningEffort)
	}
}
