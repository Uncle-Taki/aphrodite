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

	"aphrodite/internal/post/domain"
	"aphrodite/internal/post/usecase"
	"aphrodite/internal/shared/httpx/authctx"
)

// memoryRepo is the local in-memory PostRepository backing real usecases.
type memoryRepo struct {
	posts   map[uuid.UUID]*domain.Post
	listErr error
}

func newMemoryRepo(posts ...*domain.Post) *memoryRepo {
	repo := &memoryRepo{posts: map[uuid.UUID]*domain.Post{}}
	for _, p := range posts {
		cp := *p
		repo.posts[p.ID] = &cp
	}
	return repo
}

func (r *memoryRepo) Create(_ context.Context, p *domain.Post) error {
	cp := *p
	r.posts[p.ID] = &cp
	return nil
}

func (r *memoryRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Post, error) {
	p, ok := r.posts[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *memoryRepo) List(_ context.Context, _ usecase.ListFilter) ([]*domain.Post, int64, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	posts := make([]*domain.Post, 0, len(r.posts))
	for _, p := range r.posts {
		cp := *p
		posts = append(posts, &cp)
	}
	return posts, int64(len(posts)), nil
}

func (r *memoryRepo) Update(_ context.Context, p *domain.Post) error {
	if _, ok := r.posts[p.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *p
	r.posts[p.ID] = &cp
	return nil
}

func (r *memoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := r.posts[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.posts, id)
	return nil
}

func newTestRouter(repo *memoryRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewHandler(
		usecase.NewCreatePost(repo, usecase.CreatePostConfig{TitleMaxLength: 20, ContentMaxLength: 50}, nil, nil),
		usecase.NewGetPost(repo),
		usecase.NewListPosts(repo, usecase.ListConfig{DefaultLimit: 15, MaxLimit: 75}),
		usecase.NewUpdatePost(repo, usecase.UpdatePostConfig{TitleMaxLength: 20, ContentMaxLength: 50}, nil),
		usecase.NewDeletePost(repo),
	)
	Register(r.Group("/v1"), handler, authStub(), allowApplicationWrites())
	return r
}

// authStub mimics AuthMiddleware: without a caller the request is rejected.
func authStub() gin.HandlerFunc {
	return func(c *gin.Context) {
		bearer := c.GetHeader("Authorization")
		if bearer == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		authctx.Set(c, authctx.Caller{ID: uuid.MustParse("b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f"), Role: "user"})
		c.Next()
	}
}

func allowApplicationWrites() gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

func newAuthedRequest(method, target, body string) *http.Request {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	req.Header.Set("Authorization", "Bearer test")
	return req
}

func TestPostHandler_Create(t *testing.T) {
	repo := newMemoryRepo()
	router := newTestRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, newAuthedRequest(http.MethodPost, "/v1/posts", `{"title":"Hello","content":"Body"}`))
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.Code, res.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["title"] != "Hello" {
		t.Fatalf("unexpected response: %s", res.Body.String())
	}
	if len(repo.posts) != 1 {
		t.Fatal("post not persisted")
	}
}

func TestPostHandler_CreateRequiresAuth(t *testing.T) {
	router := newTestRouter(newMemoryRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(`{"title":"t","content":"c"}`)))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestPostHandler_CreateRejectsInvalidPayload(t *testing.T) {
	router := newTestRouter(newMemoryRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, newAuthedRequest(http.MethodPost, "/v1/posts", `{"title":"","content":"c"}`))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", res.Code, res.Body.String())
	}
}

func TestPostHandler_GetAndList(t *testing.T) {
	postID := uuid.New()
	authorID := uuid.New()
	now := time.Now()
	repo := newMemoryRepo(&domain.Post{ID: postID, AuthorID: authorID, Title: "T", Content: "C", CreatedAt: now, UpdatedAt: now})
	router := newTestRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/posts/"+postID.String(), nil))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	res = httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/posts", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	var list struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 {
		t.Fatalf("expected one post in list, got %d", list.Total)
	}
}

func TestPostHandler_GetRejectsInvalidUUID(t *testing.T) {
	router := newTestRouter(newMemoryRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/posts/not-a-uuid", nil))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestPostHandler_GetMissingReturns404(t *testing.T) {
	router := newTestRouter(newMemoryRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/posts/"+uuid.New().String(), nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestPostHandler_Update(t *testing.T) {
	postID := uuid.New()
	authorID := uuid.MustParse("b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f") // matches authStub caller
	now := time.Now()
	repo := newMemoryRepo(&domain.Post{ID: postID, AuthorID: authorID, Title: "Old", Content: "Old", CreatedAt: now, UpdatedAt: now})
	router := newTestRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, newAuthedRequest(http.MethodPut, "/v1/posts/"+postID.String(), `{"title":"New","content":"New body"}`))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
}

func TestPostHandler_UpdateForbidden(t *testing.T) {
	postID := uuid.New()
	now := time.Now()
	repo := newMemoryRepo(&domain.Post{ID: postID, AuthorID: uuid.New(), Title: "Old", Content: "Old", CreatedAt: now, UpdatedAt: now})
	router := newTestRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, newAuthedRequest(http.MethodPut, "/v1/posts/"+postID.String(), `{"title":"New","content":"New body"}`))
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
}

func TestPostHandler_UpdateRequiresAuth(t *testing.T) {
	router := newTestRouter(newMemoryRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPut, "/v1/posts/"+uuid.New().String(), strings.NewReader(`{}`)))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestPostHandler_Delete(t *testing.T) {
	postID := uuid.New()
	authorID := uuid.MustParse("b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f")
	now := time.Now()
	repo := newMemoryRepo(&domain.Post{ID: postID, AuthorID: authorID, Title: "T", Content: "C", CreatedAt: now, UpdatedAt: now})
	router := newTestRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, newAuthedRequest(http.MethodDelete, "/v1/posts/"+postID.String(), ""))
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
	if _, ok := repo.posts[postID]; ok {
		t.Fatal("post not deleted")
	}
}

func TestPostHandler_DeleteMissingReturns404(t *testing.T) {
	router := newTestRouter(newMemoryRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, newAuthedRequest(http.MethodDelete, "/v1/posts/"+uuid.New().String(), ""))
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestPostHandler_UnexpectedErrorReturns500(t *testing.T) {
	repo := newMemoryRepo()
	repo.listErr = &unexpectedError{}
	router := newTestRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/posts", nil))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", res.Code, res.Body.String())
	}
}

type unexpectedError struct{}

func (unexpectedError) Error() string { return "boom" }
