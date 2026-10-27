package main

import (
	"aphrodite/internal/shared/jobs"
	"aphrodite/pkg/config"
	"aphrodite/pkg/logger"
	"errors"
	"github.com/hibiken/asynq"
	"testing"
)

type stubWorker struct {
	err    error
	called bool
}

func (w *stubWorker) Run(handlers map[string]asynq.Handler) error { w.called = true; return w.err }

func TestMain_LoadsConfigurationAndRunsWorker(t *testing.T) {
	for _, failure := range []error{nil, errors.New("worker failed")} {
		t.Run("worker", func(t *testing.T) {
			previousLoad, previousNew, previousConfig, previousLogger := loadConfig, newWorker, config.C, logger.L
			t.Cleanup(func() {
				loadConfig = previousLoad
				newWorker = previousNew
				config.C = previousConfig
				logger.L = previousLogger
			})
			loadConfig = func() {
				config.C = config.Config{Redis: config.RedisConfig{Addr: "test-redis"}, Worker: config.WorkerConfig{Concurrency: 2, Queue: "test"}}
			}
			worker := &stubWorker{err: failure}
			newWorker = func(redis config.RedisConfig, cfg config.WorkerConfig) workerRunner {
				if redis.Addr != "test-redis" || cfg.Concurrency != 2 || cfg.Queue != "test" {
					t.Fatal("worker configuration not forwarded")
				}
				return worker
			}
			main()
			if !worker.called {
				t.Fatal("worker not run")
			}
		})
	}
}

func TestDefaultWorkerFactory(t *testing.T) {
	worker := newWorker(config.RedisConfig{Addr: "127.0.0.1:1"}, config.WorkerConfig{Concurrency: 1, Queue: "test"})
	concrete, ok := worker.(*jobs.Worker)
	if !ok {
		t.Fatalf("worker type=%T", worker)
	}
	concrete.Shutdown()
}
