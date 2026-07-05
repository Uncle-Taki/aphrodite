package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeRedisPinger struct {
	err error
}

func (f fakeRedisPinger) Ping(context.Context) error { return f.err }

func TestHealthHandlerHealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	r := gin.New()
	r.GET("/healthz", NewHealthHandler(sqliteHealthDB(t, false), fakeRedisPinger{}).Check)

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"postgres":"up"`) || !strings.Contains(w.Body.String(), `"redis":"up"`) {
		t.Fatalf("unexpected healthy body: %s", w.Body.String())
	}
}

func TestHealthHandlerUnhealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	r := gin.New()
	r.GET("/healthz", NewHealthHandler(sqliteHealthDB(t, true), fakeRedisPinger{err: errors.New("redis down")}).Check)

	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"postgres":"down"`) || !strings.Contains(w.Body.String(), `"redis":"down"`) {
		t.Fatalf("unexpected unhealthy body: %s", w.Body.String())
	}
}

func sqliteHealthDB(t *testing.T, closeSQL bool) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if closeSQL {
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatalf("sql db: %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close sql db: %v", err)
		}
	}
	return db
}
