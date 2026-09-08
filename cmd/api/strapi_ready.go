package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"aphrodite/pkg/config"
)

func waitForStrapi(ctx context.Context, cfg config.StrapiConfig) error {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil
	}
	readyTimeout := cfg.ReadyTimeout
	if readyTimeout <= 0 {
		readyTimeout = 30 * time.Second
	}
	pollInterval := cfg.ReadyPollInterval
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}

	readyCtx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()
	client := &http.Client{Timeout: cfg.Timeout}
	if client.Timeout <= 0 {
		client.Timeout = 3 * time.Second
	}
	return waitForStrapiWithClient(readyCtx, client, baseURL+"/_health", pollInterval)
}

func waitForStrapiWithClient(ctx context.Context, client *http.Client, endpoint string, pollInterval time.Duration) error {
	var lastErr error
	for {
		if err := strapiHealthCheck(ctx, client, endpoint); err == nil {
			return nil
		} else {
			lastErr = err
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("%s: %w", lastErr, ctx.Err())
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func strapiHealthCheck(ctx context.Context, client *http.Client, endpoint string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1024))
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("health endpoint returned HTTP %d", res.StatusCode)
	}
	return nil
}
