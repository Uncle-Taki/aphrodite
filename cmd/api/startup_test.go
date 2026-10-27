package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"aphrodite/pkg/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWaitForStrapi_DefaultsAndInvalidEndpoint(t *testing.T) {
	if err := waitForStrapi(context.Background(), config.StrapiConfig{}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_health" {
			t.Errorf("path=%s", r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer server.Close()
	if err := waitForStrapi(context.Background(), config.StrapiConfig{BaseURL: server.URL}); err != nil {
		t.Fatal(err)
	}
	if err := waitForStrapi(context.Background(), config.StrapiConfig{BaseURL: server.URL, Timeout: time.Second, ReadyTimeout: time.Second, ReadyPollInterval: time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	if err := strapiHealthCheck(context.Background(), http.DefaultClient, "http://invalid%host"); err == nil {
		t.Fatal("accepted invalid health URL")
	}
}

func TestAutoMigrate_IdentifiesFailingSchema(t *testing.T) {
	for _, table := range []string{"users", "posts", "comments"} {
		t.Run(table, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			failure := errors.New("schema permission denied")
			if err := db.Callback().Raw().Before("gorm:raw").Register("test:reject-create", func(tx *gorm.DB) {
				if strings.Contains(tx.Statement.SQL.String(), "CREATE TABLE `"+table+"`") {
					tx.AddError(failure)
				}
			}); err != nil {
				t.Fatal(err)
			}
			err = autoMigrate(db)
			if !errors.Is(err, failure) {
				t.Fatalf("migration error=%v", err)
			}
		})
	}
}

func startupDeps(t *testing.T) runDeps {
	t.Helper()
	old := config.C
	t.Cleanup(func() { config.C = old })
	return runDeps{
		loadConfig: func() {
			config.C = config.Config{Auth: config.AuthConfig{JWTSecret: "0123456789abcdef0123456789abcdef", JWTMinSecretBytes: 32, AccessTokenTTL: time.Hour}}
		},
		connectPostgres: func(config.DatabaseConfig) (*gorm.DB, error) {
			return gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		},
		newRedis: func(config.RedisConfig) (redisPinger, error) { return stubRedis{}, nil },
	}
}

func TestRunWithDeps_RejectsInvalidAdapters(t *testing.T) {
	for _, stage := range []string{"default migration", "default readiness", "Strapi token", "JWT settings"} {
		t.Run(stage, func(t *testing.T) {
			deps := startupDeps(t)
			load := deps.loadConfig
			deps.loadConfig = func() {
				load()
				switch stage {
				case "default readiness":
					config.C.Strapi = config.StrapiConfig{BaseURL: "http://127.0.0.1:1", ReadyTimeout: time.Millisecond, ReadyPollInterval: time.Millisecond}
				case "Strapi token":
					config.C.Strapi = config.StrapiConfig{BaseURL: "http://strapi.test", Timeout: time.Second}
				case "JWT settings":
					config.C.Auth.JWTSecret = "short"
				}
			}
			if stage == "default migration" {
				connect := deps.connectPostgres
				deps.connectPostgres = func(cfg config.DatabaseConfig) (*gorm.DB, error) {
					db, err := connect(cfg)
					if err == nil {
						err = db.Callback().Raw().Before("gorm:raw").Register("test:reject-create", func(tx *gorm.DB) {
							tx.AddError(errors.New("database unavailable"))
						})
					}
					return db, err
				}
			}
			if stage == "Strapi token" {
				deps.waitForStrapi = func(context.Context, config.StrapiConfig) error { return nil }
			}
			if code := runWithDeps(deps); code != 1 {
				t.Fatalf("exit code=%d", code)
			}
		})
	}
}

type failingCloseListener struct {
	net.Listener
	cancel context.CancelFunc
	once   sync.Once
}

type cacheRedis struct{ stubRedis }

func (cacheRedis) Get(context.Context, string) (string, error)                   { return "", errors.New("cache miss") }
func (cacheRedis) Set(context.Context, string, interface{}, time.Duration) error { return nil }

func TestRunWithDeps_WiresContentCache(t *testing.T) {
	deps := startupDeps(t)
	load := deps.loadConfig
	deps.loadConfig = func() {
		load()
		config.C.Strapi = config.StrapiConfig{BaseURL: "http://strapi.test", APIToken: "test-token", Timeout: time.Second, CacheTTL: time.Minute}
	}
	deps.waitForStrapi = func(context.Context, config.StrapiConfig) error { return nil }
	deps.newRedis = func(config.RedisConfig) (redisPinger, error) { return cacheRedis{}, nil }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deps.notifyContext = func(context.Context, ...os.Signal) (context.Context, context.CancelFunc) { return ctx, func() {} }
	completed := make(chan struct{})
	deps.listenAndServe = func(*http.Server) error { defer close(completed); cancel(); return http.ErrServerClosed }
	deps.exitOnUnexpectedServeError = func(int) { t.Error("unexpected serve failure") }
	if code := runWithDeps(deps); code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	<-completed
}

func (l *failingCloseListener) Accept() (net.Conn, error) {
	l.once.Do(l.cancel)
	return l.Listener.Accept()
}
func (l *failingCloseListener) Close() error {
	_ = l.Listener.Close()
	return errors.New("listener close failed")
}

func TestRunWithDeps_HandlesListenerAndShutdownFailures(t *testing.T) {
	for _, stage := range []string{"serve", "shutdown"} {
		t.Run(stage, func(t *testing.T) {
			deps := startupDeps(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			deps.notifyContext = func(context.Context, ...os.Signal) (context.Context, context.CancelFunc) { return ctx, func() {} }
			completed := make(chan struct{})
			codes := make(chan int, 1)
			deps.exitOnUnexpectedServeError = func(code int) { codes <- code; cancel() }
			deps.listenAndServe = func(server *http.Server) error {
				defer close(completed)
				if stage == "serve" {
					return errors.New("listener failed")
				}
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					cancel()
					return err
				}
				return server.Serve(&failingCloseListener{Listener: listener, cancel: cancel})
			}
			if code := runWithDeps(deps); code != 0 {
				t.Fatalf("exit code=%d", code)
			}
			<-completed
			if stage == "serve" {
				select {
				case code := <-codes:
					if code != 1 {
						t.Fatalf("unexpected error exit=%d", code)
					}
				default:
					t.Fatal("serve error not reported")
				}
			}
		})
	}
}
