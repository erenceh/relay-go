package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateRoomName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:  "valid alphanumeric",
			input: "general",
		},
		{
			name:  "valid with spaces",
			input: "general chat",
		},
		{
			name:  "valid with underscores",
			input: "off_topic",
		},
		{
			name:  "valid with mixed chars",
			input: "Room 1_A",
		},
		{
			name:  "exactly 3 characters",
			input: "abc",
		},
		{
			name:  "exactly 32 characters",
			input: strings.Repeat("a", 32),
		},
		{
			name:    "too short (2 chars)",
			input:   "ab",
			wantErr: "room name must be at least 3 characters",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: "room name must be at least 3 characters",
		},
		{
			name:    "too long (33 chars)",
			input:   strings.Repeat("a", 33),
			wantErr: "room name must be at most 32 characters",
		},
		{
			name:    "invalid special chars",
			input:   "room#1",
			wantErr: "room name may only contain letters, numbers, spaces, and underscores",
		},
		{
			name:    "invalid with slash",
			input:   "room/chat",
			wantErr: "room name may only contain letters, numbers, spaces, and underscores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRoomName(tt.input)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestNewRoom(t *testing.T) {
	tests := []struct {
		name     string
		roomName string
		wantPriv bool
	}{
		{
			name:     "creates public room",
			roomName: "general",
			wantPriv: false,
		},
		{
			name:     "creates room with spaces in name",
			roomName: "general chat",
			wantPriv: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRoom(tt.roomName)

			assert.NotEmpty(t, r.ID)
			assert.Equal(t, tt.roomName, r.Name)
			assert.Equal(t, tt.wantPriv, r.IsPrivate)
			assert.False(t, r.CreatedAt.IsZero())
			assert.Nil(t, r.DeletedAt)
		})
	}
}
