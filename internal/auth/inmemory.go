package auth

import (
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
	mu     sync.Mutex
	users  map[string]*User
	secret []byte
}

// NewInMemoryAuthService returns an InMemoryAuthService that signs tokens with secret.
func NewInMemoryAuthService(secret []byte) *InMemoryAuthService {
	return &InMemoryAuthService{
		users:  make(map[string]*User),
		secret: secret,
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
func (as *InMemoryAuthService) Login(username, password string) (string, error) {
	if len(username) == 0 {
		return "", errors.New("username must not be empty")
	}
	if len(password) == 0 {
		return "", errors.New("password must not be empty")
	}

	user, ok := as.users[username]
	if !ok {
		return "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("failed to verify password: %w", err)
	}

	claims := jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(as.secret)
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w", err)
	}

	return signed, nil
}

// Validate checks the session token and returns the associated username.
func (as *InMemoryAuthService) Validate(tokenString string) (string, error) {
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

	username, ok := claims["sub"].(string)
	if !ok {
		return "", errors.New("invalid token subject")
	}

	return username, nil
}
