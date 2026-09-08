package main

import (
	"log/slog"

	"aphrodite/internal/shared/jobs"
	"aphrodite/pkg/config"
	"aphrodite/pkg/logger"
)

func main() {
	config.Load()
	logger.Init(config.C.Debug)
	worker := jobs.NewWorker(config.C.Redis, config.C.Worker)
	if err := worker.Run(nil); err != nil {
		slog.Error("worker stopped", "err", err)
	}
}
