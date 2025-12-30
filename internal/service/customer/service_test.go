package customer

import (
	"context"
	"testing"

	"commercetools-replica/internal/domain"
	tokenrepo "commercetools-replica/internal/repository/token"
)

// memoryRepo is a lightweight in-memory customer repository for tests.
type memoryRepo struct {
	byProject map[string]map[string]domain.Customer
}

type memoryTokenRepo struct {
	tokens map[string]tokenrepo.Token
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{byProject: make(map[string]map[string]domain.Customer)}
}

func newMemoryTokenRepo() *memoryTokenRepo {
	return &memoryTokenRepo{tokens: make(map[string]tokenrepo.Token)}
}

func (r *memoryTokenRepo) Create(_ context.Context, token tokenrepo.Token) error {
	if _, exists := r.tokens[token.Token]; exists {
		return domain.ErrAlreadyExists
	}
	r.tokens[token.Token] = token
	return nil
}

func (r *memoryTokenRepo) Get(_ context.Context, token string) (*tokenrepo.Token, error) {
	t, ok := r.tokens[token]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := t
	return &clone, nil
}

func (r *memoryTokenRepo) Delete(_ context.Context, token string) error {
	if _, ok := r.tokens[token]; !ok {
		return domain.ErrNotFound
	}
	delete(r.tokens, token)
	return nil
}

func (r *memoryRepo) Create(_ context.Context, c domain.Customer) (*domain.Customer, error) {
	if r.byProject[c.ProjectID] == nil {
		r.byProject[c.ProjectID] = make(map[string]domain.Customer)
	}
	if _, exists := r.byProject[c.ProjectID][c.Email]; exists {
		return nil, domain.ErrAlreadyExists
	}
	clone := c
	if clone.ID == "" {
		clone.ID = "cust-" + c.Email
	}
	r.byProject[c.ProjectID][clone.Email] = clone
	return &clone, nil
}

func (r *memoryRepo) GetByEmail(_ context.Context, projectID, email string) (*domain.Customer, error) {
	proj := r.byProject[projectID]
	if proj == nil {
		return nil, domain.ErrNotFound
	}
	if c, ok := proj[email]; ok {
		clone := c
		return &clone, nil
	}
	return nil, domain.ErrNotFound
}

func (r *memoryRepo) GetByID(_ context.Context, projectID, id string) (*domain.Customer, error) {
	proj := r.byProject[projectID]
	if proj == nil {
		return nil, domain.ErrNotFound
	}
	for _, c := range proj {
		if c.ID == id {
			clone := c
			return &clone, nil
		}
	}
	return nil, domain.ErrNotFound
}

func TestSignupAndLogin_SucceedsWithTrimmedPassword(t *testing.T) {
	repo := newMemoryRepo()
	svc := New(repo, newMemoryTokenRepo())

	ctx := context.Background()
	projectID := "proj-1"
	rawPassword := " Abcdefg1 " // includes whitespace

	customer, err := svc.Signup(ctx, projectID, SignupInput{
		Email:     "user@example.com",
		Password:  rawPassword,
		FirstName: "T",
		LastName:  "User",
	})
	if err != nil {
		t.Fatalf("signup returned error: %v", err)
	}
	if customer == nil || customer.Email != "user@example.com" {
		t.Fatalf("unexpected customer %+v", customer)
	}

	_, _, _, err = svc.Login(ctx, projectID, "user@example.com", "Abcdefg1")
	if err != nil {
		t.Fatalf("login failed with trimmed password: %v", err)
	}
}

func TestValidatePassword_FailsOnWeakValues(t *testing.T) {
	cases := []struct {
		name string
		pass string
	}{
		{"too short", "Abc1"},
		{"no upper", "abcdefg1"},
		{"no lower", "ABCDEFG1"},
		{"no digit", "Abcdefgh"},
	}
	for _, tc := range cases {
		if err := validatePassword(tc.pass, 8); err == nil {
			t.Fatalf("expected error for case %s", tc.name)
		}
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	repo := newMemoryRepo()
	svc := New(repo, newMemoryTokenRepo())
	ctx := context.Background()

	if _, err := svc.Signup(ctx, "proj", SignupInput{
		Email:     "user@example.com",
		Password:  "Abcdefg1",
		FirstName: "T",
	}); err != nil {
		t.Fatalf("signup: %v", err)
	}

	if _, _, _, err := svc.Login(ctx, "proj", "user@example.com", "wrongpass"); err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if _, _, _, err := svc.Login(ctx, "proj", "missing@example.com", "Abcdefg1"); err != ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials for missing user, got %v", err)
	}
}
