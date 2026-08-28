package strapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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
		if got := r.URL.Query().Get("filters[slug[$eq]]"); got != "hello-world" {
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

	filters := map[string]string{"localeCode[$eq]": "fa"}
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

func TestConfigValidation(t *testing.T) {
	if _, err := New(Config{BaseURL: "", Timeout: time.Second}); err == nil {
		t.Fatal("expected empty base URL error")
	}
	if _, err := New(Config{BaseURL: "ftp://strapi.local", Timeout: time.Second}); err == nil {
		t.Fatal("expected invalid scheme error")
	}
}
