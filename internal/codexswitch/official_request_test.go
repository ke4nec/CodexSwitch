package codexswitch

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRateLimitWindowsMapsWeeklyOnlyPrimaryWindowToSecondary(t *testing.T) {
	primary, secondary := resolveRateLimitWindows(rateLimitPayload{
		PlanType: "free",
		RateLimit: &rateLimitDetails{
			PrimaryWindow: &rateLimitWindowPayload{
				UsedPercent:        17,
				LimitWindowSeconds: 7 * 24 * 60 * 60,
				ResetAt:            1731547200,
			},
		},
	})

	if primary != nil {
		t.Fatalf("expected 5h window to be unavailable, got %+v", primary)
	}
	if secondary == nil {
		t.Fatal("expected weekly window to be resolved")
	}
	if secondary.UsedPercent != 17 {
		t.Fatalf("expected weekly used_percent=17, got %+v", secondary)
	}
	if secondary.WindowDurationMins == nil || *secondary.WindowDurationMins != 10080 {
		t.Fatalf("expected weekly window_minutes=10080, got %+v", secondary.WindowDurationMins)
	}
}

func TestRefreshRateLimitsRefreshesOfficialTokenAndSyncsActiveAuth(t *testing.T) {
	const (
		accountID = "account-123"
		userID    = "user-123"
		email     = "free@example.com"
		planType  = "free"
	)

	oldIDToken, oldAccessToken := buildTestOfficialTokens(t, accountID, userID, email, planType, "old")
	newIDToken, newAccessToken := buildTestOfficialTokens(t, accountID, userID, email, planType, "new")

	var refreshRequestBody string
	var usageAuthorization string

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
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{
			  "plan_type": "free",
			  "rate_limit": {
			    "primary_window": {
			      "used_percent": 9,
			      "limit_window_seconds": 604800,
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

	authRaw := buildTestOfficialAuthRaw(t, oldIDToken, oldAccessToken, "refresh-old", accountID)
	configRaw := fmt.Sprintf(`model = "gpt-5.4"
model_reasoning_effort = "xhigh"
[model_providers.OpenAI]
base_url = %q
`, server.URL+"/backend-api")

	snapshot, err := buildProfileSnapshot(authRaw, configRaw, profileSourceImportedCurrent, service.now())
	if err != nil {
		t.Fatalf("buildProfileSnapshot returned error: %v", err)
	}
	if err := service.saveProfileSnapshot(snapshot); err != nil {
		t.Fatalf("saveProfileSnapshot failed: %v", err)
	}
	if _, err := service.SwitchProfile(snapshot.Meta.ID); err != nil {
		t.Fatalf("SwitchProfile returned error: %v", err)
	}

	state, err := service.RefreshRateLimits([]string{snapshot.Meta.ID})
	if err != nil {
		t.Fatalf("RefreshRateLimits returned error: %v", err)
	}

	if !strings.Contains(refreshRequestBody, `"refresh_token":"refresh-old"`) {
		t.Fatalf("expected refresh request to include old refresh token, got %s", refreshRequestBody)
	}
	if usageAuthorization != "Bearer "+newAccessToken {
		t.Fatalf("expected refreshed access token to be used, got %q", usageAuthorization)
	}

	stored, err := service.loadProfile(snapshot.Meta.ID)
	if err != nil {
		t.Fatalf("loadProfile failed: %v", err)
	}
	if !strings.Contains(stored.AuthRaw, newAccessToken) {
		t.Fatalf("expected stored auth to contain refreshed access token")
	}
	if !strings.Contains(stored.AuthRaw, `"refresh_token": "refresh-new"`) {
		t.Fatalf("expected stored auth to contain refreshed refresh token: %s", stored.AuthRaw)
	}

	currentAuthRaw, err := os.ReadFile(filepath.Join(codexHome, "auth.json"))
	if err != nil {
		t.Fatalf("read active auth.json failed: %v", err)
	}
	if !strings.Contains(string(currentAuthRaw), newAccessToken) {
		t.Fatalf("expected active auth.json to contain refreshed access token")
	}

	var refreshed ProfileMeta
	for _, profile := range state.Profiles {
		if profile.ID == snapshot.Meta.ID {
			refreshed = profile
			break
		}
	}
	if refreshed.ID == "" {
		t.Fatal("expected refreshed profile in app state")
	}
	if refreshed.RateLimits.Primary != nil {
		t.Fatalf("expected 5h window to remain empty for free weekly-only response, got %+v", refreshed.RateLimits.Primary)
	}
	if refreshed.RateLimits.Secondary == nil || refreshed.RateLimits.Secondary.UsedPercent != 9 {
		t.Fatalf("expected weekly window to be populated, got %+v", refreshed.RateLimits.Secondary)
	}
}

func buildTestOfficialTokens(
	t *testing.T,
	accountID, userID, email, planType, tokenLabel string,
) (string, string) {
	t.Helper()

	idToken := buildTestJWT(t, map[string]any{
		"email":          email,
		"email_verified": true,
		"jti":            "id-" + tokenLabel,
		"https://api.openai.com/profile": map[string]any{
			"email":          email,
			"email_verified": true,
		},
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_plan_type":  planType,
			"chatgpt_user_id":    userID,
			"chatgpt_account_id": accountID,
		},
	})

	accessToken := buildTestJWT(t, map[string]any{
		"email":          email,
		"email_verified": true,
		"client_id":      officialRefreshClientID,
		"jti":            "access-" + tokenLabel,
		"https://api.openai.com/profile": map[string]any{
			"email":          email,
			"email_verified": true,
		},
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_plan_type":  planType,
			"chatgpt_user_id":    userID,
			"chatgpt_account_id": accountID,
		},
	})

	return idToken, accessToken
}

func buildTestOfficialAuthRaw(t *testing.T, idToken, accessToken, refreshToken, accountID string) string {
	t.Helper()

	authMode := officialAuthModeTemplate
	lastRefresh := "2026-03-19T00:00:00Z"
	authRaw, err := marshalCanonicalJSON(authFile{
		AuthMode:     &authMode,
		OpenAIAPIKey: nil,
		Tokens: &tokenData{
			IDToken:      idToken,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			AccountID:    accountID,
		},
		LastRefresh: &lastRefresh,
	})
	if err != nil {
		t.Fatalf("marshalCanonicalJSON failed: %v", err)
	}
	return authRaw
}

func buildTestJWT(t *testing.T, payload map[string]any) string {
	t.Helper()

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadRaw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal jwt payload failed: %v", err)
	}
	body := base64.RawURLEncoding.EncodeToString(payloadRaw)
	return header + "." + body + ".sig"
}
