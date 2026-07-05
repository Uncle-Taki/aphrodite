package usecase

import (
	"context"

	"github.com/google/uuid"

	"aphrodite/internal/post/domain"
)

type GetPost struct {
	repo PostRepository
}

func NewGetPost(repo PostRepository) *GetPost {
	return &GetPost{repo: repo}
}

func (uc *GetPost) Execute(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
	return uc.repo.FindByID(ctx, id)
}
