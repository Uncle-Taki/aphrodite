package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

type memorySessionRepo struct {
	sessions map[uuid.UUID]*domain.Session
}

func (r *memorySessionRepo) Create(_ context.Context, session *domain.Session) error {
	if r.sessions == nil {
		r.sessions = make(map[uuid.UUID]*domain.Session)
	}
	cp := *session
	r.sessions[session.ID] = &cp
	return nil
}

func (r *memorySessionRepo) FindActiveByHash(_ context.Context, hash string, now time.Time) (*domain.Session, error) {
	for _, session := range r.sessions {
		if session.TokenHash == hash && session.RevokedAt == nil && session.ExpiresAt.After(now) {
			cp := *session
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *memorySessionRepo) Touch(_ context.Context, id uuid.UUID, now time.Time) error {
	if session := r.sessions[id]; session != nil {
		session.LastSeenAt = now
	}
	return nil
}

func (r *memorySessionRepo) Revoke(_ context.Context, id uuid.UUID, now time.Time) error {
	if session := r.sessions[id]; session != nil {
		session.RevokedAt = &now
	}
	return nil
}

func TestCMSSessionManager_AllowsEditorAndNormalizesEmail(t *testing.T) {
	id := uuid.New()
	user := testUser(id, domain.RoleEditor)
	repo := newUserMemoryRepo(user)
	sessions := &memorySessionRepo{}
	manager := NewCMSSessionManager(repo, fakePasswordHasher{}, sessions, time.Hour)
	manager.newToken = func() (string, error) { return "session-token", nil }
	manager.now = func() time.Time { return time.Unix(100, 0) }

	result, err := manager.Login(context.Background(), CMSLoginInput{
		Identifier: " ALICE@EXAMPLE.COM ", Password: "old-password",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.User.ID != id || result.Token != "session-token" {
		t.Fatalf("unexpected login result: %+v", result)
	}
	if len(sessions.sessions) != 1 {
		t.Fatalf("expected one persisted session, got %d", len(sessions.sessions))
	}
}

func TestCMSSessionManager_RejectsDisabledEditor(t *testing.T) {
	id := uuid.New()
	user := testUser(id, domain.RoleEditor)
	user.Disabled = true
	manager := NewCMSSessionManager(newUserMemoryRepo(user), fakePasswordHasher{}, &memorySessionRepo{}, time.Hour)

	if _, err := manager.Login(context.Background(), CMSLoginInput{Identifier: "alice", Password: "old-password"}); err != domain.ErrInvalidCredential {
		t.Fatalf("expected invalid credential for disabled user, got %v", err)
	}
}
