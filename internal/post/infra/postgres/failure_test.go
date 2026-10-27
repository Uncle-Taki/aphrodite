package postgres

import (
	"aphrodite/internal/post/domain"
	"aphrodite/internal/post/usecase"
	"context"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"testing"
)

func TestRepository_PreservesDatabaseErrors(t *testing.T) {
	failure := errors.New("database operation failed")
	db := newTestDB(t)
	repo := New(db)
	if err := db.Callback().Query().Before("gorm:query").Register("test:fail", func(tx *gorm.DB) { tx.AddError(failure) }); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Create().Before("gorm:create").Register("test:fail", func(tx *gorm.DB) { tx.AddError(failure) }); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register("test:fail", func(tx *gorm.DB) { tx.AddError(failure) }); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Delete().Before("gorm:delete").Register("test:fail", func(tx *gorm.DB) { tx.AddError(failure) }); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.FindByID(context.Background(), uuid.New()); !errors.Is(err, failure) {
		t.Fatalf("lookup: %v", err)
	}
	if _, _, err := repo.List(context.Background(), usecase.ListFilter{Limit: 10}); !errors.Is(err, failure) {
		t.Fatalf("count: %v", err)
	}
	item := &domain.Post{ID: uuid.New()}
	if err := repo.Create(context.Background(), item); !errors.Is(err, failure) {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Update(context.Background(), item); !errors.Is(err, failure) {
		t.Fatalf("update: %v", err)
	}
	if err := repo.Delete(context.Background(), item.ID); !errors.Is(err, failure) {
		t.Fatalf("delete: %v", err)
	}
	if err := db.Callback().Query().Remove("test:fail"); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Query().Before("gorm:query").Register("test:fail-find", func(tx *gorm.DB) {
		if _, counting := tx.Statement.Dest.(*int64); !counting {
			tx.AddError(failure)
		}
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.List(context.Background(), usecase.ListFilter{Limit: 10}); !errors.Is(err, failure) {
		t.Fatalf("find after successful count: %v", err)
	}
}
