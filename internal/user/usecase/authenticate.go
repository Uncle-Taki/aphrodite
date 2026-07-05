package usecase

import (
	"context"
	"errors"
	"log/slog"

	"aphrodite/internal/user/domain"
)

type AuthenticateInput struct {
	// Identifier accepts either username or email.
	Identifier string
	Password   string
}

type AuthenticateResult struct {
	User  *domain.User
	Token string
}

type AuthenticateUser struct {
	repo   UserRepository
	hasher PasswordHasher
	tokens TokenIssuer
}

func NewAuthenticateUser(repo UserRepository, hasher PasswordHasher, tokens TokenIssuer) *AuthenticateUser {
	return &AuthenticateUser{repo: repo, hasher: hasher, tokens: tokens}
}

func (uc *AuthenticateUser) Execute(ctx context.Context, in AuthenticateInput) (*AuthenticateResult, error) {
	if in.Identifier == "" || in.Password == "" {
		return nil, domain.ErrInvalidCredential
	}

	u, err := uc.lookup(ctx, in.Identifier)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrInvalidCredential
		}
		slog.ErrorContext(ctx, "user: authenticate lookup failed", "err", err)
		return nil, err
	}

	if err := uc.hasher.Verify(ctx, u.PasswordHash, in.Password); err != nil {
		return nil, domain.ErrInvalidCredential
	}

	token, err := uc.tokens.Issue(ctx, u)
	if err != nil {
		slog.ErrorContext(ctx, "user: token issue failed", "err", err, "user_id", u.ID)
		return nil, err
	}

	slog.InfoContext(ctx, "user: authenticated", "user_id", u.ID)
	return &AuthenticateResult{User: u, Token: token}, nil
}

func (uc *AuthenticateUser) lookup(ctx context.Context, identifier string) (*domain.User, error) {
	u, err := uc.repo.FindByUsername(ctx, identifier)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	return uc.repo.FindByEmail(ctx, identifier)
}
