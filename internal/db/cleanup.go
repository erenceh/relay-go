package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

func StartCleanupWorker(db *sql.DB, retention time.Duration) {
	go func() {
		for {
			if err := runCleanup(db, retention); err != nil {
				slog.Warn("cleanup worker failed", "err", err)
			}
			time.Sleep(24 * time.Hour)
		}
	}()
}

func runCleanup(db *sql.DB, retention time.Duration) error {
	days := retention.Hours() / 24
	slog.Info("running cleanup worker", "retention_days", days)

	messagesQuery := `
	DELETE FROM messages
	WHERE deleted_at IS NOT NULL
	AND deleted_at < NOW() - INTERVAL '1 day' * $1
	`

	if _, err := db.Exec(messagesQuery, days); err != nil {
		return fmt.Errorf("failed to execute delete room query: %w", err)
	}

	roomsQuery := `
	DELETE FROM rooms
	WHERE deleted_at IS NOT NULL
	AND deleted_at < NOW() - INTERVAL '1 day' * $1
	`

	if _, err := db.Exec(roomsQuery, days); err != nil {
		return fmt.Errorf("failed to execute delete room query: %w", err)
	}
	slog.Info("cleanup worker completed")

	return nil
}
