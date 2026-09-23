package token

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

func testJWTConfig(ttl time.Duration) JWTConfig {
	return JWTConfig{Secret: testJWTSecret, TTL: ttl, MinSecretBytes: 32}
}

func testJWTUser() *domain.User {
	return &domain.User{
		ID:   uuid.MustParse("b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f"),
		Role: domain.RoleAdmin,
	}
}

func TestJWT_IssueVerifyRoundTrip(t *testing.T) {
	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	issuer, err := newJWT(testJWTConfig(time.Hour), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}

	raw, err := issuer.Issue(context.Background(), testJWTUser())
	if err != nil {
		t.Fatal(err)
	}
	if parts := strings.Split(raw, "."); len(parts) != 3 {
		t.Fatalf("expected three jwt segments, got %q", raw)
	}

	id, role, err := issuer.Verify(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if id != testJWTUser().ID || role != domain.RoleAdmin {
		t.Fatalf("unexpected verified subject: id=%s role=%s", id, role)
	}
}

func TestJWT_VerifyRejectsExpiredToken(t *testing.T) {
	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	issuer, err := newJWT(testJWTConfig(time.Hour), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	raw, err := issuer.Issue(context.Background(), testJWTUser())
	if err != nil {
		t.Fatal(err)
	}

	now = now.Add(time.Hour + time.Second)
	_, _, err = issuer.Verify(context.Background(), raw)
	if !errors.Is(err, domain.ErrInvalidCredential) {
		t.Fatalf("expected invalid credential for expired token, got %v", err)
	}
}

func TestJWT_VerifyRejectsTamperedToken(t *testing.T) {
	issuer, err := newJWT(testJWTConfig(time.Hour), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := issuer.Issue(context.Background(), testJWTUser())
	if err != nil {
		t.Fatal(err)
	}

	tampered := raw[:len(raw)-1] + replacementJWTChar(raw[len(raw)-1])
	_, _, err = issuer.Verify(context.Background(), tampered)
	if !errors.Is(err, domain.ErrInvalidCredential) {
		t.Fatalf("expected invalid credential for tampered token, got %v", err)
	}
}

// replacementJWTChar returns a base64url char that decodes differently from
// last when placed in the final signature position, where only the top four
// bits of the char are significant: chars four alphabet positions apart
// (A-D, E-H, ... w-z) decode to the same byte, so the replacement shifts by
// sixteen positions to land in a different block.
func replacementJWTChar(last byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	idx := strings.IndexByte(alphabet, last)
	if idx < 0 {
		return "A"
	}
	return string(alphabet[(idx+16)%len(alphabet)])
}

func TestJWT_VerifyRejectsWrongSecret(t *testing.T) {
	issuer, err := newJWT(testJWTConfig(time.Hour), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := issuer.Issue(context.Background(), testJWTUser())
	if err != nil {
		t.Fatal(err)
	}

	verifier, err := newJWT(JWTConfig{
		Secret:         "abcdef0123456789abcdef0123456789",
		TTL:            time.Hour,
		MinSecretBytes: 32,
	}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = verifier.Verify(context.Background(), raw)
	if !errors.Is(err, domain.ErrInvalidCredential) {
		t.Fatalf("expected invalid credential for wrong secret, got %v", err)
	}
}

func TestJWT_NewRejectsWeakSettings(t *testing.T) {
	if _, err := NewJWT(JWTConfig{Secret: "short", TTL: time.Hour, MinSecretBytes: 32}); err == nil {
		t.Fatal("expected short secret to be rejected")
	}
	if _, err := NewJWT(JWTConfig{Secret: testJWTSecret, TTL: 0, MinSecretBytes: 32}); err == nil {
		t.Fatal("expected non-positive ttl to be rejected")
	}
}
