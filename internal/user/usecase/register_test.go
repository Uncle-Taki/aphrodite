package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

func TestRegisterUser_DefaultsToUserRole(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 7, 5, 14, 0, 0, 0, time.UTC)
	repo := newUserMemoryRepo()
	uc := NewRegisterUser(repo, fakePasswordHasher{}, "bootstrap-secret", func() uuid.UUID { return userID }, func() time.Time { return now })

	got, err := uc.Execute(context.Background(), RegisterInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "password",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != userID || got.Role != domain.RoleUser || got.PasswordHash != "hash:password" {
		t.Fatalf("unexpected registered user: %+v", got)
	}
}

func TestRegisterUser_RejectsAdminWithoutSuperAdminKey(t *testing.T) {
	repo := newUserMemoryRepo()
	uc := NewRegisterUser(repo, fakePasswordHasher{}, "bootstrap-secret", nil, nil)

	_, err := uc.Execute(context.Background(), RegisterInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "password",
		Role:     domain.RoleAdmin,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if len(repo.users) != 0 {
		t.Fatalf("admin user should not be persisted: %+v", repo.users)
	}
}

func TestRegisterUser_RejectsAdminWithWrongSuperAdminKey(t *testing.T) {
	repo := newUserMemoryRepo()
	uc := NewRegisterUser(repo, fakePasswordHasher{}, "bootstrap-secret", nil, nil)

	_, err := uc.Execute(context.Background(), RegisterInput{
		Username:      "alice",
		Email:         "alice@example.com",
		Password:      "password",
		Role:          domain.RoleAdmin,
		SuperAdminKey: "wrong-secret",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if len(repo.users) != 0 {
		t.Fatalf("admin user should not be persisted: %+v", repo.users)
	}
}

func TestRegisterUser_AllowsAdminWithSuperAdminKey(t *testing.T) {
	repo := newUserMemoryRepo()
	uc := NewRegisterUser(repo, fakePasswordHasher{}, "bootstrap-secret", nil, nil)

	got, err := uc.Execute(context.Background(), RegisterInput{
		Username:      "alice",
		Email:         "alice@example.com",
		Password:      "password",
		Role:          domain.RoleAdmin,
		SuperAdminKey: "bootstrap-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Role != domain.RoleAdmin {
		t.Fatalf("role not set to admin: %+v", got)
	}
}
