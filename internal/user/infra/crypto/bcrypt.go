package crypto

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"aphrodite/internal/user/domain"
	"aphrodite/internal/user/usecase"
)

type BcryptHasher struct {
	cost int
}

var _ usecase.PasswordHasher = (*BcryptHasher)(nil)

func NewBcryptHasher(cost int) *BcryptHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

func (h *BcryptHasher) Hash(_ context.Context, plaintext string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plaintext), h.cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (h *BcryptHasher) Verify(_ context.Context, hash, plaintext string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return domain.ErrInvalidCredential
	}
	return err
}
