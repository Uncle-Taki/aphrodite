package crypto

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"aphrodite/internal/user/domain"
)

func TestNewBcryptHasherClampsCost(t *testing.T) {
	if got := NewBcryptHasher(bcrypt.MinCost - 1); got.cost != bcrypt.DefaultCost {
		t.Errorf("expected below-min cost to clamp to default, got %d", got.cost)
	}
	if got := NewBcryptHasher(bcrypt.MaxCost + 1); got.cost != bcrypt.DefaultCost {
		t.Errorf("expected above-max cost to clamp to default, got %d", got.cost)
	}
	if got := NewBcryptHasher(bcrypt.MinCost); got.cost != bcrypt.MinCost {
		t.Errorf("expected in-range cost preserved, got %d", got.cost)
	}
}

func TestHashVerifyRoundTrip(t *testing.T) {
	hasher := NewBcryptHasher(bcrypt.MinCost)
	hash, err := hasher.Hash(context.Background(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" || hash == "secret" {
		t.Fatalf("expected a real hash, got %q", hash)
	}
	if err := hasher.Verify(context.Background(), hash, "secret"); err != nil {
		t.Fatalf("expected correct password to verify, got %v", err)
	}
}

func TestVerifyRejectsWrongPassword(t *testing.T) {
	hasher := NewBcryptHasher(bcrypt.MinCost)
	hash, err := hasher.Hash(context.Background(), "secret")
	if err != nil {
		t.Fatal(err)
	}
	if err := hasher.Verify(context.Background(), hash, "wrong"); !errors.Is(err, domain.ErrInvalidCredential) {
		t.Fatalf("expected ErrInvalidCredential, got %v", err)
	}
}
