package usecase

import (
	"context"
	"errors"
	"testing"

	"aphrodite/internal/user/domain"
	"github.com/google/uuid"
)

type failingUserRepo struct {
	*userMemoryRepo
	createErr, updateErr, listErr error
	updates                       int
}

func (r *failingUserRepo) Create(ctx context.Context, u *domain.User) error {
	if r.createErr != nil {
		return r.createErr
	}
	return r.userMemoryRepo.Create(ctx, u)
}
func (r *failingUserRepo) Update(ctx context.Context, u *domain.User) error {
	r.updates++
	if r.updateErr != nil {
		return r.updateErr
	}
	return r.userMemoryRepo.Update(ctx, u)
}
func (r *failingUserRepo) List(ctx context.Context, f ListFilter) ([]*domain.User, int64, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	return r.userMemoryRepo.List(ctx, f)
}

type failingHasher struct {
	hash string
	err  error
}

func (h failingHasher) Hash(context.Context, string) (string, error) { return h.hash, h.err }
func (h failingHasher) Verify(context.Context, string, string) error { return nil }

func TestChangePassword_FailureDoesNotPersist(t *testing.T) {
	failure := errors.New("dependency failed")
	for _, tc := range []struct {
		name                             string
		current, next                    string
		lookup, hashErr, updateErr, want error
		hash                             string
		updates                          int
	}{
		{name: "missing current", next: "new", want: domain.ErrInvalidCredential},
		{name: "missing new", current: "old-password", want: domain.ErrInvalidCredential},
		{name: "missing user", current: "old-password", next: "new", lookup: domain.ErrNotFound, want: domain.ErrInvalidCredential},
		{name: "lookup failure", current: "old-password", next: "new", lookup: failure, want: failure},
		{name: "hash failure", current: "old-password", next: "new", hashErr: failure, want: failure},
		{name: "invalid hash", current: "old-password", next: "new", want: domain.ErrInvalidPasswordHash},
		{name: "save failure", current: "old-password", next: "new", hash: "new-hash", updateErr: failure, want: failure, updates: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := uuid.New()
			memory := newUserMemoryRepo(testUser(id, domain.RoleUser))
			memory.findErr = tc.lookup
			repo := &failingUserRepo{userMemoryRepo: memory, updateErr: tc.updateErr}
			err := NewChangePassword(repo, failingHasher{hash: tc.hash, err: tc.hashErr}, nil).Execute(context.Background(), ChangePasswordInput{UserID: id, CurrentPassword: tc.current, NewPassword: tc.next})
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
			if repo.updates != tc.updates {
				t.Fatalf("updates = %d, want %d", repo.updates, tc.updates)
			}
			if memory.users[id].PasswordHash != "hash:old-password" {
				t.Fatal("failed change persisted password")
			}
		})
	}
}

func TestRegisterUser_FailuresDoNotCreateUser(t *testing.T) {
	failure := errors.New("dependency failed")
	for _, tc := range []struct {
		name, password, username string
		hasher                   PasswordHasher
		createErr, want          error
	}{
		{"missing password", "", "alice", fakePasswordHasher{}, nil, domain.ErrInvalidCredential},
		{"hash failure", "password", "alice", failingHasher{err: failure}, nil, failure},
		{"invalid user", "password", "", fakePasswordHasher{}, nil, domain.ErrInvalidUsername},
		{"duplicate", "password", "alice", fakePasswordHasher{}, domain.ErrAlreadyExists, domain.ErrAlreadyExists},
		{"storage failure", "password", "alice", fakePasswordHasher{}, failure, failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &failingUserRepo{userMemoryRepo: newUserMemoryRepo(), createErr: tc.createErr}
			got, err := NewRegisterUser(repo, tc.hasher, "", nil, nil).Execute(context.Background(), RegisterInput{Username: tc.username, Email: "alice@example.com", Password: tc.password})
			if got != nil || !errors.Is(err, tc.want) {
				t.Fatalf("got %v, %v; want %v", got, err, tc.want)
			}
			if len(repo.users) != 0 {
				t.Fatal("failed registration saved a user")
			}
		})
	}
}

func TestUpdateUser_PropagatesFailures(t *testing.T) {
	failure := errors.New("dependency failed")
	for _, tc := range []struct {
		name, username     string
		lookup, save, want error
	}{
		{"lookup", "alice", failure, nil, failure},
		{"validation", "", nil, nil, domain.ErrInvalidUsername},
		{"duplicate", "alice", nil, domain.ErrAlreadyExists, domain.ErrAlreadyExists},
		{"deleted during update", "alice", nil, domain.ErrNotFound, domain.ErrNotFound},
		{"storage", "alice", nil, failure, failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := uuid.New()
			memory := newUserMemoryRepo(testUser(id, domain.RoleUser))
			memory.findErr = tc.lookup
			repo := &failingUserRepo{userMemoryRepo: memory, updateErr: tc.save}
			got, err := NewUpdateUser(repo, nil).Execute(context.Background(), UpdateInput{TargetID: id, CallerID: id, CallerRole: "user", Username: tc.username, Email: "alice@example.com"})
			if got != nil || !errors.Is(err, tc.want) {
				t.Fatalf("got %v, %v; want %v", got, err, tc.want)
			}
		})
	}
}

func TestListUsers_PropagatesRepositoryError(t *testing.T) {
	failure := errors.New("query failed")
	repo := &failingUserRepo{userMemoryRepo: newUserMemoryRepo(), listErr: failure}
	got, err := NewListUsers(repo, ListConfig{DefaultLimit: 10, MaxLimit: 50}).Execute(context.Background(), ListInput{})
	if got != nil || !errors.Is(err, failure) {
		t.Fatalf("got %v, %v", got, err)
	}
}
