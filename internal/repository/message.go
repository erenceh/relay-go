package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/erenceh/relay-go/internal/domain"
)

// PostgresMessageRepository is a PostgreSQL-backed implementation of a message repository.
// It uses a *sql.DB handle for all database operations.
type PostgresMessageRepository struct {
	db *sql.DB
}

// NewPostgresMessageRepository returns a PostgresMessageRepository backed by the given *sql.DB.
func NewPostgresMessageRepository(db *sql.DB) *PostgresMessageRepository {
	return &PostgresMessageRepository{db: db}
}

// Create inserts a new message record into the messages table.
// System messages (sender ID matching domain.SystemSenderID) are silently skipped and not persisted.
// Returns an error if the database handle is nil or the insert fails.
func (mr *PostgresMessageRepository) Create(msg *domain.Message) error {
	if mr.db == nil {
		return fmt.Errorf("db must not be nil")
	}

	if msg.SenderID == domain.SystemSenderID {
		return nil
	}

	query := `
	INSERT INTO messages (id, sender_id, room_id, body, delivered, created_at)
	VALUES ($1, $2, $3, $4, $5, $6)
	`

	if _, err := mr.db.Exec(query, msg.ID, msg.SenderID, msg.RoomID, msg.Body, msg.Delivered, msg.CreatedAt); err != nil {
		return fmt.Errorf("failed to execute create message query: %w", err)
	}

	return nil
}

// ListByRoom returns up to limit non-deleted messages for the given room, ordered newest first.
// Joins against users to populate the From field with the sender's username.
// Returns an error if the database handle is nil or the query fails.
func (mr *PostgresMessageRepository) ListByRoom(roomID string, limit int) ([]*domain.Message, error) {
	if mr.db == nil {
		return nil, fmt.Errorf("db must not be nil")
	}

	var (
		id        string
		senderID  string
		roomIDRes string
		body      string
		delivered bool
		createdAt time.Time
		deletedAt *time.Time
		from      string
	)

	query := `
	SELECT m.id, m.sender_id, m.room_id, m.body, m.delivered, 
		m.created_at, m.deleted_at, u.username as from
	FROM messages m
	JOIN users u ON u.id = m.sender_id
	WHERE m.room_id = $1 AND m.deleted_at IS NULL
	ORDER BY m.created_at DESC
	LIMIT $2
	`

	rows, err := mr.db.Query(query, roomID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages table: %w", err)
	}
	defer rows.Close()

	messages := make([]*domain.Message, 0, limit)
	for rows.Next() {
		err := rows.Scan(&id, &senderID, &roomIDRes, &body, &delivered, &createdAt, &deletedAt, &from)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row in messages: %w", err)
		}

		messages = append(messages, &domain.Message{
			ID:        id,
			SenderID:  senderID,
			RoomID:    roomIDRes,
			From:      from,
			Body:      body,
			Delivered: delivered,
			CreatedAt: createdAt,
			DeletedAt: deletedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate messages row: %w", err)
	}

	return messages, nil
}

// ListUndelivered returns all undelivered messages in private rooms where the given user
// is a member, excluding messages sent by that user.
// Returns an error if the database handle is nil or the query fails.
func (mr *PostgresMessageRepository) ListUndelivered(userID string) ([]*domain.Message, error) {
	if mr.db == nil {
		return nil, fmt.Errorf("db must not be nil")
	}

	var (
		id        string
		senderID  string
		roomID    string
		body      string
		delivered bool
		createdAt time.Time
		deletedAt *time.Time
		from      string
	)

	query := `
	SELECT m.id, m.sender_id, m.room_id, m.body, m.delivered,
		m.created_at, m.deleted_at, u.username as from
	FROM messages m
	JOIN users u ON u.id = m.sender_id
	JOIN rooms r ON r.id = m.room_id
	JOIN room_members rm ON rm.room_id = r.id
	WHERE rm.user_id = $1
	AND m.sender_id != $1
	AND r.is_private = true
	AND m.delivered = false
	AND m.deleted_at IS NULL
	`

	rows, err := mr.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages table: %w", err)
	}
	defer rows.Close()

	messages := make([]*domain.Message, 0, 64)
	for rows.Next() {
		err := rows.Scan(&id, &senderID, &roomID, &body, &delivered, &createdAt, &deletedAt, &from)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row in messages: %w", err)
		}

		messages = append(messages, &domain.Message{
			ID:        id,
			SenderID:  senderID,
			RoomID:    roomID,
			From:      from,
			Body:      body,
			Delivered: delivered,
			CreatedAt: createdAt,
			DeletedAt: deletedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate messages row: %w", err)
	}

	return messages, nil
}

// MarkDelivered sets the delivered flag to true for the given message ID.
// Returns an error if the database handle is nil or the update fails.
func (mr *PostgresMessageRepository) MarkDelivered(msgID string) error {
	if mr.db == nil {
		return fmt.Errorf("db must not be nil")
	}

	query := `
	UPDATE messages
	SET delivered = true
	WHERE id = $1
	`

	if _, err := mr.db.Exec(query, msgID); err != nil {
		return fmt.Errorf("failed to execute update message query: %w", err)
	}

	return nil
}

// SoftDelete sets deleted_at to the current time for the given message ID,
// hiding it from future queries without removing the row.
// Returns an error if the database handle is nil or the update fails.
func (mr *PostgresMessageRepository) SoftDelete(msgID string) error {
	if mr.db == nil {
		return fmt.Errorf("db must not be nil")
	}

	query := `
	UPDATE messages
	SET deleted_at = NOW()
	WHERE id = $1
	AND deleted_at IS NULL
	`

	if _, err := mr.db.Exec(query, msgID); err != nil {
		return fmt.Errorf("failed to execute update message query: %w", err)
	}

	return nil
}
