package domain

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

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

func ValidateUsername(username string) error {
	if len(username) < 3 {
		return errors.New("username must be at least 3 characters")
	}
	if len(username) > 32 {
		return errors.New("username must be at most 32 characters")
	}
	if !usernameRegex.MatchString(username) {
		return errors.New("username may only contain letters, numbers, and underscores")
	}
	return nil
}
