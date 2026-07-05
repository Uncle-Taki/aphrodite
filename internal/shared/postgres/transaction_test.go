package postgres

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type txRecord struct {
	ID   uint64 `gorm:"primaryKey"`
	Name string
}

func TestWithTxCommitAndRollback(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&txRecord{}); err != nil {
		t.Fatalf("migrate tx records: %v", err)
	}

	err = WithTx(context.Background(), db, func(tx *gorm.DB) *gorm.DB { return tx }, func(tx *gorm.DB) error {
		return tx.Create(&txRecord{Name: "commit"}).Error
	})
	if err != nil {
		t.Fatalf("commit tx: %v", err)
	}
	var count int64
	if err := db.Model(&txRecord{}).Where("name = ?", "commit").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("commit not persisted: count=%d err=%v", count, err)
	}

	rollbackErr := errors.New("rollback")
	err = WithTx(context.Background(), db, func(tx *gorm.DB) *gorm.DB { return tx }, func(tx *gorm.DB) error {
		if err := tx.Create(&txRecord{Name: "rollback"}).Error; err != nil {
			return err
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("expected rollback error, got %v", err)
	}
	if err := db.Model(&txRecord{}).Where("name = ?", "rollback").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("rollback persisted unexpectedly: count=%d err=%v", count, err)
	}
}
