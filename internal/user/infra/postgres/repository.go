package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	sharedpg "aphrodite/internal/shared/postgres"
	"aphrodite/internal/user/domain"
	"aphrodite/internal/user/usecase"
)

type Repository struct {
	db *gorm.DB
}

// Ensure the adapter satisfies the usecase port at compile time.
var _ usecase.UserRepository = (*Repository)(nil)

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, u *domain.User) error {
	row := fromDomain(u)
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		if sharedpg.IsUniqueViolation(err) {
			return domain.ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return r.findOne(ctx, "id = ?", id)
}

func (r *Repository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	return r.findOne(ctx, "username = ?", username)
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.findOne(ctx, "email = ?", email)
}

func (r *Repository) findOne(ctx context.Context, query string, args ...any) (*domain.User, error) {
	var row userRow
	err := r.db.WithContext(ctx).Where(query, args...).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return row.toDomain(), nil
}
