package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func(t *testing.T, svc *InMemoryAuthService)
	}{
		{
			name: "Register successfully adds user to in-memory users",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				require.NoError(t, svc.Register("user", "password"))
			},
		},
		{
			name: "Register returns error if username field is empty",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				err := svc.Register("", "password")
				assert.Error(t, err)
			},
		},
		{
			name: "Register returns error if password field is empty",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				err := svc.Register("user", "")
				assert.Error(t, err)
			},
		},
		{
			name: "Register returns error if named user exists",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				svc.users["user"] = NewUser("user", "123")
				err := svc.Register("user", "")
				assert.Error(t, err)
			},
		},
		{
			name: "Login returns a non-empty token on successful login",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				err := svc.Register("user", "password")
				require.NoError(t, err)
				signed, err := svc.Login("user", "password")
				require.NoError(t, err)
				require.Greater(t, len(signed), 0)
			},
		},
		{
			name: "Login returns error if username or password does not exist",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				_, err := svc.Login("user", "password")
				assert.Error(t, err)
			},
		},
		{
			name: "Login returns error if username field is empty",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				_, err := svc.Login("", "password")
				assert.Error(t, err)
			},
		},
		{
			name: "Login returns error if password field is empty",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				_, err := svc.Login("user", "")
				assert.Error(t, err)
			},
		},
		{
			name: "Validate returns correct username on successful token validation",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				require.NoError(t, svc.Register("user", "password"))
				token, err := svc.Login("user", "password")
				require.NoError(t, err)
				username, err := svc.Validate(token)
				require.NoError(t, err)
				assert.Equal(t, "user", username)
			},
		},
		{
			name: "Validate returns error if token is expired",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				claims := jwt.MapClaims{
					"sub": "user",
					"exp": time.Now().Add(-1 * time.Hour).Unix(),
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				expired, _ := token.SignedString([]byte("test-secret"))
				_, err := svc.Validate(expired)
				assert.Error(t, err)
			},
		},
		{
			name: "Validate returns error if token was tampered",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				require.NoError(t, svc.Register("user", "password"))
				token, _ := svc.Login("user", "password")
				tampered := token + "x"
				_, err := svc.Validate(tampered)
				assert.Error(t, err)
			},
		},
		{
			name: "Validate returns error if token is empty",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				_, err := svc.Validate("")
				assert.Error(t, err)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewInMemoryAuthService([]byte("test-secret"))
			tt.run(t, svc)
		})
	}
}
