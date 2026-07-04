package usecase

import (
	"context"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

type GetUserProfile struct {
	repo UserRepository
}

func NewGetUserProfile(repo UserRepository) *GetUserProfile {
	return &GetUserProfile{repo: repo}
}

func (uc *GetUserProfile) Execute(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return uc.repo.FindByID(ctx, id)
}
