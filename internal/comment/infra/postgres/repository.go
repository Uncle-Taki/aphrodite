package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"aphrodite/internal/comment/domain"
	"aphrodite/internal/comment/usecase"
)

type Repository struct {
	db *gorm.DB
}

// Ensure the adapter satisfies the usecase port at compile time.
var _ usecase.CommentRepository = (*Repository)(nil)

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, c *domain.Comment) error {
	row := fromDomain(c)
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Comment, error) {
	var row commentRow
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *Repository) ListByPost(ctx context.Context, filter usecase.ListFilter) ([]*domain.Comment, int64, error) {
	var (
		rows  []commentRow
		total int64
	)
	db := r.db.WithContext(ctx).Model(&commentRow{}).Where("post_id = ?", filter.PostID)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order("created_at ASC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	comments := make([]*domain.Comment, 0, len(rows))
	for _, row := range rows {
		comments = append(comments, row.toDomain())
	}
	return comments, total, nil
}

func (r *Repository) Update(ctx context.Context, c *domain.Comment) error {
	res := r.db.WithContext(ctx).
		Model(&commentRow{}).
		Where("id = ?", c.ID).
		Updates(map[string]any{
			"content":    c.Content,
			"updated_at": c.UpdatedAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&commentRow{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
