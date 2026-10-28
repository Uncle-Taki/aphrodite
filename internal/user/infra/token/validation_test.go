package token

import (
	"aphrodite/internal/user/domain"
	"context"
	"errors"
	"testing"
	"time"
)

func TestJWT_RejectsInvalidClaimsAndEncoding(t *testing.T) {
	issuer, err := newJWT(testJWTConfig(time.Hour), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range []*domain.User{nil, {Role: domain.Role("unknown")}} {
		raw, err := issuer.Issue(context.Background(), user)
		if raw != "" || err == nil {
			t.Fatalf("invalid user issued token: %q %v", raw, err)
		}
	}
	header := encodeSegment([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims := encodeSegment([]byte(`{"sub":"b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f","role":"user","exp":9999999999}`))
	for _, tc := range []struct{ name, header, claims, signature string }{
		{"header base64", "!", claims, ""},
		{"header json", encodeSegment([]byte("{")), claims, ""},
		{"algorithm", encodeSegment([]byte(`{"alg":"none","typ":"JWT"}`)), claims, ""},
		{"signature base64", header, claims, "!"},
		{"claims base64", header, "!", ""},
		{"claims json", header, encodeSegment([]byte("{")), ""},
		{"subject", header, encodeSegment([]byte(`{"sub":"invalid","role":"user","exp":9999999999}`)), ""},
		{"role", header, encodeSegment([]byte(`{"sub":"b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f","role":"unknown","exp":9999999999}`)), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.header + "." + tc.claims
			signature := tc.signature
			if signature == "" {
				signature = encodeSegment(issuer.sign(input))
			}
			id, role, err := issuer.Verify(context.Background(), input+"."+signature)
			if !errors.Is(err, domain.ErrInvalidCredential) || id != [16]byte{} || role != "" {
				t.Fatalf("invalid token accepted: %s %s %v", id, role, err)
			}
		})
	}
	if _, _, err := issuer.Verify(context.Background(), "invalid"); !errors.Is(err, domain.ErrInvalidCredential) {
		t.Fatalf("segment count: %v", err)
	}
}
