package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"aphrodite/internal/post/domain"
	"aphrodite/internal/post/usecase"
)

type Repository struct {
	db *gorm.DB
}

// Ensure the adapter satisfies the usecase port at compile time.
var _ usecase.PostRepository = (*Repository)(nil)

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, p *domain.Post) error {
	row := fromDomain(p)
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
	var row postRow
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *Repository) List(ctx context.Context, filter usecase.ListFilter) ([]*domain.Post, int64, error) {
	var (
		rows  []postRow
		total int64
	)
	db := r.db.WithContext(ctx).Model(&postRow{})

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

	posts := make([]*domain.Post, 0, len(rows))
	for _, row := range rows {
		posts = append(posts, row.toDomain())
	}
	return posts, total, nil
}

func (r *Repository) Update(ctx context.Context, p *domain.Post) error {
	res := r.db.WithContext(ctx).
		Model(&postRow{}).
		Where("id = ?", p.ID).
		Updates(map[string]any{
			"title":      p.Title,
			"content":    p.Content,
			"updated_at": p.UpdatedAt,
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
	res := r.db.WithContext(ctx).Where("id = ?", id).Delete(&postRow{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
