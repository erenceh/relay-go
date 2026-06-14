package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserJoinedEvent(t *testing.T) {
	tests := []struct {
		name     string
		username string
		roomName string
	}{
		{
			name:     "basic join event",
			username: "alice",
			roomName: "general",
		},
		{
			name:     "join event with room name containing spaces",
			username: "bob",
			roomName: "general chat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := UserJoinedEvent{Username: tt.username, RoomName: tt.roomName}

			assert.Equal(t, tt.username, e.Username)
			assert.Equal(t, tt.roomName, e.RoomName)
		})
	}
}

func TestUserLeftEvent(t *testing.T) {
	tests := []struct {
		name     string
		username string
		roomName string
	}{
		{
			name:     "basic left event",
			username: "alice",
			roomName: "general",
		},
		{
			name:     "left event with underscore username",
			username: "alice_bob",
			roomName: "off_topic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := UserLeftEvent{Username: tt.username, RoomName: tt.roomName}

			assert.Equal(t, tt.username, e.Username)
			assert.Equal(t, tt.roomName, e.RoomName)
		})
	}
}

func TestMessageSentEvent(t *testing.T) {
	tests := []struct {
		name     string
		sender   string
		roomName string
		message  string
	}{
		{
			name:     "basic message event",
			sender:   "alice",
			roomName: "general",
			message:  "hello world",
		},
		{
			name:     "message event with longer message",
			sender:   "bob",
			roomName: "off_topic",
			message:  "this is a longer message with multiple words",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := MessageSentEvent{Sender: tt.sender, RoomName: tt.roomName, Message: tt.message}

			assert.Equal(t, tt.sender, e.Sender)
			assert.Equal(t, tt.roomName, e.RoomName)
			assert.Equal(t, tt.message, e.Message)
		})
	}
}

func TestUserPresenceEvent(t *testing.T) {
	tests := []struct {
		name     string
		username string
		isOnline bool
	}{
		{
			name:     "user comes online",
			username: "alice",
			isOnline: true,
		},
		{
			name:     "user goes offline",
			username: "bob",
			isOnline: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := UserPresenceEvent{Username: tt.username, IsOnline: tt.isOnline}

			assert.Equal(t, tt.username, e.Username)
			assert.Equal(t, tt.isOnline, e.IsOnline)
		})
	}
}
