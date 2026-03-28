package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/erenceh/relay-go/internal/domain"
)

// PostgresUserRepository is a PostgreSQL-backed implementation of a user repository.
// It uses a *sql.DB handle for all database operations.
type PostgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository returns a PostgresUserRepository backed by the given *sql.DB.
func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// Create inserts a new user record into the users table.
// Returns an error if the database handle is nil or the insert fails.
func (ur *PostgresUserRepository) Create(user *domain.User) error {
	if ur.db == nil {
		return errors.New("db must not be nil")
	}

	query := `
	INSERT INTO users (id, username, password_hash, created_at)
	VALUES ($1, $2, $3, $4);
	`

	if _, err := ur.db.Exec(query, user.ID, user.Username, user.PasswordHash, user.CreatedAt); err != nil {
		return fmt.Errorf("failed to execute create user query: %w", err)
	}

	return nil
}

// FindByUsername looks up a non-deleted user by their username.
// Returns nil, nil if no matching user is found.
// Returns an error if the database handle is nil or the query fails.
func (ur *PostgresUserRepository) FindByUsername(username string) (*domain.User, error) {
	if ur.db == nil {
		return nil, errors.New("db must not be nil")
	}

	var (
		id           string
		passwordHash string
		createdAt    time.Time
		deletedAt    *time.Time
	)

	query := `
	SELECT id, password_hash, created_at, deleted_at
	FROM users
	WHERE username = $1 AND deleted_at IS NULL
	`

	row := ur.db.QueryRow(query, username)
	if err := row.Scan(&id, &passwordHash, &createdAt, &deletedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	return &domain.User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    createdAt,
		DeletedAt:    deletedAt,
	}, nil
}

// FindByID looks up a non-deleted user by their ID.
// Returns nil, nil if no matching user is found.
// Returns an error if the database handle is nil or the query fails.
func (ur *PostgresUserRepository) FindByID(id string) (*domain.User, error) {
	if ur.db == nil {
		return nil, errors.New("db must not be nil")
	}

	var (
		username     string
		passwordHash string
		createdAt    time.Time
		deletedAt    *time.Time
	)

	query := `
	SELECT username, password_hash, created_at, deleted_at
	FROM users
	WHERE id = $1 AND deleted_at IS NULL
	`

	row := ur.db.QueryRow(query, username)
	if err := row.Scan(&username, &passwordHash, &createdAt, &deletedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	return &domain.User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    createdAt,
		DeletedAt:    deletedAt,
	}, nil
}
