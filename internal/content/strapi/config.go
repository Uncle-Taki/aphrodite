package strapi

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"aphrodite/pkg/config"
)

type Config struct {
	BaseURL         string
	APIToken        string
	Timeout         time.Duration
	MaxRetries      int
	RetryBackoff    time.Duration
	MaxRetryBackoff time.Duration
}

func ConfigFromApp(cfg config.StrapiConfig) Config {
	return Config{
		BaseURL:         cfg.BaseURL,
		APIToken:        cfg.APIToken,
		Timeout:         cfg.Timeout,
		MaxRetries:      cfg.MaxRetries,
		RetryBackoff:    cfg.RetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
	}
}

func (c Config) validate() error {
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		return fmt.Errorf("strapi base URL is required")
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("invalid strapi base URL")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("strapi timeout must be positive")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("strapi max retries cannot be negative")
	}
	if c.RetryBackoff < 0 || c.MaxRetryBackoff < 0 {
		return fmt.Errorf("strapi retry backoff cannot be negative")
	}
	return nil
}
