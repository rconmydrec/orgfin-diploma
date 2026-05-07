package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
)

// ActivationTokenRepository defines the interface for token cleanup operations.
type ActivationTokenRepository interface {
	DeleteExpired() error
}

// NewTokenCleanupTask creates a new token:cleanup task with no payload.
func NewTokenCleanupTask() *asynq.Task {
	return asynq.NewTask(
		TypeTokenCleanup,
		nil,
		asynq.MaxRetry(10),
		asynq.Queue("default"),
		asynq.Timeout(5*time.Minute),
	)
}

// TokenCleanupHandler processes token:cleanup tasks by deleting expired
// activation tokens from the database.
type TokenCleanupHandler struct {
	tokenRepo ActivationTokenRepository
	logger    *slog.Logger
}

// NewTokenCleanupHandler creates a new handler for token:cleanup tasks.
func NewTokenCleanupHandler(tokenRepo ActivationTokenRepository, logger *slog.Logger) *TokenCleanupHandler {
	return &TokenCleanupHandler{
		tokenRepo: tokenRepo,
		logger:    logger,
	}
}

// ProcessTask deletes all expired activation tokens.
func (h *TokenCleanupHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	h.logger.Info("processing token cleanup task")

	if err := h.tokenRepo.DeleteExpired(); err != nil {
		h.logger.Error("failed to delete expired tokens", "error", err)
		return fmt.Errorf("delete expired tokens: %w", err)
	}

	h.logger.Info("token cleanup completed successfully")
	return nil
}
