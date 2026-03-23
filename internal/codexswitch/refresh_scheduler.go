package codexswitch

import (
	"fmt"
	"time"
)

func (s *Service) RefreshActiveOfficialProfile() (string, bool, error) {
	if err := s.ensureAppLayout(); err != nil {
		return "", false, err
	}

	settings, err := s.loadSettings()
	if err != nil {
		return "", false, err
	}

	current, currentSnapshot := s.scanCurrentProfile(settings.CodexHomePath)
	if currentSnapshot == nil || !current.Available || !currentSnapshot.Meta.IsValid || current.Type != ProfileTypeOfficial {
		return "", false, nil
	}

	activeID := currentSnapshot.Meta.ID
	currentSnapshot.Meta.IsActive = true
	if err := s.syncCurrentProfileForActiveRefresh(currentSnapshot); err != nil {
		return activeID, false, err
	}
	s.syncProfileActiveFlags(activeID)

	stored, err := s.loadProfile(activeID)
	if err != nil {
		return activeID, false, fmt.Errorf("读取当前激活官方配置失败: %w", err)
	}
	if stored.Meta.Disabled {
		return activeID, false, nil
	}

	updated, refreshErr := s.refreshProfileRateLimit(stored)
	if refreshErr != nil {
		if updated.Meta.RateLimits.Primary != nil || updated.Meta.RateLimits.Secondary != nil {
			updated.Meta.RateLimits.Status = RateLimitStatusStale
		} else {
			updated.Meta.RateLimits.Status = RateLimitStatusError
		}
		updated.Meta.RateLimits.ErrorMessage = refreshErr.Error()
		updated.Meta.UpdatedAt = s.now().UTC().Format(time.RFC3339)
	}

	if err := s.saveProfileSnapshot(s.buildSnapshotFromExistingProfile(updated)); err != nil {
		return activeID, false, fmt.Errorf("保存当前激活官方配置失败: %w", err)
	}
	s.syncActiveProfileToCurrentCodexHome(stored, updated)

	if refreshErr != nil {
		return activeID, true, refreshErr
	}
	return activeID, true, nil
}

func (s *Service) syncCurrentProfileForActiveRefresh(currentSnapshot *profileSnapshot) error {
	if currentSnapshot == nil {
		return nil
	}

	currentSnapshot.Meta.IsActive = true

	existing, err := s.loadProfile(currentSnapshot.Meta.ID)
	if err != nil {
		if isNotFound(err) {
			return s.saveProfileSnapshot(currentSnapshot)
		}
		return fmt.Errorf("读取当前激活配置失败: %w", err)
	}

	if existing.Meta.ContentHash == currentSnapshot.Meta.ContentHash {
		if existing.Meta.IsActive {
			return nil
		}
		existing.Meta.IsActive = true
		return s.saveProfileSnapshot(s.buildSnapshotFromExistingProfile(existing))
	}

	currentSnapshot.Meta.Source = existing.Meta.Source
	currentSnapshot.Meta.CreatedAt = existing.Meta.CreatedAt
	currentSnapshot.Meta.LastRateLimitFetchAt = existing.Meta.LastRateLimitFetchAt
	currentSnapshot.Meta.RateLimits = existing.Meta.RateLimits
	currentSnapshot.Meta.LatencyTest = existing.Meta.LatencyTest
	return s.saveProfileSnapshot(currentSnapshot)
}

func (s *Service) syncProfileActiveFlags(activeID string) {
	profiles, err := s.loadAllProfiles()
	if err != nil {
		s.logger.Warn("sync profile active flags failed", "id", activeID, "error", err)
		return
	}

	for _, meta := range profiles {
		nextActive := meta.ID == activeID
		if meta.IsActive == nextActive {
			continue
		}

		stored, err := s.loadProfile(meta.ID)
		if err != nil {
			s.logger.Warn("load profile for active flag sync failed", "id", meta.ID, "error", err)
			continue
		}
		stored.Meta.IsActive = nextActive
		if err := s.saveProfileSnapshot(s.buildSnapshotFromExistingProfile(stored)); err != nil {
			s.logger.Warn("save profile active flag sync failed", "id", meta.ID, "error", err)
		}
	}
}
