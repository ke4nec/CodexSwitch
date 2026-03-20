package codexswitch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type rateLimitPayload struct {
	PlanType             string                    `json:"plan_type"`
	RateLimit            *rateLimitDetails         `json:"rate_limit"`
	AdditionalRateLimits []additionalRateLimitInfo `json:"additional_rate_limits"`
}

type rateLimitDetails struct {
	PrimaryWindow   *rateLimitWindowPayload `json:"primary_window"`
	SecondaryWindow *rateLimitWindowPayload `json:"secondary_window"`
}

type additionalRateLimitInfo struct {
	LimitName      string            `json:"limit_name"`
	MeteredFeature string            `json:"metered_feature"`
	RateLimit      *rateLimitDetails `json:"rate_limit"`
}

type rateLimitWindowPayload struct {
	UsedPercent        int   `json:"used_percent"`
	LimitWindowSeconds int64 `json:"limit_window_seconds"`
	ResetAt            int64 `json:"reset_at"`
}

type rateLimitWindowCandidate struct {
	Source string
	Window *rateLimitWindowPayload
}

func (s *Service) refreshProfileRateLimit(profile storedProfile) (storedProfile, error) {
	if profile.Meta.Type != ProfileTypeOfficial {
		return profile, nil
	}

	refreshedProfile, err := s.refreshOfficialProfileAuth(profile)
	if err != nil {
		s.logger.Warn("refresh official token before rate limit fetch failed", "id", profile.Meta.ID, "error", err)
	} else {
		profile = refreshedProfile
	}

	var auth authFile
	if err := json.Unmarshal([]byte(profile.AuthRaw), &auth); err != nil {
		return profile, fmt.Errorf("读取额度前解析 auth.json 失败: %w", err)
	}
	if auth.Tokens == nil || strings.TrimSpace(auth.Tokens.AccessToken) == "" {
		return profile, fmt.Errorf("当前官方配置缺少可用 access_token")
	}

	config := parseConfigTOML(profile.ConfigRaw)
	usageURL := officialUsageURL(config.BaseURL)

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

	primary, secondary := resolveRateLimitWindows(payload)
	profile.Meta.RateLimits = RateLimitState{
		Primary:   primary,
		Secondary: secondary,
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

func officialUsageURL(baseURL string) string {
	baseURL = normalizeOfficialBaseURL(baseURL)
	if strings.Contains(baseURL, "/backend-api") {
		return baseURL + "/wham/usage"
	}
	return baseURL + "/api/codex/usage"
}

func normalizeOfficialBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return "https://chatgpt.com/backend-api"
	}

	if (strings.HasPrefix(baseURL, "https://chatgpt.com") || strings.HasPrefix(baseURL, "https://chat.openai.com")) &&
		!strings.Contains(baseURL, "/backend-api") {
		return baseURL + "/backend-api"
	}
	return baseURL
}

func resolveRateLimitWindows(payload rateLimitPayload) (*RateLimitWindow, *RateLimitWindow) {
	candidates := collectRateLimitWindowCandidates(payload)
	primary := findRateLimitWindowByDuration(candidates, 300)
	secondary := findRateLimitWindowByDuration(candidates, 10080)

	if primary == nil {
		primary = fallbackRateLimitWindow(candidates, "primary", 10080, secondary != nil)
	}
	if secondary == nil {
		secondary = fallbackRateLimitWindow(candidates, "secondary", 300, primary != nil)
	}

	return primary, secondary
}

func collectRateLimitWindowCandidates(payload rateLimitPayload) []rateLimitWindowCandidate {
	candidates := make([]rateLimitWindowCandidate, 0, 2+len(payload.AdditionalRateLimits)*2)
	candidates = appendRateLimitWindowCandidates(candidates, payload.RateLimit)
	for _, additional := range payload.AdditionalRateLimits {
		candidates = appendRateLimitWindowCandidates(candidates, additional.RateLimit)
	}
	return candidates
}

func appendRateLimitWindowCandidates(candidates []rateLimitWindowCandidate, details *rateLimitDetails) []rateLimitWindowCandidate {
	if details == nil {
		return candidates
	}
	if details.PrimaryWindow != nil {
		candidates = append(candidates, rateLimitWindowCandidate{
			Source: "primary",
			Window: details.PrimaryWindow,
		})
	}
	if details.SecondaryWindow != nil {
		candidates = append(candidates, rateLimitWindowCandidate{
			Source: "secondary",
			Window: details.SecondaryWindow,
		})
	}
	return candidates
}

func findRateLimitWindowByDuration(candidates []rateLimitWindowCandidate, wantedMinutes int64) *RateLimitWindow {
	for _, candidate := range candidates {
		if windowDurationMinutes(candidate.Window) == wantedMinutes {
			return convertRateLimitPayloadWindow(candidate.Window)
		}
	}
	return nil
}

func fallbackRateLimitWindow(
	candidates []rateLimitWindowCandidate,
	fallbackSource string,
	excludedMinutes int64,
	excludeMatchedDuration bool,
) *RateLimitWindow {
	for _, candidate := range candidates {
		if candidate.Source != fallbackSource {
			continue
		}
		if excludeMatchedDuration && windowDurationMinutes(candidate.Window) == excludedMinutes {
			continue
		}
		return convertRateLimitPayloadWindow(candidate.Window)
	}
	return nil
}

func convertRateLimitPayloadWindow(window *rateLimitWindowPayload) *RateLimitWindow {
	if window == nil {
		return nil
	}

	duration := windowDurationMinutes(window)
	resetsAt := window.ResetAt
	return &RateLimitWindow{
		UsedPercent:        window.UsedPercent,
		WindowDurationMins: optionalInt64(duration),
		ResetsAt:           optionalInt64(resetsAt),
	}
}

func windowDurationMinutes(window *rateLimitWindowPayload) int64 {
	if window == nil || window.LimitWindowSeconds <= 0 {
		return 0
	}
	return (window.LimitWindowSeconds + 59) / 60
}

func optionalInt64(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	return &value
}
