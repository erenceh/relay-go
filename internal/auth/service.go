package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/erenceh/relay-go/internal/domain"
	"github.com/erenceh/relay-go/internal/repository"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

// authService is a repository-backed implementation of AuthService.
// It delegates user storage to a UserRepository and signs tokens with secret.
type authService struct {
	users         repository.UserRepository
	refreshTokens map[string]string
	secret        []byte
	mu            sync.Mutex
}

// NewAuthService returns an authService that delegates user storage to users and signs tokens with secret.
func NewAuthService(users repository.UserRepository, secret []byte) *authService {
	return &authService{
		users:         users,
		refreshTokens: make(map[string]string),
		secret:        secret,
	}
}

func (as *authService) Register(username, password string) error {
	if len(username) == 0 {
		return errors.New("username must not be empty")
	}
	if len(password) == 0 {
		return errors.New("password must not be empty")
	}

	existing, err := as.users.FindByUsername(username)
	if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}
	if existing != nil {
		return errors.New("username already taken")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %w", err)
	}

	user := domain.NewUser(username, string(hash))
	if err := as.users.Create(user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (as *authService) Login(username, password string) (accessToken, refreshToken string, err error) {
	if len(username) == 0 {
		return "", "", errors.New("username must not be empty")
	}
	if len(password) == 0 {
		return "", "", errors.New("password must not be empty")
	}

	user, err := as.users.FindByUsername(username)
	if err != nil {
		return "", "", fmt.Errorf("failed to find user: %w", err)
	}
	if user == nil {
		return "", "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", fmt.Errorf("failed to verify password: %w", err)
	}

	claims := jwt.MapClaims{
		"sub": username,
		"uid": user.ID,
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

func (as *authService) Validate(tokenString string) (username, userID string, err error) {
	if len(tokenString) == 0 {
		return "", "", errors.New("token must not be empty")
	}

	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return as.secret, nil
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", "", errors.New("invalid token claims")
	}

	username, ok = claims["sub"].(string)
	if !ok {
		return "", "", errors.New("invalid token subject")
	}
	userID, ok = claims["uid"].(string)
	if !ok {
		return "", "", errors.New("invalid token user id")
	}

	return username, userID, nil
}

func (as *authService) Refresh(refreshToken string) (accessToken, newRefreshToken string, err error) {
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

	user, err := as.users.FindByUsername(username)
	if err != nil || user == nil {
		return "", "", errors.New("invalid refresh token")
	}

	claims := jwt.MapClaims{
		"sub": username,
		"uid": user.ID,
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

func (as *authService) IssueRefreshToken(username string) (refreshToken string, err error) {
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
