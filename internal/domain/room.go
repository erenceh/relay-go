package domain

import (
	"time"

	"github.com/google/uuid"
)

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
