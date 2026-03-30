package domain

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var roomNameRegex = regexp.MustCompile(`^[a-zA-Z0-9 _'-]+$`)

// Room represents a chat room with a name and its currently connected members.
type Room struct {
	ID        string
	Name      string
	IsPrivate bool
	CreatedAt time.Time
	DeletedAt *time.Time
}

// NewRoom returns an initialized Room with the given name.
func NewRoom(name string) Room {
	return Room{
		ID:        uuid.New().String(),
		Name:      name,
		IsPrivate: false,
		CreatedAt: time.Now(),
	}
}

func ValidateRoomName(name string) error {
	if len(name) < 3 {
		return errors.New("room name must be at least 3 characters")
	}
	if len(name) > 32 {
		return errors.New("room name must be at most 32 characters")
	}
	if !roomNameRegex.MatchString(name) {
		return errors.New("room name may only contain letters, numbers, spaces, underscores, hyphens, and apostrophes")
	}
	return nil
}
