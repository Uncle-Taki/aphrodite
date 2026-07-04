package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	Octopus    OctopusConfig
	Heimdall   HeimdallConfig
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

type OctopusConfig struct {
	BaseURL             string
	AdminUsername       string
	AdminPassword       string
	RequestTimeout      time.Duration
	ProxyRequestTimeout time.Duration
}

type HeimdallConfig struct {
	Bypass       bool
	GRPCAddr     string
	GRPCTimeout  time.Duration
	ServiceToken string
	TokenType    string
	AdminUserIDs []string
}

type AuthConfig struct {
	JWTSecret         string
	AccessTokenTTL    time.Duration
	JWTMinSecretBytes int
	BcryptCost        int
}

type ValidationConfig struct {
	PostTitleMaxLength      int
	PostContentMaxLength    int
	CommentContentMaxLength int
}

type PaginationConfig struct {
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
	_ = godotenv.Load(".env", ".heimdal.env", ".octopus.txt")

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
		Octopus: OctopusConfig{
			BaseURL:             mustGetEnv("OCTOPUS_BASE_URL"),
			AdminUsername:       mustGetEnv("OCTOPUS_ADMIN_USERNAME"),
			AdminPassword:       mustGetEnv("OCTOPUS_ADMIN_PASSWORD"),
			RequestTimeout:      getEnvDuration("OCTOPUS_REQUEST_TIMEOUT", 30*time.Second),
			ProxyRequestTimeout: getEnvDuration("OCTOPUS_PROXY_REQUEST_TIMEOUT", 120*time.Second),
		},
		Heimdall: loadHeimdallConfig(),
		Auth: AuthConfig{
			JWTSecret:         mustGetEnv("JWT_SECRET"),
			AccessTokenTTL:    getEnvDuration("JWT_ACCESS_TOKEN_TTL", 24*time.Hour),
			JWTMinSecretBytes: mustGetPositiveEnvInt("JWT_MIN_SECRET_BYTES"),
			BcryptCost:        mustGetPositiveEnvInt("AUTH_BCRYPT_COST"),
		},
		Validation: ValidationConfig{
			PostTitleMaxLength:      mustGetPositiveEnvInt("POST_TITLE_MAX_LENGTH"),
			PostContentMaxLength:    mustGetPositiveEnvInt("POST_CONTENT_MAX_LENGTH"),
			CommentContentMaxLength: mustGetPositiveEnvInt("COMMENT_CONTENT_MAX_LENGTH"),
		},
		Pagination: PaginationConfig{
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

func loadHeimdallConfig() HeimdallConfig {
	bypass := getEnvBool("HEIMDALL_BYPASS", false)
	if bypass && getEnv("APP_ENV", "development") == "production" {
		panic("HEIMDALL_BYPASS=true is forbidden when APP_ENV=production")
	}

	cfg := HeimdallConfig{
		Bypass:       bypass,
		GRPCAddr:     os.Getenv("HEIMDALL_GRPC_ADDR"),
		GRPCTimeout:  getEnvDuration("HEIMDALL_GRPC_TIMEOUT", 5*time.Second),
		ServiceToken: os.Getenv("HEIMDALL_SERVICE_TOKEN"),
		TokenType:    getEnv("HEIMDALL_TOKEN_TYPE", "access_token"),
		AdminUserIDs: getEnvStringList("HEIMDALL_ADMIN_USER_IDS", nil),
	}
	if bypass {
		return cfg
	}
	if cfg.GRPCAddr == "" {
		panic("heimdall gRPC address not configured: set HEIMDALL_GRPC_ADDR or HEIMDALL_BYPASS=true")
	}
	return cfg
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

func getEnvStringList(key string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return fallback
	}
	return out
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
