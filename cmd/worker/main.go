package main

import (
	"log/slog"

	"github.com/hibiken/asynq"

	"aphrodite/internal/shared/jobs"
	"aphrodite/pkg/config"
	"aphrodite/pkg/logger"
)

type workerRunner interface {
	Run(map[string]asynq.Handler) error
}

var (
	loadConfig = config.Load
	newWorker  = func(redis config.RedisConfig, worker config.WorkerConfig) workerRunner {
		return jobs.NewWorker(redis, worker)
	}
)

func main() {
	loadConfig()
	logger.Init(config.C.Debug)
	worker := newWorker(config.C.Redis, config.C.Worker)
	if err := worker.Run(nil); err != nil {
		slog.Error("worker stopped", "err", err)
	}
}
