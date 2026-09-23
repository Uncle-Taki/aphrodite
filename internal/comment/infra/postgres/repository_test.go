package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"aphrodite/internal/comment/domain"
	"aphrodite/internal/comment/usecase"
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

func testComment(id, postID, authorID uuid.UUID, createdAt time.Time) *domain.Comment {
	return &domain.Comment{
		ID:        id,
		PostID:    postID,
		AuthorID:  authorID,
		Content:   "content",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func TestCommentRepository_CRUD(t *testing.T) {
	db := newTestDB(t)
	repo := New(db)
	id := uuid.New()
	postID := uuid.New()
	authorID := uuid.New()
	created := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

	if err := repo.Create(context.Background(), testComment(id, postID, authorID, created)); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "content" || got.PostID != postID {
		t.Fatalf("unexpected comment: %+v", got)
	}

	updated := testComment(id, postID, authorID, created)
	updated.Content = "new content"
	if err := repo.Update(context.Background(), updated); err != nil {
		t.Fatal(err)
	}
	got, err = repo.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "new content" {
		t.Fatalf("update not applied: %+v", got)
	}

	if err := repo.Delete(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.FindByID(context.Background(), id); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestCommentRepository_FindMissing(t *testing.T) {
	repo := New(newTestDB(t))
	if _, err := repo.FindByID(context.Background(), uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCommentRepository_UpdateMissingReturnsNotFound(t *testing.T) {
	repo := New(newTestDB(t))
	err := repo.Update(context.Background(), testComment(uuid.New(), uuid.New(), uuid.New(), time.Now()))
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCommentRepository_DeleteMissingReturnsNotFound(t *testing.T) {
	repo := New(newTestDB(t))
	if err := repo.Delete(context.Background(), uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCommentRepository_ListByPostFiltersPaginatesAndOrders(t *testing.T) {
	db := newTestDB(t)
	repo := New(db)

	postA := uuid.New()
	postB := uuid.New()
	authorID := uuid.New()
	base := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)

	// three comments on postA (oldest first expected), one on postB
	for i := 0; i < 3; i++ {
		if err := repo.Create(context.Background(), testComment(uuid.New(), postA, authorID, base.Add(time.Duration(i)*time.Minute))); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Create(context.Background(), testComment(uuid.New(), postB, authorID, base)); err != nil {
		t.Fatal(err)
	}

	comments, total, err := repo.ListByPost(context.Background(), usecase.ListFilter{PostID: postA, Limit: 2, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(comments) != 2 {
		t.Fatalf("expected page of 2 with total 3, got len=%d total=%d", len(comments), total)
	}
	if comments[0].CreatedAt.After(comments[1].CreatedAt) {
		t.Fatal("expected created_at ASC ordering")
	}

	// page 2 of postA
	comments, _, err = repo.ListByPost(context.Background(), usecase.ListFilter{PostID: postA, Limit: 2, Offset: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected remainder page of 1, got %d", len(comments))
	}

	// postB must not leak into postA listing
	for _, c := range comments {
		if c.PostID != postA {
			t.Fatalf("comment from another post leaked: %+v", c)
		}
	}
}
