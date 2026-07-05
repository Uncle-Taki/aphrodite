package postgres

import (
	"context"

	"gorm.io/gorm"
)

func WithTx[T any](ctx context.Context, db *gorm.DB, build func(*gorm.DB) T, fn func(T) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(build(tx))
	})
}
