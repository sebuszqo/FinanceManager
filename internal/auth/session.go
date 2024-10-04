package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalidSessionToken = errors.New("session token is invalid")
	ErrExpiredSessionToken = errors.New("token is expired")
)

const defaultSessionTokenDuration = 5 * time.Minute // 30 days

type SessionManagerInterface interface {
	VerifySessionToken(sessionToken string) (string, error)
	DeleteSessionToken(sessionToken string)
	StartSessionTokenCleanup(interval time.Duration)
	GenerateSessionToken(userID string, duration time.Duration) (string, error)
}

type SessionToken struct {
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type SessionManager struct {
	mu     sync.RWMutex
	tokens map[string]SessionToken
}

func NewSessionManager() SessionManagerInterface {
	return &SessionManager{
		tokens: make(map[string]SessionToken),
	}
}

func (sm *SessionManager) VerifySessionToken(sessionToken string) (string, error) {
	sm.mu.RLock()
	token, exists := sm.tokens[sessionToken]
	sm.mu.RUnlock()

	if !exists {
		return "", ErrInvalidSessionToken
	}

	if time.Now().After(token.ExpiresAt) {
		return "", ErrExpiredSessionToken
	}

	return token.UserID, nil
}

func (sm *SessionManager) DeleteSessionToken(sessionToken string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.tokens, sessionToken)
}

func (sm *SessionManager) StartSessionTokenCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			sm.mu.Lock()
			for token, session := range sm.tokens {
				if time.Now().After(session.ExpiresAt) {
					delete(sm.tokens, token)
				}
			}
			sm.mu.Unlock()
		}
	}()
}

func (sm *SessionManager) GenerateSessionToken(userID string, duration time.Duration) (string, error) {
	tokenBytes := make([]byte, 32)

	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", ErrInternalError
	}

	token := hex.EncodeToString(tokenBytes)
	expirationTime := time.Now().Add(duration)

	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.tokens[token] = SessionToken{
		UserID:    userID,
		ExpiresAt: expirationTime,
		CreatedAt: time.Now(),
	}
	return token, nil
}
