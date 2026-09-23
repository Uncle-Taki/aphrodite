package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"aphrodite/internal/post/domain"
)

func TestGetPost(t *testing.T) {
	postID := uuid.New()
	repo := newUpdatePostRepo(&domain.Post{ID: postID, AuthorID: uuid.New(), Title: "t", Content: "c"})
	uc := NewGetPost(repo)

	got, err := uc.Execute(context.Background(), postID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != postID {
		t.Fatalf("unexpected post: %+v", got)
	}
}

func TestGetPost_NotFound(t *testing.T) {
	uc := NewGetPost(newUpdatePostRepo())
	_, err := uc.Execute(context.Background(), uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
