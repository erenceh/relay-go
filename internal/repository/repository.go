package repository

import (
	"github.com/erenceh/relay-go/internal/domain"
)

type UserRepository interface {
	// Create inserts a new user record into the users table.
	Create(user *domain.User) error
	// FindByUsername looks up a non-deleted user by their username.
	FindByUsername(username string) (*domain.User, error)
	// FindByID looks up a non-deleted user by their ID.
	FindByID(id string) (*domain.User, error)
}

// MessageRepository defines persistence operations for chat messages.
type MessageRepository interface {
	// Create inserts a new message. System messages are silently skipped.
	Create(msg *domain.Message) error
	// ListByRoom returns up to limit non-deleted messages for the given room, ordered newest first.
	ListByRoom(roomID string, limit int) ([]*domain.Message, error)
	// ListUndelivered returns all undelivered private messages addressed to the given user.
	ListUndelivered(userID string) ([]*domain.Message, error)
	// MarkDelivered sets the delivered flag to true for the given message ID.
	MarkDelivered(msgID string) error
	// SoftDelete sets deleted_at on the given message, hiding it from future queries.
	SoftDelete(msgID string) error
}

// RoomMemberRepository defines persistence operations for room membership.
type RoomMemberRepository interface {
	// Add records that a user has joined a room.
	Add(roomID, userID string) error
	// Remove records that a user has left a room.
	Remove(roomID, userID string) error
	// ListByRoom returns the user IDs of all members in the given room.
	ListByRoom(roomID string) ([]string, error)
	// ListByUser returns the room IDs that the given user is a member of.
	ListByUser(userID string) ([]string, error)
}
