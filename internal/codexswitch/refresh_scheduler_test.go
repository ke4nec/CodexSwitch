package codexswitch

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRefreshActiveOfficialProfileRefreshesCurrentOfficialProfile(t *testing.T) {
	const (
		accountID = "account-123"
		userID    = "user-123"
		email     = "pro@example.com"
		planType  = "pro"
	)

	oldIDToken, oldAccessToken := buildTestOfficialTokens(t, accountID, userID, email, planType, "old")
	newIDToken, newAccessToken := buildTestOfficialTokens(t, accountID, userID, email, planType, "new")

	var refreshRequestBody string
	var usageAuthorization string
	var usageAccountID string

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/oauth/token":
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("read refresh request body failed: %v", err)
			}
			refreshRequestBody = string(body)

			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(fmt.Sprintf(`{
			  "id_token": %q,
			  "access_token": %q,
			  "refresh_token": "refresh-new"
			}`, newIDToken, newAccessToken)))
		case "/backend-api/wham/usage":
			usageAuthorization = request.Header.Get("Authorization")
			usageAccountID = request.Header.Get("ChatGPT-Account-Id")

			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{
			  "plan_type": "pro",
			  "rate_limit": {
			    "primary_window": {
			      "used_percent": 18,
			      "limit_window_seconds": 18000,
			      "reset_at": 1731547200
			    }
			  }
			}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Setenv(officialRefreshTokenURLOverrideEnv, server.URL+"/oauth/token")

	service := newTestServiceWithHTTP(t, server.Client())
	codexHome := t.TempDir()
	if err := service.saveSettings(AppSettings{CodexHomePath: codexHome}); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}

	authRaw := buildTestOfficialAuthRaw(t, oldIDToken, oldAccessToken, "refresh-old", "")
	configRaw := fmt.Sprintf(`model = "gpt-5.4"
model_reasoning_effort = "xhigh"
[model_providers.OpenAI]
base_url = %q
`, server.URL+"/backend-api")

	if err := os.WriteFile(filepath.Join(codexHome, "auth.json"), []byte(authRaw), 0o600); err != nil {
		t.Fatalf("write auth.json failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte(configRaw), 0o600); err != nil {
		t.Fatalf("write config.toml failed: %v", err)
	}

	profileID, refreshed, err := service.RefreshActiveOfficialProfile()
	if err != nil {
		t.Fatalf("RefreshActiveOfficialProfile returned error: %v", err)
	}
	if !refreshed {
		t.Fatal("expected active official profile to be refreshed")
	}
	if profileID == "" {
		t.Fatal("expected active profile id to be returned")
	}

	if !strings.Contains(refreshRequestBody, `"refresh_token":"refresh-old"`) {
		t.Fatalf("expected refresh request to include old refresh token, got %s", refreshRequestBody)
	}
	if usageAuthorization != "Bearer "+newAccessToken {
		t.Fatalf("expected refreshed access token to be used, got %q", usageAuthorization)
	}
	if usageAccountID != accountID {
		t.Fatalf("expected usage request account id %q, got %q", accountID, usageAccountID)
	}

	profiles, err := service.loadAllProfiles()
	if err != nil {
		t.Fatalf("loadAllProfiles failed: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 managed profile, got %d", len(profiles))
	}
	if profiles[0].ID != profileID {
		t.Fatalf("expected managed profile id %q, got %q", profileID, profiles[0].ID)
	}
	if !profiles[0].IsActive {
		t.Fatal("expected refreshed profile to be marked active")
	}

	stored, err := service.loadProfile(profileID)
	if err != nil {
		t.Fatalf("loadProfile failed: %v", err)
	}
	if !strings.Contains(stored.AuthRaw, newAccessToken) {
		t.Fatalf("expected stored auth to contain refreshed access token")
	}
	if !strings.Contains(stored.AuthRaw, `"refresh_token": "refresh-new"`) {
		t.Fatalf("expected stored auth to contain refreshed refresh token: %s", stored.AuthRaw)
	}
	if !strings.Contains(stored.AuthRaw, `"account_id": "account-123"`) {
		t.Fatalf("expected stored auth to contain refreshed account id: %s", stored.AuthRaw)
	}
	if stored.Meta.RateLimits.Primary == nil || stored.Meta.RateLimits.Primary.UsedPercent != 18 {
		t.Fatalf("expected refreshed primary rate limit, got %+v", stored.Meta.RateLimits)
	}
	if stored.Meta.LastRateLimitFetchAt == "" {
		t.Fatal("expected refreshed profile to record last rate limit fetch time")
	}

	currentAuthRaw, err := os.ReadFile(filepath.Join(codexHome, "auth.json"))
	if err != nil {
		t.Fatalf("read active auth.json failed: %v", err)
	}
	if !strings.Contains(string(currentAuthRaw), newAccessToken) {
		t.Fatalf("expected active auth.json to contain refreshed access token")
	}
	if !strings.Contains(string(currentAuthRaw), `"account_id": "account-123"`) {
		t.Fatalf("expected active auth.json to contain refreshed account id: %s", string(currentAuthRaw))
	}
}

func TestRefreshActiveOfficialProfileSkipsNonOfficialCurrentProfile(t *testing.T) {
	service := newTestService(t)
	codexHome := prepareCodexHome(t, "api")
	if err := service.saveSettings(AppSettings{CodexHomePath: codexHome}); err != nil {
		t.Fatalf("saveSettings failed: %v", err)
	}

	profileID, refreshed, err := service.RefreshActiveOfficialProfile()
	if err != nil {
		t.Fatalf("RefreshActiveOfficialProfile returned error: %v", err)
	}
	if refreshed {
		t.Fatal("expected api current profile to skip active official refresh")
	}
	if profileID != "" {
		t.Fatalf("expected empty profile id when skipping refresh, got %q", profileID)
	}

	profiles, err := service.loadAllProfiles()
	if err != nil {
		t.Fatalf("loadAllProfiles failed: %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("expected no managed profiles to be created, got %d", len(profiles))
	}
}
