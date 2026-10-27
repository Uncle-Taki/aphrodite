package strapi

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"aphrodite/internal/content"
)

func TestClient_LiveStrapiCredentialsAndDocumentLifecycle(t *testing.T) {
	baseURL := os.Getenv("aphrodite_TEST_STRAPI_URL")
	if baseURL == "" {
		t.Skip("run scripts/test_coverage.py --integration for live Strapi tests")
	}
	encoded, err := os.ReadFile(os.Getenv("aphrodite_TEST_STRAPI_TOKEN_FILE"))
	if err != nil {
		t.Fatal(err)
	}
	var tokens struct {
		Full     string `json:"full"`
		ReadOnly string `json:"readOnly"`
	}
	if err := json.Unmarshal(encoded, &tokens); err != nil {
		t.Fatal(err)
	}
	newLiveClient := func(token string) *Client {
		t.Helper()
		client, err := New(Config{BaseURL: baseURL, APIToken: token, Timeout: 5 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		return client
	}
	full, readOnly, invalid := newLiveClient(tokens.Full), newLiveClient(tokens.ReadOnly), newLiveClient("invalid-test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	query := content.Query{Locale: "fa", Status: "published"}
	if _, err := invalid.List(ctx, "articles", query); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("invalid token: %v", err)
	}
	slug := "coverage-" + time.Now().Format("150405000000000")
	doc, err := full.Create(ctx, "articles", map[string]any{"title": "Coverage article", "slug": slug, "body": "Integration test body"}, query)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := full.Delete(context.Background(), "articles", doc.DocumentID, content.Query{Locale: "fa"}); err != nil && !errors.Is(err, ErrNotFound) {
			t.Error(err)
		}
	})
	if doc.DocumentID == "" {
		t.Fatal("missing Strapi document ID")
	}
	found, err := readOnly.FindBySlug(ctx, "articles", slug, query)
	if err != nil || found.DocumentID != doc.DocumentID || found.Fields["title"] != "Coverage article" {
		t.Fatalf("published read: %+v %v", found, err)
	}
	if _, err := readOnly.Update(ctx, "articles", doc.DocumentID, map[string]any{"title": "Forbidden edit"}, query); !errors.Is(err, ErrForbidden) {
		t.Fatalf("read-only write: %v", err)
	}
	changed, err := full.Update(ctx, "articles", doc.DocumentID, map[string]any{"title": "Updated article"}, query)
	if err != nil || changed.Fields["title"] != "Updated article" {
		t.Fatalf("update: %+v %v", changed, err)
	}
	if err := full.Delete(ctx, "articles", doc.DocumentID, content.Query{Locale: "fa"}); err != nil {
		t.Fatal(err)
	}
	if _, err := full.FindBySlug(ctx, "articles", slug, query); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted article: %v", err)
	}
}
