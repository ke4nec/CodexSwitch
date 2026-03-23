package codexswitch

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const defaultAPIModelContextWindow = "1000000"

const apiConfigTemplate = `model_provider = "OpenAI"
model = {{ .Model }}
review_model = {{ .Model }}
model_reasoning_effort = {{ .ModelReasoningEffort }}
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true
model_context_window = {{ .ModelContextWindow }}
model_auto_compact_token_limit = {{ .ModelAutoCompactTokenLimit }}

[model_providers.OpenAI]
name = "OpenAI"
base_url = {{ .BaseURL }}
wire_api = "responses"
requires_openai_auth = true

[windows]
sandbox = "elevated"
`

func (s *Service) buildAPIProfile(input APIProfileInput) (*profileSnapshot, error) {
	input = normalizeAPIProfileInput(input)
	if err := validateAPIProfileInput(input); err != nil {
		return nil, err
	}

	authBody, err := json.MarshalIndent(map[string]string{
		"OPENAI_API_KEY": input.APIKey,
	}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("生成 API auth.json 失败: %w", err)
	}
	authRaw := string(authBody) + "\n"
	configRaw := renderAPIConfig(input)

	return buildProfileSnapshot(authRaw, configRaw, profileSourceCreatedAPIForm, s.now())
}

func normalizeAPIProfileInput(input APIProfileInput) APIProfileInput {
	input.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	input.Model = strings.TrimSpace(input.Model)
	input.ModelReasoningEffort = strings.TrimSpace(input.ModelReasoningEffort)
	input.ModelContextWindow = normalizeModelContextWindow(input.ModelContextWindow)
	input.APIKey = strings.TrimSpace(input.APIKey)
	return input
}

func validateAPIProfileInput(input APIProfileInput) error {
	switch {
	case input.BaseURL == "":
		return fmt.Errorf("Base URL 不能为空")
	case input.Model == "":
		return fmt.Errorf("模型不能为空")
	case input.ModelReasoningEffort == "":
		return fmt.Errorf("推理强度不能为空")
	case input.ModelContextWindow == "":
		return fmt.Errorf("上下文大小不能为空")
	case !isPositiveInteger(input.ModelContextWindow):
		return fmt.Errorf("上下文大小必须是正整数")
	case input.APIKey == "":
		return fmt.Errorf("API Key 不能为空")
	default:
		return nil
	}
}

func renderAPIConfig(input APIProfileInput) string {
	contextWindow := input.ModelContextWindow
	autoCompactTokenLimit := computeAutoCompactTokenLimit(contextWindow)
	replacer := strings.NewReplacer(
		"{{ .Model }}", tomlQuote(input.Model),
		"{{ .ModelReasoningEffort }}", tomlQuote(input.ModelReasoningEffort),
		"{{ .ModelContextWindow }}", contextWindow,
		"{{ .ModelAutoCompactTokenLimit }}", autoCompactTokenLimit,
		"{{ .BaseURL }}", tomlQuote(input.BaseURL),
	)
	return replacer.Replace(apiConfigTemplate)
}

func normalizeModelContextWindow(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "_", "")
	if value == "" {
		return defaultAPIModelContextWindow
	}
	return value
}

func defaultModelContextWindowOr(value string) string {
	return normalizeModelContextWindow(value)
}

func isPositiveInteger(value string) bool {
	parsed, err := strconv.ParseInt(value, 10, 64)
	return err == nil && parsed > 0
}

func computeAutoCompactTokenLimit(modelContextWindow string) string {
	parsed, err := strconv.ParseInt(modelContextWindow, 10, 64)
	if err != nil || parsed <= 0 {
		parsed, _ = strconv.ParseInt(defaultAPIModelContextWindow, 10, 64)
	}

	limit := parsed * 9 / 10
	if limit < 1 {
		limit = 1
	}

	return strconv.FormatInt(limit, 10)
}

func (s *Service) buildSnapshotFromExistingProfile(profile storedProfile) *profileSnapshot {
	return &profileSnapshot{
		Meta:      profile.Meta,
		AuthRaw:   profile.AuthRaw,
		ConfigRaw: profile.ConfigRaw,
	}
}

func preserveStoredFields(next *ProfileMeta, existing *ProfileMeta, now time.Time) {
	if next.RateLimits.Status == "" {
		next.RateLimits.Status = RateLimitStatusIdle
	}
	if next.LatencyTest.Status == "" {
		next.LatencyTest.Status = LatencyTestStatusIdle
	}

	if existing == nil {
		next.CreatedAt = now.UTC().Format(time.RFC3339)
		next.UpdatedAt = now.UTC().Format(time.RFC3339)
		return
	}

	next.CreatedAt = existing.CreatedAt
	next.UpdatedAt = now.UTC().Format(time.RFC3339)
	if next.LastRateLimitFetchAt == "" {
		next.LastRateLimitFetchAt = existing.LastRateLimitFetchAt
	}
	if next.RateLimits.Status == RateLimitStatusIdle && next.RateLimits.Primary == nil && next.RateLimits.Secondary == nil {
		next.RateLimits = existing.RateLimits
	}
	if shouldPreserveLatencyTest(next, existing) {
		next.LatencyTest = existing.LatencyTest
	}
}

func shouldPreserveLatencyTest(next *ProfileMeta, existing *ProfileMeta) bool {
	if next == nil || existing == nil {
		return false
	}
	if next.ContentHash != existing.ContentHash {
		return false
	}
	if next.LatencyTest.Status != LatencyTestStatusIdle {
		return false
	}
	if next.LatencyTest.Available {
		return false
	}
	if next.LatencyTest.LatencyMs != nil || next.LatencyTest.StatusCode != nil {
		return false
	}
	if strings.TrimSpace(next.LatencyTest.ErrorMessage) != "" || strings.TrimSpace(next.LatencyTest.CheckedAt) != "" {
		return false
	}
	return true
}
