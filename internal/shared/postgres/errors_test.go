package postgres

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	if !IsUniqueViolation(&pgconn.PgError{Code: pgUniqueViolationCode}) {
		t.Fatal("expected pg unique violation to be detected")
	}
	if IsUniqueViolation(&pgconn.PgError{Code: "23503"}) {
		t.Fatal("foreign key violation must not be treated as unique conflict")
	}
}
