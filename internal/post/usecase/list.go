package usecase

import (
	"context"

	"aphrodite/internal/post/domain"
)

type ListInput struct {
	Limit int
	Page  int
}

type ListResult struct {
	Posts []*domain.Post
	Total int64
	Limit int
	Page  int
}

type ListConfig struct {
	DefaultLimit int
	MaxLimit     int
}

type ListPosts struct {
	repo   PostRepository
	config ListConfig
}

func NewListPosts(repo PostRepository, config ListConfig) *ListPosts {
	return &ListPosts{repo: repo, config: config}
}

func (uc *ListPosts) Execute(ctx context.Context, in ListInput) (*ListResult, error) {
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

	posts, total, err := uc.repo.List(ctx, ListFilter{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	return &ListResult{Posts: posts, Total: total, Limit: limit, Page: page}, nil
}
