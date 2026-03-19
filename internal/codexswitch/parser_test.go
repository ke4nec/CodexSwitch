package codexswitch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalizeStandardOfficialAuthAddsDefaults(t *testing.T) {
	authRaw := mustReadSample(t, "codex", "auth.json")

	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(authRaw), &root); err != nil {
		t.Fatalf("unmarshal official sample failed: %v", err)
	}
	delete(root, "auth_mode")
	delete(root, "OPENAI_API_KEY")

	trimmedRaw, err := marshalCanonicalJSON(root)
	if err != nil {
		t.Fatalf("marshal trimmed official auth failed: %v", err)
	}

	normalizedRaw, inputFormat, err := normalizeAuthJSON(trimmedRaw)
	if err != nil {
		t.Fatalf("normalizeAuthJSON returned error: %v", err)
	}
	if inputFormat != officialAuthInputFormatStandard {
		t.Fatalf("expected standard format, got %s", inputFormat)
	}

	var normalized map[string]json.RawMessage
	if err := json.Unmarshal([]byte(normalizedRaw), &normalized); err != nil {
		t.Fatalf("unmarshal normalized auth failed: %v", err)
	}

	authMode, ok := optionalJSONStringField(normalized, "auth_mode")
	if !ok || authMode != officialAuthModeTemplate {
		t.Fatalf("expected auth_mode=%q, got %q", officialAuthModeTemplate, authMode)
	}
	if rawValue, ok := normalized["OPENAI_API_KEY"]; !ok || strings.TrimSpace(string(rawValue)) != "null" {
		t.Fatalf("expected OPENAI_API_KEY=null, got %s", string(rawValue))
	}
}

func TestNormalizeCLIOfficialAuthSample(t *testing.T) {
	authRaw := mustReadSample(t, "codex", "cli-auth.json")

	normalizedRaw, inputFormat, err := normalizeAuthJSON(authRaw)
	if err != nil {
		t.Fatalf("normalizeAuthJSON returned error: %v", err)
	}
	if inputFormat != officialAuthInputFormatCLI {
		t.Fatalf("expected cli format, got %s", inputFormat)
	}
	if strings.Contains(normalizedRaw, "\"access_token\":") && !strings.Contains(normalizedRaw, "\"tokens\":") {
		t.Fatalf("expected CLI auth to be wrapped into tokens: %s", normalizedRaw)
	}

	snapshot, err := buildProfileSnapshot(normalizedRaw, mustReadSample(t, "codex", "config.toml"), profileSourceImportedFileCLI, time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("buildProfileSnapshot returned error: %v", err)
	}
	if snapshot.Meta.Type != ProfileTypeOfficial {
		t.Fatalf("expected official profile, got %s", snapshot.Meta.Type)
	}
	if snapshot.Meta.Email != "codex_001@proton.me" {
		t.Fatalf("unexpected email: %s", snapshot.Meta.Email)
	}
}

func TestNormalizeCLIOfficialAuthRejectsMissingSensitiveField(t *testing.T) {
	authRaw := mustReadSample(t, "codex", "cli-auth.json")

	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(authRaw), &root); err != nil {
		t.Fatalf("unmarshal CLI sample failed: %v", err)
	}
	delete(root, "refresh_token")

	trimmedRaw, err := marshalCanonicalJSON(root)
	if err != nil {
		t.Fatalf("marshal broken CLI auth failed: %v", err)
	}

	_, _, err = normalizeAuthJSON(trimmedRaw)
	if err == nil {
		t.Fatal("expected normalizeAuthJSON to fail when refresh_token is missing")
	}
	if !strings.Contains(err.Error(), "refresh_token") {
		t.Fatalf("expected refresh_token error, got %v", err)
	}
}

func TestBuildProfileSnapshotOfficialSample(t *testing.T) {
	authRaw := mustReadSample(t, "codex", "auth.json")
	configRaw := mustReadSample(t, "codex", "config.toml")

	snapshot, err := buildProfileSnapshot(authRaw, configRaw, "imported_current", time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("buildProfileSnapshot returned error: %v", err)
	}

	if snapshot.Meta.Type != ProfileTypeOfficial {
		t.Fatalf("expected official profile, got %s", snapshot.Meta.Type)
	}
	if snapshot.Meta.Email != "minyox2025@gmail.com" {
		t.Fatalf("unexpected email: %s", snapshot.Meta.Email)
	}
	if snapshot.Meta.PlanType != "team" {
		t.Fatalf("unexpected plan type: %s", snapshot.Meta.PlanType)
	}
	if snapshot.Meta.ChatGPTAccountID == "" {
		t.Fatal("expected chatgpt account id")
	}
	if snapshot.Meta.ClientID == "" {
		t.Fatal("expected client id")
	}
	if snapshot.Meta.Model != "gpt-5.4" {
		t.Fatalf("unexpected model: %s", snapshot.Meta.Model)
	}
}

func TestBuildProfileSnapshotAPISample(t *testing.T) {
	authRaw := mustReadSample(t, "api", "auth.json")
	configRaw := mustReadSample(t, "api", "config.toml")

	snapshot, err := buildProfileSnapshot(authRaw, configRaw, "imported_current", time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("buildProfileSnapshot returned error: %v", err)
	}

	if snapshot.Meta.Type != ProfileTypeAPI {
		t.Fatalf("expected api profile, got %s", snapshot.Meta.Type)
	}
	if snapshot.Meta.BaseURL != "https://hivewa.store/v1" {
		t.Fatalf("unexpected base url: %s", snapshot.Meta.BaseURL)
	}
	if snapshot.Meta.ModelReasoningEffort != "xhigh" {
		t.Fatalf("unexpected reasoning effort: %s", snapshot.Meta.ModelReasoningEffort)
	}
	if !strings.Contains(snapshot.Meta.MaskedAPIKey, "*") {
		t.Fatalf("expected masked api key, got %s", snapshot.Meta.MaskedAPIKey)
	}
}

func TestBuildAPIProfileFromTemplate(t *testing.T) {
	service := newTestService(t)

	snapshot, err := service.buildAPIProfile(APIProfileInput{
		BaseURL:              "https://api.openai.com/v1",
		Model:                "gpt-5.4",
		ModelReasoningEffort: "xhigh",
		APIKey:               "sk-test-1234567890",
	})
	if err != nil {
		t.Fatalf("buildAPIProfile returned error: %v", err)
	}

	if snapshot.Meta.Type != ProfileTypeAPI {
		t.Fatalf("expected api profile, got %s", snapshot.Meta.Type)
	}
	if !strings.Contains(snapshot.ConfigRaw, "base_url = \"https://api.openai.com/v1\"") {
		t.Fatalf("config template not rendered correctly: %s", snapshot.ConfigRaw)
	}
	if !strings.Contains(snapshot.AuthRaw, "sk-test-1234567890") {
		t.Fatalf("auth template not rendered correctly: %s", snapshot.AuthRaw)
	}
}

func mustReadSample(t *testing.T, parts ...string) string {
	t.Helper()

	allParts := append([]string{"..", "..", "conf"}, parts...)
	path := filepath.Join(allParts...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read sample %s failed: %v", path, err)
	}
	return string(data)
}
