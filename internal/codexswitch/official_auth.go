package codexswitch

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	profileSourceImportedCurrent      = "imported_current"
	profileSourceImportedFileStandard = "imported_file_standard"
	profileSourceImportedFileCLI      = "imported_file_cli"
	profileSourceCreatedAPIForm       = "created_api_form"
)

const officialConfigTemplate = `model = "gpt-5.4"
model_reasoning_effort = "xhigh"
[windows]
sandbox = "elevated"
`

const officialAuthModeTemplate = "chatgpt"

type officialAuthInputFormat string

const (
	officialAuthInputFormatUnknown  officialAuthInputFormat = ""
	officialAuthInputFormatStandard officialAuthInputFormat = "standard"
	officialAuthInputFormatCLI      officialAuthInputFormat = "cli"
)

func normalizeAuthJSON(authRaw string) (string, officialAuthInputFormat, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(authRaw), &root); err != nil {
		return authRaw, officialAuthInputFormatUnknown, nil
	}

	if isCLIOfficialAuth(root) {
		normalized, err := normalizeCLIOfficialAuth(root)
		if err != nil {
			return "", officialAuthInputFormatCLI, err
		}
		return normalized, officialAuthInputFormatCLI, nil
	}

	var auth authFile
	if err := json.Unmarshal([]byte(authRaw), &auth); err != nil {
		return authRaw, officialAuthInputFormatUnknown, nil
	}

	if detectProfileType(auth) != ProfileTypeOfficial {
		return authRaw, officialAuthInputFormatUnknown, nil
	}

	normalized, changed, err := normalizeStandardOfficialAuth(root, auth)
	if err != nil {
		return "", officialAuthInputFormatStandard, err
	}
	if !changed {
		return authRaw, officialAuthInputFormatStandard, nil
	}
	return normalized, officialAuthInputFormatStandard, nil
}

func normalizeStandardOfficialAuth(root map[string]json.RawMessage, auth authFile) (string, bool, error) {
	changed := false

	if auth.AuthMode == nil || strings.TrimSpace(*auth.AuthMode) == "" {
		authModeRaw, err := json.Marshal(officialAuthModeTemplate)
		if err != nil {
			return "", false, err
		}
		root["auth_mode"] = authModeRaw
		changed = true
	}

	if _, ok := root["OPENAI_API_KEY"]; !ok {
		root["OPENAI_API_KEY"] = json.RawMessage("null")
		changed = true
	}

	if !changed {
		return "", false, nil
	}

	normalized, err := marshalCanonicalJSON(root)
	if err != nil {
		return "", false, fmt.Errorf("标准官方 auth.json 规范化失败: %w", err)
	}
	return normalized, true, nil
}

func normalizeCLIOfficialAuth(root map[string]json.RawMessage) (string, error) {
	idToken, err := requireJSONStringField(root, "id_token")
	if err != nil {
		return "", err
	}
	accessToken, err := requireJSONStringField(root, "access_token")
	if err != nil {
		return "", err
	}
	refreshToken, err := requireJSONStringField(root, "refresh_token")
	if err != nil {
		return "", err
	}
	accountID, err := requireJSONStringField(root, "account_id")
	if err != nil {
		return "", err
	}

	lastRefresh, _ := optionalJSONStringField(root, "last_refresh")

	normalized := map[string]any{
		"auth_mode":      officialAuthModeTemplate,
		"OPENAI_API_KEY": nil,
		"tokens": map[string]string{
			"id_token":      idToken,
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"account_id":    accountID,
		},
	}
	if strings.TrimSpace(lastRefresh) != "" {
		normalized["last_refresh"] = strings.TrimSpace(lastRefresh)
	}

	authRaw, err := marshalCanonicalJSON(normalized)
	if err != nil {
		return "", fmt.Errorf("CLI auth.json 标准化失败: %w", err)
	}
	return authRaw, nil
}

func isCLIOfficialAuth(root map[string]json.RawMessage) bool {
	if _, ok := root["tokens"]; ok {
		return false
	}
	if _, ok := root["access_token"]; ok {
		return true
	}
	if _, ok := root["id_token"]; ok {
		return true
	}
	if _, ok := root["refresh_token"]; ok {
		return true
	}
	if _, ok := root["account_id"]; ok {
		return true
	}
	authType, ok := optionalJSONStringField(root, "type")
	return ok && strings.EqualFold(strings.TrimSpace(authType), "codex")
}

func requireJSONStringField(root map[string]json.RawMessage, key string) (string, error) {
	value, ok := optionalJSONStringField(root, key)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("CLI auth.json 缺少必需敏感字段 %q", key)
	}
	return strings.TrimSpace(value), nil
}

func optionalJSONStringField(root map[string]json.RawMessage, key string) (string, bool) {
	raw, ok := root[key]
	if !ok {
		return "", false
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

func marshalCanonicalJSON(value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}
