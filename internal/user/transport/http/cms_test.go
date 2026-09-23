package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"aphrodite/internal/shared/httpx/authctx"
	"aphrodite/internal/user/domain"
	"aphrodite/internal/user/usecase"
)

// ── Session fakes ────────────────────────────────────────────────────────────

type memorySessionRepo struct {
	sessions map[uuid.UUID]*domain.Session
}

func (r *memorySessionRepo) Create(_ context.Context, s *domain.Session) error {
	if r.sessions == nil {
		r.sessions = make(map[uuid.UUID]*domain.Session)
	}
	cp := *s
	r.sessions[s.ID] = &cp
	return nil
}

func (r *memorySessionRepo) FindActiveByHash(_ context.Context, hash string, now time.Time) (*domain.Session, error) {
	for _, s := range r.sessions {
		if s.TokenHash == hash && s.RevokedAt == nil && s.ExpiresAt.After(now) {
			cp := *s
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *memorySessionRepo) Touch(_ context.Context, id uuid.UUID, now time.Time) error {
	if s := r.sessions[id]; s != nil {
		s.LastSeenAt = now
	}
	return nil
}

func (r *memorySessionRepo) Revoke(_ context.Context, id uuid.UUID, now time.Time) error {
	if s := r.sessions[id]; s != nil {
		s.RevokedAt = &now
	}
	return nil
}

func newCMSRouter(repo *memoryUserRepo, sessions *memorySessionRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	manager := usecase.NewCMSSessionManager(repo, stubHasher{}, sessions, time.Hour)
	handler := NewCMSHandler(manager, "test_cms_session", time.Hour, false)
	RegisterCMS(r.Group("/v1"), handler)
	return r
}

func cmsEditorRepo() *memoryUserRepo {
	return newMemoryUserRepo(testUser(uuid.MustParse(callerID), domain.RoleEditor))
}

// loginAndGetCookie performs a CMS login and returns the session cookie.
func loginAndGetCookie(t *testing.T, router *gin.Engine) *http.Cookie {
	t.Helper()
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/v1/cms/auth/login", strings.NewReader(`{"identifier":"alice","password":"old-password"}`)))
	if res.Code != http.StatusOK {
		t.Fatalf("cms login failed: %d %s", res.Code, res.Body.String())
	}
	for _, cookie := range res.Result().Cookies() {
		if cookie.Name == "test_cms_session" {
			return cookie
		}
	}
	t.Fatal("cms session cookie not set")
	return nil
}

// ── CMS auth flow ────────────────────────────────────────────────────────────

func TestCMSHandler_LoginSetsCookie(t *testing.T) {
	sessions := &memorySessionRepo{}
	router := newCMSRouter(cmsEditorRepo(), sessions)

	cookie := loginAndGetCookie(t, router)
	if cookie.Value == "" {
		t.Fatal("expected a non-empty session cookie value")
	}
	if len(sessions.sessions) != 1 {
		t.Fatalf("expected one persisted session, got %d", len(sessions.sessions))
	}
}

func TestCMSHandler_LoginRejectsWrongPassword(t *testing.T) {
	router := newCMSRouter(cmsEditorRepo(), &memorySessionRepo{})

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/v1/cms/auth/login", strings.NewReader(`{"identifier":"alice","password":"wrong"}`)))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", res.Code, res.Body.String())
	}
}

func TestCMSHandler_LoginRejectsInvalidJSON(t *testing.T) {
	router := newCMSRouter(cmsEditorRepo(), &memorySessionRepo{})

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/v1/cms/auth/login", strings.NewReader(`{nope`)))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestCMSHandler_MeWithSession(t *testing.T) {
	router := newCMSRouter(cmsEditorRepo(), &memorySessionRepo{})
	cookie := loginAndGetCookie(t, router)

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/cms/me", nil)
	req.AddCookie(cookie)
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Role != string(domain.RoleEditor) {
		t.Fatalf("unexpected role in response: %s", res.Body.String())
	}
}

func TestCMSHandler_MeRequiresSession(t *testing.T) {
	router := newCMSRouter(cmsEditorRepo(), &memorySessionRepo{})

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/cms/me", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestCMSHandler_MeRejectsInvalidCookie(t *testing.T) {
	router := newCMSRouter(cmsEditorRepo(), &memorySessionRepo{})

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/cms/me", nil)
	req.AddCookie(&http.Cookie{Name: "test_cms_session", Value: "forged"})
	router.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestCMSHandler_LogoutRevokesSession(t *testing.T) {
	sessions := &memorySessionRepo{}
	router := newCMSRouter(cmsEditorRepo(), sessions)
	cookie := loginAndGetCookie(t, router)

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/cms/auth/logout", nil)
	req.AddCookie(cookie)
	router.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
	for _, s := range sessions.sessions {
		if s.RevokedAt == nil {
			t.Fatal("expected session to be revoked")
		}
	}
}

// ── CMS middleware ───────────────────────────────────────────────────────────

func TestCMSAuthMiddlewareSecure_RejectsMissingCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	sessions := usecase.NewCMSSessionManager(cmsEditorRepo(), stubHasher{}, &memorySessionRepo{}, time.Hour)
	r.GET("/secure", CMSAuthMiddlewareSecure(sessions, "test_cms_session", true), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	res := httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/secure", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestCMSAuthMiddlewareSecure_AcceptsValidSession(t *testing.T) {
	sessions := &memorySessionRepo{}
	router := newCMSRouter(cmsEditorRepo(), sessions)
	cookie := loginAndGetCookie(t, router)

	// Build a fresh router whose middleware reads the same session repo.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	manager := usecase.NewCMSSessionManager(cmsEditorRepo(), stubHasher{}, sessions, time.Hour)
	r.GET("/secure", CMSAuthMiddlewareSecure(manager, "test_cms_session", true), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.AddCookie(cookie)
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
}

func TestRequireCMSRole(t *testing.T) {
	cases := []struct {
		name    string
		role    string
		allowed []string
		status  int
	}{
		{"editor allowed", "editor", []string{"editor", "admin"}, http.StatusOK},
		{"admin allowed", "admin", []string{"editor", "admin"}, http.StatusOK},
		{"user rejected", "user", []string{"editor", "admin"}, http.StatusForbidden},
		{"editor rejected for admin-only", "editor", []string{"admin"}, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.GET("/role", func(c *gin.Context) {
				setCallerForTest(c, tc.role)
				c.Next()
			}, RequireCMSRole(tc.allowed...), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			res := httptest.NewRecorder()
			r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/role", nil))
			if res.Code != tc.status {
				t.Fatalf("expected %d, got %d", tc.status, res.Code)
			}
		})
	}
}

func TestRequireCMSRole_NoCallerRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/role", RequireCMSRole("editor"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	res := httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/role", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without caller, got %d", res.Code)
	}
}

func setCallerForTest(c *gin.Context, role string) {
	authctx.Set(c, authctx.Caller{ID: uuid.MustParse(callerID), Role: role})
}
