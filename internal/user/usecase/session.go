package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

type SessionRepository interface {
	Create(context.Context, *domain.Session) error
	FindActiveByHash(context.Context, string, time.Time) (*domain.Session, error)
	Touch(context.Context, uuid.UUID, time.Time) error
	Revoke(context.Context, uuid.UUID, time.Time) error
}

type CMSLoginInput struct {
	Identifier string
	Password   string
}

type CMSLoginResult struct {
	User    *domain.User
	Session *domain.Session
	Token   string
}

type CMSSessionManager struct {
	users    UserRepository
	password PasswordHasher
	sessions SessionRepository
	now      func() time.Time
	newID    func() uuid.UUID
	newToken func() (string, error)
	ttl      time.Duration
}

func NewCMSSessionManager(users UserRepository, password PasswordHasher, sessions SessionRepository, ttl time.Duration) *CMSSessionManager {
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	return &CMSSessionManager{users: users, password: password, sessions: sessions, ttl: ttl, now: time.Now, newID: uuid.New, newToken: secureToken}
}

func (m *CMSSessionManager) Login(ctx context.Context, in CMSLoginInput) (*CMSLoginResult, error) {
	in.Identifier = strings.TrimSpace(in.Identifier)
	if in.Identifier == "" || in.Password == "" {
		return nil, domain.ErrInvalidCredential
	}
	u, err := m.users.FindByUsername(ctx, in.Identifier)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		u, err = m.users.FindByEmail(ctx, strings.ToLower(in.Identifier))
	}
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrInvalidCredential
		}
		return nil, err
	}
	if u == nil || u.Disabled || (u.Role != domain.RoleAdmin && u.Role != domain.RoleEditor) {
		return nil, domain.ErrInvalidCredential
	}
	if err := m.password.Verify(ctx, u.PasswordHash, in.Password); err != nil {
		return nil, domain.ErrInvalidCredential
	}
	token, err := m.newToken()
	if err != nil {
		return nil, err
	}
	now := m.now()
	s := &domain.Session{ID: m.newID(), UserID: u.ID, TokenHash: domain.HashSessionToken(token), ExpiresAt: now.Add(m.ttl), CreatedAt: now, LastSeenAt: now}
	if err := m.sessions.Create(ctx, s); err != nil {
		return nil, err
	}
	return &CMSLoginResult{User: u, Session: s, Token: token}, nil
}

func (m *CMSSessionManager) Authenticate(ctx context.Context, token string) (*domain.Session, *domain.User, error) {
	if token == "" {
		return nil, nil, domain.ErrInvalidCredential
	}
	now := m.now()
	s, err := m.sessions.FindActiveByHash(ctx, domain.HashSessionToken(token), now)
	if err != nil {
		return nil, nil, domain.ErrInvalidCredential
	}
	u, err := m.users.FindByID(ctx, s.UserID)
	if err != nil || u.Disabled || (u.Role != domain.RoleAdmin && u.Role != domain.RoleEditor) {
		return nil, nil, domain.ErrInvalidCredential
	}
	_ = m.sessions.Touch(ctx, s.ID, now)
	return s, u, nil
}

func (m *CMSSessionManager) Logout(ctx context.Context, s *domain.Session) error {
	if s == nil {
		return nil
	}
	return m.sessions.Revoke(ctx, s.ID, m.now())
}

func secureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
