package customer

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"commercetools-replica/internal/domain"
	tokenrepo "commercetools-replica/internal/repository/token"
)

type tokenMeta struct {
	CustomerID string
	ProjectID  string
	ExpiresAt  time.Time
}

type tokenManager struct {
	repo tokenrepo.Repository
}

func newTokenManager(repo tokenrepo.Repository) *tokenManager {
	return &tokenManager{
		repo: repo,
	}
}

func (m *tokenManager) Issue(ctx context.Context, projectID, customerID, kind string, ttl time.Duration) (string, error) {
	expiresAt := time.Now().Add(ttl)
	for i := 0; i < 5; i++ {
		token, err := randomToken()
		if err != nil {
			return "", err
		}
		customer := customerID
		err = m.repo.Create(ctx, tokenrepo.Token{
			Token:      token,
			ProjectID:  projectID,
			CustomerID: &customer,
			Kind:       kind,
			ExpiresAt:  expiresAt,
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
	if meta.Kind != "access" || meta.CustomerID == nil {
		return tokenMeta{}, false
	}
	if time.Now().After(meta.ExpiresAt) {
		_ = m.repo.Delete(ctx, token)
		return tokenMeta{}, false
	}
	return tokenMeta{
		CustomerID: *meta.CustomerID,
		ProjectID:  meta.ProjectID,
		ExpiresAt:  meta.ExpiresAt,
	}, true
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
