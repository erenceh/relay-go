package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents an authenticated user account.
type User struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	DeletedAt    *time.Time
}

// NewUser constructs a User from an already-hashed password.
func NewUser(username, hashedPassword string) *User {
	return &User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: hashedPassword,
		CreatedAt:    time.Now(),
	}
}
