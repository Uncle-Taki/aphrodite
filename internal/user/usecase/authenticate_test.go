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

func TestAuthenticateUser_EditorReceivesBearerToken(t *testing.T) {
	user := testUser(uuid.New(), domain.RoleEditor)
	uc := NewAuthenticateUser(newUserMemoryRepo(user), fakePasswordHasher{}, fakeTokenIssuer{})
	result, err := uc.Execute(context.Background(), AuthenticateInput{Identifier: "alice", Password: "old-password"})
	if err != nil {
		t.Fatalf("expected editor bearer login to succeed, got %v", err)
	}
	if result.Token == "" || result.User.Role != domain.RoleEditor {
		t.Fatalf("unexpected editor authentication result: %+v", result)
	}
}
