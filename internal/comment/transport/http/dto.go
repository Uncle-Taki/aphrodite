package http

import (
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/comment/domain"
)

type AddCommentRequest struct {
	Content string `json:"content" binding:"required" example:"Great post!"`
}

type UpdateCommentRequest struct {
	Content string `json:"content" binding:"required" example:"Updated comment."`
}

type CommentResponse struct {
	ID        uuid.UUID `json:"id"`
	PostID    uuid.UUID `json:"post_id"`
	AuthorID  uuid.UUID `json:"author_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CommentListResponse struct {
	Comments []CommentResponse `json:"comments"`
	Total    int64             `json:"total"`
	Limit    int               `json:"limit"`
	Page     int               `json:"page"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func toCommentResponse(c *domain.Comment) CommentResponse {
	return CommentResponse{
		ID:        c.ID,
		PostID:    c.PostID,
		AuthorID:  c.AuthorID,
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
