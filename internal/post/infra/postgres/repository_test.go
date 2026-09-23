package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"aphrodite/internal/post/domain"
	"aphrodite/internal/post/usecase"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func testPost(id, authorID uuid.UUID, createdAt time.Time) *domain.Post {
	return &domain.Post{
		ID:        id,
		AuthorID:  authorID,
		Title:     "Title",
		Content:   "Content",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func TestPostRepository_CRUD(t *testing.T) {
	db := newTestDB(t)
	repo := New(db)
	id := uuid.New()
	authorID := uuid.New()
	created := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

	if err := repo.Create(context.Background(), testPost(id, authorID, created)); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Title" || got.AuthorID != authorID {
		t.Fatalf("unexpected post: %+v", got)
	}

	updated := testPost(id, authorID, created)
	updated.Title = "New title"
	updated.Content = "New content"
	if err := repo.Update(context.Background(), updated); err != nil {
		t.Fatal(err)
	}
	got, err = repo.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "New title" {
		t.Fatalf("update not applied: %+v", got)
	}

	if err := repo.Delete(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.FindByID(context.Background(), id); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestPostRepository_FindMissing(t *testing.T) {
	repo := New(newTestDB(t))
	if _, err := repo.FindByID(context.Background(), uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPostRepository_UpdateMissingReturnsNotFound(t *testing.T) {
	repo := New(newTestDB(t))
	err := repo.Update(context.Background(), testPost(uuid.New(), uuid.New(), time.Now()))
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPostRepository_DeleteMissingReturnsNotFound(t *testing.T) {
	repo := New(newTestDB(t))
	if err := repo.Delete(context.Background(), uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPostRepository_ListPaginatesNewestFirst(t *testing.T) {
	db := newTestDB(t)
	repo := New(db)
	authorID := uuid.New()
	base := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		if err := repo.Create(context.Background(), testPost(uuid.New(), authorID, base.Add(time.Duration(i)*time.Minute))); err != nil {
			t.Fatal(err)
		}
	}

	posts, total, err := repo.List(context.Background(), usecase.ListFilter{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(posts) != 2 {
		t.Fatalf("expected page of 2 with total 3, got len=%d total=%d", len(posts), total)
	}
	if posts[0].CreatedAt.Before(posts[1].CreatedAt) {
		t.Fatal("expected created_at DESC ordering")
	}

	posts, _, err = repo.List(context.Background(), usecase.ListFilter{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected remainder page of 1, got %d", len(posts))
	}
}
