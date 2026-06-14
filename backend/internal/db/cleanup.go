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

	allMessagesQuery := `
	DELETE FROM messages
	WHERE created_at < NOW() - INTERVAL '1 day' * $1
	`
	if _, err := db.Exec(allMessagesQuery, days); err != nil {
		return fmt.Errorf("failed to execute delete old messages query: %w", err)
	}

	allRoomQuery := `
	DELETE FROM rooms
	WHERE created_at < NOW() - INTERVAL '1 day' * $1
	AND deleted_at IS NOT NULL
	`
	if _, err := db.Exec(allRoomQuery, days); err != nil {
		return fmt.Errorf("failed to execute delete old rooms query: %w", err)
	}

	slog.Info("cleanup worker completed")
	return nil
}
