package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"aphrodite/internal/comment/domain"
)

type listCommentRepo struct {
	updateCommentRepo
	filter ListFilter
	err    error
}

func (r *listCommentRepo) ListByPost(_ context.Context, filter ListFilter) ([]*domain.Comment, int64, error) {
	r.filter = filter
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

func TestListComments_RejectsNilPost(t *testing.T) {
	uc := NewListComments(&listCommentRepo{}, ListConfig{DefaultLimit: 30, MaxLimit: 90})
	_, err := uc.Execute(context.Background(), ListInput{})
	if !errors.Is(err, domain.ErrInvalidPost) {
		t.Fatalf("expected invalid post, got %v", err)
	}
}

func TestListComments_ClampsLimitAndPage(t *testing.T) {
	repo := &listCommentRepo{}
	uc := NewListComments(repo, ListConfig{DefaultLimit: 30, MaxLimit: 90})

	result, err := uc.Execute(context.Background(), ListInput{PostID: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Limit != 30 || result.Page != 1 {
		t.Fatalf("expected defaults, got limit=%d page=%d", result.Limit, result.Page)
	}
	if repo.filter.Limit != 30 || repo.filter.Offset != 0 {
		t.Fatalf("unexpected filter: %+v", repo.filter)
	}
}

func TestListComments_CapsAtMaxLimitAndOffsetsByPage(t *testing.T) {
	repo := &listCommentRepo{}
	uc := NewListComments(repo, ListConfig{DefaultLimit: 30, MaxLimit: 90})

	result, err := uc.Execute(context.Background(), ListInput{PostID: uuid.New(), Limit: 500, Page: 2})
	if err != nil {
		t.Fatal(err)
	}
	if result.Limit != 90 {
		t.Fatalf("expected capped limit, got %d", result.Limit)
	}
	if repo.filter.Offset != 90 {
		t.Fatalf("expected offset 90 for page 2, got %d", repo.filter.Offset)
	}
	if repo.filter.PostID == uuid.Nil {
		t.Fatal("expected post filter to be forwarded")
	}
}

func TestListComments_PropagatesRepoError(t *testing.T) {
	repo := &listCommentRepo{err: errors.New("db down")}
	uc := NewListComments(repo, ListConfig{DefaultLimit: 30, MaxLimit: 90})
	if _, err := uc.Execute(context.Background(), ListInput{PostID: uuid.New()}); err == nil {
		t.Fatal("expected repo error to propagate")
	}
}
