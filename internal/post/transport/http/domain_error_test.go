package http

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreatePost_MapsDomainValidationFailure(t *testing.T) {
	repo := newMemoryRepo()
	router := newTestRouter(repo)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, newAuthedRequest("POST", "/v1/posts", `{"title":"`+strings.Repeat("x", 21)+`","content":"body"}`))
	if response.Code != 400 || len(repo.posts) != 0 {
		t.Fatalf("status=%d posts=%d", response.Code, len(repo.posts))
	}
}
