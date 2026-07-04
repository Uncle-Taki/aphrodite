package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

type RegisterInput struct {
	Username    string
	Email       string
	Password    string
	PhoneNumber *string
	Role        domain.Role
}

type RegisterUser struct {
	repo   UserRepository
	hasher PasswordHasher
	newID  func() uuid.UUID
	now    func() time.Time
}

func NewRegisterUser(repo UserRepository, hasher PasswordHasher, newID func() uuid.UUID, now func() time.Time) *RegisterUser {
	if newID == nil {
		newID = uuid.New
	}
	if now == nil {
		now = time.Now
	}
	return &RegisterUser{repo: repo, hasher: hasher, newID: newID, now: now}
}

func (uc *RegisterUser) Execute(ctx context.Context, in RegisterInput) (*domain.User, error) {
	role := in.Role
	if role == "" {
		role = domain.RoleUser
	}
	if in.Password == "" {
		return nil, domain.ErrInvalidCredential
	}

	hash, err := uc.hasher.Hash(ctx, in.Password)
	if err != nil {
		slog.ErrorContext(ctx, "user: password hash failed", "err", err)
		return nil, err
	}

	u, err := domain.NewUser(domain.UserDTO{
		ID:           uc.newID(),
		Username:     in.Username,
		Email:        in.Email,
		PasswordHash: hash,
		PhoneNumber:  in.PhoneNumber,
		Role:         role,
		Now:          uc.now(),
	})
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, u); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return nil, err
		}
		slog.ErrorContext(ctx, "user: repository create failed", "err", err, "user_id", u.ID)
		return nil, err
	}

	slog.InfoContext(ctx, "user: registered", "user_id", u.ID, "username", u.Username, "role", u.Role)
	return u, nil
}
