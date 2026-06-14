package domain

// UserJoinedEvent is emitted when a user joins a room.
type UserJoinedEvent struct {
	Username string
	RoomName string
}

// UserLeftEvent is emitted when a user leaves a room.
type UserLeftEvent struct {
	Username string
	RoomName string
}

// MessageSentEvent is emitted when a user sends a message to a room.
type MessageSentEvent struct {
	Sender   string
	RoomName string
	Message  string
}

// UserPresenceEvent is emitted when a user's online status changes.
type UserPresenceEvent struct {
	Username string
	IsOnline bool
}
