package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

// InMemoryAuthService is an in-process implementation of AuthService.
// User records and credentials are stored in a mutex-protected map with no persistence.
// It is safe for concurrent use.
type InMemoryAuthService struct {
	mu            sync.Mutex
	users         map[string]*User
	refreshTokens map[string]string
	secret        []byte
}

// NewInMemoryAuthService returns an InMemoryAuthService that signs tokens with secret.
func NewInMemoryAuthService(secret []byte) *InMemoryAuthService {
	return &InMemoryAuthService{
		users:         make(map[string]*User),
		refreshTokens: make(map[string]string),
		secret:        secret,
	}
}

// Register creates a new user with the given username and plaintext password.
// Returns an error if the username is already taken or the input is invalid.
func (as *InMemoryAuthService) Register(username, password string) error {
	if len(username) == 0 {
		return errors.New("username must not be empty")
	}
	if len(password) == 0 {
		return errors.New("password must not be empty")
	}

	as.mu.Lock()
	defer as.mu.Unlock()

	_, ok := as.users[username]
	if ok {
		return errors.New("that user already exist")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %w", err)
	}

	user := NewUser(username, string(hash))
	as.users[username] = user

	return nil
}

// Login authenticates the user and returns a session token on success.
func (as *InMemoryAuthService) Login(username, password string) (accessToken, refreshToken string, err error) {
	if len(username) == 0 {
		return "", "", errors.New("username must not be empty")
	}
	if len(password) == 0 {
		return "", "", errors.New("password must not be empty")
	}

	user, ok := as.users[username]
	if !ok {
		return "", "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", fmt.Errorf("failed to verify password: %w", err)
	}

	claims := jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err = token.SignedString(as.secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to create token: %w", err)
	}

	refreshToken, err = as.IssueRefreshToken(username)
	if err != nil {
		return "", "", fmt.Errorf("failed to issue refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// Validate checks the session token and returns the associated username.
func (as *InMemoryAuthService) Validate(tokenString string) (username string, err error) {
	if len(tokenString) == 0 {
		return "", errors.New("token must not be empty")
	}

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return as.secret, nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token claims")
	}

	username, ok = claims["sub"].(string)
	if !ok {
		return "", errors.New("invalid token subject")
	}

	return username, nil
}

// Refresh validates the given refresh token, revokes it, and returns a new access token
// and a new refresh token. Returns an error if the token is missing or unrecognized.
func (as *InMemoryAuthService) Refresh(refreshToken string) (accessToken, newRefreshToken string, err error) {
	if len(refreshToken) == 0 {
		return "", "", errors.New("token must not be empty")
	}

	as.mu.Lock()
	defer as.mu.Unlock()

	username, ok := as.refreshTokens[refreshToken]
	if !ok {
		return "", "", errors.New("invalid refresh token")
	}

	delete(as.refreshTokens, refreshToken)

	claims := jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err = token.SignedString(as.secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to create access token: %w", err)
	}

	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	newRefreshToken = hex.EncodeToString(bytes)
	as.refreshTokens[newRefreshToken] = username

	return accessToken, newRefreshToken, nil
}

// IssueRefreshToken generates a cryptographically random refresh token for the given username
// and stores it for later validation. Returns an error if the username is empty or token generation fails.
func (as *InMemoryAuthService) IssueRefreshToken(username string) (refreshToken string, err error) {
	if len(username) == 0 {
		return "", errors.New("username must not be empty")
	}

	as.mu.Lock()
	defer as.mu.Unlock()

	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshToken = hex.EncodeToString(bytes)
	as.refreshTokens[refreshToken] = username

	return refreshToken, nil
}
