package usecase

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"aphrodite/internal/post/domain"
)

type DeleteInput struct {
	PostID     uuid.UUID
	CallerID   uuid.UUID
	CallerRole string
}

type DeletePost struct {
	repo PostRepository
}

func NewDeletePost(repo PostRepository) *DeletePost {
	return &DeletePost{repo: repo}
}

func (uc *DeletePost) Execute(ctx context.Context, in DeleteInput) error {
	p, err := uc.repo.FindByID(ctx, in.PostID)
	if err != nil {
		return err
	}

	if in.CallerRole != RoleAdmin && p.AuthorID != in.CallerID {
		slog.WarnContext(ctx, "post: delete forbidden",
			"post_id", in.PostID, "caller_id", in.CallerID, "caller_role", in.CallerRole)
		return domain.ErrForbidden
	}

	if err := uc.repo.Delete(ctx, in.PostID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return err
		}
		slog.ErrorContext(ctx, "post: repository delete failed", "err", err, "post_id", in.PostID)
		return err
	}

	slog.InfoContext(ctx, "post: deleted",
		"post_id", in.PostID, "caller_id", in.CallerID, "caller_role", in.CallerRole)
	return nil
}
