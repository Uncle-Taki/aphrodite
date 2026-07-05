package usecase

import (
	"context"

	"github.com/google/uuid"

	"aphrodite/internal/comment/domain"
)

type ListInput struct {
	PostID uuid.UUID
	Limit  int
	Page   int
}

type ListResult struct {
	Comments []*domain.Comment
	Total    int64
	Limit    int
	Page     int
}

type ListConfig struct {
	DefaultLimit int
	MaxLimit     int
}

type ListComments struct {
	repo   CommentRepository
	config ListConfig
}

func NewListComments(repo CommentRepository, config ListConfig) *ListComments {
	return &ListComments{repo: repo, config: config}
}

func (uc *ListComments) Execute(ctx context.Context, in ListInput) (*ListResult, error) {
	if in.PostID == uuid.Nil {
		return nil, domain.ErrInvalidPost
	}

	limit := in.Limit
	if limit <= 0 {
		limit = uc.config.DefaultLimit
	}
	if limit > uc.config.MaxLimit {
		limit = uc.config.MaxLimit
	}
	page := in.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	comments, total, err := uc.repo.ListByPost(ctx, ListFilter{PostID: in.PostID, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	return &ListResult{Comments: comments, Total: total, Limit: limit, Page: page}, nil
}
