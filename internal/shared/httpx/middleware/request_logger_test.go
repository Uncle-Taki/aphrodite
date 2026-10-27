package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"aphrodite/pkg/logger"
	"github.com/gin-gonic/gin"
)

func TestRequestLogger_RecordsCompletedRequest(t *testing.T) {
	var output bytes.Buffer
	previous := logger.L
	logger.L = slog.New(slog.NewJSONHandler(&output, nil))
	t.Cleanup(func() { logger.L = previous })
	r := gin.New()
	r.Use(RequestID(), RequestLogger())
	r.GET("/example", func(c *gin.Context) {
		if got := RequestIDFromStdContext(c.Request.Context()); got != "trace-123" {
			t.Fatalf("context request ID = %q", got)
		}
		c.Status(http.StatusAccepted)
	})
	req := httptest.NewRequest(http.MethodGet, "/example?private=value", nil)
	req.Header.Set(HeaderRequestID, "trace-123")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["method"] != "GET" || entry["path"] != "/example" || entry["status"] != float64(202) || entry["request_id"] != "trace-123" {
		t.Fatalf("unexpected log: %s", output.String())
	}
	if got := RequestIDFromStdContext(context.Background()); got != "" {
		t.Fatalf("unexpected request ID: %q", got)
	}
}
