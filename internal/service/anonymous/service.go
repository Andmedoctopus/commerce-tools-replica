package anonymous

import (
	"context"
	"errors"
	"time"
)

var ErrInvalidToken = errors.New("invalid token")

type Service struct {
	tokens     *tokenManager
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func New() *Service {
	return &Service{
		tokens:     newTokenManager(),
		accessTTL:  3 * time.Hour,
		refreshTTL: 30 * 24 * time.Hour,
	}
}

func (s *Service) Issue(ctx context.Context, projectID string) (accessToken, refreshToken, anonymousID string, err error) {
	anonID, err := randomID()
	if err != nil {
		return "", "", "", err
	}
	accessToken, err = s.tokens.Issue(projectID, anonID, s.accessTTL)
	if err != nil {
		return "", "", "", err
	}
	refreshToken, err = s.tokens.Issue(projectID, anonID, s.refreshTTL)
	if err != nil {
		return "", "", "", err
	}
	return accessToken, refreshToken, anonID, nil
}

func (s *Service) LookupByToken(ctx context.Context, projectID, token string) (string, error) {
	meta, ok := s.tokens.Validate(token)
	if !ok || meta.ProjectID != projectID {
		return "", ErrInvalidToken
	}
	return meta.AnonymousID, nil
}

func (s *Service) AccessTTLSeconds() int {
	return int(s.accessTTL.Seconds())
}
