package codexswitch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (s *Service) settingsPath() string {
	return filepath.Join(s.appConfigDir, "settings.json")
}

func (s *Service) profilesRoot() string {
	return filepath.Join(s.appConfigDir, "profiles")
}

func (s *Service) profileDir(id string) string {
	return filepath.Join(s.profilesRoot(), id)
}

func (s *Service) sharedOfficialConfigPath() string {
	return filepath.Join(s.appConfigDir, "official-config.toml")
}

func (s *Service) ensureAppLayout() error {
	if err := ensureDir(s.appConfigDir); err != nil {
		return fmt.Errorf("初始化应用配置目录失败: %w", err)
	}
	if err := ensureDir(s.profilesRoot()); err != nil {
		return fmt.Errorf("初始化 profiles 目录失败: %w", err)
	}
	return nil
}

func (s *Service) loadProfile(id string) (storedProfile, error) {
	dir := s.profileDir(id)
	metaRaw, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return storedProfile{}, err
	}

	var meta ProfileMeta
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		return storedProfile{}, fmt.Errorf("读取 meta.json 失败: %w", err)
	}

	authRaw, err := readTextFile(filepath.Join(dir, "auth.json"))
	if err != nil {
		return storedProfile{}, err
	}
	configRaw, err := s.loadStoredConfigRaw(meta, dir)
	if err != nil {
		return storedProfile{}, err
	}

	profile := storedProfile{
		Meta:      meta,
		AuthRaw:   authRaw,
		ConfigRaw: configRaw,
	}

	normalizeStoredProfileMeta(&profile)
	if profile.Meta.Type == ProfileTypeAPI {
		if err := s.seedAPILatencyHistoryIfEmpty(profile.Meta.ID, profile.Meta.LatencyTest.History); err != nil {
			s.logger.Warn("seed api latency history failed", "id", profile.Meta.ID, "error", err)
		}
		history, err := s.loadAPILatencyHistory(profile.Meta.ID, maxLatencyHistoryEntries)
		if err != nil {
			s.logger.Warn("load api latency history failed", "id", profile.Meta.ID, "error", err)
			profile.Meta.LatencyTest.History = nil
		} else {
			profile.Meta.LatencyTest.History = history
		}
	}
	return profile, nil
}

func (s *Service) loadAllProfiles() ([]ProfileMeta, error) {
	if err := s.ensureAppLayout(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.profilesRoot())
	if err != nil {
		return nil, err
	}

	profiles := make([]ProfileMeta, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		profile, err := s.loadProfile(entry.Name())
		if err != nil {
			s.logger.Warn("skip invalid profile", "id", entry.Name(), "error", err)
			continue
		}
		profiles = append(profiles, profile.Meta)
	}

	sortProfiles(profiles)
	return profiles, nil
}

func (s *Service) saveProfileSnapshot(snapshot *profileSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("快照不能为空")
	}
	if err := s.ensureAppLayout(); err != nil {
		return err
	}

	normalizedSnapshot, err := s.applyOfficialSharedConfig(snapshot)
	if err != nil {
		return err
	}
	snapshot = normalizedSnapshot

	var existing *ProfileMeta
	if stored, err := s.loadProfile(snapshot.Meta.ID); err == nil {
		existing = &stored.Meta
	}
	preserveStoredFields(&snapshot.Meta, existing, s.now())

	dir := s.profileDir(snapshot.Meta.ID)
	if err := ensureDir(dir); err != nil {
		return err
	}
	if err := safeWriteText(filepath.Join(dir, "auth.json"), snapshot.AuthRaw); err != nil {
		return fmt.Errorf("写入 auth.json 失败: %w", err)
	}
	if snapshot.Meta.Type == ProfileTypeOfficial {
		if err := os.Remove(filepath.Join(dir, "config.toml")); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("删除官方 profile 冗余 config.toml 失败: %w", err)
		}
	} else if err := safeWriteText(filepath.Join(dir, "config.toml"), snapshot.ConfigRaw); err != nil {
		return fmt.Errorf("写入 config.toml 失败: %w", err)
	}
	if snapshot.Meta.Type == ProfileTypeAPI {
		if err := s.seedAPILatencyHistoryIfEmpty(snapshot.Meta.ID, snapshot.Meta.LatencyTest.History); err != nil {
			s.logger.Warn("seed api latency history before save failed", "id", snapshot.Meta.ID, "error", err)
		}
	}
	metaForDisk := snapshot.Meta
	metaForDisk.LatencyTest.History = nil
	if err := safeWriteJSON(filepath.Join(dir, "meta.json"), metaForDisk); err != nil {
		return fmt.Errorf("写入 meta.json 失败: %w", err)
	}
	return nil
}

func (s *Service) deleteProfileDirectory(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("配置 ID 不能为空")
	}
	if err := s.deleteAPILatencyHistory(id); err != nil {
		return err
	}
	if err := os.RemoveAll(s.profileDir(id)); err != nil {
		return fmt.Errorf("删除配置失败: %w", err)
	}
	return nil
}

func sortProfiles(profiles []ProfileMeta) {
	sort.SliceStable(profiles, func(i, j int) bool {
		left := profiles[i]
		right := profiles[j]
		if left.IsActive != right.IsActive {
			return left.IsActive
		}
		if left.Type != right.Type {
			return left.Type == ProfileTypeOfficial
		}
		if left.UpdatedAt != right.UpdatedAt {
			return left.UpdatedAt > right.UpdatedAt
		}
		return strings.ToLower(left.DisplayName) < strings.ToLower(right.DisplayName)
	})
}

func isNotFound(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

func normalizeStoredProfileMeta(profile *storedProfile) {
	if profile == nil {
		return
	}
	if profile.Meta.RateLimits.Status == "" {
		profile.Meta.RateLimits.Status = RateLimitStatusIdle
	}
	if profile.Meta.LatencyTest.Status == "" {
		profile.Meta.LatencyTest.Status = LatencyTestStatusIdle
	}
	profile.Meta.LatencyTest.History = trimLatencyHistoryEntries(profile.Meta.LatencyTest.History)

	config := parseConfigTOML(profile.ConfigRaw)
	if strings.TrimSpace(config.BaseURL) != "" {
		profile.Meta.BaseURL = strings.TrimSpace(config.BaseURL)
	}
	if strings.TrimSpace(config.Model) != "" {
		profile.Meta.Model = strings.TrimSpace(config.Model)
	}
	if strings.TrimSpace(config.ModelReasoningEffort) != "" {
		profile.Meta.ModelReasoningEffort = strings.TrimSpace(config.ModelReasoningEffort)
	}

	if profile.Meta.Type != ProfileTypeAPI {
		return
	}

	var auth authFile
	if err := json.Unmarshal([]byte(profile.AuthRaw), &auth); err != nil {
		return
	}

	apiKey := ""
	if auth.OpenAIAPIKey != nil {
		apiKey = strings.TrimSpace(*auth.OpenAIAPIKey)
	}
	if apiKey == "" {
		return
	}

	profile.Meta.MaskedAPIKey = maskAPIKey(apiKey)
	profile.Meta.DisplayName = trimmedFirst(profile.Meta.MaskedAPIKey, profile.Meta.BaseURL, "API 配置")
}

func (s *Service) loadStoredConfigRaw(meta ProfileMeta, dir string) (string, error) {
	if meta.Type != ProfileTypeOfficial {
		return readTextFile(filepath.Join(dir, "config.toml"))
	}

	legacyConfigRaw := ""
	if raw, err := readTextFile(filepath.Join(dir, "config.toml")); err == nil {
		legacyConfigRaw = raw
	}
	return s.sharedOfficialConfigRaw(legacyConfigRaw)
}

func (s *Service) sharedOfficialConfigRaw(seed string) (string, error) {
	if err := s.ensureAppLayout(); err != nil {
		return "", err
	}

	path := s.sharedOfficialConfigPath()
	if raw, err := readTextFile(path); err == nil && strings.TrimSpace(raw) != "" {
		return normalizeConfigRaw(raw), nil
	} else if err != nil && !isNotFound(err) {
		return "", err
	}

	configRaw := normalizeConfigRaw(seed)
	if err := safeWriteText(path, configRaw); err != nil {
		return "", fmt.Errorf("写入官方共享 config.toml 失败: %w", err)
	}
	return configRaw, nil
}

func (s *Service) applyOfficialSharedConfig(snapshot *profileSnapshot) (*profileSnapshot, error) {
	if snapshot == nil || snapshot.Meta.Type != ProfileTypeOfficial {
		return snapshot, nil
	}

	configRaw, err := s.sharedOfficialConfigRaw(snapshot.ConfigRaw)
	if err != nil {
		return nil, err
	}
	if normalizeConfigRaw(snapshot.ConfigRaw) == configRaw {
		snapshot.ConfigRaw = configRaw
		return snapshot, nil
	}

	rebuilt, err := buildProfileSnapshot(snapshot.AuthRaw, configRaw, snapshot.Meta.Source, s.now())
	if err != nil {
		return nil, fmt.Errorf("应用官方共享 config.toml 失败: %w", err)
	}
	rebuilt.Meta.IsActive = snapshot.Meta.IsActive
	rebuilt.Meta.LastRateLimitFetchAt = snapshot.Meta.LastRateLimitFetchAt
	rebuilt.Meta.RateLimits = snapshot.Meta.RateLimits
	rebuilt.Meta.LatencyTest = snapshot.Meta.LatencyTest
	return rebuilt, nil
}

func normalizeConfigRaw(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = strings.TrimSpace(officialConfigTemplate)
	}
	return raw + "\n"
}
