package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/post/domain"
)

type CreateInput struct {
	AuthorID uuid.UUID
	Title    string
	Content  string
}

type CreatePostConfig struct {
	TitleMaxLength   int
	ContentMaxLength int
}

type CreatePost struct {
	repo   PostRepository
	config CreatePostConfig
	newID  func() uuid.UUID
	now    func() time.Time
}

func NewCreatePost(repo PostRepository, config CreatePostConfig, newID func() uuid.UUID, now func() time.Time) *CreatePost {
	if newID == nil {
		newID = uuid.New
	}
	if now == nil {
		now = time.Now
	}
	return &CreatePost{repo: repo, config: config, newID: newID, now: now}
}

func (uc *CreatePost) Execute(ctx context.Context, in CreateInput) (*domain.Post, error) {
	p, err := domain.NewPost(domain.PostDTO{
		ID:               uc.newID(),
		AuthorID:         in.AuthorID,
		Title:            in.Title,
		Content:          in.Content,
		TitleMaxLength:   uc.config.TitleMaxLength,
		ContentMaxLength: uc.config.ContentMaxLength,
		Now:              uc.now(),
	})
	if err != nil {
		return nil, err
	}
	if err := uc.repo.Create(ctx, p); err != nil {
		slog.ErrorContext(ctx, "post: repository create failed", "err", err, "post_id", p.ID)
		return nil, err
	}
	slog.InfoContext(ctx, "post: created", "post_id", p.ID, "author_id", p.AuthorID)
	return p, nil
}
