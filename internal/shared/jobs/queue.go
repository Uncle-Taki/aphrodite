package jobs

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"

	"aphrodite/pkg/config"
)

type Queue struct {
	client *asynq.Client
	queue  string
}

func New(cfg config.RedisConfig, queue string) *Queue {
	return &Queue{
		client: asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB}),
		queue:  queue,
	}
}

func (q *Queue) Close() error { return q.client.Close() }

func (q *Queue) Enqueue(ctx context.Context, taskType string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = q.client.EnqueueContext(ctx, asynq.NewTask(taskType, body), asynq.Queue(q.queue))
	return err
}

type Worker struct {
	server *asynq.Server
}

func NewWorker(cfg config.RedisConfig, worker config.WorkerConfig) *Worker {
	return &Worker{server: asynq.NewServer(asynq.RedisClientOpt{
		Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB,
	}, asynq.Config{Concurrency: worker.Concurrency, Queues: map[string]int{worker.Queue: 10}})}
}

func (w *Worker) Run(handler map[string]asynq.Handler) error {
	mux := asynq.NewServeMux()
	for taskType, taskHandler := range handler {
		mux.Handle(taskType, taskHandler)
	}
	return w.server.Run(mux)
}

func (w *Worker) Shutdown() { w.server.Shutdown() }
