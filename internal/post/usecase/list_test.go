package usecase

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"aphrodite/internal/post/domain"
)

type listPostRepo struct {
	updatePostRepo
	filter ListFilter
	err    error
}

func (r *listPostRepo) List(_ context.Context, filter ListFilter) ([]*domain.Post, int64, error) {
	r.filter = filter
	if r.err != nil {
		return nil, 0, r.err
	}
	return nil, 0, nil
}

func TestListPosts_ClampsLimitAndPage(t *testing.T) {
	repo := &listPostRepo{}
	uc := NewListPosts(repo, ListConfig{DefaultLimit: 15, MaxLimit: 75})

	result, err := uc.Execute(context.Background(), ListInput{Limit: 0, Page: -2})
	if err != nil {
		t.Fatal(err)
	}
	if result.Limit != 15 || result.Page != 1 {
		t.Fatalf("expected defaults applied, got limit=%d page=%d", result.Limit, result.Page)
	}
	if repo.filter.Limit != 15 || repo.filter.Offset != 0 {
		t.Fatalf("expected filter with default limit and zero offset, got %+v", repo.filter)
	}
}

func TestListPosts_CapsAtMaxLimit(t *testing.T) {
	repo := &listPostRepo{}
	uc := NewListPosts(repo, ListConfig{DefaultLimit: 15, MaxLimit: 75})

	result, err := uc.Execute(context.Background(), ListInput{Limit: 500, Page: 3})
	if err != nil {
		t.Fatal(err)
	}
	if result.Limit != 75 {
		t.Fatalf("expected limit capped at max, got %d", result.Limit)
	}
	// page 3 of 75 → offset 150
	if repo.filter.Offset != 150 {
		t.Fatalf("expected offset 150, got %d", repo.filter.Offset)
	}
}

func TestListPosts_PropagatesRepoError(t *testing.T) {
	repo := &listPostRepo{err: errFake}
	uc := NewListPosts(repo, ListConfig{DefaultLimit: 15, MaxLimit: 75})
	if _, err := uc.Execute(context.Background(), ListInput{}); err != errFake {
		t.Fatalf("expected repo error, got %v", err)
	}
}

var errFake = &fakeError{}

type fakeError struct{}

func (e *fakeError) Error() string { return "fake" }

func TestListPosts_KeepsPostsAndTotal(t *testing.T) {
	posts := []*domain.Post{{ID: uuid.New()}}
	repo := &countingListRepo{posts: posts, total: 7}
	uc := NewListPosts(repo, ListConfig{DefaultLimit: 15, MaxLimit: 75})

	result, err := uc.Execute(context.Background(), ListInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Posts) != 1 || result.Total != 7 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

type countingListRepo struct {
	listPostRepo
	posts []*domain.Post
	total int64
}

func (r *countingListRepo) List(_ context.Context, _ ListFilter) ([]*domain.Post, int64, error) {
	return r.posts, r.total, nil
}
