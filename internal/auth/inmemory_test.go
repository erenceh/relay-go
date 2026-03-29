package auth

import (
	"testing"
	"time"

	"github.com/erenceh/relay-go/internal/domain"
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
				svc.users["user"] = domain.NewUser("user", "123")
				err := svc.Register("user", "")
				assert.Error(t, err)
			},
		},
		{
			name: "Login returns a non-empty token on successful login",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				err := svc.Register("user", "password")
				require.NoError(t, err)
				accessToken, refreshToken, err := svc.Login("user", "password")
				require.NoError(t, err)
				require.Greater(t, len(accessToken), 0)
				require.Greater(t, len(refreshToken), 0)
			},
		},
		{
			name: "Login returns error if username or password does not exist",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				_, _, err := svc.Login("user", "password")
				assert.Error(t, err)
			},
		},
		{
			name: "Login returns error if username field is empty",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				_, _, err := svc.Login("", "password")
				assert.Error(t, err)
			},
		},
		{
			name: "Login returns error if password field is empty",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				_, _, err := svc.Login("user", "")
				assert.Error(t, err)
			},
		},
		{
			name: "Validate returns correct username on successful token validation",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				require.NoError(t, svc.Register("user", "password"))
				accessToken, _, err := svc.Login("user", "password")
				require.NoError(t, err)
				username, _, err := svc.Validate(accessToken)
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
				_, _, err := svc.Validate(expired)
				assert.Error(t, err)
			},
		},
		{
			name: "Validate returns error if token was tampered",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				require.NoError(t, svc.Register("user", "password"))
				accessToken, _, _ := svc.Login("user", "password")
				tampered := accessToken + "x"
				_, _, err := svc.Validate(tampered)
				assert.Error(t, err)
			},
		},
		{
			name: "Validate returns error if token is empty",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				_, _, err := svc.Validate("")
				assert.Error(t, err)
			},
		},
		{
			name: "Refresh returns error if token is empty",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				_, _, err := svc.Refresh("")
				assert.Error(t, err)
			},
		},
		{
			name: "Refresh returns error for unknown refresh token",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				_, _, err := svc.Refresh("not-a-real-token")
				assert.Error(t, err)
			},
		},
		{
			name: "Refresh returns new access and refresh tokens for a valid refresh token",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				require.NoError(t, svc.Register("user", "password"))
				refreshToken, err := svc.IssueRefreshToken("user")
				require.NoError(t, err)
				accessToken, newRefreshToken, err := svc.Refresh(refreshToken)
				require.NoError(t, err)
				assert.Greater(t, len(accessToken), 0)
				assert.Greater(t, len(newRefreshToken), 0)
			},
		},
		{
			name: "Refresh new access token contains correct username",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				require.NoError(t, svc.Register("user", "password"))
				refreshToken, err := svc.IssueRefreshToken("user")
				require.NoError(t, err)
				accessToken, _, err := svc.Refresh(refreshToken)
				require.NoError(t, err)
				username, _, err := svc.Validate(accessToken)
				require.NoError(t, err)
				assert.Equal(t, "user", username)
			},
		},
		{
			name: "Refresh invalidates the old refresh token after use",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				require.NoError(t, svc.Register("user", "password"))
				refreshToken, err := svc.IssueRefreshToken("user")
				require.NoError(t, err)
				_, _, err = svc.Refresh(refreshToken)
				require.NoError(t, err)
				_, _, err = svc.Refresh(refreshToken)
				assert.Error(t, err)
			},
		},
		{
			name: "IssueRefreshToken returns error if username is empty",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				_, err := svc.IssueRefreshToken("")
				assert.Error(t, err)
			},
		},
		{
			name: "IssueRefreshToken returns a non-empty token for a valid username",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				token, err := svc.IssueRefreshToken("user")
				require.NoError(t, err)
				assert.Greater(t, len(token), 0)
			},
		},
		{
			name: "IssueRefreshToken issued token can be used with Refresh",
			run: func(t *testing.T, svc *InMemoryAuthService) {
				require.NoError(t, svc.Register("user", "password"))
				token, err := svc.IssueRefreshToken("user")
				require.NoError(t, err)
				_, _, err = svc.Refresh(token)
				assert.NoError(t, err)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewInMemoryAuthService([]byte("test-secret"))
			tt.run(t, svc)
		})
	}
}
