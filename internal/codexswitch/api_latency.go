package codexswitch

import (
	"bytes"
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

const latencyTestPrompt = "hi"

func (s *Service) refreshProfileLatencyTest(profile storedProfile) (storedProfile, error) {
	if profile.Meta.Type == ProfileTypeOfficial {
		refreshedProfile, err := s.refreshOfficialProfileAuth(profile)
		if err != nil {
			s.logger.Warn("refresh official token before latency test failed", "id", profile.Meta.ID, "error", err)
		} else {
			profile = refreshedProfile
		}
	}

	req, requestBody, err := buildLatencyTestRequest(profile)
	if err != nil {
		return profile, err
	}
	if req == nil {
		return profile, nil
	}

	s.logLatencyTestRequest(profile, req, requestBody)

	startedAt := time.Now()
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return profile, err
	}
	defer resp.Body.Close()

	responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	latencyMs := time.Since(startedAt).Milliseconds()
	if latencyMs <= 0 {
		latencyMs = 1
	}
	if readErr != nil {
		return profile, readErr
	}

	s.logLatencyTestResponse(profile, req, resp, latencyMs, responseBody)

	profile.Meta.LatencyTest = LatencyTestState{
		Status:     LatencyTestStatusSuccess,
		Available:  resp.StatusCode >= 200 && resp.StatusCode < 300,
		LatencyMs:  optionalInt64(latencyMs),
		StatusCode: optionalInt(resp.StatusCode),
		CheckedAt:  s.now().UTC().Format(time.RFC3339),
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		profile.Meta.LatencyTest.ErrorMessage = formatAPIProbeFailure(resp.StatusCode, resp.Status, responseBody)
	}

	profile.Meta.UpdatedAt = s.now().UTC().Format(time.RFC3339)
	return profile, nil
}

func buildLatencyTestRequest(profile storedProfile) (*http.Request, string, error) {
	switch profile.Meta.Type {
	case ProfileTypeAPI:
		return buildAPILatencyTestRequest(profile)
	case ProfileTypeOfficial:
		req, err := buildOfficialLatencyTestRequest(profile)
		return req, "", err
	default:
		return nil, "", nil
	}
}

func buildAPILatencyTestRequest(profile storedProfile) (*http.Request, string, error) {
	var auth authFile
	if err := json.Unmarshal([]byte(profile.AuthRaw), &auth); err != nil {
		return nil, "", fmt.Errorf("读取延迟测试前解析 auth.json 失败: %w", err)
	}

	apiKey := ""
	if auth.OpenAIAPIKey != nil {
		apiKey = strings.TrimSpace(*auth.OpenAIAPIKey)
	}
	if apiKey == "" {
		return nil, "", fmt.Errorf("当前 API 配置缺少可用 OPENAI_API_KEY")
	}

	config := parseConfigTOML(profile.ConfigRaw)
	model := strings.TrimSpace(trimmedFirst(config.Model, profile.Meta.Model))
	if model == "" {
		return nil, "", fmt.Errorf("当前 API 配置缺少可用 model")
	}

	probeURL, requestBody, err := buildAPIInteractionProbe(config.BaseURL, config.WireAPI, model)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest(http.MethodPost, probeURL, strings.NewReader(requestBody))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CodexSwitch/0.1")
	return req, requestBody, nil
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

func buildAPIInteractionProbe(baseURL, wireAPI, model string) (string, string, error) {
	switch normalizeWireAPI(wireAPI) {
	case "", "responses":
		probeURL, err := apiEndpointURL(baseURL, "/responses")
		if err != nil {
			return "", "", err
		}
		requestBody, err := json.Marshal(map[string]any{
			"model": model,
			"input": latencyTestPrompt,
		})
		if err != nil {
			return "", "", fmt.Errorf("构造 responses 测试请求失败: %w", err)
		}
		return probeURL, string(requestBody), nil
	case "chat_completions":
		probeURL, err := apiEndpointURL(baseURL, "/chat/completions")
		if err != nil {
			return "", "", err
		}
		requestBody, err := json.Marshal(map[string]any{
			"model": model,
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": latencyTestPrompt,
				},
			},
		})
		if err != nil {
			return "", "", fmt.Errorf("构造 chat completions 测试请求失败: %w", err)
		}
		return probeURL, string(requestBody), nil
	default:
		return "", "", fmt.Errorf("当前 API 配置的 wire_api=%q 暂不支持测试", strings.TrimSpace(wireAPI))
	}
}

func normalizeWireAPI(raw string) string {
	replacer := strings.NewReplacer("-", "_", ".", "_", "/", "_", " ", "_")
	return strings.ToLower(strings.TrimSpace(replacer.Replace(raw)))
}

func apiEndpointURL(baseURL, endpointPath string) (string, error) {
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
	parsed.Path = path + endpointPath
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func formatAPIProbeFailure(statusCode int, status string, body []byte) string {
	if statusCode <= 0 && strings.TrimSpace(status) == "" {
		return "延迟测试失败"
	}

	var payload apiProbeErrorPayload
	if err := json.Unmarshal(body, &payload); err == nil && payload.Error != nil {
		message := strings.TrimSpace(payload.Error.Message)
		if message != "" {
			return fmt.Sprintf("HTTP %d: %s", statusCode, message)
		}
	}

	statusText := strings.TrimSpace(status)
	if statusText == "" {
		statusText = fmt.Sprintf("HTTP %d", statusCode)
	}
	return statusText
}

func optionalInt(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func (s *Service) logLatencyTestRequest(profile storedProfile, req *http.Request, requestBody string) {
	if req == nil {
		return
	}

	s.logger.Info(
		"latency test request",
		"profile_id", profile.Meta.ID,
		"profile_type", profile.Meta.Type,
		"model", strings.TrimSpace(profile.Meta.Model),
		"method", req.Method,
		"url", req.URL.String(),
		"authorization", maskAuthorizationHeader(req.Header.Get("Authorization")),
		"request_body", formatProbeLogBody([]byte(requestBody)),
	)
}

func (s *Service) logLatencyTestResponse(
	profile storedProfile,
	req *http.Request,
	resp *http.Response,
	latencyMs int64,
	responseBody []byte,
) {
	if req == nil || resp == nil {
		return
	}

	s.logger.Info(
		"latency test response",
		"profile_id", profile.Meta.ID,
		"profile_type", profile.Meta.Type,
		"method", req.Method,
		"url", req.URL.String(),
		"status_code", resp.StatusCode,
		"latency_ms", latencyMs,
		"response_body", formatProbeLogBody(responseBody),
	)
}

func maskAuthorizationHeader(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "bearer ") {
		return "Bearer " + maskAPIKey(strings.TrimSpace(trimmed[len("bearer "):]))
	}
	return maskAPIKey(trimmed)
}

func formatProbeLogBody(body []byte) string {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return ""
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, trimmed); err == nil {
		trimmed = compact.Bytes()
	}

	const maxLogBodyLength = 4096
	if len(trimmed) > maxLogBodyLength {
		return string(trimmed[:maxLogBodyLength]) + "...(truncated)"
	}
	return string(trimmed)
}
