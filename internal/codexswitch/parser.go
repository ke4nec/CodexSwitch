package codexswitch

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type authFile struct {
	AuthMode     *string    `json:"auth_mode"`
	OpenAIAPIKey *string    `json:"OPENAI_API_KEY"`
	Tokens       *tokenData `json:"tokens"`
	LastRefresh  *string    `json:"last_refresh"`
}

type tokenData struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id"`
}

type idTokenClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Profile       struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	} `json:"https://api.openai.com/profile"`
	Auth struct {
		ChatGPTPlanType  string `json:"chatgpt_plan_type"`
		ChatGPTUserID    string `json:"chatgpt_user_id"`
		UserID           string `json:"user_id"`
		ChatGPTAccountID string `json:"chatgpt_account_id"`
	} `json:"https://api.openai.com/auth"`
}

type accessTokenClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	ClientID      string `json:"client_id"`
	Profile       struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	} `json:"https://api.openai.com/profile"`
	Auth struct {
		ChatGPTPlanType  string `json:"chatgpt_plan_type"`
		ChatGPTUserID    string `json:"chatgpt_user_id"`
		UserID           string `json:"user_id"`
		ChatGPTAccountID string `json:"chatgpt_account_id"`
	} `json:"https://api.openai.com/auth"`
}

func buildProfileSnapshot(authRaw, configRaw, source string, now time.Time) (*profileSnapshot, error) {
	var auth authFile
	if err := json.Unmarshal([]byte(authRaw), &auth); err != nil {
		return nil, fmt.Errorf("auth.json 解析失败: %w", err)
	}

	config := parseConfigTOML(configRaw)
	profileType := detectProfileType(auth)
	meta := ProfileMeta{
		Type:                 profileType,
		Model:                strings.TrimSpace(config.Model),
		ModelReasoningEffort: strings.TrimSpace(config.ModelReasoningEffort),
		BaseURL:              strings.TrimSpace(config.BaseURL),
		Source:               source,
		IsValid:              profileType != ProfileTypeUnknown,
		ContentHash:          computeContentHash(authRaw, configRaw),
		CreatedAt:            now.UTC().Format(time.RFC3339),
		UpdatedAt:            now.UTC().Format(time.RFC3339),
		RateLimits: RateLimitState{
			Status: RateLimitStatusIdle,
		},
	}

	switch profileType {
	case ProfileTypeOfficial:
		if err := fillOfficialProfileMeta(&meta, auth); err != nil {
			return nil, err
		}
	case ProfileTypeAPI:
		fillAPIProfileMeta(&meta, auth)
	default:
		meta.DisplayName = "未识别配置"
	}

	stableIdentity := selectStableIdentity(meta, auth)
	if stableIdentity == "" {
		stableIdentity = meta.ContentHash
	}
	meta.StableKeyHash = sha256Hex(stableIdentity)
	meta.ID = buildProfileID(meta.Type, stableIdentity)

	return &profileSnapshot{
		Meta:      meta,
		AuthRaw:   authRaw,
		ConfigRaw: configRaw,
	}, nil
}

func detectProfileType(auth authFile) ProfileType {
	if auth.AuthMode != nil {
		mode := strings.ToLower(strings.TrimSpace(*auth.AuthMode))
		switch {
		case strings.HasPrefix(mode, "chatgpt"):
			return ProfileTypeOfficial
		case strings.Contains(mode, "api"):
			return ProfileTypeAPI
		default:
			return ProfileTypeUnknown
		}
	}

	if auth.OpenAIAPIKey != nil && strings.TrimSpace(*auth.OpenAIAPIKey) != "" {
		return ProfileTypeAPI
	}
	if auth.Tokens != nil {
		return ProfileTypeOfficial
	}
	return ProfileTypeUnknown
}

func fillOfficialProfileMeta(meta *ProfileMeta, auth authFile) error {
	if auth.Tokens == nil {
		return errors.New("官方配置缺少 tokens 字段")
	}

	var idClaims idTokenClaims
	var accessClaims accessTokenClaims

	if strings.TrimSpace(auth.Tokens.IDToken) != "" {
		if err := decodeJWTClaims(auth.Tokens.IDToken, &idClaims); err != nil {
			return fmt.Errorf("id_token 解析失败: %w", err)
		}
	}
	if strings.TrimSpace(auth.Tokens.AccessToken) != "" {
		if err := decodeJWTClaims(auth.Tokens.AccessToken, &accessClaims); err != nil {
			return fmt.Errorf("access_token 解析失败: %w", err)
		}
	}

	meta.Email = strings.ToLower(trimmedFirst(
		idClaims.Email,
		idClaims.Profile.Email,
		accessClaims.Email,
		accessClaims.Profile.Email,
	))
	meta.EmailVerified = idClaims.EmailVerified || idClaims.Profile.EmailVerified ||
		accessClaims.EmailVerified || accessClaims.Profile.EmailVerified
	meta.PlanType = strings.ToLower(trimmedFirst(
		idClaims.Auth.ChatGPTPlanType,
		accessClaims.Auth.ChatGPTPlanType,
	))
	meta.ChatGPTUserID = trimmedFirst(
		idClaims.Auth.ChatGPTUserID,
		idClaims.Auth.UserID,
		accessClaims.Auth.ChatGPTUserID,
		accessClaims.Auth.UserID,
	)
	meta.ChatGPTAccountID = trimmedFirst(
		auth.Tokens.AccountID,
		idClaims.Auth.ChatGPTAccountID,
		accessClaims.Auth.ChatGPTAccountID,
	)
	meta.ClientID = trimmedFirst(accessClaims.ClientID)
	meta.DisplayName = trimmedFirst(meta.Email, meta.ChatGPTAccountID, meta.ChatGPTUserID, "官方账号")

	return nil
}

func fillAPIProfileMeta(meta *ProfileMeta, auth authFile) {
	apiKey := ""
	if auth.OpenAIAPIKey != nil {
		apiKey = strings.TrimSpace(*auth.OpenAIAPIKey)
	}
	meta.MaskedAPIKey = maskAPIKey(apiKey)
	meta.DisplayName = trimmedFirst(meta.MaskedAPIKey, meta.BaseURL, "API 配置")
}

func selectStableIdentity(meta ProfileMeta, auth authFile) string {
	switch meta.Type {
	case ProfileTypeOfficial:
		return trimmedFirst(
			strings.ToLower(meta.Email),
			meta.ChatGPTAccountID,
			meta.ChatGPTUserID,
			meta.ContentHash,
		)
	case ProfileTypeAPI:
		if auth.OpenAIAPIKey != nil {
			return trimmedFirst(*auth.OpenAIAPIKey, meta.ContentHash)
		}
	}
	return meta.ContentHash
}

func decodeJWTClaims(token string, dest any) error {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return errors.New("JWT 格式无效")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return err
	}

	return json.Unmarshal(payload, dest)
}
