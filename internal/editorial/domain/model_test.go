package domain

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func validEditorialPost() *Post {
	return &Post{
		ID:       uuid.MustParse("b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f"),
		Slug:     "my-post",
		Language: "fa",
		Status:   StatusDraft,
		Blocks: []Block{
			{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Position: 0, Type: "paragraph", Payload: json.RawMessage(`{"text":"hi"}`)},
			{ID: uuid.MustParse("22222222-2222-2222-2222-222222222222"), Position: 1, Type: "image", Payload: json.RawMessage(`{"src":"a.png"}`)},
		},
	}
}

func TestEditorialPostValidate(t *testing.T) {
	if err := validEditorialPost().Validate(); err != nil {
		t.Fatalf("expected valid post, got %v", err)
	}
}

func TestEditorialPostValidateErrors(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Post)
		want   error
	}{
		{"empty slug", func(p *Post) { p.Slug = "  " }, ErrInvalidSlug},
		{"slug with space", func(p *Post) { p.Slug = "my post" }, ErrInvalidSlug},
		{"slug with slash", func(p *Post) { p.Slug = "my/post" }, ErrInvalidSlug},
		{"slug with question", func(p *Post) { p.Slug = "my?post" }, ErrInvalidSlug},
		{"slug with hash", func(p *Post) { p.Slug = "my#post" }, ErrInvalidSlug},
		{"empty language", func(p *Post) { p.Language = " " }, ErrInvalidLanguage},
		{"negative position", func(p *Post) { p.Blocks[0].Position = -1 }, ErrInvalidBlock},
		{"empty block type", func(p *Post) { p.Blocks[0].Type = "" }, ErrInvalidBlock},
		{"empty payload", func(p *Post) { p.Blocks[0].Payload = nil }, ErrInvalidBlock},
		{"invalid payload json", func(p *Post) { p.Blocks[0].Payload = json.RawMessage(`{broken`) }, ErrInvalidBlock},
		{"duplicate positions", func(p *Post) { p.Blocks[1].Position = 0 }, ErrInvalidBlock},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			post := validEditorialPost()
			tc.mutate(post)
			if err := post.Validate(); !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}

func TestEditorialPostValidateTrimsSlugAndLanguage(t *testing.T) {
	post := validEditorialPost()
	post.Slug = "  my-post  "
	post.Language = " fa "
	if err := post.Validate(); err != nil {
		t.Fatal(err)
	}
	if post.Slug != "my-post" || post.Language != "fa" {
		t.Fatalf("fields not trimmed: %q %q", post.Slug, post.Language)
	}
}

func TestCanTransition(t *testing.T) {
	all := []Status{StatusDraft, StatusInReview, StatusApproved, StatusScheduled, StatusPublished, StatusArchived}
	allowed := map[Status][]Status{
		StatusDraft:     {StatusInReview},
		StatusInReview:  {StatusDraft, StatusApproved},
		StatusApproved:  {StatusScheduled, StatusPublished, StatusInReview},
		StatusScheduled: {StatusPublished, StatusInReview},
		StatusPublished: {StatusArchived, StatusInReview},
		StatusArchived:  {StatusInReview},
	}
	for _, from := range all {
		for _, to := range all {
			want := false
			for _, candidate := range allowed[from] {
				if candidate == to {
					want = true
					break
				}
			}
			if got := CanTransition(from, to); got != want {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", from, to, got, want)
			}
		}
	}
	if CanTransition("bogus", StatusDraft) {
		t.Error("unknown source status must not transition")
	}
}
