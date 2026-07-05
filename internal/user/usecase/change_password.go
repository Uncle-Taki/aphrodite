package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

type ChangePasswordInput struct {
	UserID          uuid.UUID
	CurrentPassword string
	NewPassword     string
}

type ChangePassword struct {
	repo   UserRepository
	hasher PasswordHasher
	now    func() time.Time
}

func NewChangePassword(repo UserRepository, hasher PasswordHasher, now func() time.Time) *ChangePassword {
	if now == nil {
		now = time.Now
	}
	return &ChangePassword{repo: repo, hasher: hasher, now: now}
}

func (uc *ChangePassword) Execute(ctx context.Context, in ChangePasswordInput) error {
	if in.CurrentPassword == "" || in.NewPassword == "" {
		return domain.ErrInvalidCredential
	}

	u, err := uc.repo.FindByID(ctx, in.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrInvalidCredential
		}
		slog.ErrorContext(ctx, "user: change password lookup failed", "err", err, "user_id", in.UserID)
		return err
	}

	if err := uc.hasher.Verify(ctx, u.PasswordHash, in.CurrentPassword); err != nil {
		return domain.ErrInvalidCredential
	}

	hash, err := uc.hasher.Hash(ctx, in.NewPassword)
	if err != nil {
		slog.ErrorContext(ctx, "user: password hash failed", "err", err, "user_id", in.UserID)
		return err
	}
	if err := u.ChangePassword(hash, uc.now()); err != nil {
		return err
	}

	if err := uc.repo.Update(ctx, u); err != nil {
		slog.ErrorContext(ctx, "user: repository password update failed", "err", err, "user_id", in.UserID)
		return err
	}

	slog.InfoContext(ctx, "user: password changed", "user_id", in.UserID)
	return nil
}
