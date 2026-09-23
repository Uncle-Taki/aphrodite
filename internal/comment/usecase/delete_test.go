package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"aphrodite/internal/comment/domain"
)

func TestDeleteComment_AuthorCanDelete(t *testing.T) {
	commentID := uuid.New()
	authorID := uuid.New()
	repo := newUpdateCommentRepo(&domain.Comment{ID: commentID, PostID: uuid.New(), AuthorID: authorID, Content: "c"})
	uc := NewDeleteComment(repo)

	if err := uc.Execute(context.Background(), DeleteInput{CommentID: commentID, CallerID: authorID}); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.comments[commentID]; ok {
		t.Fatal("comment not removed from repository")
	}
}

func TestDeleteComment_AdminCanDeleteAnyComment(t *testing.T) {
	commentID := uuid.New()
	repo := newUpdateCommentRepo(&domain.Comment{ID: commentID, PostID: uuid.New(), AuthorID: uuid.New(), Content: "c"})
	uc := NewDeleteComment(repo)

	if err := uc.Execute(context.Background(), DeleteInput{
		CommentID:  commentID,
		CallerID:   uuid.New(),
		CallerRole: RoleAdmin,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteComment_RejectsUnauthorizedCaller(t *testing.T) {
	commentID := uuid.New()
	repo := newUpdateCommentRepo(&domain.Comment{ID: commentID, PostID: uuid.New(), AuthorID: uuid.New(), Content: "c"})
	uc := NewDeleteComment(repo)

	err := uc.Execute(context.Background(), DeleteInput{CommentID: commentID, CallerID: uuid.New()})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
	if _, ok := repo.comments[commentID]; !ok {
		t.Fatal("comment must remain after a forbidden delete")
	}
}

func TestDeleteComment_MissingCommentReturnsNotFound(t *testing.T) {
	uc := NewDeleteComment(newUpdateCommentRepo())
	err := uc.Execute(context.Background(), DeleteInput{CommentID: uuid.New(), CallerID: uuid.New()})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
