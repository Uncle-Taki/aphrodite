package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/post/domain"
)

func TestCreatePost(t *testing.T) {
	authorID := uuid.New()
	postID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	repo := newUpdatePostRepo()
	uc := NewCreatePost(repo, CreatePostConfig{TitleMaxLength: 20, ContentMaxLength: 50},
		func() uuid.UUID { return postID }, func() time.Time { return now })

	got, err := uc.Execute(context.Background(), CreateInput{
		AuthorID: authorID,
		Title:    "  Title  ",
		Content:  "  Body  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != postID || got.AuthorID != authorID {
		t.Fatalf("identity not applied: %+v", got)
	}
	if got.Title != "Title" || got.Content != "Body" {
		t.Fatalf("content not trimmed: %+v", got)
	}
	if got.CreatedAt != now || got.UpdatedAt != now {
		t.Fatalf("timestamps not set: %+v", got)
	}
	if _, ok := repo.posts[postID]; !ok {
		t.Fatal("post not persisted in repository")
	}
}

func TestCreatePostDefaultsIDAndNow(t *testing.T) {
	uc := NewCreatePost(newUpdatePostRepo(), CreatePostConfig{TitleMaxLength: 20, ContentMaxLength: 50}, nil, nil)
	got, err := uc.Execute(context.Background(), CreateInput{AuthorID: uuid.New(), Title: "t", Content: "c"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == uuid.Nil {
		t.Fatal("default newID must produce a real UUID")
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("default now must produce a real timestamp")
	}
}

func TestCreatePostRejectsInvalidPayload(t *testing.T) {
	uc := NewCreatePost(newUpdatePostRepo(), CreatePostConfig{TitleMaxLength: 5, ContentMaxLength: 50}, nil, nil)
	_, err := uc.Execute(context.Background(), CreateInput{AuthorID: uuid.New(), Title: "too long", Content: "c"})
	if !errors.Is(err, domain.ErrInvalidTitle) {
		t.Fatalf("expected invalid title, got %v", err)
	}
}

func TestCreatePostPropagatesRepoError(t *testing.T) {
	repo := newUpdatePostRepo()
	repo.createErr = errors.New("db down")
	uc := NewCreatePost(repo, CreatePostConfig{TitleMaxLength: 20, ContentMaxLength: 50}, nil, nil)
	if _, err := uc.Execute(context.Background(), CreateInput{AuthorID: uuid.New(), Title: "t", Content: "c"}); err == nil {
		t.Fatal("expected repository error to propagate")
	}
}
