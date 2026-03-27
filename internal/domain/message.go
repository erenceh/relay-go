package domain

import (
	"time"

	"github.com/google/uuid"
)

// Message represents a chat message with a sender and body.
type Message struct {
	ID        string
	SenderID  string
	RoomID    string
	From      string
	Body      string
	Delivered bool
	CreatedAt time.Time
	DeletedAt *time.Time
}

// NewMessage returns an initialized Message with the given senderID, roomID, and body.
func NewMessage(senderID, roomID, from, body string) Message {
	return Message{
		ID:        uuid.New().String(),
		SenderID:  senderID,
		RoomID:    roomID,
		From:      from,
		Body:      body,
		Delivered: false,
		CreatedAt: time.Now(),
	}
}
