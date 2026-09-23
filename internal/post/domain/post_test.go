package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

var testNow = time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

var authorID = uuid.MustParse("b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f")

func TestNewPost(t *testing.T) {
	post, err := NewPost(PostDTO{
		AuthorID:         authorID,
		Title:            "  Hello  ",
		Content:          "  Body  ",
		TitleMaxLength:   100,
		ContentMaxLength: 1000,
		Now:              testNow,
	})
	if err != nil {
		t.Fatal(err)
	}
	if post.Title != "Hello" || post.Content != "Body" {
		t.Errorf("content not trimmed: %+v", post)
	}
	if post.CreatedAt != testNow || post.UpdatedAt != testNow {
		t.Errorf("timestamps not set")
	}
}

func TestNewPostValidation(t *testing.T) {
	long := strings.Repeat("x", 20)
	cases := []struct {
		name string
		dto  PostDTO
		want error
	}{
		{"nil author", PostDTO{Title: "t", Content: "c"}, ErrInvalidAuthor},
		{"empty title", PostDTO{AuthorID: authorID, Title: "  ", Content: "c"}, ErrInvalidTitle},
		{"title too long", PostDTO{AuthorID: authorID, Title: long, Content: "c", TitleMaxLength: 10}, ErrInvalidTitle},
		{"empty content", PostDTO{AuthorID: authorID, Title: "t", Content: " "}, ErrInvalidContent},
		{"content too long", PostDTO{AuthorID: authorID, Title: "t", Content: long, ContentMaxLength: 10}, ErrInvalidContent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewPost(tc.dto)
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

func TestPost_Update(t *testing.T) {
	post, err := NewPost(PostDTO{AuthorID: authorID, Title: "Old", Content: "Old body"})
	if err != nil {
		t.Fatal(err)
	}
	later := testNow.Add(time.Hour)
	if err := post.Update(UpdateDTO{Title: " New ", Content: " New body ", TitleMaxLength: 50, ContentMaxLength: 500, Now: later}); err != nil {
		t.Fatal(err)
	}
	if post.Title != "New" || post.Content != "New body" {
		t.Errorf("update not applied: %+v", post)
	}
	if post.UpdatedAt != later {
		t.Errorf("UpdatedAt not bumped")
	}
}

func TestPost_UpdateValidation(t *testing.T) {
	post, _ := NewPost(PostDTO{AuthorID: authorID, Title: "Old", Content: "Old body"})
	if err := post.Update(UpdateDTO{Title: "", Content: "body", TitleMaxLength: 50, ContentMaxLength: 500}); !errors.Is(err, ErrInvalidTitle) {
		t.Fatalf("expected ErrInvalidTitle, got %v", err)
	}
	if err := post.Update(UpdateDTO{Title: "t", Content: "", TitleMaxLength: 50, ContentMaxLength: 500}); !errors.Is(err, ErrInvalidContent) {
		t.Fatalf("expected ErrInvalidContent, got %v", err)
	}
}

func TestPost_ZeroMaxLengthSkipsLengthCheck(t *testing.T) {
	long := strings.Repeat("x", 500)
	if _, err := NewPost(PostDTO{AuthorID: authorID, Title: long, Content: long}); err != nil {
		t.Fatalf("zero max length must skip checks, got %v", err)
	}
}
