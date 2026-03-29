package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/erenceh/relay-go/internal/domain"
)

// PostgresRoomRepository implements RoomRepository using a PostgreSQL database.
type PostgresRoomRepository struct {
	db *sql.DB
}

// NewPostgresRoomRepository returns a PostgresRoomRepository backed by the given *sql.DB.
func NewPostgresRoomRepository(db *sql.DB) *PostgresRoomRepository {
	return &PostgresRoomRepository{db: db}
}

// Create inserts a new room record into the rooms table.
func (rr *PostgresRoomRepository) Create(room *domain.Room) error {
	if rr.db == nil {
		return fmt.Errorf("db must not be nil")
	}

	query := `
	INSERT INTO rooms (id, name, is_private, created_at)
	VALUES ($1, $2, $3, $4)
	`

	if _, err := rr.db.Exec(query, room.ID, room.Name, room.IsPrivate, room.CreatedAt); err != nil {
		return fmt.Errorf("failed to execute create room query: %w", err)
	}

	return nil
}

// FindByRoomName looks up a non-deleted room by its name. Returns nil, nil if not found.
func (rr *PostgresRoomRepository) FindByRoomName(roomName string) (*domain.Room, error) {
	if rr.db == nil {
		return nil, errors.New("db must not be nil")
	}

	var (
		id        string
		isPrivate bool
		createdAt time.Time
		deletedAt *time.Time
	)

	query := `
	SELECT id, is_private, created_at, deleted_at
	FROM rooms
	WHERE name = $1 AND deleted_at IS NULL
	`

	row := rr.db.QueryRow(query, roomName)
	if err := row.Scan(&id, &isPrivate, &createdAt, &deletedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan room: %w", err)
	}

	return &domain.Room{
		ID:        id,
		Name:      roomName,
		IsPrivate: isPrivate,
		CreatedAt: createdAt,
		DeletedAt: deletedAt,
	}, nil
}

// FindByRoomID looks up a non-deleted room by its ID. Returns nil, nil if not found.
func (rr *PostgresRoomRepository) FindByRoomID(roomID string) (*domain.Room, error) {
	if rr.db == nil {
		return nil, errors.New("db must not be nil")
	}

	var (
		name      string
		isPrivate bool
		createdAt time.Time
		deletedAt *time.Time
	)

	query := `
	SELECT name, is_private, created_at, deleted_at
	FROM rooms
	WHERE id = $1 AND deleted_at IS NULL
	`

	row := rr.db.QueryRow(query, roomID)
	if err := row.Scan(&name, &isPrivate, &createdAt, &deletedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan room: %w", err)
	}

	return &domain.Room{
		ID:        roomID,
		Name:      name,
		IsPrivate: isPrivate,
		CreatedAt: createdAt,
		DeletedAt: deletedAt,
	}, nil
}
