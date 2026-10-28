package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"aphrodite/pkg/config"
	"github.com/hibiken/asynq"
)

func TestQueue_RejectsInvalidPayloadAndConnectionFailure(t *testing.T) {
	queue := New(config.RedisConfig{Addr: "127.0.0.1:1"}, "test")
	defer queue.Close()
	if err := queue.Enqueue(context.Background(), "test", make(chan int)); err == nil {
		t.Fatal("expected payload encoding error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := queue.Enqueue(ctx, "test", map[string]string{"title": "hello"}); err == nil {
		t.Fatal("expected connection error")
	}
}

func TestWorker_RejectsRestartAfterShutdown(t *testing.T) {
	worker := NewWorker(config.RedisConfig{Addr: "127.0.0.1:1"}, config.WorkerConfig{Concurrency: 1, Queue: "test"})
	// Starting the server before Run exercises its lifecycle rejection without
	// sending process-wide signals from a unit test.
	if err := worker.server.Start(asynq.HandlerFunc(func(context.Context, *asynq.Task) error { return nil })); err != nil {
		t.Fatal(err)
	}
	worker.Shutdown()
	if err := worker.Run(map[string]asynq.Handler{"test": asynq.HandlerFunc(func(context.Context, *asynq.Task) error { return nil })}); !errors.Is(err, asynq.ErrServerClosed) {
		t.Fatalf("restart: %v", err)
	}
}

func TestQueue_EnqueuesTaskInRedis(t *testing.T) {
	addr := os.Getenv("aphrodite_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set aphrodite_TEST_REDIS_ADDR to run Redis queue integration test")
	}
	password := os.Getenv("aphrodite_TEST_REDIS_PASSWORD")
	name := "coverage-" + time.Now().Format("150405.000000000")
	queue := New(config.RedisConfig{Addr: addr, Password: password}, name)
	defer queue.Close()
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: addr, Password: password})
	defer inspector.Close()
	t.Cleanup(func() { _, _ = inspector.DeleteAllPendingTasks(name); _ = inspector.DeleteQueue(name, false) })
	if err := queue.Enqueue(context.Background(), "article:publish", map[string]string{"documentId": "doc-1"}); err != nil {
		t.Fatal(err)
	}
	tasks, err := inspector.ListPendingTasks(name)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Type != "article:publish" {
		t.Fatalf("unexpected tasks: %+v", tasks)
	}
	var payload map[string]string
	if err := json.Unmarshal(tasks[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["documentId"] != "doc-1" {
		t.Fatalf("payload=%v", payload)
	}
}
