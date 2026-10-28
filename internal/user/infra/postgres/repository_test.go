package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"aphrodite/internal/user/domain"
	"aphrodite/internal/user/usecase"
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

func testUser(id uuid.UUID, role domain.Role) *domain.User {
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	return &domain.User{
		ID:           id,
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hash",
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestUserRepository_CreateAndFind(t *testing.T) {
	db := newTestDB(t)
	repo := New(db)
	id := uuid.New()

	if err := repo.Create(context.Background(), testUser(id, domain.RoleUser)); err != nil {
		t.Fatal(err)
	}

	byID, err := repo.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if byID.Username != "alice" || byID.Role != domain.RoleUser {
		t.Fatalf("unexpected user from FindByID: %+v", byID)
	}

	byUsername, err := repo.FindByUsername(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if byUsername.ID != id {
		t.Fatalf("unexpected user from FindByUsername: %+v", byUsername)
	}

	byEmail, err := repo.FindByEmail(context.Background(), "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if byEmail.ID != id {
		t.Fatalf("unexpected user from FindByEmail: %+v", byEmail)
	}
}

func TestUserRepository_FindReturnsNotFound(t *testing.T) {
	repo := New(newTestDB(t))

	for name, err := range map[string]error{
		"by id":       func() error { _, err := repo.FindByID(context.Background(), uuid.New()); return err }(),
		"by username": func() error { _, err := repo.FindByUsername(context.Background(), "ghost"); return err }(),
		"by email":    func() error { _, err := repo.FindByEmail(context.Background(), "ghost@example.com"); return err }(),
	} {
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s: expected ErrNotFound, got %v", name, err)
		}
	}
}

func TestUserRepository_ListPaginates(t *testing.T) {
	db := newTestDB(t)
	repo := New(db)

	base := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		u := testUser(uuid.New(), domain.RoleUser)
		u.Username = "alice" + string(rune('a'+i))
		u.Email = string(rune('a'+i)) + "@example.com"
		u.CreatedAt = base.Add(time.Duration(i) * time.Minute)
		if err := repo.Create(context.Background(), u); err != nil {
			t.Fatal(err)
		}
	}

	users, total, err := repo.List(context.Background(), usecase.ListFilter{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(users) != 2 {
		t.Fatalf("expected page of 2 with total 3, got len=%d total=%d", len(users), total)
	}
	// newest first: the last-created user must be first
	if users[0].CreatedAt.Before(users[1].CreatedAt) {
		t.Fatal("expected created_at DESC ordering")
	}

	users, _, err = repo.List(context.Background(), usecase.ListFilter{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 {
		t.Fatalf("expected remainder page of 1, got %d", len(users))
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := newTestDB(t)
	repo := New(db)
	id := uuid.New()
	if err := repo.Create(context.Background(), testUser(id, domain.RoleUser)); err != nil {
		t.Fatal(err)
	}

	updated := testUser(id, domain.RoleAdmin)
	updated.Username = "alice2"
	updated.Email = "alice2@example.com"
	if err := repo.Update(context.Background(), updated); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "alice2" || got.Role != domain.RoleAdmin {
		t.Fatalf("update not applied: %+v", got)
	}
}

func TestUserRepository_UpdateMissingReturnsNotFound(t *testing.T) {
	repo := New(newTestDB(t))
	err := repo.Update(context.Background(), testUser(uuid.New(), domain.RoleUser))
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
