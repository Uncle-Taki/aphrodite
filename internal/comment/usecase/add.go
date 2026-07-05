package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/comment/domain"
)

type AddInput struct {
	PostID   uuid.UUID
	AuthorID uuid.UUID
	Content  string
}

type AddCommentConfig struct {
	ContentMaxLength int
}

type AddComment struct {
	repo   CommentRepository
	config AddCommentConfig
	newID  func() uuid.UUID
	now    func() time.Time
}

func NewAddComment(repo CommentRepository, config AddCommentConfig, newID func() uuid.UUID, now func() time.Time) *AddComment {
	if newID == nil {
		newID = uuid.New
	}
	if now == nil {
		now = time.Now
	}
	return &AddComment{repo: repo, config: config, newID: newID, now: now}
}

func (uc *AddComment) Execute(ctx context.Context, in AddInput) (*domain.Comment, error) {
	c, err := domain.NewComment(domain.CommentDTO{
		ID:               uc.newID(),
		PostID:           in.PostID,
		AuthorID:         in.AuthorID,
		Content:          in.Content,
		ContentMaxLength: uc.config.ContentMaxLength,
		Now:              uc.now(),
	})
	if err != nil {
		return nil, err
	}
	if err := uc.repo.Create(ctx, c); err != nil {
		slog.ErrorContext(ctx, "comment: repository create failed", "err", err, "comment_id", c.ID)
		return nil, err
	}
	slog.InfoContext(ctx, "comment: added",
		"comment_id", c.ID, "post_id", c.PostID, "author_id", c.AuthorID)
	return c, nil
}
