package usecase

import (
	"context"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, filter ListFilter) ([]*domain.User, int64, error)
	Update(ctx context.Context, u *domain.User) error
}

type ListFilter struct {
	Limit  int
	Offset int
}

type PasswordHasher interface {
	Hash(ctx context.Context, plaintext string) (string, error)
	Verify(ctx context.Context, hash, plaintext string) error
}

type TokenIssuer interface {
	Issue(ctx context.Context, u *domain.User) (string, error)
	Verify(ctx context.Context, token string) (uuid.UUID, domain.Role, error)
}
