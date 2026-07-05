package usecase

import (
	"context"

	"github.com/google/uuid"

	"aphrodite/internal/post/domain"
)

const RoleAdmin = "admin"

type ListFilter struct {
	Limit  int
	Offset int
}

type PostRepository interface {
	Create(ctx context.Context, p *domain.Post) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Post, error)
	List(ctx context.Context, filter ListFilter) ([]*domain.Post, int64, error)
	Update(ctx context.Context, p *domain.Post) error
	Delete(ctx context.Context, id uuid.UUID) error
}
