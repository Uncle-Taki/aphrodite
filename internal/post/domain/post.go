package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Post struct {
	ID        uuid.UUID
	AuthorID  uuid.UUID
	Title     string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PostDTO struct {
	ID               uuid.UUID
	AuthorID         uuid.UUID
	Title            string
	Content          string
	TitleMaxLength   int
	ContentMaxLength int
	Now              time.Time
}

type UpdateDTO struct {
	Title            string
	Content          string
	TitleMaxLength   int
	ContentMaxLength int
	Now              time.Time
}

func NewPost(dto PostDTO) (*Post, error) {
	if dto.AuthorID == uuid.Nil {
		return nil, ErrInvalidAuthor
	}

	title, content, err := normalizeContent(dto.Title, dto.Content, dto.TitleMaxLength, dto.ContentMaxLength)
	if err != nil {
		return nil, err
	}

	return &Post{
		ID:        dto.ID,
		AuthorID:  dto.AuthorID,
		Title:     title,
		Content:   content,
		CreatedAt: dto.Now,
		UpdatedAt: dto.Now,
	}, nil
}

func (p *Post) Update(dto UpdateDTO) error {
	title, content, err := normalizeContent(dto.Title, dto.Content, dto.TitleMaxLength, dto.ContentMaxLength)
	if err != nil {
		return err
	}

	p.Title = title
	p.Content = content
	p.UpdatedAt = dto.Now
	return nil
}

func normalizeContent(title, content string, titleMaxLength, contentMaxLength int) (string, string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", "", ErrInvalidTitle
	}
	if titleMaxLength > 0 && len(title) > titleMaxLength {
		return "", "", ErrInvalidTitle
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return "", "", ErrInvalidContent
	}
	if contentMaxLength > 0 && len(content) > contentMaxLength {
		return "", "", ErrInvalidContent
	}

	return title, content, nil
}
