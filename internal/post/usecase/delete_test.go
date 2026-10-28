package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"aphrodite/internal/post/domain"
)

func TestDeletePost_AuthorCanDelete(t *testing.T) {
	postID := uuid.New()
	authorID := uuid.New()
	repo := newUpdatePostRepo(&domain.Post{ID: postID, AuthorID: authorID, Title: "t", Content: "c"})
	uc := NewDeletePost(repo)

	if err := uc.Execute(context.Background(), DeleteInput{PostID: postID, CallerID: authorID}); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.posts[postID]; ok {
		t.Fatal("post not removed from repository")
	}
}

func TestDeletePost_AdminCanDeleteAnyPost(t *testing.T) {
	postID := uuid.New()
	repo := newUpdatePostRepo(&domain.Post{ID: postID, AuthorID: uuid.New(), Title: "t", Content: "c"})
	uc := NewDeletePost(repo)

	if err := uc.Execute(context.Background(), DeleteInput{
		PostID:     postID,
		CallerID:   uuid.New(),
		CallerRole: RoleAdmin,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDeletePost_RejectsUnauthorizedCaller(t *testing.T) {
	postID := uuid.New()
	repo := newUpdatePostRepo(&domain.Post{ID: postID, AuthorID: uuid.New(), Title: "t", Content: "c"})
	uc := NewDeletePost(repo)

	err := uc.Execute(context.Background(), DeleteInput{PostID: postID, CallerID: uuid.New()})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if _, ok := repo.posts[postID]; !ok {
		t.Fatal("post must remain after a forbidden delete")
	}
}

func TestDeletePost_MissingPostReturnsNotFound(t *testing.T) {
	uc := NewDeletePost(newUpdatePostRepo())
	err := uc.Execute(context.Background(), DeleteInput{PostID: uuid.New(), CallerID: uuid.New()})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
