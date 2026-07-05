package usecase

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

type userMemoryRepo struct {
	users map[uuid.UUID]*domain.User
}

func newUserMemoryRepo(users ...*domain.User) *userMemoryRepo {
	repo := &userMemoryRepo{users: map[uuid.UUID]*domain.User{}}
	for _, u := range users {
		cp := *u
		repo.users[u.ID] = &cp
	}
	return repo
}

func (r *userMemoryRepo) Create(_ context.Context, u *domain.User) error {
	cp := *u
	r.users[u.ID] = &cp
	return nil
}

func (r *userMemoryRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *userMemoryRepo) FindByUsername(_ context.Context, username string) (*domain.User, error) {
	for _, u := range r.users {
		if u.Username == username {
			cp := *u
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *userMemoryRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *userMemoryRepo) List(_ context.Context, filter ListFilter) ([]*domain.User, int64, error) {
	users := make([]*domain.User, 0, len(r.users))
	for _, u := range r.users {
		cp := *u
		users = append(users, &cp)
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].CreatedAt.After(users[j].CreatedAt)
	})

	start := filter.Offset
	if start > len(users) {
		start = len(users)
	}
	end := start + filter.Limit
	if end > len(users) {
		end = len(users)
	}
	return users[start:end], int64(len(users)), nil
}

func (r *userMemoryRepo) Update(_ context.Context, u *domain.User) error {
	if _, ok := r.users[u.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *u
	r.users[u.ID] = &cp
	return nil
}

type fakePasswordHasher struct{}

func (fakePasswordHasher) Hash(_ context.Context, plaintext string) (string, error) {
	return "hash:" + plaintext, nil
}

func (fakePasswordHasher) Verify(_ context.Context, hash, plaintext string) error {
	if hash != "hash:"+plaintext {
		return domain.ErrInvalidCredential
	}
	return nil
}

func testUser(id uuid.UUID, role domain.Role) *domain.User {
	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	return &domain.User{
		ID:           id,
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: "hash:old-password",
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestUpdateUser_SelfCanUpdateProfileButNotRole(t *testing.T) {
	userID := uuid.New()
	updatedAt := time.Date(2026, 7, 5, 13, 0, 0, 0, time.UTC)
	repo := newUserMemoryRepo(testUser(userID, domain.RoleUser))
	uc := NewUpdateUser(repo, func() time.Time { return updatedAt })

	got, err := uc.Execute(context.Background(), UpdateInput{
		TargetID:   userID,
		CallerID:   userID,
		CallerRole: string(domain.RoleUser),
		Username:   "  Alice2  ",
		Email:      "ALICE2@EXAMPLE.COM",
		Role:       domain.RoleAdmin,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "Alice2" || got.Email != "alice2@example.com" || got.Role != domain.RoleUser {
		t.Fatalf("unexpected updated user: %+v", got)
	}
	if !got.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("updated_at not changed: %v", got.UpdatedAt)
	}
}

func TestUpdateUser_AdminCanUpdateRole(t *testing.T) {
	targetID := uuid.New()
	adminID := uuid.New()
	repo := newUserMemoryRepo(testUser(targetID, domain.RoleUser))
	uc := NewUpdateUser(repo, time.Now)

	got, err := uc.Execute(context.Background(), UpdateInput{
		TargetID:   targetID,
		CallerID:   adminID,
		CallerRole: string(domain.RoleAdmin),
		Username:   "alice",
		Email:      "alice@example.com",
		Role:       domain.RoleAdmin,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Role != domain.RoleAdmin {
		t.Fatalf("role not updated: %+v", got)
	}
}

func TestUpdateUser_RejectsOtherUser(t *testing.T) {
	targetID := uuid.New()
	repo := newUserMemoryRepo(testUser(targetID, domain.RoleUser))
	uc := NewUpdateUser(repo, time.Now)

	_, err := uc.Execute(context.Background(), UpdateInput{
		TargetID:   targetID,
		CallerID:   uuid.New(),
		CallerRole: string(domain.RoleUser),
		Username:   "alice",
		Email:      "alice@example.com",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestChangePassword_VerifiesCurrentPassword(t *testing.T) {
	userID := uuid.New()
	repo := newUserMemoryRepo(testUser(userID, domain.RoleUser))
	uc := NewChangePassword(repo, fakePasswordHasher{}, time.Now)

	if err := uc.Execute(context.Background(), ChangePasswordInput{
		UserID:          userID,
		CurrentPassword: "old-password",
		NewPassword:     "new-password",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := repo.FindByID(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PasswordHash != "hash:new-password" {
		t.Fatalf("password hash not updated: %q", got.PasswordHash)
	}
}

func TestChangePassword_RejectsWrongCurrentPassword(t *testing.T) {
	userID := uuid.New()
	repo := newUserMemoryRepo(testUser(userID, domain.RoleUser))
	uc := NewChangePassword(repo, fakePasswordHasher{}, time.Now)

	err := uc.Execute(context.Background(), ChangePasswordInput{
		UserID:          userID,
		CurrentPassword: "wrong-password",
		NewPassword:     "new-password",
	})
	if !errors.Is(err, domain.ErrInvalidCredential) {
		t.Fatalf("expected invalid credential, got %v", err)
	}
}
