package postgres

import (
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/comment/domain"
)

type commentRow struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	PostID    uuid.UUID `gorm:"type:uuid;index;not null"`
	AuthorID  uuid.UUID `gorm:"type:uuid;index;not null"`
	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"index;not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (commentRow) TableName() string { return "comments" }

func fromDomain(c *domain.Comment) commentRow {
	return commentRow{
		ID:        c.ID,
		PostID:    c.PostID,
		AuthorID:  c.AuthorID,
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (r commentRow) toDomain() *domain.Comment {
	return &domain.Comment{
		ID:        r.ID,
		PostID:    r.PostID,
		AuthorID:  r.AuthorID,
		Content:   r.Content,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
