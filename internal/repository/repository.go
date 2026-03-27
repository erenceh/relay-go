package repository

import (
	"github.com/erenceh/relay-go/internal/domain"
)

type UserRepository interface {
	Create(user *domain.User) error
	FindByUsername(username string) (*domain.User, error)
}

type MessageRepository interface {
	Create(msg *domain.Message) error
	ListByRoom(roomID string, limit int) ([]*domain.Message, error)
	ListUndelivered(userID string) ([]*domain.Message, error)
	MarkDelivered(msgID string) error
	SoftDelete(msgID string) error
}
