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

	"aphrodite/internal/comment/domain"
	"aphrodite/internal/comment/usecase"
	"aphrodite/internal/shared/httpx/authctx"
)

// memoryRepo is the local in-memory CommentRepository backing real usecases.
type memoryRepo struct {
	comments map[uuid.UUID]*domain.Comment
	listErr  error
}

func newMemoryRepo(comments ...*domain.Comment) *memoryRepo {
	repo := &memoryRepo{comments: map[uuid.UUID]*domain.Comment{}}
	for _, c := range comments {
		cp := *c
		repo.comments[c.ID] = &cp
	}
	return repo
}

func (r *memoryRepo) Create(_ context.Context, c *domain.Comment) error {
	cp := *c
	r.comments[c.ID] = &cp
	return nil
}

func (r *memoryRepo) FindByID(_ context.Context, id uuid.UUID) (*domain.Comment, error) {
	c, ok := r.comments[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *memoryRepo) ListByPost(_ context.Context, _ usecase.ListFilter) ([]*domain.Comment, int64, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	comments := make([]*domain.Comment, 0, len(r.comments))
	for _, c := range r.comments {
		cp := *c
		comments = append(comments, &cp)
	}
	return comments, int64(len(comments)), nil
}

func (r *memoryRepo) Update(_ context.Context, c *domain.Comment) error {
	if _, ok := r.comments[c.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *c
	r.comments[c.ID] = &cp
	return nil
}

func (r *memoryRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := r.comments[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.comments, id)
	return nil
}

func newTestRouter(repo *memoryRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewHandler(
		usecase.NewAddComment(repo, usecase.AddCommentConfig{ContentMaxLength: 100}, nil, nil),
		usecase.NewListComments(repo, usecase.ListConfig{DefaultLimit: 30, MaxLimit: 90}),
		usecase.NewUpdateComment(repo, usecase.UpdateCommentConfig{ContentMaxLength: 100}, nil),
		usecase.NewDeleteComment(repo),
	)
	Register(r.Group("/v1"), handler, authStub(), allowApplicationWrites())
	return r
}

// authStub mimics AuthMiddleware: without a bearer header the request is rejected.
func authStub() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" {
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
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test")
	return req
}

func TestCommentHandler_Add(t *testing.T) {
	postID := uuid.New()
	repo := newMemoryRepo()
	router := newTestRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, newAuthedRequest(http.MethodPost, "/v1/posts/"+postID.String()+"/comments", `{"content":"  Nice  "}`))
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.Code, res.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["content"] != "Nice" {
		t.Fatalf("unexpected response: %s", res.Body.String())
	}
	if len(repo.comments) != 1 {
		t.Fatal("comment not persisted")
	}
}

func TestCommentHandler_AddRequiresAuth(t *testing.T) {
	postID := uuid.New()
	router := newTestRouter(newMemoryRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/v1/posts/"+postID.String()+"/comments", strings.NewReader(`{"content":"c"}`)))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestCommentHandler_AddRejectsInvalidPostID(t *testing.T) {
	router := newTestRouter(newMemoryRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, newAuthedRequest(http.MethodPost, "/v1/posts/not-a-uuid/comments", `{"content":"c"}`))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestCommentHandler_AddRejectsInvalidPayload(t *testing.T) {
	postID := uuid.New()
	router := newTestRouter(newMemoryRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, newAuthedRequest(http.MethodPost, "/v1/posts/"+postID.String()+"/comments", `{"content":"  "}`))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestCommentHandler_List(t *testing.T) {
	postID := uuid.New()
	authorID := uuid.New()
	now := time.Now()
	repo := newMemoryRepo(&domain.Comment{ID: uuid.New(), PostID: postID, AuthorID: authorID, Content: "c1", CreatedAt: now, UpdatedAt: now})
	router := newTestRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/posts/"+postID.String()+"/comments", nil))
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
		t.Fatalf("expected one comment, got %d", list.Total)
	}
}

func TestCommentHandler_ListRejectsInvalidPostID(t *testing.T) {
	router := newTestRouter(newMemoryRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/posts/not-a-uuid/comments", nil))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestCommentHandler_ListEmptyPostReturns200(t *testing.T) {
	router := newTestRouter(newMemoryRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/posts/"+uuid.New().String()+"/comments", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 for a post with no comments, got %d", res.Code)
	}
}

func TestCommentHandler_Update(t *testing.T) {
	commentID := uuid.New()
	authorID := uuid.MustParse("b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f") // matches authStub caller
	now := time.Now()
	repo := newMemoryRepo(&domain.Comment{ID: commentID, PostID: uuid.New(), AuthorID: authorID, Content: "old", CreatedAt: now, UpdatedAt: now})
	router := newTestRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, newAuthedRequest(http.MethodPut, "/v1/comments/"+commentID.String(), `{"content":"new"}`))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
}

func TestCommentHandler_UpdateForbidden(t *testing.T) {
	commentID := uuid.New()
	now := time.Now()
	repo := newMemoryRepo(&domain.Comment{ID: commentID, PostID: uuid.New(), AuthorID: uuid.New(), Content: "old", CreatedAt: now, UpdatedAt: now})
	router := newTestRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, newAuthedRequest(http.MethodPut, "/v1/comments/"+commentID.String(), `{"content":"new"}`))
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}
}

func TestCommentHandler_UpdateRequiresAuth(t *testing.T) {
	router := newTestRouter(newMemoryRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPut, "/v1/comments/"+uuid.New().String(), strings.NewReader(`{}`)))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestCommentHandler_UpdateMissingReturns404(t *testing.T) {
	router := newTestRouter(newMemoryRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, newAuthedRequest(http.MethodPut, "/v1/comments/"+uuid.New().String(), `{"content":"new"}`))
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestCommentHandler_Delete(t *testing.T) {
	commentID := uuid.New()
	authorID := uuid.MustParse("b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f")
	now := time.Now()
	repo := newMemoryRepo(&domain.Comment{ID: commentID, PostID: uuid.New(), AuthorID: authorID, Content: "c", CreatedAt: now, UpdatedAt: now})
	router := newTestRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, newAuthedRequest(http.MethodDelete, "/v1/comments/"+commentID.String(), ""))
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
	if _, ok := repo.comments[commentID]; ok {
		t.Fatal("comment not deleted")
	}
}

func TestCommentHandler_DeleteMissingReturns404(t *testing.T) {
	router := newTestRouter(newMemoryRepo())
	res := httptest.NewRecorder()
	router.ServeHTTP(res, newAuthedRequest(http.MethodDelete, "/v1/comments/"+uuid.New().String(), ""))
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestCommentHandler_UnexpectedErrorReturns500(t *testing.T) {
	repo := newMemoryRepo()
	repo.listErr = &unexpectedError{}
	router := newTestRouter(repo)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/posts/"+uuid.New().String()+"/comments", nil))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", res.Code, res.Body.String())
	}
}

type unexpectedError struct{}

func (unexpectedError) Error() string { return "boom" }
