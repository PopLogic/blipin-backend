package worker

import (
	"context"

	db "github.com/PopLogic/blipin-backend/db/sqlc"
	"github.com/hibiken/asynq"
)

type Processor interface {
	Start() error
	ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error
}

type RedisTaskProcessor struct {
	server *asynq.Server
	store  *db.Store
}

func NewRedisTaskProcessor(redisOpt asynq.RedisClientOpt, store db.Store) Processor {
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{},
	)

	return &RedisTaskProcessor{
		server: server,
		store:  &store,
	}
}

func (processor *RedisTaskProcessor) Start() error {
    mux := asynq.NewServeMux()
    mux.HandleFunc(TaskSendVerifyEmail, processor.ProcessTaskSendVerifyEmail)
    return processor.server.Start(mux)
}