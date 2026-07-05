package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/comment/domain"
)

type updateCommentRepo struct {
	comments map[uuid.UUID]*domain.Comment
}

func newUpdateCommentRepo(comments ...*domain.Comment) *updateCommentRepo {
	repo := &updateCommentRepo{comments: map[uuid.UUID]*domain.Comment{}}
	for _, c := range comments {
		cp := *c
		repo.comments[c.ID] = &cp
	}
	return repo
}

func (r *updateCommentRepo) Create(_ context.Context, c *domain.Comment) error {
	cp := *c
	r.comments[c.ID] = &cp
	return nil
}

func (r *updateCommentRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Comment, error) {
	c, ok := r.comments[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *updateCommentRepo) ListByPost(_ context.Context, _ ListFilter) ([]*domain.Comment, int64, error) {
	return nil, 0, nil
}

func (r *updateCommentRepo) Update(_ context.Context, c *domain.Comment) error {
	if _, ok := r.comments[c.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *c
	r.comments[c.ID] = &cp
	return nil
}

func (r *updateCommentRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := r.comments[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.comments, id)
	return nil
}

func TestUpdateComment_AuthorCanUpdate(t *testing.T) {
	commentID := uuid.New()
	authorID := uuid.New()
	updatedAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	repo := newUpdateCommentRepo(&domain.Comment{
		ID:        commentID,
		PostID:    uuid.New(),
		AuthorID:  authorID,
		Content:   "old content",
		CreatedAt: updatedAt.Add(-time.Hour),
		UpdatedAt: updatedAt.Add(-time.Hour),
	})
	uc := NewUpdateComment(repo, UpdateCommentConfig{ContentMaxLength: 50}, func() time.Time { return updatedAt })

	got, err := uc.Execute(context.Background(), UpdateInput{
		CommentID: commentID,
		CallerID:  authorID,
		Content:   "  new content  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "new content" || !got.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("comment not updated: %+v", got)
	}
}

func TestUpdateComment_RejectsUnauthorizedCaller(t *testing.T) {
	commentID := uuid.New()
	repo := newUpdateCommentRepo(&domain.Comment{
		ID:        commentID,
		PostID:    uuid.New(),
		AuthorID:  uuid.New(),
		Content:   "old content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	uc := NewUpdateComment(repo, UpdateCommentConfig{ContentMaxLength: 50}, time.Now)

	_, err := uc.Execute(context.Background(), UpdateInput{
		CommentID: commentID,
		CallerID:  uuid.New(),
		Content:   "new content",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestUpdateComment_RejectsInvalidContent(t *testing.T) {
	commentID := uuid.New()
	authorID := uuid.New()
	repo := newUpdateCommentRepo(&domain.Comment{
		ID:        commentID,
		PostID:    uuid.New(),
		AuthorID:  authorID,
		Content:   "old content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	uc := NewUpdateComment(repo, UpdateCommentConfig{ContentMaxLength: 3}, time.Now)

	_, err := uc.Execute(context.Background(), UpdateInput{
		CommentID: commentID,
		CallerID:  authorID,
		Content:   "too long",
	})
	if !errors.Is(err, domain.ErrInvalidContent) {
		t.Fatalf("expected invalid content, got %v", err)
	}
}
