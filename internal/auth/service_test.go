package auth

import (
	"testing"
	"time"

	"github.com/erenceh/relay-go/internal/domain"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUserRepo struct {
	users map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*domain.User)}
}

func (m *mockUserRepo) Create(user *domain.User) error {
	m.users[user.Username] = user
	return nil
}

func (m *mockUserRepo) FindByUsername(username string) (*domain.User, error) {
	user, ok := m.users[username]
	if !ok {
		return nil, nil
	}
	return user, nil
}

func (m *mockUserRepo) FindByID(id string) (*domain.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, nil
}

func newAuthService() AuthService {
	return NewAuthService(newMockUserRepo(), []byte("test-secret"))
}

func TestAuthService(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func(t *testing.T, svc AuthService)
	}{
		{
			name: "Register successfully adds user",
			run: func(t *testing.T, svc AuthService) {
				require.NoError(t, svc.Register("user", "password"))
			},
		},
		{
			name: "Register returns error if username field is empty",
			run: func(t *testing.T, svc AuthService) {
				err := svc.Register("", "password")
				assert.Error(t, err)
			},
		},
		{
			name: "Register returns error if password field is empty",
			run: func(t *testing.T, svc AuthService) {
				err := svc.Register("user", "")
				assert.Error(t, err)
			},
		},
		{
			name: "Register returns error if named user exists",
			run: func(t *testing.T, svc AuthService) {
				require.NoError(t, svc.Register("user", "password"))
				err := svc.Register("user", "password")
				assert.Error(t, err)
			},
		},
		{
			name: "Register returns error for invalid username",
			run: func(t *testing.T, svc AuthService) {
				err := svc.Register("a", "password")
				assert.Error(t, err)
			},
		},
		{
			name: "Register returns error for username with special chars",
			run: func(t *testing.T, svc AuthService) {
				err := svc.Register("user@name", "password")
				assert.Error(t, err)
			},
		},
		{
			name: "Login returns a non-empty token on successful login",
			run: func(t *testing.T, svc AuthService) {
				require.NoError(t, svc.Register("user", "password"))
				accessToken, refreshToken, err := svc.Login("user", "password")
				require.NoError(t, err)
				assert.Greater(t, len(accessToken), 0)
				assert.Greater(t, len(refreshToken), 0)
			},
		},
		{
			name: "Login returns error if username or password does not exist",
			run: func(t *testing.T, svc AuthService) {
				_, _, err := svc.Login("user", "password")
				assert.Error(t, err)
			},
		},
		{
			name: "Login returns error if username field is empty",
			run: func(t *testing.T, svc AuthService) {
				_, _, err := svc.Login("", "password")
				assert.Error(t, err)
			},
		},
		{
			name: "Login returns error if password field is empty",
			run: func(t *testing.T, svc AuthService) {
				_, _, err := svc.Login("user", "")
				assert.Error(t, err)
			},
		},
		{
			name: "Validate returns correct username on successful token validation",
			run: func(t *testing.T, svc AuthService) {
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
			run: func(t *testing.T, svc AuthService) {
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
			run: func(t *testing.T, svc AuthService) {
				require.NoError(t, svc.Register("user", "password"))
				accessToken, _, _ := svc.Login("user", "password")
				tampered := accessToken + "x"
				_, _, err := svc.Validate(tampered)
				assert.Error(t, err)
			},
		},
		{
			name: "Validate returns error if token is empty",
			run: func(t *testing.T, svc AuthService) {
				_, _, err := svc.Validate("")
				assert.Error(t, err)
			},
		},
		{
			name: "Refresh returns error if token is empty",
			run: func(t *testing.T, svc AuthService) {
				_, _, err := svc.Refresh("")
				assert.Error(t, err)
			},
		},
		{
			name: "Refresh returns error for unknown refresh token",
			run: func(t *testing.T, svc AuthService) {
				_, _, err := svc.Refresh("not-a-real-token")
				assert.Error(t, err)
			},
		},
		{
			name: "Refresh returns new access and refresh tokens for a valid refresh token",
			run: func(t *testing.T, svc AuthService) {
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
			run: func(t *testing.T, svc AuthService) {
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
			run: func(t *testing.T, svc AuthService) {
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
			run: func(t *testing.T, svc AuthService) {
				_, err := svc.IssueRefreshToken("")
				assert.Error(t, err)
			},
		},
		{
			name: "IssueRefreshToken returns a non-empty token for a valid username",
			run: func(t *testing.T, svc AuthService) {
				token, err := svc.IssueRefreshToken("user")
				require.NoError(t, err)
				assert.Greater(t, len(token), 0)
			},
		},
		{
			name: "IssueRefreshToken issued token can be used with Refresh",
			run: func(t *testing.T, svc AuthService) {
				require.NoError(t, svc.Register("user", "password"))
				token, err := svc.IssueRefreshToken("user")
				require.NoError(t, err)
				_, _, err = svc.Refresh(token)
				assert.NoError(t, err)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t, newAuthService())
		})
	}
}
