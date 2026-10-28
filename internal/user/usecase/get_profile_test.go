package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

func TestGetUserProfile(t *testing.T) {
	id := uuid.New()
	repo := newUserMemoryRepo(testUser(id, domain.RoleUser))
	uc := NewGetUserProfile(repo)

	got, err := uc.Execute(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != id {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestGetUserProfile_NotFound(t *testing.T) {
	uc := NewGetUserProfile(newUserMemoryRepo())
	_, err := uc.Execute(context.Background(), uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
