package codexswitch

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (s *Service) ImportOfficialProfileFile(filePath string) (AppState, error) {
	snapshot, err := s.buildOfficialProfileSnapshotFromFile(filePath)
	if err != nil {
		return AppState{}, err
	}
	if err := s.saveProfileSnapshot(snapshot); err != nil {
		return AppState{}, err
	}
	return s.syncAndBuildState(true)
}

func (s *Service) buildOfficialProfileSnapshotFromFile(filePath string) (*profileSnapshot, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("官方账号文件路径不能为空")
	}

	authRaw, err := readTextFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取官方账号文件失败: %w", err)
	}

	normalizedAuthRaw, inputFormat, err := normalizeAuthJSON(authRaw)
	if err != nil {
		return nil, err
	}

	source := sourceForOfficialImportFormat(inputFormat)
	if source == "" {
		return nil, fmt.Errorf("所选文件不是受支持的官方账号 auth.json 或 CLI auth.json: %s", filepath.Base(filePath))
	}

	configRaw, err := s.sharedOfficialConfigRaw("")
	if err != nil {
		return nil, err
	}

	snapshot, err := buildProfileSnapshot(normalizedAuthRaw, configRaw, source, s.now())
	if err != nil {
		return nil, err
	}
	if snapshot.Meta.Type != ProfileTypeOfficial {
		return nil, fmt.Errorf("所选文件不是官方账号配置: %s", filepath.Base(filePath))
	}
	return snapshot, nil
}

func sourceForOfficialImportFormat(inputFormat officialAuthInputFormat) string {
	switch inputFormat {
	case officialAuthInputFormatStandard:
		return profileSourceImportedFileStandard
	case officialAuthInputFormatCLI:
		return profileSourceImportedFileCLI
	default:
		return ""
	}
}
