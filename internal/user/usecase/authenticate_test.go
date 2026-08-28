package usecase

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

type fakeTokenIssuer struct{}

func (fakeTokenIssuer) Issue(context.Context, *domain.User) (string, error) { return "token", nil }
func (fakeTokenIssuer) Verify(context.Context, string) (uuid.UUID, domain.Role, error) {
	return uuid.Nil, domain.RoleUser, nil
}

func TestAuthenticateUser_EditorUsesCMSSessionInsteadOfBearerToken(t *testing.T) {
	user := testUser(uuid.New(), domain.RoleEditor)
	uc := NewAuthenticateUser(newUserMemoryRepo(user), fakePasswordHasher{}, fakeTokenIssuer{})
	if _, err := uc.Execute(context.Background(), AuthenticateInput{Identifier: "alice", Password: "old-password"}); err != domain.ErrInvalidCredential {
		t.Fatalf("expected editor bearer login to be rejected, got %v", err)
	}
}
