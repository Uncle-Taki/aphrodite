package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

var testNow = time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

var (
	postID   = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	authorID = uuid.MustParse("b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f")
)

func TestNewComment(t *testing.T) {
	comment, err := NewComment(CommentDTO{
		PostID:           postID,
		AuthorID:         authorID,
		Content:          "  Nice post  ",
		ContentMaxLength: 100,
		Now:              testNow,
	})
	if err != nil {
		t.Fatal(err)
	}
	if comment.Content != "Nice post" {
		t.Errorf("content not trimmed: %+v", comment)
	}
	if comment.CreatedAt != testNow || comment.UpdatedAt != testNow {
		t.Errorf("timestamps not set")
	}
}

func TestNewCommentValidation(t *testing.T) {
	long := strings.Repeat("x", 20)
	cases := []struct {
		name string
		dto  CommentDTO
		want error
	}{
		{"nil post", CommentDTO{AuthorID: authorID, Content: "c"}, ErrInvalidPost},
		{"nil author", CommentDTO{PostID: postID, Content: "c"}, ErrInvalidAuthor},
		{"empty content", CommentDTO{PostID: postID, AuthorID: authorID, Content: "  "}, ErrInvalidContent},
		{"content too long", CommentDTO{PostID: postID, AuthorID: authorID, Content: long, ContentMaxLength: 10}, ErrInvalidContent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewComment(tc.dto)
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

func TestComment_Update(t *testing.T) {
	comment, err := NewComment(CommentDTO{PostID: postID, AuthorID: authorID, Content: "Old"})
	if err != nil {
		t.Fatal(err)
	}
	later := testNow.Add(time.Minute)
	if err := comment.Update(UpdateDTO{Content: " New ", ContentMaxLength: 100, Now: later}); err != nil {
		t.Fatal(err)
	}
	if comment.Content != "New" || comment.UpdatedAt != later {
		t.Errorf("update not applied: %+v", comment)
	}
}

func TestComment_UpdateValidation(t *testing.T) {
	comment, _ := NewComment(CommentDTO{PostID: postID, AuthorID: authorID, Content: "Old"})
	if err := comment.Update(UpdateDTO{Content: " ", ContentMaxLength: 100}); !errors.Is(err, ErrInvalidContent) {
		t.Fatalf("expected ErrInvalidContent, got %v", err)
	}
	long := strings.Repeat("x", 20)
	if err := comment.Update(UpdateDTO{Content: long, ContentMaxLength: 10}); !errors.Is(err, ErrInvalidContent) {
		t.Fatalf("expected ErrInvalidContent for long content, got %v", err)
	}
}
