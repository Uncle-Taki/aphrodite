package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

func listTestUser(index int, createdAt time.Time) *domain.User {
	return &domain.User{
		ID:           uuid.New(),
		Username:     fmt.Sprintf("user-%d", index),
		Email:        fmt.Sprintf("user-%d@example.com", index),
		PasswordHash: "hash:password",
		Role:         domain.RoleUser,
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
	}
}

func TestListUsers_UsesDefaultsAndNewestFirst(t *testing.T) {
	base := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	repo := newUserMemoryRepo(
		listTestUser(1, base.Add(1*time.Minute)),
		listTestUser(2, base.Add(2*time.Minute)),
		listTestUser(3, base.Add(3*time.Minute)),
	)
	uc := NewListUsers(repo, ListConfig{DefaultLimit: 2, MaxLimit: 5})

	got, err := uc.Execute(context.Background(), ListInput{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 3 || got.Limit != 2 || got.Page != 1 || len(got.Users) != 2 {
		t.Fatalf("unexpected list metadata: %+v", got)
	}
	if got.Users[0].Username != "user-3" || got.Users[1].Username != "user-2" {
		t.Fatalf("users not newest first: %+v", got.Users)
	}
}

func TestListUsers_CapsLimitAndAppliesPage(t *testing.T) {
	base := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	repo := newUserMemoryRepo(
		listTestUser(1, base.Add(1*time.Minute)),
		listTestUser(2, base.Add(2*time.Minute)),
		listTestUser(3, base.Add(3*time.Minute)),
		listTestUser(4, base.Add(4*time.Minute)),
	)
	uc := NewListUsers(repo, ListConfig{DefaultLimit: 2, MaxLimit: 2})

	got, err := uc.Execute(context.Background(), ListInput{Limit: 100, Page: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 4 || got.Limit != 2 || got.Page != 2 || len(got.Users) != 2 {
		t.Fatalf("unexpected list metadata: %+v", got)
	}
	if got.Users[0].Username != "user-2" || got.Users[1].Username != "user-1" {
		t.Fatalf("unexpected page contents: %+v", got.Users)
	}
}
