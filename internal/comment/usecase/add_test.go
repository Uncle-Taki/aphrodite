package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/comment/domain"
)

func TestAddComment(t *testing.T) {
	postID := uuid.New()
	authorID := uuid.New()
	commentID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	repo := newUpdateCommentRepo()
	uc := NewAddComment(repo, AddCommentConfig{ContentMaxLength: 100},
		func() uuid.UUID { return commentID }, func() time.Time { return now })

	got, err := uc.Execute(context.Background(), AddInput{
		PostID:   postID,
		AuthorID: authorID,
		Content:  "  Nice  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != commentID || got.PostID != postID || got.AuthorID != authorID {
		t.Fatalf("identity not applied: %+v", got)
	}
	if got.Content != "Nice" {
		t.Fatalf("content not trimmed: %+v", got)
	}
	if _, ok := repo.comments[commentID]; !ok {
		t.Fatal("comment not persisted in repository")
	}
}

func TestAddCommentDefaultsIDAndNow(t *testing.T) {
	uc := NewAddComment(newUpdateCommentRepo(), AddCommentConfig{ContentMaxLength: 100}, nil, nil)
	got, err := uc.Execute(context.Background(), AddInput{PostID: uuid.New(), AuthorID: uuid.New(), Content: "c"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == uuid.Nil || got.CreatedAt.IsZero() {
		t.Fatalf("defaults not applied: %+v", got)
	}
}

func TestAddCommentRejectsInvalidContent(t *testing.T) {
	uc := NewAddComment(newUpdateCommentRepo(), AddCommentConfig{ContentMaxLength: 5}, nil, nil)
	_, err := uc.Execute(context.Background(), AddInput{PostID: uuid.New(), AuthorID: uuid.New(), Content: "too long"})
	if !errors.Is(err, domain.ErrInvalidContent) {
		t.Fatalf("expected invalid content, got %v", err)
	}
}

func TestAddCommentPropagatesRepoError(t *testing.T) {
	repo := newUpdateCommentRepo()
	repo.createErr = errors.New("db down")
	uc := NewAddComment(repo, AddCommentConfig{ContentMaxLength: 100}, nil, nil)
	if _, err := uc.Execute(context.Background(), AddInput{PostID: uuid.New(), AuthorID: uuid.New(), Content: "c"}); err == nil {
		t.Fatal("expected repository error to propagate")
	}
}
