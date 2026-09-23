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

func TestRunWithDeps_AutoMigrateFailureExitsNonZero(t *testing.T) {
	origConfig := config.C
	t.Cleanup(func() { config.C = origConfig })

	deps := runDeps{
		loadConfig: func() { config.C = config.Config{Env: "test", Port: "0"} },
		initLogger: func(bool) {},
		connectPostgres: func(config.DatabaseConfig) (*gorm.DB, error) {
			return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		},
		autoMigrate: func(*gorm.DB) error { return errors.New("schema unavailable") },
		newRedis:    func(config.RedisConfig) (redisPinger, error) { return stubRedis{}, nil },
		listenAndServe: func(*http.Server) error {
			t.Fatal("server should not start after AutoMigrate failure")
			return nil
		},
	}

	if code := runWithDeps(deps); code == 0 {
		t.Fatal("expected non-zero exit on AutoMigrate failure")
	}
}

func TestRunWithDeps_StrapiContentWiring(t *testing.T) {
	origConfig := config.C
	t.Cleanup(func() { config.C = origConfig })

	shutdownCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deps := runDeps{
		loadConfig: func() {
			config.C = config.Config{
				Env:  "test",
				Port: "0",
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
					ContentDefaultLimit: 15,
					ContentMaxLimit:     75,
				},
				Strapi: config.StrapiConfig{
					BaseURL:        "http://strapi.test",
					APIToken:       "test-token",
					Timeout:        time.Second,
					DefaultLocale:  "fa",
					FallbackLocale: "fa",
				},
			}
		},
		initLogger: func(bool) {},
		connectPostgres: func(config.DatabaseConfig) (*gorm.DB, error) {
			return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		},
		waitForStrapi: func(context.Context, config.StrapiConfig) error { return nil },
		newRedis: func(config.RedisConfig) (redisPinger, error) {
			return stubRedis{}, nil
		},
		notifyContext: func(_ context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
			return shutdownCtx, func() {}
		},
		listenAndServe: func(*http.Server) error {
			cancel()
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

func TestRunWithDeps_StrapiReadinessFailureExitsNonZero(t *testing.T) {
	origConfig := config.C
	t.Cleanup(func() { config.C = origConfig })

	deps := runDeps{
		loadConfig: func() {
			config.C = config.Config{
				Env:  "test",
				Port: "0",
				Strapi: config.StrapiConfig{
					BaseURL:      "http://strapi.test",
					ReadyTimeout: time.Second,
				},
			}
		},
		initLogger: func(bool) {},
		connectPostgres: func(config.DatabaseConfig) (*gorm.DB, error) {
			return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		},
		waitForStrapi:              func(context.Context, config.StrapiConfig) error { return errors.New("strapi unreachable") },
		newRedis:                   func(config.RedisConfig) (redisPinger, error) { return stubRedis{}, nil },
		listenAndServe:             func(*http.Server) error { return nil },
		exitOnUnexpectedServeError: func(int) {},
	}

	if code := runWithDeps(deps); code == 0 {
		t.Fatal("expected non-zero exit on strapi readiness failure")
	}
}

func TestRunWithDeps_RedisFailureExitsNonZero(t *testing.T) {
	origConfig := config.C
	t.Cleanup(func() { config.C = origConfig })

	deps := runDeps{
		loadConfig: func() { config.C = config.Config{Env: "test", Port: "0"} },
		initLogger: func(bool) {},
		connectPostgres: func(config.DatabaseConfig) (*gorm.DB, error) {
			return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		},
		newRedis:                   func(config.RedisConfig) (redisPinger, error) { return nil, errors.New("redis down") },
		listenAndServe:             func(*http.Server) error { return nil },
		exitOnUnexpectedServeError: func(int) {},
	}

	if code := runWithDeps(deps); code == 0 {
		t.Fatal("expected non-zero exit on redis failure")
	}
}
