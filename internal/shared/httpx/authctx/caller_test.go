package authctx

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func newTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c
}

func TestCallerRoundTrip(t *testing.T) {
	c := newTestContext()
	caller := Caller{ID: uuid.MustParse("b3f6a3a8-6df0-4e9e-a9c0-d4a1c1e58a9f"), Role: "admin"}
	Set(c, caller)
	got, ok := From(c)
	if !ok {
		t.Fatal("expected caller to be found after Set")
	}
	if got != caller {
		t.Fatalf("round-trip mismatch: %+v vs %+v", got, caller)
	}
}

func TestFromMissingCaller(t *testing.T) {
	c := newTestContext()
	if _, ok := From(c); ok {
		t.Fatal("expected no caller on a fresh context")
	}
}

func TestFromWrongType(t *testing.T) {
	c := newTestContext()
	c.Set(ContextKey, "not-a-caller")
	if _, ok := From(c); ok {
		t.Fatal("expected From to reject non-Caller values")
	}
}
