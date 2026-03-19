package codexswitch

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const apiConfigTemplate = `model_provider = "OpenAI"
model = {{ .Model }}
review_model = {{ .Model }}
model_reasoning_effort = {{ .ModelReasoningEffort }}
disable_response_storage = true
network_access = "enabled"
windows_wsl_setup_acknowledged = true
model_context_window = 1000000
model_auto_compact_token_limit = 900000

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

	return buildProfileSnapshot(authRaw, configRaw, "created_api_form", s.now())
}

func normalizeAPIProfileInput(input APIProfileInput) APIProfileInput {
	input.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	input.Model = strings.TrimSpace(input.Model)
	input.ModelReasoningEffort = strings.TrimSpace(input.ModelReasoningEffort)
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
	case input.APIKey == "":
		return fmt.Errorf("API Key 不能为空")
	default:
		return nil
	}
}

func renderAPIConfig(input APIProfileInput) string {
	replacer := strings.NewReplacer(
		"{{ .Model }}", tomlQuote(input.Model),
		"{{ .ModelReasoningEffort }}", tomlQuote(input.ModelReasoningEffort),
		"{{ .BaseURL }}", tomlQuote(input.BaseURL),
	)
	return replacer.Replace(apiConfigTemplate)
}

func (s *Service) buildSnapshotFromExistingProfile(profile storedProfile) *profileSnapshot {
	return &profileSnapshot{
		Meta:      profile.Meta,
		AuthRaw:   profile.AuthRaw,
		ConfigRaw: profile.ConfigRaw,
	}
}

func preserveStoredFields(next *ProfileMeta, existing *ProfileMeta, now time.Time) {
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
}
