package config

import (
	"testing"
	"time"
)

func TestGetEnv_FallbackWhenUnset(t *testing.T) {
	t.Setenv("aphrodite_TEST_VAR", "")
	if got := getEnv("aphrodite_TEST_VAR", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
}

func TestGetEnv_PrefersEnvValue(t *testing.T) {
	t.Setenv("aphrodite_TEST_VAR", "set")
	if got := getEnv("aphrodite_TEST_VAR", "fallback"); got != "set" {
		t.Fatalf("expected set, got %q", got)
	}
}

func TestMustGetEnv_PanicsWhenUnset(t *testing.T) {
	t.Setenv("aphrodite_TEST_REQ", "")
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for missing required var")
		}
	}()
	mustGetEnv("aphrodite_TEST_REQ")
}

func TestMustGetEnv_ReturnsValueWhenSet(t *testing.T) {
	t.Setenv("aphrodite_TEST_REQ", "ok")
	if got := mustGetEnv("aphrodite_TEST_REQ"); got != "ok" {
		t.Fatalf("expected ok, got %q", got)
	}
}

func TestGetEnvBool_ParsesValid(t *testing.T) {
	cases := map[string]bool{"true": true, "false": false, "1": true, "0": false}
	for in, want := range cases {
		t.Setenv("aphrodite_TEST_BOOL", in)
		if got := getEnvBool("aphrodite_TEST_BOOL", !want); got != want {
			t.Fatalf("getEnvBool(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestGetEnvBool_FallsBackOnInvalid(t *testing.T) {
	t.Setenv("aphrodite_TEST_BOOL", "not-a-bool")
	if got := getEnvBool("aphrodite_TEST_BOOL", true); got != true {
		t.Fatal("expected fallback on invalid value")
	}
}

func TestGetEnvBool_FallsBackWhenUnset(t *testing.T) {
	t.Setenv("aphrodite_TEST_BOOL", "")
	if got := getEnvBool("aphrodite_TEST_BOOL", true); got != true {
		t.Fatal("expected fallback when unset")
	}
}

func TestGetEnvInt_ParsesValid(t *testing.T) {
	t.Setenv("aphrodite_TEST_INT", "42")
	if got := getEnvInt("aphrodite_TEST_INT", 0); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestGetEnvInt_FallsBackOnInvalid(t *testing.T) {
	t.Setenv("aphrodite_TEST_INT", "abc")
	if got := getEnvInt("aphrodite_TEST_INT", 7); got != 7 {
		t.Fatalf("expected fallback 7, got %d", got)
	}
}

func TestGetEnvInt_FallsBackWhenUnset(t *testing.T) {
	t.Setenv("aphrodite_TEST_INT", "")
	if got := getEnvInt("aphrodite_TEST_INT", 9); got != 9 {
		t.Fatalf("expected fallback 9, got %d", got)
	}
}

func TestGetEnvDuration_ParsesValid(t *testing.T) {
	t.Setenv("aphrodite_TEST_DUR", "1h30m")
	if got := getEnvDuration("aphrodite_TEST_DUR", 0); got != 90*time.Minute {
		t.Fatalf("expected 1h30m, got %v", got)
	}
}

func TestGetEnvDuration_FallsBackOnInvalid(t *testing.T) {
	t.Setenv("aphrodite_TEST_DUR", "not-a-duration")
	if got := getEnvDuration("aphrodite_TEST_DUR", 5*time.Second); got != 5*time.Second {
		t.Fatalf("expected fallback, got %v", got)
	}
}

func TestGetEnvDuration_FallsBackWhenUnset(t *testing.T) {
	t.Setenv("aphrodite_TEST_DUR", "")
	if got := getEnvDuration("aphrodite_TEST_DUR", 7*time.Second); got != 7*time.Second {
		t.Fatalf("expected fallback, got %v", got)
	}
}

func TestDSN_FormatIsStable(t *testing.T) {
	d := DatabaseConfig{
		Host: "h", Port: "p", User: "u", Password: "x", Name: "n", SSLMode: "disable",
	}
	got := d.DSN()
	want := "host=h port=p user=u password=x dbname=n sslmode=disable"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func setRequiredDatabaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("POSTGRES_HOST", "postgres")
	t.Setenv("POSTGRES_PORT", "")
	t.Setenv("POSTGRES_USER", "aphrodite")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("POSTGRES_DB", "aphrodite")
	t.Setenv("POSTGRES_SSLMODE", "")
}

func TestLoadDatabaseConfig_AssignmentIndexMaintenanceDefaultsFalse(t *testing.T) {
	setRequiredDatabaseEnv(t)
	t.Setenv("POSTGRES_MAINTAIN_ASSIGNMENT_INDEXES", "")

	cfg := loadDatabaseConfig()
	if cfg.MaintainIndexes {
		t.Fatal("expected assignment index maintenance disabled by default")
	}
}

func TestLoadDatabaseConfig_AssignmentIndexMaintenanceCanBeEnabled(t *testing.T) {
	setRequiredDatabaseEnv(t)
	t.Setenv("POSTGRES_MAINTAIN_ASSIGNMENT_INDEXES", "true")

	cfg := loadDatabaseConfig()
	if !cfg.MaintainIndexes {
		t.Fatal("expected assignment index maintenance enabled")
	}
}

func TestLoadReadsTopLevelSettings(t *testing.T) {
	setRequiredDatabaseEnv(t)
	t.Setenv("APP_ENV", "test")
	t.Setenv("APP_DEBUG", "true")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("POSTGRES_MAINTAIN_ASSIGNMENT_INDEXES", "true")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("REDIS_PASSWORD", "redis-pass")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("JWT_ACCESS_TOKEN_TTL", "2h")
	t.Setenv("JWT_MIN_SECRET_BYTES", "48")
	t.Setenv("AUTH_BCRYPT_COST", "12")
	t.Setenv("SUPER_ADMIN_KEY", "bootstrap-secret")
	t.Setenv("POST_TITLE_MAX_LENGTH", "150")
	t.Setenv("POST_CONTENT_MAX_LENGTH", "5000")
	t.Setenv("COMMENT_CONTENT_MAX_LENGTH", "700")
	t.Setenv("USER_DEFAULT_LIMIT", "10")
	t.Setenv("USER_MAX_LIMIT", "50")
	t.Setenv("POST_DEFAULT_LIMIT", "15")
	t.Setenv("POST_MAX_LIMIT", "75")
	t.Setenv("COMMENT_DEFAULT_LIMIT", "30")
	t.Setenv("COMMENT_MAX_LIMIT", "90")
	t.Setenv("CRYPTO_MASTER_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("USAGE_RECONCILE_INTERVAL", "1m")

	Load()

	if C.Env != "test" || !C.Debug || C.Port != "9090" {
		t.Fatalf("unexpected app config: %+v", C)
	}
	if !C.Database.MaintainIndexes || C.Database.Port != "5432" {
		t.Fatalf("unexpected database config: %+v", C.Database)
	}
	if C.Redis.Addr != "redis:6379" || C.Redis.Password != "redis-pass" || C.Redis.DB != 2 {
		t.Fatalf("unexpected redis config: %+v", C.Redis)
	}
	if C.Auth.JWTSecret != "0123456789abcdef0123456789abcdef" ||
		C.Auth.AccessTokenTTL != 2*time.Hour ||
		C.Auth.JWTMinSecretBytes != 48 ||
		C.Auth.BcryptCost != 12 ||
		C.Auth.SuperAdminKey != "bootstrap-secret" {
		t.Fatalf("unexpected auth config: %+v", C.Auth)
	}
	if C.Validation.PostTitleMaxLength != 150 ||
		C.Validation.PostContentMaxLength != 5000 ||
		C.Validation.CommentContentMaxLength != 700 {
		t.Fatalf("unexpected validation config: %+v", C.Validation)
	}
	if C.Pagination.UserDefaultLimit != 10 ||
		C.Pagination.UserMaxLimit != 50 ||
		C.Pagination.PostDefaultLimit != 15 ||
		C.Pagination.PostMaxLimit != 75 ||
		C.Pagination.CommentDefaultLimit != 30 ||
		C.Pagination.CommentMaxLimit != 90 {
		t.Fatalf("unexpected pagination config: %+v", C.Pagination)
	}
	if C.Crypto.MasterKey != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected crypto config: %+v", C.Crypto)
	}
	if C.Usage.ReconcileInterval != time.Minute {
		t.Fatalf("unexpected usage config: %+v", C.Usage)
	}
}
