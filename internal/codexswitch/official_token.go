package codexswitch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	officialRefreshTokenURL            = "https://auth.openai.com/oauth/token"
	officialRefreshTokenURLOverrideEnv = "CODEX_REFRESH_TOKEN_URL_OVERRIDE"
	officialRefreshClientID            = "app_EMoamEEZ73f0CkXaXp7hrann"
)

type officialTokenRefreshRequest struct {
	ClientID     string `json:"client_id"`
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
}

type officialTokenRefreshResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (s *Service) refreshOfficialProfileAuth(profile storedProfile) (storedProfile, error) {
	if profile.Meta.Type != ProfileTypeOfficial {
		return profile, nil
	}

	var auth authFile
	if err := json.Unmarshal([]byte(profile.AuthRaw), &auth); err != nil {
		return profile, fmt.Errorf("读取刷新前 auth.json 失败: %w", err)
	}
	if auth.Tokens == nil {
		return profile, fmt.Errorf("当前官方配置缺少 tokens")
	}

	refreshToken := strings.TrimSpace(auth.Tokens.RefreshToken)
	if refreshToken == "" {
		return profile, nil
	}

	requestBody, err := json.Marshal(officialTokenRefreshRequest{
		ClientID:     officialRefreshClientID,
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
	})
	if err != nil {
		return profile, fmt.Errorf("生成刷新 token 请求失败: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, officialRefreshTokenEndpoint(), bytes.NewReader(requestBody))
	if err != nil {
		return profile, fmt.Errorf("创建刷新 token 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "CodexSwitch/0.1")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return profile, fmt.Errorf("刷新官方 token 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return profile, fmt.Errorf("刷新官方 token 失败: %s", formatOfficialTokenRefreshFailure(resp))
	}

	var payload officialTokenRefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return profile, fmt.Errorf("刷新官方 token 响应解析失败: %w", err)
	}

	changed := false
	if token := strings.TrimSpace(payload.IDToken); token != "" && token != auth.Tokens.IDToken {
		auth.Tokens.IDToken = token
		changed = true
	}
	if token := strings.TrimSpace(payload.AccessToken); token != "" && token != auth.Tokens.AccessToken {
		auth.Tokens.AccessToken = token
		changed = true
	}
	if token := strings.TrimSpace(payload.RefreshToken); token != "" && token != auth.Tokens.RefreshToken {
		auth.Tokens.RefreshToken = token
		changed = true
	}

	lastRefresh := s.now().UTC().Format(time.RFC3339)
	if auth.LastRefresh == nil || strings.TrimSpace(*auth.LastRefresh) != lastRefresh {
		auth.LastRefresh = &lastRefresh
		changed = true
	}

	if auth.AuthMode == nil || strings.TrimSpace(*auth.AuthMode) == "" {
		authMode := officialAuthModeTemplate
		auth.AuthMode = &authMode
		changed = true
	}

	if !changed {
		return profile, nil
	}

	authRaw, err := marshalCanonicalJSON(auth)
	if err != nil {
		return profile, fmt.Errorf("序列化刷新后的 auth.json 失败: %w", err)
	}

	snapshot, err := buildProfileSnapshot(authRaw, profile.ConfigRaw, profile.Meta.Source, s.now())
	if err != nil {
		return profile, fmt.Errorf("刷新官方 token 后重建配置失败: %w", err)
	}
	if snapshot.Meta.ID != profile.Meta.ID {
		return profile, fmt.Errorf("刷新官方 token 返回了不同账号，已拒绝覆盖本地配置")
	}

	refreshedMeta := snapshot.Meta
	refreshedMeta.Source = profile.Meta.Source
	refreshedMeta.IsActive = profile.Meta.IsActive
	refreshedMeta.CreatedAt = profile.Meta.CreatedAt
	refreshedMeta.UpdatedAt = profile.Meta.UpdatedAt
	refreshedMeta.LastRateLimitFetchAt = profile.Meta.LastRateLimitFetchAt
	refreshedMeta.RateLimits = profile.Meta.RateLimits
	refreshedMeta.LatencyTest = profile.Meta.LatencyTest

	profile.AuthRaw = snapshot.AuthRaw
	profile.ConfigRaw = snapshot.ConfigRaw
	profile.Meta = refreshedMeta
	return profile, nil
}

func (s *Service) syncActiveProfileToCurrentCodexHome(original, updated storedProfile) {
	if !updated.Meta.IsActive {
		return
	}
	if original.AuthRaw == updated.AuthRaw && original.ConfigRaw == updated.ConfigRaw {
		return
	}

	settings, err := s.loadSettings()
	if err != nil {
		s.logger.Warn("load settings for active profile sync failed", "id", updated.Meta.ID, "error", err)
		return
	}
	if strings.TrimSpace(settings.CodexHomePath) == "" {
		return
	}

	if err := s.writeTargetProfile(settings.CodexHomePath, s.buildSnapshotFromExistingProfile(updated)); err != nil {
		s.logger.Warn("sync active profile to codex home failed", "id", updated.Meta.ID, "error", err)
	}
}

func officialRefreshTokenEndpoint() string {
	if override := strings.TrimSpace(os.Getenv(officialRefreshTokenURLOverrideEnv)); override != "" {
		return override
	}
	return officialRefreshTokenURL
}

func formatOfficialTokenRefreshFailure(resp *http.Response) string {
	if resp == nil {
		return "刷新 token 失败"
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	message := strings.TrimSpace(string(body))
	if message != "" {
		var payload struct {
			Message          string `json:"message"`
			ErrorDescription string `json:"error_description"`
			Error            *struct {
				Message string `json:"message"`
				Code    any    `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &payload); err == nil {
			message = trimmedFirst(
				payload.Message,
				payload.ErrorDescription,
			)
			if payload.Error != nil {
				message = trimmedFirst(message, payload.Error.Message)
			}
		}
	}

	if message != "" {
		return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, message)
	}

	statusText := strings.TrimSpace(resp.Status)
	if statusText == "" {
		statusText = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return statusText
}
