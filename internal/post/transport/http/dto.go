package http

import (
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/post/domain"
)

type CreatePostRequest struct {
	Title   string `json:"title" binding:"required" example:"Hello, world"`
	Content string `json:"content" binding:"required" example:"This is my first post."`
}

type UpdatePostRequest struct {
	Title   string `json:"title" binding:"required" example:"Updated title"`
	Content string `json:"content" binding:"required" example:"Updated post content."`
}

type PostResponse struct {
	ID        uuid.UUID `json:"id"`
	AuthorID  uuid.UUID `json:"author_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PostListResponse struct {
	Posts []PostResponse `json:"posts"`
	Total int64          `json:"total"`
	Limit int            `json:"limit"`
	Page  int            `json:"page"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func toPostResponse(p *domain.Post) PostResponse {
	return PostResponse{
		ID:        p.ID,
		AuthorID:  p.AuthorID,
		Title:     p.Title,
		Content:   p.Content,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
