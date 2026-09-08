package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusInReview  Status = "in_review"
	StatusApproved  Status = "approved"
	StatusScheduled Status = "scheduled"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
)

type Block struct {
	ID       uuid.UUID       `json:"id"`
	Position int             `json:"position"`
	Type     string          `json:"type"`
	Payload  json.RawMessage `json:"payload"`
}

type Post struct {
	ID              uuid.UUID  `json:"id"`
	LegacyPostID    *uuid.UUID `json:"legacy_post_id,omitempty"`
	Slug            string     `json:"slug"`
	Language        string     `json:"language"`
	Status          Status     `json:"status"`
	Version         int64      `json:"version"`
	CanonicalSource *uuid.UUID `json:"canonical_source_id,omitempty"`
	Blocks          []Block    `json:"blocks"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

var (
	ErrInvalidSlug       = errors.New("editorial: invalid slug")
	ErrInvalidLanguage   = errors.New("editorial: invalid language")
	ErrInvalidBlock      = errors.New("editorial: invalid content block")
	ErrInvalidTransition = errors.New("editorial: invalid status transition")
)

func (p *Post) Validate() error {
	p.Slug = strings.TrimSpace(p.Slug)
	p.Language = strings.TrimSpace(p.Language)
	if p.Slug == "" || strings.ContainsAny(p.Slug, " /?#") {
		return ErrInvalidSlug
	}
	if p.Language == "" {
		return ErrInvalidLanguage
	}
	seen := make(map[int]struct{}, len(p.Blocks))
	for _, block := range p.Blocks {
		if block.Position < 0 || block.Type == "" || len(block.Payload) == 0 || !json.Valid(block.Payload) {
			return ErrInvalidBlock
		}
		if _, ok := seen[block.Position]; ok {
			return ErrInvalidBlock
		}
		seen[block.Position] = struct{}{}
	}
	return nil
}

func CanTransition(from, to Status) bool {
	switch from {
	case StatusDraft:
		return to == StatusInReview
	case StatusInReview:
		return to == StatusDraft || to == StatusApproved
	case StatusApproved:
		return to == StatusScheduled || to == StatusPublished || to == StatusInReview
	case StatusScheduled:
		return to == StatusPublished || to == StatusInReview
	case StatusPublished:
		return to == StatusArchived || to == StatusInReview
	case StatusArchived:
		return to == StatusInReview
	default:
		return false
	}
}
