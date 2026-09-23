package strapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"aphrodite/internal/content"
)

func testClient(t *testing.T, handler http.Handler) (*Client, func()) {
	t.Helper()
	c, err := newClient(Config{
		BaseURL:         "http://strapi.test",
		APIToken:        "test-token",
		Timeout:         time.Second,
		MaxRetries:      2,
		RetryBackoff:    time.Millisecond,
		MaxRetryBackoff: 2 * time.Millisecond,
	}, &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		result := recorder.Result()
		body := append([]byte(nil), recorder.Body.Bytes()...)
		return &http.Response{
			StatusCode: result.StatusCode,
			Status:     result.Status,
			Header:     result.Header,
			Body:       io.NopCloser(bytes.NewReader(body)),
			Request:    req,
		}, nil
	})}, func(context.Context, time.Duration) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	return c, func() {}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestClientFindBySlugDecodesStrapiV5Document(t *testing.T) {
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("missing API token")
		}
		if r.URL.Path != "/api/articles" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("filters[slug][$eq]"); got != "hello-world" {
			t.Fatalf("unexpected slug filter %q", got)
		}
		if got := r.URL.Query().Get("locale"); got != "fa" {
			t.Fatalf("unexpected locale %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":7,"documentId":"doc-7","title":"سلام","locale":"fa","author":{"documentId":"author-1"}}],"meta":{"pagination":{"page":1,"pageSize":1,"pageCount":1,"total":1}}}`))
	}))
	defer closeServer()

	doc, err := c.FindBySlug(context.Background(), "articles", "hello-world", content.Query{Locale: "fa", Status: "published", Populate: []string{"author"}})
	if err != nil {
		t.Fatal(err)
	}
	if doc.DocumentID != "doc-7" || doc.ID != 7 {
		t.Fatalf("unexpected document identity: %+v", doc)
	}
	if doc.Fields["title"] != "سلام" {
		t.Fatalf("unexpected fields: %+v", doc.Fields)
	}
}

func TestClientFindBySlugDoesNotMutateFilters(t *testing.T) {
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[],"meta":{}}`))
	}))
	defer closeServer()

	filters := map[string]string{"localeCode": "fa"}
	_, _ = c.FindBySlug(context.Background(), "articles", "hello", content.Query{Filters: filters})
	if len(filters) != 1 {
		t.Fatalf("input filters were mutated: %+v", filters)
	}
}

func TestClientMapsNotFound(t *testing.T) {
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"status":404,"name":"NotFoundError","message":"Not Found"}}`))
	}))
	defer closeServer()

	_, err := c.Get(context.Background(), "articles", "missing", content.Query{})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestClientRetriesTransientResponses(t *testing.T) {
	var calls atomic.Int32
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[],"meta":{"pagination":{"page":1,"pageSize":25,"pageCount":0,"total":0}}}`))
	}))
	defer closeServer()

	result, err := c.List(context.Background(), "articles", content.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 || len(result.Items) != 0 {
		t.Fatalf("expected three calls and empty result, calls=%d result=%+v", calls.Load(), result)
	}
}

func TestClientDoesNotRetryValidationErrors(t *testing.T) {
	var calls atomic.Int32
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"status":400,"name":"ValidationError","message":"bad query"}}`))
	}))
	defer closeServer()

	_, err := c.List(context.Background(), "articles", content.Query{})
	if calls.Load() != 1 {
		t.Fatalf("expected one call, got %d", calls.Load())
	}
	if !strings.Contains(err.Error(), "bad query") {
		t.Fatalf("expected mapped error message, got %v", err)
	}
}

func TestClientMapsValidationErrorsToInvalidRequest(t *testing.T) {
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"status":400,"name":"ValidationError","message":"Invalid key slug"}}`))
	}))
	defer closeServer()

	_, err := c.List(context.Background(), "articles", content.Query{})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}
}

func TestClientEncodesFiltersAsStrapiV5Operators(t *testing.T) {
	var gotQuery url.Values
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[],"meta":{"pagination":{"page":1,"pageSize":1,"pageCount":0,"total":0}}}`))
	}))
	defer closeServer()

	_, err := c.List(context.Background(), "articles", content.Query{
		Filters: map[string]string{"mainCategory": "beauty"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := gotQuery.Get("filters[mainCategory][$eq]"); got != "beauty" {
		t.Fatalf("expected nested $eq filter, got %q (raw %v)", got, gotQuery)
	}
}

func TestConfigValidation(t *testing.T) {
	if _, err := New(Config{BaseURL: "", Timeout: time.Second}); err == nil {
		t.Fatal("expected empty base URL error")
	}
	if _, err := New(Config{BaseURL: "ftp://strapi.local", Timeout: time.Second}); err == nil {
		t.Fatal("expected invalid scheme error")
	}
	if _, err := New(Config{BaseURL: "http://strapi.local", Timeout: time.Second}); err == nil {
		t.Fatal("expected missing API token error")
	}
	if _, err := New(Config{BaseURL: "http://strapi.local", APIToken: "replace-with-a-scoped-server-token", Timeout: time.Second}); err == nil {
		t.Fatal("expected placeholder API token error")
	}
}

func TestClientCreateSendsStrapiV5Body(t *testing.T) {
	var gotMethod, gotPath, gotContentType string
	var gotBody map[string]any
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":9,"documentId":"doc-9","name":"Beauty","slug":"beauty"}}`))
	}))
	defer closeServer()

	doc, err := c.Create(context.Background(), "main-categories", map[string]any{"name": "Beauty"}, content.Query{Locale: "fa"})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/main-categories" {
		t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
	}
	if gotContentType != "application/json" {
		t.Fatalf("unexpected content type %q", gotContentType)
	}
	if gotBody["data"].(map[string]any)["name"] != "Beauty" {
		t.Fatalf("payload not wrapped in data envelope: %+v", gotBody)
	}
	if doc.DocumentID != "doc-9" {
		t.Fatalf("unexpected document: %+v", doc)
	}
}

func TestClientUpdateSendsPutToDocumentPath(t *testing.T) {
	var gotMethod, gotPath string
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":9,"documentId":"doc-9","name":"New name"}}`))
	}))
	defer closeServer()

	doc, err := c.Update(context.Background(), "main-categories", "doc-9", map[string]any{"name": "New name"}, content.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/main-categories/doc-9" {
		t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
	}
	if doc.DocumentID != "doc-9" {
		t.Fatalf("unexpected document: %+v", doc)
	}
}

func TestClientDeleteUsesDocumentPath(t *testing.T) {
	var gotMethod, gotPath string
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":9,"documentId":"doc-9"}}`))
	}))
	defer closeServer()

	if err := c.Delete(context.Background(), "main-categories", "doc-9", content.Query{}); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/main-categories/doc-9" {
		t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
	}
}

func TestClientCreateMapsValidationErrors(t *testing.T) {
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"status":400,"name":"ValidationError","message":"name is required"}}`))
	}))
	defer closeServer()

	_, err := c.Create(context.Background(), "main-categories", map[string]any{}, content.Query{})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}
}

func TestClientGetRejectsEmptyDocumentID(t *testing.T) {
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer closeServer()
	if _, err := c.Get(context.Background(), "articles", "  ", content.Query{}); err == nil {
		t.Fatal("expected empty document ID to be rejected")
	}
}

func TestClientGetDecodesSingleDocument(t *testing.T) {
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/articles/doc-7" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":7,"documentId":"doc-7","title":"سلام"},"meta":{"pagination":{"total":1}}}`))
	}))
	defer closeServer()

	doc, err := c.Get(context.Background(), "articles", "doc-7", content.Query{})
	if err != nil {
		t.Fatal(err)
	}
	if doc.DocumentID != "doc-7" || doc.Fields["title"] != "سلام" {
		t.Fatalf("unexpected document: %+v", doc)
	}
}

func TestClientUpdateDeleteRejectEmptyDocumentID(t *testing.T) {
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer closeServer()
	if _, err := c.Update(context.Background(), "articles", "", nil, content.Query{}); err == nil {
		t.Fatal("expected empty document ID to be rejected on update")
	}
	if err := c.Delete(context.Background(), "articles", " ", content.Query{}); err == nil {
		t.Fatal("expected empty document ID to be rejected on delete")
	}
}

func TestClientRejectsInvalidResource(t *testing.T) {
	c, closeServer := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer closeServer()
	if _, err := c.List(context.Background(), "", content.Query{}); err == nil {
		t.Fatal("expected empty resource to be rejected")
	}
	if _, err := c.List(context.Background(), "/articles", content.Query{}); err == nil {
		t.Fatal("expected absolute path resource to be rejected")
	}
}
