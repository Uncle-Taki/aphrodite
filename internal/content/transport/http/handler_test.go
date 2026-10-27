package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"aphrodite/internal/content"
)

type fakeSlugReader struct {
	queries   []content.Query
	resources []string
	result    content.Document
	callErr   error
	list      content.DocumentList
	listErr   error
}

func (f *fakeSlugReader) List(_ context.Context, resource string, query content.Query) (content.DocumentList, error) {
	f.queries = append(f.queries, query)
	f.resources = append(f.resources, resource)
	return f.list, f.listErr
}

func (f *fakeSlugReader) Get(context.Context, string, string, content.Query) (content.Document, error) {
	return content.Document{}, nil
}

func (f *fakeSlugReader) FindBySlug(_ context.Context, _ string, _ string, query content.Query) (content.Document, error) {
	f.queries = append(f.queries, query)
	return f.result, f.callErr
}

func TestNewHandlerDefaultsLocales(t *testing.T) {
	h := NewHandler(&fakeSlugReader{}, "", "")
	if h.defaultLocale != "fa" || h.fallbackLocale != "fa" {
		t.Fatalf("unexpected defaults: %+v", h)
	}
}

func newListRouter(t *testing.T, reader *fakeSlugReader) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandlerWithList(reader, nil, "", "", ListConfig{DefaultLimit: 15, MaxLimit: 75})
	Register(r.Group("/v1"), h)
	return r
}

func TestListArticlesDefaults(t *testing.T) {
	reader := &fakeSlugReader{}
	router := newListRouter(t, reader)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/articles", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("unexpected status %d body %s", res.Code, res.Body.String())
	}
	if len(reader.queries) != 1 {
		t.Fatalf("expected one list call, got %d", len(reader.queries))
	}
	query := reader.queries[0]
	if query.Locale != "fa" || query.Page != 1 || query.PageSize != 15 || query.Status != "published" {
		t.Fatalf("unexpected query defaults: %+v", query)
	}
	if len(query.Populate) == 0 {
		t.Fatal("expected populate list to be forwarded")
	}

	var body struct {
		Total int `json:"total"`
		Limit int `json:"limit"`
		Page  int `json:"page"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Limit != 15 || body.Page != 1 {
		t.Fatalf("unexpected response pagination: %+v", body)
	}
}

func TestListArticlesClampsPageSize(t *testing.T) {
	reader := &fakeSlugReader{}
	router := newListRouter(t, reader)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/articles?page=2&pageSize=500", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", res.Code)
	}
	if reader.queries[0].PageSize != 75 || reader.queries[0].Page != 2 {
		t.Fatalf("expected clamped pagination, got %+v", reader.queries[0])
	}
}

func TestListArticlesForwardsSortAndAllowlistedFilters(t *testing.T) {
	reader := &fakeSlugReader{}
	router := newListRouter(t, reader)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/articles?sort=title:asc&sort=createdAt:desc&mainCategory=beauty&author=sara&locale=en", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", res.Code, res.Body.String())
	}
	query := reader.queries[0]
	if len(query.Sort) != 2 || query.Sort[0] != "title:asc" {
		t.Fatalf("unexpected sort: %+v", query.Sort)
	}
	if query.Filters["mainCategory.slug"] != "beauty" || query.Filters["author.slug"] != "sara" {
		t.Fatalf("unexpected filters: %+v", query.Filters)
	}
	if query.Locale != "en" {
		t.Fatalf("unexpected locale %q", query.Locale)
	}
}

func TestListRelationSlugFilters(t *testing.T) {
	for _, path := range []string{
		"/v1/articles?mainCategory=beauty&author=sara",
		"/v1/recipes?mainCategory=beauty&author=sara",
		"/v1/products?brand=example",
	} {
		t.Run(path, func(t *testing.T) {
			reader := &fakeSlugReader{}
			router := newListRouter(t, reader)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path+"&ignored=value&slug=untrusted", nil))
			if res.Code != http.StatusOK {
				t.Fatalf("unexpected status %d: %s", res.Code, res.Body.String())
			}
			want := map[string]string{"mainCategory.slug": "beauty", "author.slug": "sara"}
			if strings.HasPrefix(path, "/v1/products") {
				want = map[string]string{"brand.slug": "example"}
			}
			filters := reader.queries[0].Filters
			if len(filters) != len(want) {
				t.Fatalf("unexpected filters: %+v", filters)
			}
			for key, value := range want {
				if filters[key] != value {
					t.Errorf("filter %s = %q, want %q", key, filters[key], value)
				}
			}
		})
	}
}

func TestWriteMapsUpstreamErrors(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, tc := range []struct {
			name    string
			err     error
			status  int
			message string
		}{
			{"validation", content.ErrInvalidRequest, http.StatusBadRequest, "invalid content payload"},
			{"not found", content.ErrNotFound, http.StatusNotFound, "content not found"},
			{"unavailable", content.ErrUnavailable, http.StatusServiceUnavailable, "content service unavailable"},
			{"forbidden server credential", content.ErrForbidden, http.StatusBadGateway, "content service unavailable"},
		} {
			t.Run(method+"/"+tc.name, func(t *testing.T) {
				writer := &fakeWriter{reader: &fakeSlugReader{}, createErr: tc.err, updateErr: tc.err}
				router := newWriteRouter(t, writer, authGate(false), allow(), allow())
				path := "/v1/articles"
				if method == http.MethodPut {
					path += "/doc-9"
				}
				res := httptest.NewRecorder()
				router.ServeHTTP(res, httptest.NewRequest(method, path, strings.NewReader(`{}`)))
				if res.Code != tc.status {
					t.Fatalf("expected %d, got %d: %s", tc.status, res.Code, res.Body.String())
				}
				var body ErrorResponse
				if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Error != tc.message {
					t.Fatalf("expected %q, got %q", tc.message, body.Error)
				}
			})
		}
	}
}

func TestListMapsUpstreamErrors(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{"rate limited", content.ErrRateLimited, http.StatusServiceUnavailable, "content service unavailable"},
		{"unavailable", content.ErrUnavailable, http.StatusServiceUnavailable, "content service unavailable"},
		{"forbidden", content.ErrForbidden, http.StatusBadGateway, "content service unavailable"},
		{"invalid request", content.ErrInvalidRequest, http.StatusBadGateway, "content service unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reader := &fakeSlugReader{listErr: tc.err}
			router := newListRouter(t, reader)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/articles", nil))
			if res.Code != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, res.Code, res.Body.String())
			}
			if !strings.Contains(res.Body.String(), tc.body) {
				t.Fatalf("expected body %q, got %s", tc.body, res.Body.String())
			}
		})
	}
}

func TestGetBySlugNotFoundUsesHonestMessage(t *testing.T) {
	reader := &fakeSlugReader{callErr: content.ErrNotFound}
	router := newListRouter(t, reader)

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/v1/articles/missing", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "content not found") {
		t.Fatalf("expected not found message, got %s", res.Body.String())
	}
}

func TestGetBySlugFallsBackToFallbackLocale(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reader := &fakeSlugReader{callErr: content.ErrNotFound}
	r := gin.New()
	h := NewHandler(reader, "fa", "en")
	r.GET("/v1/articles/:slug", h.Article)

	res := httptest.NewRecorder()
	router := r
	req := httptest.NewRequest(http.MethodGet, "/v1/articles/hello", nil)
	router.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after fallback, got %d", res.Code)
	}
	if len(reader.queries) != 2 || reader.queries[1].Locale != "en" {
		t.Fatalf("expected fallback retry in en, got %+v", reader.queries)
	}
	if !errors.Is(reader.callErr, content.ErrNotFound) {
		t.Fatal("callErr should be preserved")
	}
}

type fakeWriter struct {
	reader    *fakeSlugReader
	create    payloadCall
	update    payloadCall
	deleteID  string
	deleteErr error
	createErr error
	updateErr error
}

type payloadCall struct {
	resource string
	payload  map[string]any
	query    content.Query
	called   bool
}

func (f *fakeWriter) List(ctx context.Context, resource string, query content.Query) (content.DocumentList, error) {
	return f.reader.List(ctx, resource, query)
}

func (f *fakeWriter) Get(ctx context.Context, resource, documentID string, query content.Query) (content.Document, error) {
	return f.reader.Get(ctx, resource, documentID, query)
}

func (f *fakeWriter) FindBySlug(ctx context.Context, resource, slug string, query content.Query) (content.Document, error) {
	return f.reader.FindBySlug(ctx, resource, slug, query)
}

func (f *fakeWriter) Create(_ context.Context, resource string, payload map[string]any, query content.Query) (content.Document, error) {
	f.create = payloadCall{resource: resource, payload: payload, query: query, called: true}
	if f.createErr != nil {
		return content.Document{}, f.createErr
	}
	return content.Document{ID: 1, DocumentID: "doc-1", Fields: payload}, nil
}

func (f *fakeWriter) Update(_ context.Context, resource, documentID string, payload map[string]any, query content.Query) (content.Document, error) {
	f.update = payloadCall{resource: resource, payload: payload, query: query, called: true}
	if f.updateErr != nil {
		return content.Document{}, f.updateErr
	}
	return content.Document{ID: 1, DocumentID: documentID, Fields: payload}, nil
}

func (f *fakeWriter) Delete(_ context.Context, resource, documentID string, query content.Query) error {
	f.deleteID = documentID
	return f.deleteErr
}

// authGate mimics bearer authentication for write routes.
func authGate(abort bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if abort {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		c.Next()
	}
}

func allow() gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

func roleGate(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c *gin.Context) {
		if _, ok := allowed[c.GetHeader("X-Test-Role")]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "required role not granted"})
			return
		}
		c.Next()
	}
}

func newWriteRouter(t *testing.T, writer *fakeWriter, auth, adminRole, editorRole gin.HandlerFunc) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandlerWithList(writer, writer, "", "", ListConfig{DefaultLimit: 15, MaxLimit: 75})
	RegisterRoutes(r.Group("/v1"), h, auth, adminRole, editorRole)
	return r
}

func TestWriteRoutesRequireAuth(t *testing.T) {
	writer := &fakeWriter{reader: &fakeSlugReader{}}
	router := newWriteRouter(t, writer, authGate(true), allow(), allow())

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/v1/categories", strings.NewReader(`{"name":"Beauty"}`)))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", res.Code, res.Body.String())
	}
	if writer.create.called {
		t.Fatal("writer must not be called without a bearer token")
	}
}

func TestWriteRoutesEnforceAdminEditorPermissionMatrix(t *testing.T) {
	cases := []struct {
		name   string
		method string
		role   string
		status int
		called func(*fakeWriter) bool
	}{
		{"admin creates", http.MethodPost, "admin", http.StatusCreated, func(w *fakeWriter) bool { return w.create.called }},
		{"editor cannot create", http.MethodPost, "editor", http.StatusForbidden, func(w *fakeWriter) bool { return w.create.called }},
		{"admin updates", http.MethodPut, "admin", http.StatusOK, func(w *fakeWriter) bool { return w.update.called }},
		{"editor updates", http.MethodPut, "editor", http.StatusOK, func(w *fakeWriter) bool { return w.update.called }},
		{"admin deletes", http.MethodDelete, "admin", http.StatusNoContent, func(w *fakeWriter) bool { return w.deleteID != "" }},
		{"editor cannot delete", http.MethodDelete, "editor", http.StatusForbidden, func(w *fakeWriter) bool { return w.deleteID != "" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writer := &fakeWriter{reader: &fakeSlugReader{}}
			router := newWriteRouter(t, writer, authGate(false), roleGate("admin"), roleGate("admin", "editor"))
			req := httptest.NewRequest(tc.method, "/v1/categories/doc-9", strings.NewReader(`{"name":"Beauty"}`))
			if tc.method == http.MethodPost {
				req = httptest.NewRequest(tc.method, "/v1/categories", strings.NewReader(`{"name":"Beauty"}`))
			}
			req.Header.Set("X-Test-Role", tc.role)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			if res.Code != tc.status {
				t.Fatalf("expected %d, got %d: %s", tc.status, res.Code, res.Body.String())
			}
			if got := tc.called(writer); got != (tc.status < http.StatusBadRequest) {
				t.Fatalf("writer called=%t for status %d", got, tc.status)
			}
		})
	}
}

func TestCreateCategoryForwardsPayload(t *testing.T) {
	writer := &fakeWriter{reader: &fakeSlugReader{}}
	router := newWriteRouter(t, writer, authGate(false), allow(), allow())

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/v1/categories?locale=en", strings.NewReader(`{"name":"Beauty","slug":"beauty"}`)))
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.Code, res.Body.String())
	}
	if !writer.create.called || writer.create.resource != "main-categories" {
		t.Fatalf("unexpected create call: %+v", writer.create)
	}
	if writer.create.payload["name"] != "Beauty" || writer.create.query.Locale != "en" {
		t.Fatalf("payload/locale not forwarded: %+v", writer.create)
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["document_id"] != "doc-1" {
		t.Fatalf("unexpected response: %s", res.Body.String())
	}
}

func TestUpdateCategoryMapsNotFound(t *testing.T) {
	writer := &fakeWriter{reader: &fakeSlugReader{}, updateErr: content.ErrNotFound}
	router := newWriteRouter(t, writer, authGate(false), allow(), allow())

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPut, "/v1/categories/doc-9", strings.NewReader(`{"name":"New"}`)))
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", res.Code, res.Body.String())
	}
}

func TestDeleteCategoryNoContent(t *testing.T) {
	writer := &fakeWriter{reader: &fakeSlugReader{}}
	router := newWriteRouter(t, writer, authGate(false), allow(), allow())

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodDelete, "/v1/categories/doc-9", nil))
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", res.Code, res.Body.String())
	}
	if writer.deleteID != "doc-9" {
		t.Fatalf("unexpected delete target %q", writer.deleteID)
	}
}

func TestWriteRoutesNotRegisteredWithoutWriter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	reader := &fakeSlugReader{}
	h := NewHandler(reader, "", "")
	RegisterRoutes(r.Group("/v1"), h, authGate(false), allow(), allow())

	res := httptest.NewRecorder()
	router := r
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/v1/categories", strings.NewReader(`{"name":"Beauty"}`)))
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unregistered write route, got %d", res.Code)
	}
}

func TestCreateInvalidJSONRejected(t *testing.T) {
	writer := &fakeWriter{reader: &fakeSlugReader{}}
	router := newWriteRouter(t, writer, authGate(false), allow(), allow())

	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/v1/articles", strings.NewReader(`{not json`)))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
	if writer.create.called {
		t.Fatal("writer must not be called for invalid JSON")
	}
}
