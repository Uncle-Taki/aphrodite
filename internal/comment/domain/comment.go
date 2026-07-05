package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID        uuid.UUID
	PostID    uuid.UUID
	AuthorID  uuid.UUID
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CommentDTO struct {
	ID               uuid.UUID
	PostID           uuid.UUID
	AuthorID         uuid.UUID
	Content          string
	ContentMaxLength int
	Now              time.Time
}

type UpdateDTO struct {
	Content          string
	ContentMaxLength int
	Now              time.Time
}

func NewComment(dto CommentDTO) (*Comment, error) {
	if dto.PostID == uuid.Nil {
		return nil, ErrInvalidPost
	}
	if dto.AuthorID == uuid.Nil {
		return nil, ErrInvalidAuthor
	}
	content := strings.TrimSpace(dto.Content)
	if content == "" || (dto.ContentMaxLength > 0 && len(content) > dto.ContentMaxLength) {
		return nil, ErrInvalidContent
	}
	return &Comment{
		ID:        dto.ID,
		PostID:    dto.PostID,
		AuthorID:  dto.AuthorID,
		Content:   content,
		CreatedAt: dto.Now,
		UpdatedAt: dto.Now,
	}, nil
}

func (c *Comment) Update(dto UpdateDTO) error {
	content := strings.TrimSpace(dto.Content)
	if content == "" || (dto.ContentMaxLength > 0 && len(content) > dto.ContentMaxLength) {
		return ErrInvalidContent
	}
	c.Content = content
	c.UpdatedAt = dto.Now
	return nil
}
