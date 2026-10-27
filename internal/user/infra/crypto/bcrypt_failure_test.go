package crypto

import (
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"testing"
)

func TestBcryptHasher_RejectsOversizedPassword(t *testing.T) {
	hash, err := NewBcryptHasher(bcrypt.MinCost).Hash(context.Background(), strings.Repeat("a", 73))
	if hash != "" || !errors.Is(err, bcrypt.ErrPasswordTooLong) {
		t.Fatalf("got %q, %v", hash, err)
	}
}
