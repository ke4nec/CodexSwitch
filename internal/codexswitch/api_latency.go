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

type apiProbeFailureInfo struct {
	Message string
	Type    string
	Code    string
}

type apiProbeRequestContext struct {
	APIKey  string
	BaseURL string
	WireAPI string
	Model   string
}

type officialProbeRequestContext struct {
	AccessToken string
	BaseURL     string
}

const latencyTestPrompt = "hi"
const maxLatencyHistoryEntries = 48

func (s *Service) refreshProfileLatencyTest(profile storedProfile) (storedProfile, error) {
	if profile.Meta.Type == ProfileTypeOfficial {
		refreshedProfile, err := s.refreshOfficialProfileAuth(profile)
		if err != nil {
			s.logger.Warn("refresh official token before latency test failed", "id", profile.Meta.ID, "error", err)
		} else {
			profile = refreshedProfile
		}
	}

	req, err := buildLatencyAvailabilityRequest(profile)
	if err != nil {
		return profile, err
	}
	if req == nil {
		return profile, nil
	}

	s.logLatencyTestRequest(profile, req)

	latencyMs, err := s.measureProfileLatency(profile)
	if err != nil {
		s.logger.Debug(
			"latency probe failed",
			"profile_id", profile.Meta.ID,
			"profile_type", profile.Meta.Type,
			"error", err,
		)
	}

	startedAt := time.Now()
	resp, err := s.httpClient.Do(req)
	if err != nil {
		profile.Meta.LatencyTest = LatencyTestState{
			LatencyMs: latencyMs,
		}
		return profile, err
	}
	defer resp.Body.Close()

	responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	availabilityLatencyMs := time.Since(startedAt).Milliseconds()
	if availabilityLatencyMs <= 0 {
		availabilityLatencyMs = 1
	}
	if readErr != nil {
		profile.Meta.LatencyTest = LatencyTestState{
			LatencyMs:  latencyMs,
			StatusCode: optionalInt(resp.StatusCode),
		}
		return profile, readErr
	}

	if latencyMs == nil && profile.Meta.Type != ProfileTypeAPI {
		latencyMs = optionalInt64(availabilityLatencyMs)
	}

	available := resp.StatusCode >= 200 && resp.StatusCode < 300
	var failureInfo apiProbeFailureInfo

	profile.Meta.LatencyTest = LatencyTestState{
		Status:     LatencyTestStatusSuccess,
		Available:  available,
		LatencyMs:  latencyMs,
		StatusCode: optionalInt(resp.StatusCode),
		CheckedAt:  s.now().UTC().Format(time.RFC3339),
	}

	if !available {
		failureInfo = extractAPIProbeFailure(resp.StatusCode, resp.Status, responseBody)
		profile.Meta.LatencyTest.ErrorMessage = failureInfo.Message
		profile.Meta.LatencyTest.ErrorType = failureInfo.Type
		profile.Meta.LatencyTest.ErrorCode = failureInfo.Code
	}

	s.logLatencyTestResult(profile, req, resp, profile.Meta.LatencyTest.LatencyMs, failureInfo)
	return profile, nil
}

func (s *Service) measureProfileLatency(profile storedProfile) (*int64, error) {
	if profile.Meta.Type != ProfileTypeAPI {
		return nil, nil
	}

	req, err := buildLatencyProbeRequest(profile)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, nil
	}

	startedAt := time.Now()
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	latencyMs := time.Since(startedAt).Milliseconds()
	if latencyMs <= 0 {
		latencyMs = 1
	}
	return optionalInt64(latencyMs), nil
}

func buildLatencyAvailabilityRequest(profile storedProfile) (*http.Request, error) {
	switch profile.Meta.Type {
	case ProfileTypeAPI:
		return buildAPILatencyAvailabilityRequest(profile)
	case ProfileTypeOfficial:
		req, err := buildOfficialLatencyAvailabilityRequest(profile)
		return req, err
	default:
		return nil, nil
	}
}

func buildLatencyProbeRequest(profile storedProfile) (*http.Request, error) {
	switch profile.Meta.Type {
	case ProfileTypeAPI:
		return buildAPILatencyProbeRequest(profile)
	default:
		return nil, nil
	}
}

func buildAPILatencyAvailabilityRequest(profile storedProfile) (*http.Request, error) {
	ctx, err := loadAPIProbeRequestContext(profile)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(ctx.Model) == "" {
		return nil, fmt.Errorf("当前 API 配置缺少可用 model")
	}

	probeURL, requestBody, err := buildAPIInteractionProbe(ctx.BaseURL, ctx.WireAPI, ctx.Model)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, probeURL, strings.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.APIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CodexSwitch/0.1")
	return req, nil
}

func buildAPILatencyProbeRequest(profile storedProfile) (*http.Request, error) {
	ctx, err := loadAPIProbeRequestContext(profile)
	if err != nil {
		return nil, err
	}

	probeURL, err := apiLatencyProbeURL(ctx.BaseURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, probeURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CodexSwitch/0.1")
	return req, nil
}

func buildOfficialLatencyAvailabilityRequest(profile storedProfile) (*http.Request, error) {
	ctx, err := loadOfficialProbeRequestContext(profile)
	if err != nil {
		return nil, err
	}

	probeURL, err := officialLatencyModelsURL(ctx.BaseURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, probeURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+ctx.AccessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "CodexSwitch/0.1")
	return req, nil
}

func loadAPIProbeRequestContext(profile storedProfile) (apiProbeRequestContext, error) {
	var auth authFile
	if err := json.Unmarshal([]byte(profile.AuthRaw), &auth); err != nil {
		return apiProbeRequestContext{}, fmt.Errorf("读取延迟测试前解析 auth.json 失败: %w", err)
	}

	apiKey := ""
	if auth.OpenAIAPIKey != nil {
		apiKey = strings.TrimSpace(*auth.OpenAIAPIKey)
	}
	if apiKey == "" {
		return apiProbeRequestContext{}, fmt.Errorf("当前 API 配置缺少可用 OPENAI_API_KEY")
	}

	config := parseConfigTOML(profile.ConfigRaw)
	return apiProbeRequestContext{
		APIKey:  apiKey,
		BaseURL: config.BaseURL,
		WireAPI: config.WireAPI,
		Model:   strings.TrimSpace(trimmedFirst(config.Model, profile.Meta.Model)),
	}, nil
}

func loadOfficialProbeRequestContext(profile storedProfile) (officialProbeRequestContext, error) {
	var auth authFile
	if err := json.Unmarshal([]byte(profile.AuthRaw), &auth); err != nil {
		return officialProbeRequestContext{}, fmt.Errorf("读取延迟测试前解析 auth.json 失败: %w", err)
	}
	if auth.Tokens == nil || strings.TrimSpace(auth.Tokens.AccessToken) == "" {
		return officialProbeRequestContext{}, fmt.Errorf("当前官方配置缺少可用 access_token")
	}

	config := parseConfigTOML(profile.ConfigRaw)
	return officialProbeRequestContext{
		AccessToken: strings.TrimSpace(auth.Tokens.AccessToken),
		BaseURL:     config.BaseURL,
	}, nil
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

func apiLatencyProbeURL(baseURL string) (string, error) {
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
	lowerPath := strings.ToLower(path)
	if lowerPath == "/v1" {
		path = ""
	} else if strings.HasSuffix(lowerPath, "/v1") {
		path = path[:len(path)-len("/v1")]
	}

	if path == "" {
		parsed.Path = "/"
	} else {
		parsed.Path = path
	}
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func officialLatencyModelsURL(baseURL string) (string, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return "https://api.openai.com/v1/models", nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("Base URL 无法解析: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("Base URL 格式无效")
	}

	path := strings.TrimRight(parsed.Path, "/")
	lowerPath := strings.ToLower(path)
	if strings.HasSuffix(lowerPath, "/backend-api") {
		path = path[:len(path)-len("/backend-api")]
		lowerPath = strings.ToLower(path)
	}
	if path == "" {
		path = "/v1"
		lowerPath = "/v1"
	}
	if !strings.HasSuffix(lowerPath, "/v1") {
		path += "/v1"
	}

	parsed.Path = path + "/models"
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func extractAPIProbeFailure(statusCode int, status string, body []byte) apiProbeFailureInfo {
	if statusCode <= 0 && strings.TrimSpace(status) == "" {
		return apiProbeFailureInfo{Message: "延迟测试失败"}
	}

	var payload apiProbeErrorPayload
	if err := json.Unmarshal(body, &payload); err == nil && payload.Error != nil {
		info := apiProbeFailureInfo{
			Message: strings.TrimSpace(payload.Error.Message),
			Type:    strings.TrimSpace(payload.Error.Type),
			Code:    normalizeProbeErrorCode(payload.Error.Code),
		}
		if info.Message != "" || info.Type != "" || info.Code != "" {
			if info.Message == "" {
				info.Message = fmt.Sprintf("HTTP %d", statusCode)
			}
			return info
		}
	}

	statusText := strings.TrimSpace(status)
	if statusText == "" {
		statusText = fmt.Sprintf("HTTP %d", statusCode)
	}
	return apiProbeFailureInfo{Message: statusText}
}

func optionalInt(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}

func (s *Service) logLatencyTestRequest(profile storedProfile, req *http.Request) {
	if req == nil {
		return
	}

	logArgs := []any{
		"profile_id", profile.Meta.ID,
		"profile_type", profile.Meta.Type,
		"method", req.Method,
		"target", latencyTestLogTarget(req),
	}
	if model := strings.TrimSpace(profile.Meta.Model); model != "" {
		logArgs = append(logArgs, "model", model)
	}

	s.logger.Debug(
		"latency test start",
		logArgs...,
	)
}

func (s *Service) logLatencyTestResult(
	profile storedProfile,
	req *http.Request,
	resp *http.Response,
	latencyMs *int64,
	failureInfo apiProbeFailureInfo,
) {
	if req == nil || resp == nil {
		return
	}

	available := resp.StatusCode >= 200 && resp.StatusCode < 300
	logArgs := []any{
		"profile_id", profile.Meta.ID,
		"profile_type", profile.Meta.Type,
		"method", req.Method,
		"target", latencyTestLogTarget(req),
		"status_code", resp.StatusCode,
		"available", available,
	}
	if latencyMs != nil {
		logArgs = append(logArgs, "latency_ms", *latencyMs)
	}

	if !available {
		if failureInfo.Type != "" {
			logArgs = append(logArgs, "error_type", failureInfo.Type)
		}
		if failureInfo.Code != "" {
			logArgs = append(logArgs, "error_code", failureInfo.Code)
		}
		if failureInfo.Message != "" {
			logArgs = append(logArgs, "error_message", failureInfo.Message)
		}
	}

	s.logger.Info(
		"latency test completed",
		logArgs...,
	)
}

func latencyTestLogTarget(req *http.Request) string {
	if req == nil || req.URL == nil {
		return ""
	}

	target := strings.TrimSpace(req.URL.Host)
	path := strings.TrimSpace(req.URL.EscapedPath())
	if path == "" {
		path = "/"
	}
	if target == "" {
		return path
	}
	return target + path
}

func trimLatencyHistoryEntries(history []LatencyHistoryEntry) []LatencyHistoryEntry {
	if len(history) == 0 {
		return nil
	}
	if len(history) <= maxLatencyHistoryEntries {
		return history
	}
	return history[len(history)-maxLatencyHistoryEntries:]
}

func copyOptionalInt(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func copyOptionalInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func normalizeProbeErrorCode(value any) string {
	if value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
