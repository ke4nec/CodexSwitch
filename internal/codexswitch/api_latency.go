package codexswitch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type apiProbeErrorPayload struct {
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

func (s *Service) refreshProfileLatencyTest(profile storedProfile) (storedProfile, error) {
	if profile.Meta.Type == ProfileTypeOfficial {
		refreshedProfile, err := s.refreshOfficialProfileAuth(profile)
		if err != nil {
			s.logger.Warn("refresh official token before latency test failed", "id", profile.Meta.ID, "error", err)
		} else {
			profile = refreshedProfile
		}
	}

	req, err := buildLatencyTestRequest(profile)
	if err != nil {
		return profile, err
	}
	if req == nil {
		return profile, nil
	}

	startedAt := time.Now()
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return profile, err
	}
	defer resp.Body.Close()

	latencyMs := time.Since(startedAt).Milliseconds()
	if latencyMs <= 0 {
		latencyMs = 1
	}

	profile.Meta.LatencyTest = LatencyTestState{
		Status:     LatencyTestStatusSuccess,
		Available:  resp.StatusCode >= 200 && resp.StatusCode < 300,
		LatencyMs:  optionalInt64(latencyMs),
		StatusCode: optionalInt(resp.StatusCode),
		CheckedAt:  s.now().UTC().Format(time.RFC3339),
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		profile.Meta.LatencyTest.ErrorMessage = formatAPIProbeFailure(resp)
	} else {
		_, _ = io.Copy(io.Discard, resp.Body)
	}

	profile.Meta.UpdatedAt = s.now().UTC().Format(time.RFC3339)
	return profile, nil
}

func buildLatencyTestRequest(profile storedProfile) (*http.Request, error) {
	switch profile.Meta.Type {
	case ProfileTypeAPI:
		return buildAPILatencyTestRequest(profile)
	case ProfileTypeOfficial:
		return buildOfficialLatencyTestRequest(profile)
	default:
		return nil, nil
	}
}

func buildAPILatencyTestRequest(profile storedProfile) (*http.Request, error) {
	var auth authFile
	if err := json.Unmarshal([]byte(profile.AuthRaw), &auth); err != nil {
		return nil, fmt.Errorf("读取延迟测试前解析 auth.json 失败: %w", err)
	}

	apiKey := ""
	if auth.OpenAIAPIKey != nil {
		apiKey = strings.TrimSpace(*auth.OpenAIAPIKey)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("当前 API 配置缺少可用 OPENAI_API_KEY")
	}

	config := parseConfigTOML(profile.ConfigRaw)
	probeURL, err := apiModelsURL(config.BaseURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, probeURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "CodexSwitch/0.1")
	return req, nil
}

func buildOfficialLatencyTestRequest(profile storedProfile) (*http.Request, error) {
	var auth authFile
	if err := json.Unmarshal([]byte(profile.AuthRaw), &auth); err != nil {
		return nil, fmt.Errorf("读取延迟测试前解析 auth.json 失败: %w", err)
	}
	if auth.Tokens == nil || strings.TrimSpace(auth.Tokens.AccessToken) == "" {
		return nil, fmt.Errorf("当前官方配置缺少可用 access_token")
	}

	config := parseConfigTOML(profile.ConfigRaw)
	probeURL := officialUsageURL(config.BaseURL)

	req, err := http.NewRequest(http.MethodGet, probeURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+auth.Tokens.AccessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "CodexSwitch/0.1")
	if accountID := trimmedFirst(auth.Tokens.AccountID, profile.Meta.ChatGPTAccountID); accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}
	return req, nil
}

func apiModelsURL(baseURL string) (string, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return "", fmt.Errorf("Base URL 不能为空")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("Base URL 无法解析: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("Base URL 格式无效")
	}

	path := strings.TrimRight(parsed.Path, "/")
	if path == "" && strings.EqualFold(parsed.Host, "api.openai.com") {
		path = "/v1"
	}
	parsed.Path = path + "/models"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func formatAPIProbeFailure(resp *http.Response) string {
	if resp == nil {
		return "延迟测试失败"
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	var payload apiProbeErrorPayload
	if err := json.Unmarshal(body, &payload); err == nil && payload.Error != nil {
		message := strings.TrimSpace(payload.Error.Message)
		if message != "" {
			return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, message)
		}
	}

	statusText := strings.TrimSpace(resp.Status)
	if statusText == "" {
		statusText = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return statusText
}

func optionalInt(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}
