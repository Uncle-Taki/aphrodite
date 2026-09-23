package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
)

// loginSession performs a CMS login with fixed time and returns the manager,
// the session repo, and the raw token for Authenticate exercises.
func loginSession(t *testing.T, role domain.Role, disabled bool) (*CMSSessionManager, *memorySessionRepo, string) {
	t.Helper()
	id := uuid.New()
	user := testUser(id, role)
	user.Disabled = disabled
	repo := newUserMemoryRepo(user)
	sessions := &memorySessionRepo{}
	manager := NewCMSSessionManager(repo, fakePasswordHasher{}, sessions, time.Hour)
	manager.newToken = func() (string, error) { return "session-token", nil }
	manager.now = func() time.Time { return time.Unix(1000, 0) }

	result, err := manager.Login(context.Background(), CMSLoginInput{Identifier: "alice", Password: "old-password"})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	return manager, sessions, result.Token
}

func TestCMSSessionManager_AuthenticateHappyPath(t *testing.T) {
	manager, sessions, token := loginSession(t, domain.RoleEditor, false)

	s, u, err := manager.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if u.Role != domain.RoleEditor || s.UserID != u.ID {
		t.Fatalf("unexpected authenticate result: user=%+v session=%+v", u, s)
	}
	for _, session := range sessions.sessions {
		if !session.LastSeenAt.Equal(time.Unix(1000, 0)) {
			t.Fatalf("session not touched: %+v", session)
		}
	}
}

func TestCMSSessionManager_AuthenticateRejectsEmptyToken(t *testing.T) {
	manager, _, _ := loginSession(t, domain.RoleEditor, false)
	if _, _, err := manager.Authenticate(context.Background(), ""); err != domain.ErrInvalidCredential {
		t.Fatalf("expected invalid credential, got %v", err)
	}
}

func TestCMSSessionManager_AuthenticateRejectsUnknownToken(t *testing.T) {
	manager, _, _ := loginSession(t, domain.RoleEditor, false)
	_, _, err := manager.Authenticate(context.Background(), "not-a-session-token")
	if err != domain.ErrInvalidCredential {
		t.Fatalf("expected invalid credential, got %v", err)
	}
}

func TestCMSSessionManager_AuthenticateRejectsRevokedSession(t *testing.T) {
	manager, sessions, token := loginSession(t, domain.RoleEditor, false)
	for _, session := range sessions.sessions {
		now := time.Unix(1000, 0)
		session.RevokedAt = &now
	}
	if _, _, err := manager.Authenticate(context.Background(), token); err != domain.ErrInvalidCredential {
		t.Fatalf("expected invalid credential for revoked session, got %v", err)
	}
}

func TestCMSSessionManager_AuthenticateRejectsExpiredSession(t *testing.T) {
	manager, sessions, token := loginSession(t, domain.RoleEditor, false)
	for _, session := range sessions.sessions {
		session.ExpiresAt = time.Unix(999, 0) // before manager.now (1000)
	}
	if _, _, err := manager.Authenticate(context.Background(), token); err != domain.ErrInvalidCredential {
		t.Fatalf("expected invalid credential for expired session, got %v", err)
	}
}

func TestCMSSessionManager_AuthenticateRejectsDisabledUser(t *testing.T) {
	manager, sessions, token := loginSession(t, domain.RoleEditor, false)
	for _, session := range sessions.sessions {
		userID := session.UserID
		manager.users.(*userMemoryRepo).users[userID].Disabled = true
	}
	if _, _, err := manager.Authenticate(context.Background(), token); err != domain.ErrInvalidCredential {
		t.Fatalf("expected invalid credential for disabled user, got %v", err)
	}
}

func TestCMSSessionManager_AuthenticateRejectsUserRole(t *testing.T) {
	manager, sessions, token := loginSession(t, domain.RoleEditor, false)
	for _, session := range sessions.sessions {
		userID := session.UserID
		manager.users.(*userMemoryRepo).users[userID].Role = domain.RoleUser
	}
	if _, _, err := manager.Authenticate(context.Background(), token); err != domain.ErrInvalidCredential {
		t.Fatalf("expected invalid credential for plain user role, got %v", err)
	}
}

func TestCMSSessionManager_LogoutRevokesSession(t *testing.T) {
	manager, sessions, token := loginSession(t, domain.RoleEditor, false)
	s, _, err := manager.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Logout(context.Background(), s); err != nil {
		t.Fatal(err)
	}
	for _, session := range sessions.sessions {
		if session.RevokedAt == nil {
			t.Fatal("expected session to be revoked")
		}
	}
}

func TestCMSSessionManager_LogoutNilSessionIsNoop(t *testing.T) {
	manager, _, _ := loginSession(t, domain.RoleEditor, false)
	if err := manager.Logout(context.Background(), nil); err != nil {
		t.Fatalf("nil session logout must be a no-op, got %v", err)
	}
}

func TestCMSSessionManager_LoginRejectsEmptyInput(t *testing.T) {
	manager := NewCMSSessionManager(newUserMemoryRepo(), fakePasswordHasher{}, &memorySessionRepo{}, time.Hour)
	if _, err := manager.Login(context.Background(), CMSLoginInput{Identifier: " ", Password: ""}); err != domain.ErrInvalidCredential {
		t.Fatalf("expected invalid credential, got %v", err)
	}
}

func TestCMSSessionManager_LoginUnknownIdentifier(t *testing.T) {
	manager := NewCMSSessionManager(newUserMemoryRepo(), fakePasswordHasher{}, &memorySessionRepo{}, time.Hour)
	if _, err := manager.Login(context.Background(), CMSLoginInput{Identifier: "ghost", Password: "x"}); err != domain.ErrInvalidCredential {
		t.Fatalf("expected invalid credential, got %v", err)
	}
}

func TestCMSSessionManager_LoginByEmailFallback(t *testing.T) {
	id := uuid.New()
	user := testUser(id, domain.RoleAdmin)
	user.Username = "not-the-identifier"
	repo := newUserMemoryRepo(user)
	sessions := &memorySessionRepo{}
	manager := NewCMSSessionManager(repo, fakePasswordHasher{}, sessions, time.Hour)
	manager.newToken = func() (string, error) { return "session-token", nil }

	result, err := manager.Login(context.Background(), CMSLoginInput{Identifier: "alice@example.com", Password: "old-password"})
	if err != nil {
		t.Fatal(err)
	}
	if result.User.ID != id {
		t.Fatalf("email fallback login failed: %+v", result)
	}
}

func TestCMSSessionManager_DefaultsTTLAndUsesSecureToken(t *testing.T) {
	manager := NewCMSSessionManager(newUserMemoryRepo(testUser(uuid.New(), domain.RoleEditor)), fakePasswordHasher{}, &memorySessionRepo{}, 0)
	if manager.ttl != 12*time.Hour {
		t.Fatalf("expected default 12h ttl, got %v", manager.ttl)
	}
	// secureToken is the production default; call it once to cover it.
	if _, err := manager.newToken(); err != nil {
		t.Fatalf("secureToken failed: %v", err)
	}
}
