package usecase

import (
	"context"

	"aphrodite/internal/user/domain"
)

type ListInput struct {
	Limit int
	Page  int
}

type ListResult struct {
	Users []*domain.User
	Total int64
	Limit int
	Page  int
}

type ListConfig struct {
	DefaultLimit int
	MaxLimit     int
}

type ListUsers struct {
	repo   UserRepository
	config ListConfig
}

func NewListUsers(repo UserRepository, config ListConfig) *ListUsers {
	return &ListUsers{repo: repo, config: config}
}

func (uc *ListUsers) Execute(ctx context.Context, in ListInput) (*ListResult, error) {
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

	users, total, err := uc.repo.List(ctx, ListFilter{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	return &ListResult{Users: users, Total: total, Limit: limit, Page: page}, nil
}
