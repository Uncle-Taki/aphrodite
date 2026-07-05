package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/post/domain"
)

type updatePostRepo struct {
	posts map[uuid.UUID]*domain.Post
}

func newUpdatePostRepo(posts ...*domain.Post) *updatePostRepo {
	repo := &updatePostRepo{posts: map[uuid.UUID]*domain.Post{}}
	for _, p := range posts {
		cp := *p
		repo.posts[p.ID] = &cp
	}
	return repo
}

func (r *updatePostRepo) Create(_ context.Context, p *domain.Post) error {
	cp := *p
	r.posts[p.ID] = &cp
	return nil
}

func (r *updatePostRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Post, error) {
	p, ok := r.posts[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *updatePostRepo) List(_ context.Context, _ ListFilter) ([]*domain.Post, int64, error) {
	return nil, 0, nil
}

func (r *updatePostRepo) Update(_ context.Context, p *domain.Post) error {
	if _, ok := r.posts[p.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *p
	r.posts[p.ID] = &cp
	return nil
}

func (r *updatePostRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := r.posts[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.posts, id)
	return nil
}

func TestUpdatePost_AuthorCanUpdate(t *testing.T) {
	postID := uuid.New()
	authorID := uuid.New()
	createdAt := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	repo := newUpdatePostRepo(&domain.Post{
		ID:        postID,
		AuthorID:  authorID,
		Title:     "old title",
		Content:   "old content",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	})
	uc := NewUpdatePost(repo, UpdatePostConfig{TitleMaxLength: 20, ContentMaxLength: 50}, func() time.Time {
		return updatedAt
	})

	got, err := uc.Execute(context.Background(), UpdateInput{
		PostID:   postID,
		CallerID: authorID,
		Title:    "  new title  ",
		Content:  "  new content  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "new title" || got.Content != "new content" {
		t.Fatalf("post not updated: %+v", got)
	}
	if !got.CreatedAt.Equal(createdAt) || !got.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("timestamps not preserved/updated: %+v", got)
	}
}

func TestUpdatePost_AdminCanUpdateAnotherUsersPost(t *testing.T) {
	postID := uuid.New()
	repo := newUpdatePostRepo(&domain.Post{
		ID:        postID,
		AuthorID:  uuid.New(),
		Title:     "old title",
		Content:   "old content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	uc := NewUpdatePost(repo, UpdatePostConfig{TitleMaxLength: 20, ContentMaxLength: 50}, time.Now)

	_, err := uc.Execute(context.Background(), UpdateInput{
		PostID:     postID,
		CallerID:   uuid.New(),
		CallerRole: RoleAdmin,
		Title:      "new title",
		Content:    "new content",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUpdatePost_RejectsUnauthorizedCaller(t *testing.T) {
	postID := uuid.New()
	repo := newUpdatePostRepo(&domain.Post{
		ID:        postID,
		AuthorID:  uuid.New(),
		Title:     "old title",
		Content:   "old content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	uc := NewUpdatePost(repo, UpdatePostConfig{TitleMaxLength: 20, ContentMaxLength: 50}, time.Now)

	_, err := uc.Execute(context.Background(), UpdateInput{
		PostID:   postID,
		CallerID: uuid.New(),
		Title:    "new title",
		Content:  "new content",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestUpdatePost_RejectsInvalidPayload(t *testing.T) {
	postID := uuid.New()
	authorID := uuid.New()
	repo := newUpdatePostRepo(&domain.Post{
		ID:        postID,
		AuthorID:  authorID,
		Title:     "old title",
		Content:   "old content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	uc := NewUpdatePost(repo, UpdatePostConfig{TitleMaxLength: 5, ContentMaxLength: 50}, time.Now)

	_, err := uc.Execute(context.Background(), UpdateInput{
		PostID:   postID,
		CallerID: authorID,
		Title:    "too long",
		Content:  "new content",
	})
	if !errors.Is(err, domain.ErrInvalidTitle) {
		t.Fatalf("expected invalid title, got %v", err)
	}
}
