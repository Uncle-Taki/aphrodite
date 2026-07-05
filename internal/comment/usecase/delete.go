package usecase

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"aphrodite/internal/comment/domain"
)

type DeleteInput struct {
	CommentID  uuid.UUID
	CallerID   uuid.UUID
	CallerRole string
}

type DeleteComment struct {
	repo CommentRepository
}

func NewDeleteComment(repo CommentRepository) *DeleteComment {
	return &DeleteComment{repo: repo}
}

func (uc *DeleteComment) Execute(ctx context.Context, in DeleteInput) error {
	c, err := uc.repo.FindByID(ctx, in.CommentID)
	if err != nil {
		return err
	}

	if in.CallerRole != RoleAdmin && c.AuthorID != in.CallerID {
		slog.WarnContext(ctx, "comment: delete forbidden",
			"comment_id", in.CommentID, "caller_id", in.CallerID, "caller_role", in.CallerRole)
		return domain.ErrForbidden
	}

	if err := uc.repo.Delete(ctx, in.CommentID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return err
		}
		slog.ErrorContext(ctx, "comment: repository delete failed", "err", err, "comment_id", in.CommentID)
		return err
	}

	slog.InfoContext(ctx, "comment: deleted",
		"comment_id", in.CommentID, "caller_id", in.CallerID, "caller_role", in.CallerRole)
	return nil
}
