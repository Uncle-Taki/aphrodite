package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"aphrodite/internal/user/domain"
	"aphrodite/internal/user/usecase"
)

// ── Local fakes ──────────────────────────────────────────────────────────────

type memoryUserRepo struct {
	users   map[uuid.UUID]*domain.User
	findErr error
}

func newMemoryUserRepo(users ...*domain.User) *memoryUserRepo {
	repo := &memoryUserRepo{users: map[uuid.UUID]*domain.User{}}
	for _, u := range users {
		cp := *u
		repo.users[u.ID] = &cp
	}
	return repo
}

func (r *memoryUserRepo) Create(_ context.Context, u *domain.User) error {
	for _, existing := range r.users {
		if existing.Username == u.Username || existing.Email == u.Email {
			return domain.ErrAlreadyExists
		}
	}
	cp := *u
	r.users[u.ID] = &cp
	return nil
}

func (r *memoryUserRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *memoryUserRepo) FindByUsername(_ context.Context, username string) (*domain.User, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	for _, u := range r.users {
		if u.Username == username {
			cp := *u
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *memoryUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *memoryUserRepo) List(_ context.Context, _ usecase.ListFilter) ([]*domain.User, int64, error) {
	users := make([]*domain.User, 0, len(r.users))
	for _, u := range r.users {
		cp := *u
		users = append(users, &cp)
	}
	return users, int64(len(users)), nil
}

func (r *memoryUserRepo) Update(_ context.Context, u *domain.User) error {
	if _, ok := r.users[u.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *u
	r.users[u.ID] = &cp
	return nil
}

type stubHasher struct{}

func (stubHasher) Hash(_ context.Context, plaintext string) (string, error) {
	return "hash:" + plaintext, nil
}

func (stubHasher) Verify(_ context.Context, hash, plaintext string) error {
	if hash != "hash:"+plaintext {
		return domain.ErrInvalidCredential
	}
	return nil
}

// verifyTokenIssuer issues "token" and verifies "token"/"admin-token".
type verifyTokenIssuer struct {
	admin bool
}

func (i verifyTokenIssuer) Issue(context.Context, *domain.User) (string, error) {
	return "token", nil
}

func (i verifyTokenIssuer) Verify(_ context.Context, token string) (uuid.UUID, domain.Role, error) {
	switch token {
	case "token":
		return uuid.MustParse("b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f"), domain.RoleUser, nil
	case "admin-token":
		return uuid.MustParse("a3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9e"), domain.RoleAdmin, nil
	default:
		return uuid.Nil, "", domain.ErrInvalidCredential
	}
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

const (
	callerID   = "b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f"
	adminIDStr = "a3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9e"
)

func newUserRouter(repo *memoryUserRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewHandler(
		usecase.NewRegisterUser(repo, stubHasher{}, "", nil, nil),
		usecase.NewAuthenticateUser(repo, stubHasher{}, verifyTokenIssuer{}),
		usecase.NewGetUserProfile(repo),
		usecase.NewListUsers(repo, usecase.ListConfig{DefaultLimit: 10, MaxLimit: 50}),
		usecase.NewUpdateUser(repo, time.Now),
		usecase.NewChangePassword(repo, stubHasher{}, time.Now),
	)
	Register(r.Group("/v1"), handler, verifyTokenIssuer{})
	return r
}

func authedRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer token")
	return req
}

func adminRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer admin-token")
	return req
}

// ── Register / Login ─────────────────────────────────────────────────────────

func TestUserHandler_Register(t *testing.T) {
	repo := newMemoryUserRepo()
	router := newUserRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/v1/users/register", strings.NewReader(`{"username":"bob","email":"bob@example.com","password":"secret"}`)))
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.Code, res.Body.String())
	}
	if len(repo.users) != 1 {
		t.Fatal("user not persisted")
	}
}

func TestUserHandler_RegisterRejectsDuplicate(t *testing.T) {
	existing := testUser(uuid.New(), domain.RoleUser)
	router := newUserRouter(newMemoryUserRepo(existing))

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/v1/users/register", strings.NewReader(`{"username":"alice","email":"alice@example.com","password":"secret"}`)))
	if res.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", res.Code, res.Body.String())
	}
}

func TestUserHandler_Login(t *testing.T) {
	repo := newMemoryUserRepo(testUser(uuid.MustParse(callerID), domain.RoleUser))
	router := newUserRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/v1/users/login", strings.NewReader(`{"identifier":"alice","password":"old-password"}`)))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Token != "token" {
		t.Fatalf("unexpected token: %s", res.Body.String())
	}
}

func TestUserHandler_LoginRejectsBadCredentials(t *testing.T) {
	repo := newMemoryUserRepo(testUser(uuid.MustParse(callerID), domain.RoleUser))
	router := newUserRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/v1/users/login", strings.NewReader(`{"identifier":"alice","password":"wrong"}`)))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

// ── Me / UpdateMe / ChangePassword ───────────────────────────────────────────

func TestUserHandler_Me(t *testing.T) {
	repo := newMemoryUserRepo(testUser(uuid.MustParse(callerID), domain.RoleUser))
	router := newUserRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, authedRequest(http.MethodGet, "/v1/users/me", ""))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
}

func TestUserHandler_MeRequiresAuth(t *testing.T) {
	router := newUserRouter(newMemoryUserRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/users/me", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestUserHandler_MeMissingUserReturns404(t *testing.T) {
	router := newUserRouter(newMemoryUserRepo()) // caller not in repo
	res := httptest.NewRecorder()
	router.ServeHTTP(res, authedRequest(http.MethodGet, "/v1/users/me", ""))
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestUserHandler_UpdateMe(t *testing.T) {
	repo := newMemoryUserRepo(testUser(uuid.MustParse(callerID), domain.RoleUser))
	router := newUserRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, authedRequest(http.MethodPut, "/v1/users/me", `{"username":"alice2","email":"alice2@example.com"}`))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
}

func TestUserHandler_ChangePassword(t *testing.T) {
	repo := newMemoryUserRepo(testUser(uuid.MustParse(callerID), domain.RoleUser))
	router := newUserRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, authedRequest(http.MethodPut, "/v1/users/me/password", `{"current_password":"old-password","new_password":"new-password"}`))
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", res.Code, res.Body.String())
	}
}

func TestUserHandler_ChangePasswordRejectsWrongCurrent(t *testing.T) {
	repo := newMemoryUserRepo(testUser(uuid.MustParse(callerID), domain.RoleUser))
	router := newUserRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, authedRequest(http.MethodPut, "/v1/users/me/password", `{"current_password":"wrong","new_password":"new-password"}`))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

// ── Admin routes ─────────────────────────────────────────────────────────────

func TestUserHandler_AdminList(t *testing.T) {
	repo := newMemoryUserRepo(testUser(uuid.MustParse(adminIDStr), domain.RoleAdmin))
	router := newUserRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, adminRequest(http.MethodGet, "/v1/users", ""))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var list struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 {
		t.Fatalf("expected one user, got %d", list.Total)
	}
}

func TestUserHandler_ListRequiresAdmin(t *testing.T) {
	repo := newMemoryUserRepo(testUser(uuid.MustParse(callerID), domain.RoleUser))
	router := newUserRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, authedRequest(http.MethodGet, "/v1/users", ""))
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin, got %d", res.Code)
	}
}

func TestUserHandler_AdminGetByID(t *testing.T) {
	id := uuid.MustParse(adminIDStr)
	repo := newMemoryUserRepo(testUser(id, domain.RoleAdmin))
	router := newUserRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, adminRequest(http.MethodGet, "/v1/users/"+id.String(), ""))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
}

func TestUserHandler_AdminGetByIDInvalidUUID(t *testing.T) {
	router := newUserRouter(newMemoryUserRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, adminRequest(http.MethodGet, "/v1/users/not-a-uuid", ""))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestUserHandler_AdminGetByIDMissing(t *testing.T) {
	router := newUserRouter(newMemoryUserRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, adminRequest(http.MethodGet, "/v1/users/"+uuid.New().String(), ""))
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestUserHandler_AdminUpdateByID(t *testing.T) {
	id := uuid.MustParse(adminIDStr)
	repo := newMemoryUserRepo(testUser(id, domain.RoleAdmin))
	router := newUserRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, adminRequest(http.MethodPut, "/v1/users/"+id.String(), `{"username":"alice","email":"alice@example.com","role":"admin"}`))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
}

func TestUserHandler_AdminUpdateRequiresAdmin(t *testing.T) {
	router := newUserRouter(newMemoryUserRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, authedRequest(http.MethodPut, "/v1/users/"+uuid.New().String(), `{"username":"a","email":"a@b.co"}`))
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
}

func TestUserHandler_UnexpectedErrorReturns500(t *testing.T) {
	repo := newMemoryUserRepo()
	repo.findErr = errors.New("db down")
	router := newUserRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, authedRequest(http.MethodGet, "/v1/users/me", ""))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", res.Code, res.Body.String())
	}
}

// ── Middleware ───────────────────────────────────────────────────────────────

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/whoami", AuthMiddleware(verifyTokenIssuer{}), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	cases := []struct {
		name   string
		header string
		status int
	}{
		{"valid token", "Bearer token", http.StatusOK},
		{"admin token", "Bearer admin-token", http.StatusOK},
		{"missing header", "", http.StatusUnauthorized},
		{"not bearer", "Basic abc", http.StatusUnauthorized},
		{"unknown token", "Bearer nope", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			r.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("expected %d, got %d", tc.status, res.Code)
			}
		})
	}
}
