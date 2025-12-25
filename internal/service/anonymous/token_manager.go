package anonymous

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

type tokenMeta struct {
	AnonymousID string
	ProjectID   string
	ExpiresAt   time.Time
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

func (m *tokenManager) Issue(projectID, anonymousID string, ttl time.Duration) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	meta := tokenMeta{
		AnonymousID: anonymousID,
		ProjectID:   projectID,
		ExpiresAt:   time.Now().Add(ttl),
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

func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	// UUID v4 (random).
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		b[0], b[1], b[2], b[3],
		b[4], b[5],
		b[6], b[7],
		b[8], b[9],
		b[10], b[11], b[12], b[13], b[14], b[15],
	), nil
}
