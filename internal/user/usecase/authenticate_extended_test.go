package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

func TestAuthenticateUser_HappyPathByUsername(t *testing.T) {
	id := uuid.New()
	uc := NewAuthenticateUser(newUserMemoryRepo(testUser(id, domain.RoleUser)), fakePasswordHasher{}, fakeTokenIssuer{})

	result, err := uc.Execute(context.Background(), AuthenticateInput{Identifier: "alice", Password: "old-password"})
	if err != nil {
		t.Fatal(err)
	}
	if result.User.ID != id || result.Token != "token" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestAuthenticateUser_HappyPathByEmailLookup(t *testing.T) {
	id := uuid.New()
	repo := newUserMemoryRepo(testUser(id, domain.RoleUser))
	repo.users[id].Username = "not-the-identifier"
	uc := NewAuthenticateUser(repo, fakePasswordHasher{}, fakeTokenIssuer{})

	result, err := uc.Execute(context.Background(), AuthenticateInput{Identifier: "alice@example.com", Password: "old-password"})
	if err != nil {
		t.Fatal(err)
	}
	if result.User.ID != id {
		t.Fatalf("email lookup failed: %+v", result)
	}
}

func TestAuthenticateUser_RejectsEmptyInput(t *testing.T) {
	uc := NewAuthenticateUser(newUserMemoryRepo(), fakePasswordHasher{}, fakeTokenIssuer{})
	cases := []AuthenticateInput{
		{Identifier: "", Password: "x"},
		{Identifier: "alice", Password: ""},
	}
	for _, in := range cases {
		if _, err := uc.Execute(context.Background(), in); !errors.Is(err, domain.ErrInvalidCredential) {
			t.Fatalf("expected invalid credential for %+v, got %v", in, err)
		}
	}
}

func TestAuthenticateUser_UnknownIdentifier(t *testing.T) {
	uc := NewAuthenticateUser(newUserMemoryRepo(), fakePasswordHasher{}, fakeTokenIssuer{})
	if _, err := uc.Execute(context.Background(), AuthenticateInput{Identifier: "ghost", Password: "x"}); !errors.Is(err, domain.ErrInvalidCredential) {
		t.Fatalf("expected invalid credential, got %v", err)
	}
}

func TestAuthenticateUser_RejectsDisabledUser(t *testing.T) {
	id := uuid.New()
	user := testUser(id, domain.RoleUser)
	user.Disabled = true
	uc := NewAuthenticateUser(newUserMemoryRepo(user), fakePasswordHasher{}, fakeTokenIssuer{})

	if _, err := uc.Execute(context.Background(), AuthenticateInput{Identifier: "alice", Password: "old-password"}); !errors.Is(err, domain.ErrInvalidCredential) {
		t.Fatalf("expected invalid credential, got %v", err)
	}
}

func TestAuthenticateUser_RejectsWrongPassword(t *testing.T) {
	uc := NewAuthenticateUser(newUserMemoryRepo(testUser(uuid.New(), domain.RoleUser)), fakePasswordHasher{}, fakeTokenIssuer{})

	if _, err := uc.Execute(context.Background(), AuthenticateInput{Identifier: "alice", Password: "wrong"}); !errors.Is(err, domain.ErrInvalidCredential) {
		t.Fatalf("expected invalid credential, got %v", err)
	}
}

func TestAuthenticateUser_PropagatesRepoError(t *testing.T) {
	repo := newUserMemoryRepo()
	repo.findErr = errors.New("db down")
	uc := NewAuthenticateUser(repo, fakePasswordHasher{}, fakeTokenIssuer{})

	if _, err := uc.Execute(context.Background(), AuthenticateInput{Identifier: "alice", Password: "x"}); err == nil || errors.Is(err, domain.ErrInvalidCredential) {
		t.Fatalf("expected repo error to propagate, got %v", err)
	}
}

func TestAuthenticateUser_PropagatesTokenIssueError(t *testing.T) {
	uc := NewAuthenticateUser(
		newUserMemoryRepo(testUser(uuid.New(), domain.RoleUser)),
		fakePasswordHasher{},
		failingTokenIssuer{},
	)
	if _, err := uc.Execute(context.Background(), AuthenticateInput{Identifier: "alice", Password: "old-password"}); err == nil {
		t.Fatal("expected token issue error to propagate")
	}
}

type failingTokenIssuer struct{}

func (failingTokenIssuer) Issue(context.Context, *domain.User) (string, error) {
	return "", errors.New("signing failed")
}

func (failingTokenIssuer) Verify(context.Context, string) (uuid.UUID, domain.Role, error) {
	return uuid.Nil, domain.RoleUser, nil
}
