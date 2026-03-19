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
	configRaw, err := readTextFile(filepath.Join(dir, "config.toml"))
	if err != nil {
		return storedProfile{}, err
	}

	return storedProfile{
		Meta:      meta,
		AuthRaw:   authRaw,
		ConfigRaw: configRaw,
	}, nil
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
	if err := safeWriteText(filepath.Join(dir, "config.toml"), snapshot.ConfigRaw); err != nil {
		return fmt.Errorf("写入 config.toml 失败: %w", err)
	}
	if err := safeWriteJSON(filepath.Join(dir, "meta.json"), snapshot.Meta); err != nil {
		return fmt.Errorf("写入 meta.json 失败: %w", err)
	}
	return nil
}

func (s *Service) deleteProfileDirectory(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("配置 ID 不能为空")
	}
	err := os.RemoveAll(s.profileDir(id))
	if err != nil {
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
