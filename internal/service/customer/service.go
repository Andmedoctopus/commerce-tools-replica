package customer

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"commercetools-replica/internal/domain"
	custrepo "commercetools-replica/internal/repository/customer"
	tokenrepo "commercetools-replica/internal/repository/token"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidCredentials is returned when email/password do not match.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrInvalidToken indicates the provided token could not be validated.
	ErrInvalidToken = errors.New("invalid token")
)

// Service handles customer signup/login flows.
type Service struct {
	repo        custrepo.Repository
	tokens      *tokenManager
	accessTTL   time.Duration
	refreshTTL  time.Duration
	passwordMin int
}

// New creates a Service with sane defaults.
func New(repo custrepo.Repository, tokens tokenrepo.Repository) *Service {
	return &Service{
		repo:        repo,
		tokens:      newTokenManager(tokens),
		accessTTL:   48 * time.Hour,
		refreshTTL:  30 * 24 * time.Hour,
		passwordMin: 8,
	}
}

// AddressInput mirrors incoming address payloads.
type AddressInput struct {
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Country    string `json:"country"`
	StreetName string `json:"streetName"`
	PostalCode string `json:"postalCode"`
	City       string `json:"city"`
	Email      string `json:"email"`
	Department string `json:"department"`
}

// SignupInput captures fields expected by the signup endpoint.
type SignupInput struct {
	Email                  string         `json:"email"`
	Password               string         `json:"password"`
	FirstName              string         `json:"firstName"`
	LastName               string         `json:"lastName"`
	DateOfBirth            string         `json:"dateOfBirth"`
	Addresses              []AddressInput `json:"addresses"`
	DefaultShippingAddress *int           `json:"defaultShippingAddress"`
	DefaultBillingAddress  *int           `json:"defaultBillingAddress"`
}

// Signup registers a new customer within the given project.
func (s *Service) Signup(ctx context.Context, projectID string, in SignupInput) (*domain.Customer, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	if email == "" {
		return nil, errors.New("email required")
	}
	password := strings.TrimSpace(in.Password)
	if err := validatePassword(password, s.passwordMin); err != nil {
		return nil, err
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	addresses := make([]domain.CustomerAddress, 0, len(in.Addresses))
	for _, a := range in.Addresses {
		addresses = append(addresses, domain.CustomerAddress{
			ID:         randomAddressID(),
			FirstName:  a.FirstName,
			LastName:   a.LastName,
			Country:    a.Country,
			StreetName: a.StreetName,
			PostalCode: a.PostalCode,
			City:       a.City,
			Email:      a.Email,
			Department: a.Department,
		})
	}

	shippingID := addressIDFromIndex(addresses, in.DefaultShippingAddress)
	if shippingID == "" && len(addresses) > 0 {
		shippingID = addresses[0].ID
	}
	billingID := addressIDFromIndex(addresses, in.DefaultBillingAddress)
	if billingID == "" && len(addresses) > 0 {
		billingID = addresses[0].ID
	}

	customer := domain.Customer{
		ProjectID:                projectID,
		Email:                    email,
		PasswordHash:             string(hashed),
		FirstName:                in.FirstName,
		LastName:                 in.LastName,
		DateOfBirth:              in.DateOfBirth,
		Addresses:                addresses,
		DefaultShippingAddressID: shippingID,
		DefaultBillingAddressID:  billingID,
	}
	if shippingID != "" {
		customer.ShippingAddressIDs = []string{shippingID}
	}
	if billingID != "" {
		customer.BillingAddressIDs = []string{billingID}
	}

	return s.repo.Create(ctx, customer)
}

// Login validates credentials and returns issued tokens plus the customer.
func (s *Service) Login(ctx context.Context, projectID, email, password string) (*domain.Customer, string, string, error) {
	password = strings.TrimSpace(password)
	c, err := s.repo.GetByEmail(ctx, projectID, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, "", "", ErrInvalidCredentials
		}
		return nil, "", "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(c.PasswordHash), []byte(password)); err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	access, err := s.tokens.Issue(ctx, c.ProjectID, c.ID, "access", s.accessTTL)
	if err != nil {
		return nil, "", "", err
	}
	refresh, err := s.tokens.Issue(ctx, c.ProjectID, c.ID, "refresh", s.refreshTTL)
	if err != nil {
		return nil, "", "", err
	}
	return c, access, refresh, nil
}

// LookupByToken returns the customer bound to a valid access token.
func (s *Service) LookupByToken(ctx context.Context, projectID, token string) (*domain.Customer, error) {
	meta, ok := s.tokens.Validate(ctx, token)
	if !ok || meta.ProjectID != projectID {
		return nil, ErrInvalidToken
	}
	c, err := s.repo.GetByID(ctx, projectID, meta.CustomerID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	return c, nil
}

// AccessTTLSeconds exposes the access token lifetime in seconds.
func (s *Service) AccessTTLSeconds() int {
	return int(s.accessTTL.Seconds())
}

func addressIDFromIndex(addresses []domain.CustomerAddress, idx *int) string {
	if idx == nil {
		return ""
	}
	if *idx < 0 || *idx >= len(addresses) {
		return ""
	}
	return addresses[*idx].ID
}

func randomAddressID() string {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	}
	return base64.RawURLEncoding.EncodeToString(buf[:])
}

func validatePassword(p string, min int) error {
	trimmed := strings.TrimSpace(p)
	if len(trimmed) < min {
		return fmt.Errorf("password must be at least %d characters", min)
	}
	hasUpper := false
	hasLower := false
	hasDigit := false
	for _, r := range trimmed {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return errors.New("password must contain at least 1 uppercase letter, 1 lowercase letter, and 1 number")
	}
	return nil
}
