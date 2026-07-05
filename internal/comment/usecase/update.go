package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/comment/domain"
)

type UpdateInput struct {
	CommentID  uuid.UUID
	CallerID   uuid.UUID
	CallerRole string
	Content    string
}

type UpdateCommentConfig struct {
	ContentMaxLength int
}

type UpdateComment struct {
	repo   CommentRepository
	config UpdateCommentConfig
	now    func() time.Time
}

func NewUpdateComment(repo CommentRepository, config UpdateCommentConfig, now func() time.Time) *UpdateComment {
	if now == nil {
		now = time.Now
	}
	return &UpdateComment{repo: repo, config: config, now: now}
}

func (uc *UpdateComment) Execute(ctx context.Context, in UpdateInput) (*domain.Comment, error) {
	c, err := uc.repo.FindByID(ctx, in.CommentID)
	if err != nil {
		return nil, err
	}

	if in.CallerRole != RoleAdmin && c.AuthorID != in.CallerID {
		slog.WarnContext(ctx, "comment: update forbidden",
			"comment_id", in.CommentID, "caller_id", in.CallerID, "caller_role", in.CallerRole)
		return nil, domain.ErrForbidden
	}

	if err := c.Update(domain.UpdateDTO{
		Content:          in.Content,
		ContentMaxLength: uc.config.ContentMaxLength,
		Now:              uc.now(),
	}); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(ctx, c); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		slog.ErrorContext(ctx, "comment: repository update failed", "err", err, "comment_id", in.CommentID)
		return nil, err
	}

	slog.InfoContext(ctx, "comment: updated",
		"comment_id", in.CommentID, "caller_id", in.CallerID, "caller_role", in.CallerRole)
	return c, nil
}
