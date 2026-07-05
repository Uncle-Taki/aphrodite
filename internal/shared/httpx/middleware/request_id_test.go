package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"aphrodite/internal/shared/httpx/middleware"
)

func TestRequestID_GeneratesWhenAbsentAndEchoesOnResponse(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestID())
	var captured string
	r.GET("/", func(c *gin.Context) {
		captured = middleware.RequestIDFromContext(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	echoed := w.Header().Get(middleware.HeaderRequestID)
	if echoed == "" {
		t.Fatal("X-Request-ID should be echoed on the response when missing inbound")
	}
	if captured != echoed {
		t.Fatalf("handler-side context value (%q) and response header (%q) must agree", captured, echoed)
	}
	if len(echoed) < 16 {
		t.Fatalf("generated id looks too short: %q", echoed)
	}
	if strings.ContainsAny(echoed, " \n\t") {
		t.Fatalf("generated id contains whitespace: %q", echoed)
	}
}

func TestRequestID_PreservesInboundHeader(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/", func(c *gin.Context) {
		if got := middleware.RequestIDFromContext(c); got != "client-supplied-id" {
			t.Fatalf("expected inbound id to be preserved, got %q", got)
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(middleware.HeaderRequestID, "client-supplied-id")
	r.ServeHTTP(w, req)
	if w.Header().Get(middleware.HeaderRequestID) != "client-supplied-id" {
		t.Fatal("inbound request id must be echoed back")
	}
}

func TestRequestIDFromContext_BlankWhenAbsent(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if middleware.RequestIDFromContext(c) != "" {
		t.Fatal("expected empty string when middleware hasn't run")
	}
}
