package objectstore

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"aphrodite/pkg/config"
)

func TestStore_ObjectLifecycle(t *testing.T) {
	exists := false
	var uploaded, contentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			t.Error("missing S3 request signature")
		}
		switch {
		case r.Method == http.MethodHead && r.URL.Path == "/test-bucket/":
			if !exists {
				w.WriteHeader(http.StatusNotFound)
			}
		case r.Method == http.MethodPut && r.URL.Path == "/test-bucket/":
			exists = true
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && r.URL.Path == "/test-bucket/article.txt":
			data, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			uploaded = string(data)
			contentType = r.Header.Get("Content-Type")
			w.Header().Set("ETag", `"etag"`)
		case r.Method == http.MethodDelete && r.URL.Path == "/test-bucket/article.txt":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()
	store, err := NewS3(config.StorageConfig{Endpoint: strings.TrimPrefix(server.URL, "http://"), AccessKey: "test-access", SecretKey: "test-secret", Region: "us-east-1", Bucket: "test-bucket", ForcePathStyle: true})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.EnsureBucket(ctx); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("bucket not created")
	}
	if err := store.EnsureBucket(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(ctx, "article.txt", strings.NewReader("content"), 7, "text/plain"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(uploaded, "content") || contentType != "text/plain" {
		t.Fatalf("upload body=%q content-type=%q", uploaded, contentType)
	}
	signed, err := store.PresignedGet(ctx, "article.txt", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(signed)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != "/test-bucket/article.txt" || parsed.Query().Get("X-Amz-Signature") == "" || parsed.Query().Get("X-Amz-Expires") != "60" {
		t.Fatalf("invalid presigned URL: %s", signed)
	}
	if err := store.Delete(ctx, "article.txt"); err != nil {
		t.Fatal(err)
	}
}

func TestStore_ReportsFailures(t *testing.T) {
	if _, err := New(config.StorageConfig{Endpoint: "http://invalid/path"}); err == nil {
		t.Fatal("expected invalid endpoint error")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>denied</Message></Error>`)
	}))
	defer server.Close()
	store, err := New(config.StorageConfig{Endpoint: strings.TrimPrefix(server.URL, "http://"), Region: "us-east-1", Bucket: "test-bucket"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureBucket(context.Background()); err == nil {
		t.Fatal("expected bucket permission error")
	}
	if err := store.Put(context.Background(), "key", strings.NewReader("x"), 1, "text/plain"); err == nil {
		t.Fatal("expected upload permission error")
	}
	if err := store.Delete(context.Background(), "key"); err == nil {
		t.Fatal("expected delete permission error")
	}
	if signed, err := store.PresignedGet(context.Background(), "key", 0); err == nil || signed != "" {
		t.Fatalf("invalid expiration: %q, %v", signed, err)
	}
}
