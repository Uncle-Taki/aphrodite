package postgres

import (
	"aphrodite/internal/user/domain"
	"aphrodite/internal/user/usecase"
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
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

	if _, err := repo.FindByID(context.Background(), uuid.New()); !errors.Is(err, failure) {
		t.Fatalf("lookup: %v", err)
	}
	if _, _, err := repo.List(context.Background(), usecase.ListFilter{Limit: 10}); !errors.Is(err, failure) {
		t.Fatalf("count: %v", err)
	}
	item := &domain.User{ID: uuid.New()}
	if err := repo.Create(context.Background(), item); !errors.Is(err, failure) {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Update(context.Background(), item); !errors.Is(err, failure) {
		t.Fatalf("update: %v", err)
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

func TestRepository_MapsUniqueConstraintViolations(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		t.Run(operation, func(t *testing.T) {
			db := newTestDB(t)
			unique := &pgconn.PgError{Code: "23505", Message: "duplicate key"}
			callback := func(tx *gorm.DB) { tx.AddError(unique) }
			if operation == "create" {
				if err := db.Callback().Create().Before("gorm:create").Register("test:unique", callback); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := db.Callback().Update().Before("gorm:update").Register("test:unique", callback); err != nil {
					t.Fatal(err)
				}
			}
			repo := New(db)
			user := testUser(uuid.New(), domain.RoleUser)
			var err error
			if operation == "create" {
				err = repo.Create(context.Background(), user)
			} else {
				err = repo.Update(context.Background(), user)
			}
			if !errors.Is(err, domain.ErrAlreadyExists) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
