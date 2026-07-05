package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/post/domain"
)

type UpdateInput struct {
	PostID     uuid.UUID
	CallerID   uuid.UUID
	CallerRole string
	Title      string
	Content    string
}

type UpdatePostConfig struct {
	TitleMaxLength   int
	ContentMaxLength int
}

type UpdatePost struct {
	repo   PostRepository
	config UpdatePostConfig
	now    func() time.Time
}

func NewUpdatePost(repo PostRepository, config UpdatePostConfig, now func() time.Time) *UpdatePost {
	if now == nil {
		now = time.Now
	}
	return &UpdatePost{repo: repo, config: config, now: now}
}

func (uc *UpdatePost) Execute(ctx context.Context, in UpdateInput) (*domain.Post, error) {
	p, err := uc.repo.FindByID(ctx, in.PostID)
	if err != nil {
		return nil, err
	}

	if in.CallerRole != RoleAdmin && p.AuthorID != in.CallerID {
		slog.WarnContext(ctx, "post: update forbidden",
			"post_id", in.PostID, "caller_id", in.CallerID, "caller_role", in.CallerRole)
		return nil, domain.ErrForbidden
	}

	if err := p.Update(domain.UpdateDTO{
		Title:            in.Title,
		Content:          in.Content,
		TitleMaxLength:   uc.config.TitleMaxLength,
		ContentMaxLength: uc.config.ContentMaxLength,
		Now:              uc.now(),
	}); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(ctx, p); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		slog.ErrorContext(ctx, "post: repository update failed", "err", err, "post_id", in.PostID)
		return nil, err
	}

	slog.InfoContext(ctx, "post: updated",
		"post_id", in.PostID, "caller_id", in.CallerID, "caller_role", in.CallerRole)
	return p, nil
}
