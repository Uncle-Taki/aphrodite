package main

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitForStrapiImmediateHealthy(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusOK), nil
	})}

	err := waitForStrapiWithClient(context.Background(), client, "http://strapi/_health", time.Millisecond)
	if err != nil {
		t.Fatalf("waitForStrapi: %v", err)
	}
}

func TestWaitForStrapiRetriesUntilHealthy(t *testing.T) {
	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		if calls.Add(1) < 3 {
			return response(http.StatusServiceUnavailable), nil
		}
		return response(http.StatusOK), nil
	})}

	err := waitForStrapiWithClient(context.Background(), client, "http://strapi/_health", time.Millisecond)
	if err != nil {
		t.Fatalf("waitForStrapi: %v", err)
	}
	if got := calls.Load(); got < 3 {
		t.Fatalf("expected retries, got %d calls", got)
	}
}

func TestWaitForStrapiTimesOut(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(http.StatusServiceUnavailable), nil
	})}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := waitForStrapiWithClient(ctx, client, "http://strapi/_health", time.Millisecond)
	if err == nil {
		t.Fatal("expected readiness timeout")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func response(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       http.NoBody,
		Header:     make(http.Header),
	}
}
