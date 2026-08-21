package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"
)

const TaskSendVerifyEmail = "task:send_verify_email"

type PayloadSendVerifyEmail struct {
	Email string `json:"email"`
}

func (distributor *RedisTaskDistributor) DistributeTaskSendVerifyEmail(
	ctx context.Context,
	payload *PayloadSendVerifyEmail,
	opts ...asynq.Option) error {

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TaskSendVerifyEmail, jsonPayload)
	info, err := distributor.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	log.Info().
		Str("task_type", task.Type()).
		Bytes("payload", task.Payload()).
		Str("queue", info.Queue).
		Int("max_retry", info.MaxRetry).
		Str("task_id", info.ID).
		Msg("Task enqueued successfully")
	return nil
}

func (processor *RedisTaskProcessor) ProcessTaskSendVerifyEmail(ctx context.Context, task *asynq.Task) error {
	var payload PayloadSendVerifyEmail
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", asynq.SkipRetry) // Skip retrying this task if payload is invalid
	}

	log.Info().
		Str("task_type", task.Type()).
		Bytes("payload", task.Payload()).
		Msg("Processing task")

	password_credential, err := processor.store.GetPasswordCredentialByEmail(ctx, payload.Email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("no password credential found for email: %s", payload.Email)
		}
		return fmt.Errorf("failed to get password credential by email: %w", err)
	}

	// Send verification email using the email from the password credential
	// err := processor.store.SendVerifyEmail(ctx, password_credential.Email)
	// if err != nil {
	// 	return fmt.Errorf("failed to send verify email: %w", err)
	// }

	log.Info().
		Str("task_type", task.Type()).
		Str("email", password_credential.Email).
		Bytes("payload", task.Payload()).
		Msg("Task processed successfully")
	return nil
}
