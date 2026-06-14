package repository

import (
	"database/sql"
	"errors"
	"fmt"
)

// PostgresRoomMemberRepository is a PostgreSQL-backed implementation of a room membership repository.
// It uses a *sql.DB handle for all database operations against the room_members join table.
type PostgresRoomMemberRepository struct {
	db *sql.DB
}

// NewPostgresRoomMemberRepository returns a PostgresRoomMemberRepository backed by the given *sql.DB.
func NewPostgresRoomMemberRepository(db *sql.DB) *PostgresRoomMemberRepository {
	return &PostgresRoomMemberRepository{db: db}
}

// Add inserts a membership record linking the given user to the given room.
// Returns an error if the database handle is nil or the insert fails.
func (rmr *PostgresRoomMemberRepository) Add(roomID, userID string) error {
	if rmr.db == nil {
		return errors.New("db must not be nil")
	}

	query := `
	INSERT INTO room_members (room_id, user_id)
	VALUES ($1, $2)
	`

	if _, err := rmr.db.Exec(query, roomID, userID); err != nil {
		return fmt.Errorf("failed to execute add room member query: %w", err)
	}

	return nil
}

// Remove deletes the membership record for the given user in the given room.
// Returns an error if the database handle is nil or the delete fails.
func (rmr *PostgresRoomMemberRepository) Remove(roomID, userID string) error {
	if rmr.db == nil {
		return errors.New("db must not be nil")
	}

	query := `
	DELETE FROM room_members
	WHERE room_id = $1 AND user_id = $2
	`

	if _, err := rmr.db.Exec(query, roomID, userID); err != nil {
		return fmt.Errorf("failed to execute remove room member query: %w", err)
	}

	return nil
}

// ListByRoom returns the user IDs of all members belonging to the given room.
// Returns an error if the database handle is nil or the query fails.
func (rmr *PostgresRoomMemberRepository) ListByRoom(roomID string) ([]string, error) {
	if rmr.db == nil {
		return nil, errors.New("db must not be nil")
	}

	query := `
	SELECT user_id
	FROM room_members
	WHERE room_id = $1
	`

	rows, err := rmr.db.Query(query, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to query room_members table: %w", err)
	}
	defer rows.Close()

	userIDs := make([]string, 0)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("failed to scan row in room_members: %w", err)
		}
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate room_members rows: %w", err)
	}

	return userIDs, nil
}

// ListByUser returns the room IDs of all rooms the given user is a member of.
// Returns an error if the database handle is nil or the query fails.
func (rmr *PostgresRoomMemberRepository) ListByUser(userID string) ([]string, error) {
	if rmr.db == nil {
		return nil, errors.New("db must not be nil")
	}

	query := `
	SELECT room_id
	FROM room_members
	WHERE user_id = $1
	`

	rows, err := rmr.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query room_members table: %w", err)
	}
	defer rows.Close()

	roomIDs := make([]string, 0)
	for rows.Next() {
		var roomID string
		if err := rows.Scan(&roomID); err != nil {
			return nil, fmt.Errorf("failed to scan row in room_members: %w", err)
		}
		roomIDs = append(roomIDs, roomID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate room_members rows: %w", err)
	}

	return roomIDs, nil
}
