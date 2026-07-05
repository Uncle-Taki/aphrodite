package postgres

import (
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/post/domain"
)

type postRow struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	AuthorID  uuid.UUID `gorm:"type:uuid;index;not null"`
	Title     string    `gorm:"type:varchar(200);not null"`
	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"index;not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (postRow) TableName() string { return "posts" }

func fromDomain(p *domain.Post) postRow {
	return postRow{
		ID:        p.ID,
		AuthorID:  p.AuthorID,
		Title:     p.Title,
		Content:   p.Content,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func (r postRow) toDomain() *domain.Post {
	return &domain.Post{
		ID:        r.ID,
		AuthorID:  r.AuthorID,
		Title:     r.Title,
		Content:   r.Content,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
