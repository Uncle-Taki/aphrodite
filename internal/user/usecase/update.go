package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

type UpdateInput struct {
	TargetID    uuid.UUID
	CallerID    uuid.UUID
	CallerRole  string
	Username    string
	Email       string
	PhoneNumber *string
	Role        domain.Role
}

type UpdateUser struct {
	repo UserRepository
	now  func() time.Time
}

func NewUpdateUser(repo UserRepository, now func() time.Time) *UpdateUser {
	if now == nil {
		now = time.Now
	}
	return &UpdateUser{repo: repo, now: now}
}

func (uc *UpdateUser) Execute(ctx context.Context, in UpdateInput) (*domain.User, error) {
	if in.CallerRole != string(domain.RoleAdmin) && in.TargetID != in.CallerID {
		slog.WarnContext(ctx, "user: update forbidden",
			"target_id", in.TargetID, "caller_id", in.CallerID, "caller_role", in.CallerRole)
		return nil, domain.ErrForbidden
	}

	u, err := uc.repo.FindByID(ctx, in.TargetID)
	if err != nil {
		return nil, err
	}

	role := u.Role
	if in.CallerRole == string(domain.RoleAdmin) && in.Role != "" {
		role = in.Role
	}

	if err := u.Update(domain.UpdateDTO{
		Username:    in.Username,
		Email:       in.Email,
		PhoneNumber: in.PhoneNumber,
		Role:        role,
		Now:         uc.now(),
	}); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(ctx, u); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) || errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		slog.ErrorContext(ctx, "user: repository update failed", "err", err, "user_id", in.TargetID)
		return nil, err
	}

	slog.InfoContext(ctx, "user: updated",
		"user_id", in.TargetID, "caller_id", in.CallerID, "caller_role", in.CallerRole)
	return u, nil
}
