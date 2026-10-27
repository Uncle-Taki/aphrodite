package postgres

import (
	"aphrodite/pkg/config"
	"aphrodite/pkg/logger"
	"errors"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io"
	"log/slog"
	"testing"
)

func TestConnect_ConfiguresConnectionPool(t *testing.T) {
	previous := logger.L
	logger.L = slog.New(slog.NewTextHandler(io.Discard, nil))
	t.Cleanup(func() { logger.L = previous })
	db, err := connect(config.DatabaseConfig{}, func(dialector gorm.Dialector, options ...gorm.Option) (*gorm.DB, error) {
		if dialector.Name() != "postgres" {
			t.Fatalf("unexpected dialect: %s", dialector.Name())
		}
		return gorm.Open(sqlite.Open(":memory:"), options...)
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if sqlDB.Stats().MaxOpenConnections != 25 {
		t.Fatalf("pool size: %+v", sqlDB.Stats())
	}
}

func TestConnect_RejectsInvalidConnectionPool(t *testing.T) {
	db, err := connect(config.DatabaseConfig{}, func(gorm.Dialector, ...gorm.Option) (*gorm.DB, error) { return &gorm.DB{Config: &gorm.Config{}}, nil })
	if db != nil || !errors.Is(err, gorm.ErrInvalidDB) {
		t.Fatalf("got %v, %v", db, err)
	}
}
