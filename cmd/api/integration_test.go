package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"aphrodite/pkg/config"
)

func TestAPI_LivePostgresRedisAndStrapi(t *testing.T) {
	port := os.Getenv("aphrodite_TEST_POSTGRES_PORT")
	if port == "" {
		t.Skip("run scripts/test_coverage.py --integration for the live API integration test")
	}
	encoded, err := os.ReadFile(os.Getenv("aphrodite_TEST_STRAPI_TOKEN_FILE"))
	if err != nil {
		t.Fatal(err)
	}
	var tokens struct {
		Full string `json:"full"`
	}
	if err := json.Unmarshal(encoded, &tokens); err != nil {
		t.Fatal(err)
	}
	previous := config.C
	t.Cleanup(func() { config.C = previous })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deps := defaultRunDeps()
	deps.loadConfig = func() {
		config.C = config.Config{
			Env: "test", Port: "0",
			Database:   config.DatabaseConfig{Host: "127.0.0.1", Port: port, User: "coverage", Password: "coverage-only", Name: "aphrodite_go_test", SSLMode: "disable"},
			Redis:      config.RedisConfig{Addr: os.Getenv("aphrodite_TEST_REDIS_ADDR")},
			Strapi:     config.StrapiConfig{BaseURL: os.Getenv("aphrodite_TEST_STRAPI_URL"), APIToken: tokens.Full, Timeout: 5 * time.Second, ReadyTimeout: 5 * time.Second, ReadyPollInterval: time.Millisecond, DefaultLocale: "fa", FallbackLocale: "fa"},
			Auth:       config.AuthConfig{JWTSecret: "0123456789abcdef0123456789abcdef", JWTMinSecretBytes: 32, AccessTokenTTL: time.Hour, BcryptCost: 4, SuperAdminKey: "test-bootstrap"},
			Validation: config.ValidationConfig{PostTitleMaxLength: 200, PostContentMaxLength: 1000, CommentContentMaxLength: 500},
			Pagination: config.PaginationConfig{UserDefaultLimit: 10, UserMaxLimit: 50, PostDefaultLimit: 10, PostMaxLimit: 50, CommentDefaultLimit: 10, CommentMaxLimit: 50, ContentDefaultLimit: 15, ContentMaxLimit: 75},
		}
	}
	deps.notifyContext = func(context.Context, ...os.Signal) (context.Context, context.CancelFunc) { return ctx, func() {} }
	completed := make(chan struct{})
	deps.exitOnUnexpectedServeError = func(int) { t.Error("unexpected HTTP serve failure") }
	deps.listenAndServe = func(server *http.Server) error {
		defer close(completed)
		defer cancel()
		live := httptest.NewServer(server.Handler)
		defer live.Close()
		request := func(method, path, token string, payload any, status int) map[string]any {
			t.Helper()
			var body io.Reader
			if payload != nil {
				encoded, err := json.Marshal(payload)
				if err != nil {
					t.Fatal(err)
				}
				body = bytes.NewReader(encoded)
			}
			req, err := http.NewRequest(method, live.URL+path, body)
			if err != nil {
				t.Fatal(err)
			}
			if payload != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			res, err := live.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			data, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != status {
				t.Fatalf("%s %s: status=%d body=%s", method, path, res.StatusCode, data)
			}
			var result map[string]any
			if len(data) > 0 {
				if err := json.Unmarshal(data, &result); err != nil {
					t.Fatal(err)
				}
			}
			return result
		}
		request("GET", "/healthz", "", nil, 200)
		name := fmt.Sprintf("coverage%d", time.Now().UnixNano())
		register := map[string]any{"username": name, "email": name + "@example.com", "password": "test-password", "role": "admin", "super_admin_key": "test-bootstrap"}
		request("POST", "/v1/users/register", "", register, 201)
		request("POST", "/v1/users/register", "", register, 409)
		login := request("POST", "/v1/users/login", "", map[string]any{"identifier": name, "password": "test-password"}, 200)
		token, ok := login["token"].(string)
		if !ok || token == "" {
			t.Fatal("login did not issue a token")
		}
		request("GET", "/v1/users/me", token, nil, 200)
		post := request("POST", "/v1/posts", token, map[string]any{"title": "Integration post", "content": "Stored in PostgreSQL"}, 201)
		postID := post["id"].(string)
		comment := request("POST", "/v1/posts/"+postID+"/comments", token, map[string]any{"content": "Integration comment"}, 201)
		request("GET", "/v1/posts/"+postID+"/comments", "", nil, 200)
		request("DELETE", "/v1/comments/"+comment["id"].(string), token, nil, 204)
		request("PUT", "/v1/posts/"+postID, token, map[string]any{"title": "Updated post", "content": "Updated in PostgreSQL"}, 200)
		request("DELETE", "/v1/posts/"+postID, token, nil, 204)
		request("GET", "/v1/posts/"+postID, "", nil, 404)
		request("POST", "/v1/articles", "", map[string]any{}, 401)
		article := request("POST", "/v1/articles", token, map[string]any{"title": "Live article", "slug": name, "body": "Stored in Strapi"}, 201)
		documentID := article["document_id"].(string)
		request("PUT", "/v1/articles/"+documentID, token, map[string]any{"title": "Updated live article"}, 200)
		request("GET", "/v1/articles", "", nil, 200)
		request("DELETE", "/v1/articles/"+documentID, token, nil, 204)
		return http.ErrServerClosed
	}
	if code := runWithDeps(deps); code != 0 {
		t.Fatalf("startup failed with code %d", code)
	}
	<-completed
}
