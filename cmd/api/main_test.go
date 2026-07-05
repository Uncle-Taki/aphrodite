package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"aphrodite/pkg/config"
)

type stubRedis struct{ pingErr error }

func (s stubRedis) Ping(context.Context) error { return s.pingErr }

func TestRunWithDeps_ShutdownOnSignal(t *testing.T) {
	origConfig := config.C
	t.Cleanup(func() { config.C = origConfig })

	shutdownCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deps := runDeps{
		loadConfig: func() {
			config.C = config.Config{
				Env:   "test",
				Debug: false,
				Port:  "0",
				Auth: config.AuthConfig{
					JWTSecret:         "0123456789abcdef0123456789abcdef",
					AccessTokenTTL:    time.Hour,
					JWTMinSecretBytes: 32,
					BcryptCost:        10,
				},
				Validation: config.ValidationConfig{
					PostTitleMaxLength:      200,
					PostContentMaxLength:    100000,
					CommentContentMaxLength: 10000,
				},
				Pagination: config.PaginationConfig{
					PostDefaultLimit:    20,
					PostMaxLimit:        100,
					CommentDefaultLimit: 50,
					CommentMaxLimit:     200,
				},
			}
		},
		initLogger: func(bool) {},
		connectPostgres: func(config.DatabaseConfig) (*gorm.DB, error) {
			return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		},
		newRedis: func(config.RedisConfig) (redisPinger, error) {
			return stubRedis{}, nil
		},
		notifyContext: func(_ context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
			return shutdownCtx, func() {}
		},
		listenAndServe: func(*http.Server) error {
			// Simulate a signal arriving as soon as the server starts serving.
			cancel()
			// Block briefly so runWithDeps's shutdown path is the one that
			// closes the server, mirroring production ordering.
			time.Sleep(10 * time.Millisecond)
			return http.ErrServerClosed
		},
		exitOnUnexpectedServeError: func(int) {
			t.Fatal("unexpected serve-error exit was invoked")
		},
	}

	if code := runWithDeps(deps); code != 0 {
		t.Fatalf("expected clean exit (0), got %d", code)
	}
}

func TestRunWithDeps_PostgresFailureExitsNonZero(t *testing.T) {
	origConfig := config.C
	t.Cleanup(func() { config.C = origConfig })

	deps := runDeps{
		loadConfig:      func() { config.C = config.Config{Env: "test", Port: "0"} },
		initLogger:      func(bool) {},
		connectPostgres: func(config.DatabaseConfig) (*gorm.DB, error) { return nil, errors.New("boom") },
		newRedis:        func(config.RedisConfig) (redisPinger, error) { return stubRedis{}, nil },
		notifyContext: func(ctx context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
			return ctx, func() {}
		},
		listenAndServe:             func(*http.Server) error { return nil },
		exitOnUnexpectedServeError: func(int) {},
	}

	if code := runWithDeps(deps); code == 0 {
		t.Fatal("expected non-zero exit on postgres failure")
	}
}
