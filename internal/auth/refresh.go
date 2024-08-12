package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

type RefreshManagerInterface interface {
	GenerateRefreshToken(userID string, duration time.Duration) (string, error)
	VerifyRefreshToken(token string) (string, error)
}

var (
	ErrInvalidRefreshToken = errors.New("refresh token is invalid")
	ErrExpiredRefreshToken = errors.New("refresh token is expired")
)

type RefreshTokenManager struct {
	tokens map[string]string
	mu     sync.Mutex
}

func NewRefreshTokenManager() RefreshManagerInterface {
	return &RefreshTokenManager{
		tokens: make(map[string]string),
	}
}

func (m *RefreshTokenManager) GenerateRefreshToken(userID string, duration time.Duration) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", ErrInternalError
	}
	refreshToken := hex.EncodeToString(tokenBytes)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[refreshToken] = userID

	go func() {
		time.Sleep(duration)
		m.mu.Lock()
		delete(m.tokens, refreshToken)
		m.mu.Unlock()
	}()

	return refreshToken, nil
}

func (m *RefreshTokenManager) VerifyRefreshToken(token string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	userID, exists := m.tokens[token]
	if !exists {
		return "", errors.New("invalid or expired refresh token")
	}

	return userID, nil
}
