package customer

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

type tokenMeta struct {
	CustomerID string
	ProjectID  string
	ExpiresAt  time.Time
}

type tokenManager struct {
	mu     sync.RWMutex
	tokens map[string]tokenMeta
}

func newTokenManager() *tokenManager {
	return &tokenManager{
		tokens: make(map[string]tokenMeta),
	}
}

func (m *tokenManager) Issue(projectID, customerID string, ttl time.Duration) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	meta := tokenMeta{
		CustomerID: customerID,
		ProjectID:  projectID,
		ExpiresAt:  time.Now().Add(ttl),
	}
	m.mu.Lock()
	m.tokens[token] = meta
	m.mu.Unlock()
	return token, nil
}

func (m *tokenManager) Validate(token string) (tokenMeta, bool) {
	m.mu.RLock()
	meta, ok := m.tokens[token]
	m.mu.RUnlock()
	if !ok {
		return tokenMeta{}, false
	}
	if time.Now().After(meta.ExpiresAt) {
		m.mu.Lock()
		delete(m.tokens, token)
		m.mu.Unlock()
		return tokenMeta{}, false
	}
	return meta, true
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
