package usecase

import (
	"context"

	"github.com/google/uuid"

	"aphrodite/internal/comment/domain"
)

const RoleAdmin = "admin"

type ListFilter struct {
	PostID uuid.UUID
	Limit  int
	Offset int
}

type CommentRepository interface {
	Create(ctx context.Context, c *domain.Comment) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error)
	ListByPost(ctx context.Context, filter ListFilter) ([]*domain.Comment, int64, error)
	Update(ctx context.Context, c *domain.Comment) error
	Delete(ctx context.Context, id uuid.UUID) error
}
