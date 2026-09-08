package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var C Config

type Config struct {
	Env        string
	Debug      bool
	Port       string
	Database   DatabaseConfig
	Redis      RedisConfig
	Storage    StorageConfig
	Strapi     StrapiConfig
	Worker     WorkerConfig
	Session    SessionConfig
	Auth       AuthConfig
	Validation ValidationConfig
	Pagination PaginationConfig
	Crypto     CryptoConfig
	Usage      UsageConfig
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaintainIndexes bool
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type StorageConfig struct {
	Provider       string
	Endpoint       string
	Region         string
	AccessKey      string
	SecretKey      string
	Bucket         string
	UseSSL         bool
	ForcePathStyle bool
	PublicBaseURL  string
	ImgProxyURL    string
	ImgProxyKey    string
	ImgProxySalt   string
}

type StrapiConfig struct {
	BaseURL           string
	APIToken          string
	Timeout           time.Duration
	ReadyTimeout      time.Duration
	ReadyPollInterval time.Duration
	MaxRetries        int
	RetryBackoff      time.Duration
	MaxRetryBackoff   time.Duration
	DefaultLocale     string
	FallbackLocale    string
	CacheTTL          time.Duration
	WebhookSecret     string
}

type WorkerConfig struct {
	Concurrency int
	Queue       string
}

type SessionConfig struct {
	CookieName string
	TTL        time.Duration
	Secure     bool
}

type AuthConfig struct {
	JWTSecret         string
	AccessTokenTTL    time.Duration
	JWTMinSecretBytes int
	BcryptCost        int
	SuperAdminKey     string
}

type ValidationConfig struct {
	PostTitleMaxLength      int
	PostContentMaxLength    int
	CommentContentMaxLength int
}

type PaginationConfig struct {
	UserDefaultLimit    int
	UserMaxLimit        int
	PostDefaultLimit    int
	PostMaxLimit        int
	CommentDefaultLimit int
	CommentMaxLimit     int
}

type CryptoConfig struct {
	MasterKey string
}

type UsageConfig struct {
	ReconcileInterval time.Duration // 0 disables the job
}

func Load() {
	_ = godotenv.Load(".env")

	C = Config{
		Env:      getEnv("APP_ENV", "development"),
		Debug:    getEnvBool("APP_DEBUG", false),
		Port:     getEnv("APP_PORT", "8080"),
		Database: loadDatabaseConfig(),
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Storage: StorageConfig{
			Provider:       getEnv("MEDIA_PROVIDER", "minio"),
			Endpoint:       getEnv("MEDIA_ENDPOINT", getEnv("MINIO_ENDPOINT", "localhost:9000")),
			Region:         getEnv("MEDIA_REGION", "us-east-1"),
			AccessKey:      getEnv("MEDIA_ACCESS_KEY", getEnv("MINIO_ACCESS_KEY", "minioadmin")),
			SecretKey:      getEnv("MEDIA_SECRET_KEY", getEnv("MINIO_SECRET_KEY", "minioadmin")),
			Bucket:         getEnv("MEDIA_BUCKET", getEnv("MINIO_BUCKET", "aphrodite-media")),
			UseSSL:         getEnvBool("MEDIA_USE_SSL", getEnvBool("MINIO_USE_SSL", false)),
			ForcePathStyle: getEnvBool("MEDIA_FORCE_PATH_STYLE", false),
			PublicBaseURL:  getEnv("MEDIA_PUBLIC_BASE_URL", ""),
			ImgProxyURL:    getEnv("IMGPROXY_URL", "http://localhost:8081"),
			ImgProxyKey:    getEnv("IMGPROXY_KEY", ""),
			ImgProxySalt:   getEnv("IMGPROXY_SALT", ""),
		},
		Strapi: StrapiConfig{
			BaseURL:           getEnv("STRAPI_BASE_URL", "http://localhost:1337"),
			APIToken:          getEnv("STRAPI_API_TOKEN", ""),
			Timeout:           getEnvDuration("STRAPI_TIMEOUT", 3*time.Second),
			ReadyTimeout:      getEnvDuration("STRAPI_READY_TIMEOUT", 30*time.Second),
			ReadyPollInterval: getEnvDuration("STRAPI_READY_POLL_INTERVAL", 500*time.Millisecond),
			MaxRetries:        getEnvInt("STRAPI_MAX_RETRIES", 2),
			RetryBackoff:      getEnvDuration("STRAPI_RETRY_BACKOFF", 100*time.Millisecond),
			MaxRetryBackoff:   getEnvDuration("STRAPI_MAX_RETRY_BACKOFF", 2*time.Second),
			DefaultLocale:     getEnv("STRAPI_DEFAULT_LOCALE", "fa"),
			FallbackLocale:    getEnv("STRAPI_FALLBACK_LOCALE", "fa"),
			CacheTTL:          getEnvDuration("STRAPI_CACHE_TTL", 60*time.Second),
			WebhookSecret:     getEnv("STRAPI_WEBHOOK_SECRET", ""),
		},
		Worker: WorkerConfig{
			Concurrency: getEnvInt("WORKER_CONCURRENCY", 10),
			Queue:       getEnv("WORKER_QUEUE", "default"),
		},
		Session: SessionConfig{
			CookieName: getEnv("CMS_SESSION_COOKIE", "aphrodite_cms_session"),
			TTL:        getEnvDuration("CMS_SESSION_TTL", 12*time.Hour),
			Secure:     getEnvBool("CMS_SESSION_SECURE", false),
		},
		Auth: AuthConfig{
			JWTSecret:         mustGetEnv("JWT_SECRET"),
			AccessTokenTTL:    mustGetEnvDuration("JWT_ACCESS_TOKEN_TTL"),
			JWTMinSecretBytes: mustGetPositiveEnvInt("JWT_MIN_SECRET_BYTES"),
			BcryptCost:        mustGetPositiveEnvInt("AUTH_BCRYPT_COST"),
			SuperAdminKey:     mustGetEnv("SUPER_ADMIN_KEY"),
		},
		Validation: ValidationConfig{
			PostTitleMaxLength:      mustGetPositiveEnvInt("POST_TITLE_MAX_LENGTH"),
			PostContentMaxLength:    mustGetPositiveEnvInt("POST_CONTENT_MAX_LENGTH"),
			CommentContentMaxLength: mustGetPositiveEnvInt("COMMENT_CONTENT_MAX_LENGTH"),
		},
		Pagination: PaginationConfig{
			UserDefaultLimit:    mustGetPositiveEnvInt("USER_DEFAULT_LIMIT"),
			UserMaxLimit:        mustGetPositiveEnvInt("USER_MAX_LIMIT"),
			PostDefaultLimit:    mustGetPositiveEnvInt("POST_DEFAULT_LIMIT"),
			PostMaxLimit:        mustGetPositiveEnvInt("POST_MAX_LIMIT"),
			CommentDefaultLimit: mustGetPositiveEnvInt("COMMENT_DEFAULT_LIMIT"),
			CommentMaxLimit:     mustGetPositiveEnvInt("COMMENT_MAX_LIMIT"),
		},
		Crypto: CryptoConfig{
			MasterKey: mustGetEnv("CRYPTO_MASTER_KEY"),
		},
		Usage: UsageConfig{
			ReconcileInterval: getEnvDuration("USAGE_RECONCILE_INTERVAL", 0*time.Second),
		},
	}
}

func loadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:            mustGetEnv("POSTGRES_HOST"),
		Port:            getEnv("POSTGRES_PORT", "5432"),
		User:            mustGetEnv("POSTGRES_USER"),
		Password:        mustGetEnv("POSTGRES_PASSWORD"),
		Name:            mustGetEnv("POSTGRES_DB"),
		SSLMode:         getEnv("POSTGRES_SSLMODE", "disable"),
		MaintainIndexes: getEnvBool("POSTGRES_MAINTAIN_ASSIGNMENT_INDEXES", false),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

func mustGetEnvInt(key string) int {
	v := mustGetEnv(key)
	n, err := strconv.Atoi(v)
	if err != nil {
		panic(fmt.Sprintf("required environment variable %q must be an integer", key))
	}
	return n
}

func mustGetPositiveEnvInt(key string) int {
	n := mustGetEnvInt(key)
	if n <= 0 {
		panic(fmt.Sprintf("required environment variable %q must be positive", key))
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func mustGetEnvDuration(key string) time.Duration {
	v := mustGetEnv(key)
	d, err := time.ParseDuration(v)
	if err != nil {
		panic(fmt.Sprintf("required environment variable %q must be a duration", key))
	}
	return d
}
