package codexswitch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type rateLimitPayload struct {
	PlanType  string            `json:"plan_type"`
	RateLimit *rateLimitDetails `json:"rate_limit"`
}

type rateLimitDetails struct {
	PrimaryWindow   *rateLimitWindowPayload `json:"primary_window"`
	SecondaryWindow *rateLimitWindowPayload `json:"secondary_window"`
}

type rateLimitWindowPayload struct {
	UsedPercent        int   `json:"used_percent"`
	LimitWindowSeconds int64 `json:"limit_window_seconds"`
	ResetAt            int64 `json:"reset_at"`
}

func (s *Service) refreshProfileRateLimit(profile storedProfile) (storedProfile, error) {
	if profile.Meta.Type != ProfileTypeOfficial {
		return profile, nil
	}

	var auth authFile
	if err := json.Unmarshal([]byte(profile.AuthRaw), &auth); err != nil {
		return profile, fmt.Errorf("读取额度前解析 auth.json 失败: %w", err)
	}
	if auth.Tokens == nil || strings.TrimSpace(auth.Tokens.AccessToken) == "" {
		return profile, fmt.Errorf("当前官方配置缺少可用 access_token")
	}

	config := parseConfigTOML(profile.ConfigRaw)
	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		baseURL = "https://chatgpt.com/backend-api"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	var usageURL string
	if strings.Contains(baseURL, "/backend-api") {
		usageURL = baseURL + "/wham/usage"
	} else {
		usageURL = baseURL + "/api/codex/usage"
	}

	req, err := http.NewRequest(http.MethodGet, usageURL, nil)
	if err != nil {
		return profile, err
	}
	req.Header.Set("Authorization", "Bearer "+auth.Tokens.AccessToken)
	req.Header.Set("User-Agent", "CodexSwitch/0.1")
	if accountID := trimmedFirst(auth.Tokens.AccountID, profile.Meta.ChatGPTAccountID); accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return profile, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return profile, fmt.Errorf("额度接口返回异常状态码: %s", resp.Status)
	}

	var payload rateLimitPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return profile, fmt.Errorf("额度接口返回解析失败: %w", err)
	}

	profile.Meta.RateLimits = RateLimitState{
		Primary:   convertRateLimitWindow(payload.RateLimit, true),
		Secondary: convertRateLimitWindow(payload.RateLimit, false),
		Status:    RateLimitStatusSuccess,
	}
	profile.Meta.RateLimits.ErrorMessage = ""
	if strings.TrimSpace(payload.PlanType) != "" {
		profile.Meta.PlanType = strings.ToLower(strings.TrimSpace(payload.PlanType))
	}
	profile.Meta.LastRateLimitFetchAt = s.now().UTC().Format(time.RFC3339)
	profile.Meta.UpdatedAt = s.now().UTC().Format(time.RFC3339)

	return profile, nil
}

func convertRateLimitWindow(details *rateLimitDetails, primary bool) *RateLimitWindow {
	if details == nil {
		return nil
	}
	var window *rateLimitWindowPayload
	if primary {
		window = details.PrimaryWindow
	} else {
		window = details.SecondaryWindow
	}
	if window == nil {
		return nil
	}

	duration := int64(0)
	if window.LimitWindowSeconds > 0 {
		duration = (window.LimitWindowSeconds + 59) / 60
	}
	resetsAt := window.ResetAt
	return &RateLimitWindow{
		UsedPercent:        window.UsedPercent,
		WindowDurationMins: optionalInt64(duration),
		ResetsAt:           optionalInt64(resetsAt),
	}
}

func optionalInt64(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	return &value
}
