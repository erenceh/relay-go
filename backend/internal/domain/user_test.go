package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:  "valid lowercase",
			input: "alice",
		},
		{
			name:  "valid with numbers",
			input: "user123",
		},
		{
			name:  "valid with underscore",
			input: "alice_bob",
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
			wantErr: "username must be at least 3 characters",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: "username must be at least 3 characters",
		},
		{
			name:    "too long (33 chars)",
			input:   strings.Repeat("a", 33),
			wantErr: "username must be at most 32 characters",
		},
		{
			name:    "invalid with space",
			input:   "alice bob",
			wantErr: "username may only contain letters, numbers, and underscores",
		},
		{
			name:    "invalid with hyphen",
			input:   "alice-bob",
			wantErr: "username may only contain letters, numbers, and underscores",
		},
		{
			name:    "invalid with special char",
			input:   "alice@bob",
			wantErr: "username may only contain letters, numbers, and underscores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.input)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestNewUser(t *testing.T) {
	tests := []struct {
		name           string
		username       string
		hashedPassword string
	}{
		{
			name:           "creates user with hashed password",
			username:       "alice",
			hashedPassword: "$2a$10$hashedvalue",
		},
		{
			name:           "creates user with underscore username",
			username:       "alice_bob",
			hashedPassword: "$2a$10$anotherhash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := NewUser(tt.username, tt.hashedPassword)

			assert.NotEmpty(t, u.ID)
			assert.Equal(t, tt.username, u.Username)
			assert.Equal(t, tt.hashedPassword, u.PasswordHash)
			assert.False(t, u.CreatedAt.IsZero())
			assert.Nil(t, u.DeletedAt)
		})
	}
}
