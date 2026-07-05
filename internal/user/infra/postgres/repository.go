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

func (r *Repository) List(ctx context.Context, filter usecase.ListFilter) ([]*domain.User, int64, error) {
	var (
		rows  []userRow
		total int64
	)
	db := r.db.WithContext(ctx).Model(&userRow{})

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Order("created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	users := make([]*domain.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, row.toDomain())
	}
	return users, total, nil
}

func (r *Repository) Update(ctx context.Context, u *domain.User) error {
	res := r.db.WithContext(ctx).
		Model(&userRow{}).
		Where("id = ?", u.ID).
		Updates(map[string]any{
			"username":      u.Username,
			"email":         u.Email,
			"password_hash": u.PasswordHash,
			"phone_number":  u.PhoneNumber,
			"role":          string(u.Role),
			"updated_at":    u.UpdatedAt,
		})
	if res.Error != nil {
		if sharedpg.IsUniqueViolation(res.Error) {
			return domain.ErrAlreadyExists
		}
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
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
