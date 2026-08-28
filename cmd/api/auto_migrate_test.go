package main

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAutoMigrateCreatesApplicationSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := autoMigrate(db); err != nil {
		t.Fatalf("autoMigrate: %v", err)
	}

	for _, table := range []string{"users", "cms_sessions", "posts", "comments"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected table %q", table)
		}
	}
	for _, index := range []struct {
		table string
		name  string
	}{
		{"users", "idx_users_username"},
		{"users", "idx_users_email"},
		{"cms_sessions", "idx_cms_sessions_token_hash"},
		{"cms_sessions", "idx_cms_sessions_user_id"},
		{"cms_sessions", "idx_cms_sessions_expires_at"},
		{"posts", "idx_posts_author_id"},
		{"posts", "idx_posts_created_at"},
		{"comments", "idx_comments_post_id"},
		{"comments", "idx_comments_author_id"},
	} {
		if !db.Migrator().HasIndex(index.table, index.name) {
			t.Errorf("expected index %s on %s", index.name, index.table)
		}
	}

	for _, relation := range []struct {
		table string
		name  string
	}{
		{"posts", "fk_posts_author"},
		{"comments", "fk_comments_post"},
		{"comments", "fk_comments_author"},
		{"cms_sessions", "fk_cms_sessions_user"},
	} {
		if !db.Migrator().HasConstraint(relation.table, relation.name) {
			t.Errorf("expected foreign key %s on %s", relation.name, relation.table)
		}
	}
}
