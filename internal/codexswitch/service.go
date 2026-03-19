package codexswitch

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ServiceOptions struct {
	AppConfigDir string
	HTTPClient   *http.Client
	Logger       *slog.Logger
	Now          func() time.Time
}

type Service struct {
	appConfigDir string
	httpClient   *http.Client
	logger       *slog.Logger
	now          func() time.Time
}

func NewService(options ServiceOptions) (*Service, error) {
	appConfigDir, err := resolveAppConfigDir(options.AppConfigDir)
	if err != nil {
		return nil, err
	}

	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	now := options.Now
	if now == nil {
		now = time.Now
	}

	return &Service{
		appConfigDir: appConfigDir,
		httpClient:   httpClient,
		logger:       logger,
		now:          now,
	}, nil
}

func (s *Service) GetAppState() (AppState, error) {
	return s.syncAndBuildState(true)
}

func (s *Service) UpdateSettings(input UpdateSettingsInput) (AppState, error) {
	settings, err := s.loadSettings()
	if err != nil {
		return AppState{}, err
	}
	if strings.TrimSpace(input.CodexHomePath) == "" {
		return AppState{}, fmt.Errorf("目标 Codex 配置目录不能为空")
	}
	settings.CodexHomePath = filepath.Clean(strings.TrimSpace(input.CodexHomePath))
	settings.LastOpenedAt = s.now().UTC().Format(time.RFC3339)
	if err := s.saveSettings(settings); err != nil {
		return AppState{}, fmt.Errorf("保存设置失败: %w", err)
	}
	return s.syncAndBuildState(true)
}

func (s *Service) ImportCurrentProfile() (AppState, error) {
	settings, err := s.loadSettings()
	if err != nil {
		return AppState{}, err
	}
	_, snapshot := s.scanCurrentProfile(settings.CodexHomePath)
	if snapshot == nil || !snapshot.Meta.IsValid {
		return AppState{}, fmt.Errorf("当前配置不可导入，请先检查 auth.json 和 config.toml")
	}
	if err := s.saveProfileSnapshot(snapshot); err != nil {
		return AppState{}, err
	}
	return s.syncAndBuildState(true)
}

func (s *Service) CreateAPIProfile(input APIProfileInput) (AppState, error) {
	snapshot, err := s.buildAPIProfile(input)
	if err != nil {
		return AppState{}, err
	}
	if err := s.saveProfileSnapshot(snapshot); err != nil {
		return AppState{}, err
	}
	return s.syncAndBuildState(true)
}

func (s *Service) UpdateAPIProfile(id string, input APIProfileInput) (AppState, error) {
	existing, err := s.loadProfile(id)
	if err != nil {
		return AppState{}, fmt.Errorf("读取待编辑配置失败: %w", err)
	}
	if existing.Meta.Type != ProfileTypeAPI {
		return AppState{}, fmt.Errorf("仅 API 配置支持编辑")
	}

	snapshot, err := s.buildAPIProfile(input)
	if err != nil {
		return AppState{}, err
	}
	snapshot.Meta.Source = existing.Meta.Source
	snapshot.Meta.IsActive = false

	if err := s.saveProfileSnapshot(snapshot); err != nil {
		return AppState{}, err
	}

	if snapshot.Meta.ID != id {
		if err := s.deleteProfileDirectory(id); err != nil {
			return AppState{}, err
		}
	}

	if existing.Meta.IsActive {
		settings, err := s.loadSettings()
		if err != nil {
			return AppState{}, err
		}
		if err := s.writeTargetProfile(settings.CodexHomePath, snapshot); err != nil {
			return AppState{}, err
		}
	}

	return s.syncAndBuildState(true)
}

func (s *Service) GetAPIProfileInput(id string) (APIProfileInput, error) {
	profile, err := s.loadProfile(id)
	if err != nil {
		return APIProfileInput{}, fmt.Errorf("读取配置失败: %w", err)
	}
	if profile.Meta.Type != ProfileTypeAPI {
		return APIProfileInput{}, fmt.Errorf("仅 API 配置支持编辑")
	}

	var auth authFile
	if err := json.Unmarshal([]byte(profile.AuthRaw), &auth); err != nil {
		return APIProfileInput{}, fmt.Errorf("解析 API auth.json 失败: %w", err)
	}

	config := parseConfigTOML(profile.ConfigRaw)
	apiKey := ""
	if auth.OpenAIAPIKey != nil {
		apiKey = strings.TrimSpace(*auth.OpenAIAPIKey)
	}

	return APIProfileInput{
		BaseURL:              strings.TrimSpace(config.BaseURL),
		Model:                strings.TrimSpace(config.Model),
		ModelReasoningEffort: strings.TrimSpace(config.ModelReasoningEffort),
		APIKey:               apiKey,
	}, nil
}

func (s *Service) SwitchProfile(id string) (AppState, error) {
	settings, err := s.loadSettings()
	if err != nil {
		return AppState{}, err
	}

	target, err := s.loadProfile(id)
	if err != nil {
		return AppState{}, fmt.Errorf("读取目标配置失败: %w", err)
	}

	current, currentSnapshot := s.scanCurrentProfile(settings.CodexHomePath)
	if currentSnapshot != nil && currentSnapshot.Meta.IsValid && currentSnapshot.Meta.ID != target.Meta.ID {
		if err := s.saveProfileSnapshot(currentSnapshot); err != nil {
			return AppState{}, err
		}
	}

	if currentSnapshot != nil && current.ProfileID == target.Meta.ID && current.ContentHash == target.Meta.ContentHash {
		return s.syncAndBuildState(true)
	}

	if err := s.writeTargetProfile(settings.CodexHomePath, s.buildSnapshotFromExistingProfile(target)); err != nil {
		return AppState{}, err
	}

	return s.syncAndBuildState(true)
}

func (s *Service) DeleteProfile(id string) (AppState, error) {
	state, err := s.syncAndBuildState(false)
	if err != nil {
		return AppState{}, err
	}

	for _, profile := range state.Profiles {
		if profile.ID == id && profile.IsActive {
			return AppState{}, fmt.Errorf("当前激活配置不能直接删除，请先切换到其他配置")
		}
	}

	if err := s.deleteProfileDirectory(id); err != nil {
		return AppState{}, err
	}
	return s.syncAndBuildState(true)
}

func (s *Service) RefreshRateLimits(ids []string) (AppState, error) {
	if err := s.ensureAppLayout(); err != nil {
		return AppState{}, err
	}

	targetSet := map[string]bool{}
	for _, id := range ids {
		if strings.TrimSpace(id) != "" {
			targetSet[id] = true
		}
	}

	profiles, err := s.loadAllProfiles()
	if err != nil {
		return AppState{}, err
	}

	for _, meta := range profiles {
		if meta.Type != ProfileTypeOfficial {
			continue
		}
		if len(targetSet) > 0 && !targetSet[meta.ID] {
			continue
		}

		stored, err := s.loadProfile(meta.ID)
		if err != nil {
			s.logger.Warn("load profile for rate limits failed", "id", meta.ID, "error", err)
			continue
		}
		updated, err := s.refreshProfileRateLimit(stored)
		if err != nil {
			s.logger.Warn("refresh rate limits failed", "id", meta.ID, "error", err)
			updated = stored
			if updated.Meta.RateLimits.Primary != nil || updated.Meta.RateLimits.Secondary != nil {
				updated.Meta.RateLimits.Status = RateLimitStatusStale
			} else {
				updated.Meta.RateLimits.Status = RateLimitStatusError
			}
			updated.Meta.RateLimits.ErrorMessage = err.Error()
			updated.Meta.UpdatedAt = s.now().UTC().Format(time.RFC3339)
		}

		if err := s.saveProfileSnapshot(s.buildSnapshotFromExistingProfile(updated)); err != nil {
			s.logger.Warn("save rate limits failed", "id", meta.ID, "error", err)
		}
	}

	return s.syncAndBuildState(false)
}

func (s *Service) syncAndBuildState(autoSyncCurrent bool) (AppState, error) {
	if err := s.ensureAppLayout(); err != nil {
		return AppState{}, err
	}

	settings, err := s.loadSettings()
	if err != nil {
		return AppState{}, err
	}
	settings.LastOpenedAt = s.now().UTC().Format(time.RFC3339)
	if err := s.saveSettings(settings); err != nil {
		return AppState{}, fmt.Errorf("更新 settings.json 失败: %w", err)
	}

	current, currentSnapshot := s.scanCurrentProfile(settings.CodexHomePath)
	if autoSyncCurrent && currentSnapshot != nil && currentSnapshot.Meta.IsValid {
		if existing, err := s.loadProfile(currentSnapshot.Meta.ID); err != nil {
			if isNotFound(err) {
				if err := s.saveProfileSnapshot(currentSnapshot); err != nil {
					return AppState{}, err
				}
			} else {
				return AppState{}, err
			}
		} else if existing.Meta.ContentHash != currentSnapshot.Meta.ContentHash {
			currentSnapshot.Meta.Source = existing.Meta.Source
			currentSnapshot.Meta.CreatedAt = existing.Meta.CreatedAt
			currentSnapshot.Meta.LastRateLimitFetchAt = existing.Meta.LastRateLimitFetchAt
			currentSnapshot.Meta.RateLimits = existing.Meta.RateLimits
			if err := s.saveProfileSnapshot(currentSnapshot); err != nil {
				return AppState{}, err
			}
		}
	}

	profiles, err := s.loadAllProfiles()
	if err != nil {
		return AppState{}, err
	}

	activeID := ""
	if currentSnapshot != nil && currentSnapshot.Meta.IsValid {
		activeID = currentSnapshot.Meta.ID
	}

	for i := range profiles {
		nextActive := profiles[i].ID == activeID && activeID != ""
		if profiles[i].IsActive == nextActive {
			continue
		}
		stored, err := s.loadProfile(profiles[i].ID)
		if err != nil {
			continue
		}
		stored.Meta.IsActive = nextActive
		if err := s.saveProfileSnapshot(s.buildSnapshotFromExistingProfile(stored)); err != nil {
			s.logger.Warn("persist active status failed", "id", profiles[i].ID, "error", err)
		}
		profiles[i].IsActive = nextActive
	}

	sortProfiles(profiles)
	current.Managed = activeID != ""
	if activeID != "" {
		for _, profile := range profiles {
			if profile.ID == activeID {
				current.Managed = true
				break
			}
		}
	}

	return AppState{
		Settings: settings,
		Current:  current,
		Profiles: profiles,
	}, nil
}

func (s *Service) scanCurrentProfile(codexHomePath string) (CurrentProfileState, *profileSnapshot) {
	state := CurrentProfileState{
		Path: filepath.Clean(codexHomePath),
		Type: ProfileTypeUnknown,
	}

	if strings.TrimSpace(codexHomePath) == "" {
		state.Error = "目标 Codex 配置目录为空"
		return state, nil
	}

	if _, err := os.Stat(codexHomePath); err != nil {
		if os.IsNotExist(err) {
			state.Error = "目标 Codex 配置目录不存在"
			return state, nil
		}
		state.Error = fmt.Sprintf("访问目标目录失败: %v", err)
		return state, nil
	}

	authPath := filepath.Join(codexHomePath, "auth.json")
	configPath := filepath.Join(codexHomePath, "config.toml")
	authRaw, err := readTextFile(authPath)
	if err != nil {
		state.Error = "当前目录缺少 auth.json"
		return state, nil
	}
	configRaw, err := readTextFile(configPath)
	if err != nil {
		state.Error = "当前目录缺少 config.toml"
		return state, nil
	}

	snapshot, err := buildProfileSnapshot(authRaw, configRaw, "imported_current", s.now())
	state.Available = true
	if err != nil {
		state.Error = err.Error()
		return state, nil
	}

	state.ProfileID = snapshot.Meta.ID
	state.Type = snapshot.Meta.Type
	state.DisplayName = snapshot.Meta.DisplayName
	state.ContentHash = snapshot.Meta.ContentHash
	if !snapshot.Meta.IsValid {
		state.Error = "当前配置类型无法识别"
		return state, nil
	}

	return state, snapshot
}

func (s *Service) writeTargetProfile(codexHomePath string, snapshot *profileSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("目标配置不能为空")
	}
	if err := ensureDir(codexHomePath); err != nil {
		return fmt.Errorf("创建目标 Codex 目录失败: %w", err)
	}
	if err := safeWriteText(filepath.Join(codexHomePath, "auth.json"), snapshot.AuthRaw); err != nil {
		return fmt.Errorf("写入目标 auth.json 失败: %w", err)
	}
	if err := safeWriteText(filepath.Join(codexHomePath, "config.toml"), snapshot.ConfigRaw); err != nil {
		return fmt.Errorf("写入目标 config.toml 失败: %w", err)
	}
	return nil
}
