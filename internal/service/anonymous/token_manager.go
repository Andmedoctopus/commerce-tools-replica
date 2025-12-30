package anonymous

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"commercetools-replica/internal/domain"
	tokenrepo "commercetools-replica/internal/repository/token"
)

type tokenMeta struct {
	AnonymousID string
	ProjectID   string
	ExpiresAt   time.Time
}

type tokenManager struct {
	repo tokenrepo.Repository
}

func newTokenManager(repo tokenrepo.Repository) *tokenManager {
	return &tokenManager{
		repo: repo,
	}
}

func (m *tokenManager) Issue(ctx context.Context, projectID, anonymousID, kind string, ttl time.Duration) (string, error) {
	expiresAt := time.Now().Add(ttl)
	for i := 0; i < 5; i++ {
		token, err := randomToken()
		if err != nil {
			return "", err
		}
		anon := anonymousID
		err = m.repo.Create(ctx, tokenrepo.Token{
			Token:       token,
			ProjectID:   projectID,
			AnonymousID: &anon,
			Kind:        kind,
			ExpiresAt:   expiresAt,
		})
		if err == nil {
			return token, nil
		}
		if errors.Is(err, domain.ErrAlreadyExists) {
			continue
		}
		return "", err
	}
	return "", errors.New("token collision")
}

func (m *tokenManager) Validate(ctx context.Context, token string) (tokenMeta, bool) {
	meta, err := m.repo.Get(ctx, token)
	if err != nil {
		return tokenMeta{}, false
	}
	if meta.Kind != "access" || meta.AnonymousID == nil {
		return tokenMeta{}, false
	}
	if time.Now().After(meta.ExpiresAt) {
		_ = m.repo.Delete(ctx, token)
		return tokenMeta{}, false
	}
	return tokenMeta{
		AnonymousID: *meta.AnonymousID,
		ProjectID:   meta.ProjectID,
		ExpiresAt:   meta.ExpiresAt,
	}, true
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
