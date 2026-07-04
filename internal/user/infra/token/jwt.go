// Package token holds adapters for the usecase.TokenIssuer port.
package token

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
	"aphrodite/internal/user/usecase"
)

const (
	jwtAlgorithm      = "HS256"
	jwtType           = "JWT"
	minJWTSecretBytes = 32
)

type JWT struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type jwtClaims struct {
	Subject   string `json:"sub"`
	Role      string `json:"role"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

var _ usecase.TokenIssuer = (*JWT)(nil)

func NewJWT(secret string, ttl time.Duration) (*JWT, error) {
	return newJWT(secret, ttl, time.Now)
}

func newJWT(secret string, ttl time.Duration, now func() time.Time) (*JWT, error) {
	if len([]byte(secret)) < minJWTSecretBytes {
		return nil, fmt.Errorf("jwt secret must be at least %d bytes", minJWTSecretBytes)
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("jwt ttl must be positive")
	}
	if now == nil {
		now = time.Now
	}
	return &JWT{secret: []byte(secret), ttl: ttl, now: now}, nil
}

func (j *JWT) Issue(_ context.Context, u *domain.User) (string, error) {
	if u == nil {
		return "", domain.ErrInvalidCredential
	}
	if !u.Role.Valid() {
		return "", domain.ErrInvalidRole
	}

	now := j.now().UTC()
	header := jwtHeader{Algorithm: jwtAlgorithm, Type: jwtType}
	claims := jwtClaims{
		Subject:   u.ID.String(),
		Role:      string(u.Role),
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(j.ttl).Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	signingInput := encodeSegment(headerJSON) + "." + encodeSegment(claimsJSON)
	return signingInput + "." + encodeSegment(j.sign(signingInput)), nil
}

func (j *JWT) Verify(_ context.Context, token string) (uuid.UUID, domain.Role, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return uuid.Nil, "", domain.ErrInvalidCredential
	}

	var header jwtHeader
	if err := decodeJSONSegment(parts[0], &header); err != nil {
		return uuid.Nil, "", domain.ErrInvalidCredential
	}
	if header.Algorithm != jwtAlgorithm || header.Type != jwtType {
		return uuid.Nil, "", domain.ErrInvalidCredential
	}

	signingInput := parts[0] + "." + parts[1]
	signature, err := decodeSegment(parts[2])
	if err != nil {
		return uuid.Nil, "", domain.ErrInvalidCredential
	}
	if !hmac.Equal(signature, j.sign(signingInput)) {
		return uuid.Nil, "", domain.ErrInvalidCredential
	}

	var claims jwtClaims
	if err := decodeJSONSegment(parts[1], &claims); err != nil {
		return uuid.Nil, "", domain.ErrInvalidCredential
	}
	if claims.ExpiresAt <= j.now().UTC().Unix() {
		return uuid.Nil, "", domain.ErrInvalidCredential
	}

	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, "", domain.ErrInvalidCredential
	}
	role := domain.Role(claims.Role)
	if !role.Valid() {
		return uuid.Nil, "", domain.ErrInvalidCredential
	}
	return id, role, nil
}

func (j *JWT) sign(input string) []byte {
	mac := hmac.New(sha256.New, j.secret)
	mac.Write([]byte(input))
	return mac.Sum(nil)
}

func encodeSegment(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeSegment(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

func decodeJSONSegment(s string, out any) error {
	b, err := decodeSegment(s)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}
