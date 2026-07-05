package redis

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	goredis "github.com/go-redis/redis/v8"

	"aphrodite/pkg/config"
)

func TestClientErrorPathsAndNilDetection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	c := &Client{rdb: goredis.NewClient(&goredis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 10 * time.Millisecond,
		ReadTimeout: 10 * time.Millisecond,
	})}
	defer c.rdb.Close()

	if err := c.Set(ctx, "k", "v", time.Second); err == nil {
		t.Fatal("expected set error")
	}
	if _, err := c.Get(ctx, "k"); err == nil {
		t.Fatal("expected get error")
	}
	if err := c.Del(ctx, "k"); err == nil {
		t.Fatal("expected del error")
	}
	if err := c.Ping(ctx); err == nil {
		t.Fatal("expected ping error")
	}
	if !IsNil(goredis.Nil) {
		t.Fatal("expected go-redis nil to be recognized")
	}
	if IsNil(errors.New("other")) {
		t.Fatal("unexpected nil match")
	}
}

func TestNewReturnsPingError(t *testing.T) {
	if _, err := New(config.RedisConfig{Addr: "127.0.0.1:1"}); err == nil {
		t.Fatal("expected connection error")
	}
}

func TestClientLifecycle_GatedRedis(t *testing.T) {
	addr := os.Getenv("aphrodite_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set aphrodite_TEST_REDIS_ADDR to run Redis integration test")
	}
	client, err := New(config.RedisConfig{
		Addr:     addr,
		Password: os.Getenv("aphrodite_TEST_REDIS_PASSWORD"),
	})
	if err != nil {
		t.Fatalf("connect redis: %v", err)
	}
	defer client.rdb.Close()

	ctx := context.Background()
	key := fmt.Sprintf("aphrodite:test:%d", time.Now().UnixNano())
	if err := client.Set(ctx, key, "value", time.Minute); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := client.Get(ctx, key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != "value" {
		t.Fatalf("unexpected cached value: %q", got)
	}
	if err := client.Del(ctx, key); err != nil {
		t.Fatalf("del: %v", err)
	}
	if _, err := client.Get(ctx, key); !IsNil(err) {
		t.Fatalf("expected redis nil after delete, got %v", err)
	}
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
}
