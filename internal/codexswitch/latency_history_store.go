package codexswitch

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type latencyHistoryWriteMode string

const (
	latencyHistoryWriteModeAuto   latencyHistoryWriteMode = "auto"
	latencyHistoryWriteModeManual latencyHistoryWriteMode = "manual"
)

func (s *Service) latencyHistoryDBPath() string {
	return filepath.Join(s.appConfigDir, "latency-history.sqlite")
}

func openLatencyHistoryDB(appConfigDir string) (*sql.DB, error) {
	dbPath := filepath.Join(appConfigDir, "latency-history.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开延迟历史 SQLite 失败: %w", err)
	}
	if err := initLatencyHistoryDB(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func initLatencyHistoryDB(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("延迟历史 SQLite 未初始化")
	}

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS api_latency_history (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	profile_id TEXT NOT NULL,
	checked_at TEXT NOT NULL,
	record_source TEXT NOT NULL,
	status TEXT NOT NULL,
	available INTEGER NOT NULL,
	latency_ms INTEGER,
	status_code INTEGER,
	error_message TEXT NOT NULL DEFAULT '',
	error_type TEXT NOT NULL DEFAULT '',
	error_code TEXT NOT NULL DEFAULT ''
);
`); err != nil {
		return fmt.Errorf("初始化延迟历史表失败: %w", err)
	}

	if _, err := db.Exec(`
CREATE INDEX IF NOT EXISTS idx_api_latency_history_profile_checked
ON api_latency_history(profile_id, checked_at DESC, id DESC);
`); err != nil {
		return fmt.Errorf("初始化延迟历史索引失败: %w", err)
	}

	return nil
}

func (s *Service) loadAPILatencyHistory(profileID string, limit int) ([]LatencyHistoryEntry, error) {
	if s == nil || s.latencyHistoryDB == nil || strings.TrimSpace(profileID) == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = maxLatencyHistoryEntries
	}

	rows, err := s.latencyHistoryDB.Query(`
SELECT status, available, latency_ms, status_code, error_message, error_type, error_code, checked_at
FROM api_latency_history
WHERE profile_id = ?
ORDER BY checked_at DESC, id DESC
LIMIT ?;
`, profileID, limit)
	if err != nil {
		return nil, fmt.Errorf("查询 API 延迟历史失败: %w", err)
	}
	defer rows.Close()

	descending := make([]LatencyHistoryEntry, 0, limit)
	for rows.Next() {
		var (
			status     string
			available  int
			latencyMs  sql.NullInt64
			statusCode sql.NullInt64
			errorMsg   string
			errorType  string
			errorCode  string
			checkedAt  string
		)

		if err := rows.Scan(&status, &available, &latencyMs, &statusCode, &errorMsg, &errorType, &errorCode, &checkedAt); err != nil {
			return nil, fmt.Errorf("读取 API 延迟历史失败: %w", err)
		}

		entry := LatencyHistoryEntry{
			Status:       LatencyTestStatus(strings.TrimSpace(status)),
			Available:    available != 0,
			ErrorMessage: strings.TrimSpace(errorMsg),
			ErrorType:    strings.TrimSpace(errorType),
			ErrorCode:    strings.TrimSpace(errorCode),
			CheckedAt:    strings.TrimSpace(checkedAt),
		}
		if latencyMs.Valid && latencyMs.Int64 > 0 {
			entry.LatencyMs = optionalInt64(latencyMs.Int64)
		}
		if statusCode.Valid && statusCode.Int64 > 0 {
			entry.StatusCode = optionalInt(int(statusCode.Int64))
		}
		descending = append(descending, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 API 延迟历史失败: %w", err)
	}

	ascending := make([]LatencyHistoryEntry, 0, len(descending))
	for index := len(descending) - 1; index >= 0; index-- {
		ascending = append(ascending, descending[index])
	}
	return trimLatencyHistoryEntries(ascending), nil
}

func (s *Service) recordAPILatencyHistory(profileID string, state LatencyTestState, mode latencyHistoryWriteMode) error {
	if s == nil || s.latencyHistoryDB == nil || strings.TrimSpace(profileID) == "" || strings.TrimSpace(state.CheckedAt) == "" {
		return nil
	}

	if mode == latencyHistoryWriteModeManual {
		updated, err := s.updateLatestAPILatencyHistory(profileID, state)
		if err != nil {
			return err
		}
		if updated {
			return nil
		}
	}

	_, err := s.latencyHistoryDB.Exec(`
INSERT INTO api_latency_history (
	profile_id,
	checked_at,
	record_source,
	status,
	available,
	latency_ms,
	status_code,
	error_message,
	error_type,
	error_code
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
`,
		profileID,
		strings.TrimSpace(state.CheckedAt),
		string(mode),
		string(state.Status),
		boolToSQLiteInt(state.Available),
		sqliteNullableInt64(state.LatencyMs),
		sqliteNullableInt(state.StatusCode),
		strings.TrimSpace(state.ErrorMessage),
		strings.TrimSpace(state.ErrorType),
		strings.TrimSpace(state.ErrorCode),
	)
	if err != nil {
		return fmt.Errorf("写入 API 延迟历史失败: %w", err)
	}
	return nil
}

func (s *Service) updateLatestAPILatencyHistory(profileID string, state LatencyTestState) (bool, error) {
	var id int64
	err := s.latencyHistoryDB.QueryRow(`
SELECT id
FROM api_latency_history
WHERE profile_id = ?
ORDER BY checked_at DESC, id DESC
LIMIT 1;
`, profileID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("查询最近 API 延迟历史失败: %w", err)
	}

	result, err := s.latencyHistoryDB.Exec(`
UPDATE api_latency_history
SET checked_at = ?,
	status = ?,
	available = ?,
	latency_ms = ?,
	status_code = ?,
	error_message = ?,
	error_type = ?,
	error_code = ?
WHERE id = ?;
`,
		strings.TrimSpace(state.CheckedAt),
		string(state.Status),
		boolToSQLiteInt(state.Available),
		sqliteNullableInt64(state.LatencyMs),
		sqliteNullableInt(state.StatusCode),
		strings.TrimSpace(state.ErrorMessage),
		strings.TrimSpace(state.ErrorType),
		strings.TrimSpace(state.ErrorCode),
		id,
	)
	if err != nil {
		return false, fmt.Errorf("更新最近 API 延迟历史失败: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("读取 API 延迟历史更新结果失败: %w", err)
	}
	return affected > 0, nil
}

func (s *Service) deleteAPILatencyHistory(profileID string) error {
	if s == nil || s.latencyHistoryDB == nil || strings.TrimSpace(profileID) == "" {
		return nil
	}

	if _, err := s.latencyHistoryDB.Exec(`DELETE FROM api_latency_history WHERE profile_id = ?;`, profileID); err != nil {
		return fmt.Errorf("删除 API 延迟历史失败: %w", err)
	}
	return nil
}

func (s *Service) seedAPILatencyHistoryIfEmpty(profileID string, history []LatencyHistoryEntry) error {
	if s == nil || s.latencyHistoryDB == nil || strings.TrimSpace(profileID) == "" || len(history) == 0 {
		return nil
	}

	var existing int
	if err := s.latencyHistoryDB.QueryRow(`
SELECT COUNT(1)
FROM api_latency_history
WHERE profile_id = ?;
`, profileID).Scan(&existing); err != nil {
		return fmt.Errorf("检查 API 延迟历史是否为空失败: %w", err)
	}
	if existing > 0 {
		return nil
	}

	tx, err := s.latencyHistoryDB.Begin()
	if err != nil {
		return fmt.Errorf("初始化 API 延迟历史迁移事务失败: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, entry := range trimLatencyHistoryEntries(history) {
		if strings.TrimSpace(entry.CheckedAt) == "" {
			continue
		}
		if _, err = tx.Exec(`
INSERT INTO api_latency_history (
	profile_id,
	checked_at,
	record_source,
	status,
	available,
	latency_ms,
	status_code,
	error_message,
	error_type,
	error_code
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
`,
			profileID,
			strings.TrimSpace(entry.CheckedAt),
			"legacy",
			string(entry.Status),
			boolToSQLiteInt(entry.Available),
			sqliteNullableInt64(entry.LatencyMs),
			sqliteNullableInt(entry.StatusCode),
			strings.TrimSpace(entry.ErrorMessage),
			strings.TrimSpace(entry.ErrorType),
			strings.TrimSpace(entry.ErrorCode),
		); err != nil {
			return fmt.Errorf("迁移旧版 API 延迟历史失败: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("提交 API 延迟历史迁移事务失败: %w", err)
	}
	return nil
}

func boolToSQLiteInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func sqliteNullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func sqliteNullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}
