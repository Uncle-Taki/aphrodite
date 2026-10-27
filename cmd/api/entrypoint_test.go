package main

import (
	"net/http"
	"testing"

	"aphrodite/pkg/config"
	"aphrodite/pkg/logger"
)

func TestMain_ReturnsFailureWhenDatabaseUnavailable(t *testing.T) {
	previousExit, previousConfig, previousLogger := exitProcess, config.C, logger.L
	t.Cleanup(func() { exitProcess = previousExit; config.C = previousConfig; logger.L = previousLogger })
	for key, value := range map[string]string{
		"POSTGRES_HOST": "127.0.0.1", "POSTGRES_PORT": "1", "POSTGRES_USER": "coverage", "POSTGRES_PASSWORD": "coverage", "POSTGRES_DB": "coverage", "POSTGRES_SSLMODE": "disable",
		"JWT_SECRET": "0123456789abcdef0123456789abcdef", "JWT_ACCESS_TOKEN_TTL": "1h", "JWT_MIN_SECRET_BYTES": "32", "AUTH_BCRYPT_COST": "4", "SUPER_ADMIN_KEY": "test-only",
		"POST_TITLE_MAX_LENGTH": "100", "POST_CONTENT_MAX_LENGTH": "1000", "COMMENT_CONTENT_MAX_LENGTH": "100", "USER_DEFAULT_LIMIT": "10", "USER_MAX_LIMIT": "50", "POST_DEFAULT_LIMIT": "10", "POST_MAX_LIMIT": "50", "COMMENT_DEFAULT_LIMIT": "10", "COMMENT_MAX_LIMIT": "50", "CRYPTO_MASTER_KEY": "test-only",
	} {
		t.Setenv(key, value)
	}
	code := -1
	exitProcess = func(value int) { code = value }
	main()
	if code != 1 {
		t.Fatalf("exit code=%d", code)
	}
}

func TestDefaultRunDeps_PropagatesAdapterFailures(t *testing.T) {
	deps := defaultRunDeps()
	if _, err := deps.newRedis(config.RedisConfig{Addr: "127.0.0.1:1"}); err == nil {
		t.Fatal("expected Redis connection failure")
	}
	if err := deps.listenAndServe(&http.Server{Addr: "invalid address"}); err == nil {
		t.Fatal("expected listener failure")
	}
}
