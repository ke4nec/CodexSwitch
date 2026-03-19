package codexswitch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolveAppConfigDir(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return filepath.Clean(explicit), nil
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("获取用户配置目录失败: %w", err)
	}
	return filepath.Join(userConfigDir, "CodexSwitch"), nil
}

func defaultCodexHomePath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ".codex"
	}
	return filepath.Join(home, ".codex")
}

func (s *Service) loadSettings() (AppSettings, error) {
	settings := AppSettings{
		CodexHomePath: defaultCodexHomePath(),
	}

	data, err := os.ReadFile(s.settingsPath())
	if err != nil {
		if isNotFound(err) {
			return settings, nil
		}
		return settings, fmt.Errorf("读取 settings.json 失败: %w", err)
	}

	if err := json.Unmarshal(data, &settings); err != nil {
		return settings, fmt.Errorf("解析 settings.json 失败: %w", err)
	}
	if strings.TrimSpace(settings.CodexHomePath) == "" {
		settings.CodexHomePath = defaultCodexHomePath()
	}
	settings.CodexHomePath = filepath.Clean(settings.CodexHomePath)
	return settings, nil
}

func (s *Service) saveSettings(settings AppSettings) error {
	if strings.TrimSpace(settings.CodexHomePath) == "" {
		settings.CodexHomePath = defaultCodexHomePath()
	}
	settings.CodexHomePath = filepath.Clean(settings.CodexHomePath)
	return safeWriteJSON(s.settingsPath(), settings)
}
