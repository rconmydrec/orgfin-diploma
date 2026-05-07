package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	exchangerates "github.com/go-budget/backend/internal/services/exchange_rates"
	"github.com/hibiken/asynq"
)

// NewExchangeRateUpdateTask creates a new exchange_rate:update task with no payload.
func NewExchangeRateUpdateTask() *asynq.Task {
	return asynq.NewTask(
		TypeExchangeRateUpdate,
		nil,
		asynq.MaxRetry(24),
		asynq.Queue("default"),
		asynq.Timeout(10*time.Minute),
	)
}

// ExchangeRateUpdateHandler processes exchange_rate:update tasks by fetching
// the latest exchange rates from CurrencyBeacon API and storing them.
type ExchangeRateUpdateHandler struct {
	exchangeRateSvc *exchangerates.Service
	enqueuer        TaskEnqueuer
	notifConfig     NotificationConfig
	logger          *slog.Logger
}

// NewExchangeRateUpdateHandler creates a new handler for exchange_rate:update tasks.
func NewExchangeRateUpdateHandler(
	exchangeRateSvc *exchangerates.Service,
	enqueuer TaskEnqueuer,
	notifConfig NotificationConfig,
	logger *slog.Logger,
) *ExchangeRateUpdateHandler {
	return &ExchangeRateUpdateHandler{
		exchangeRateSvc: exchangeRateSvc,
		enqueuer:        enqueuer,
		notifConfig:     notifConfig,
		logger:          logger,
	}
}

// ProcessTask fetches the latest exchange rates from the CurrencyBeacon API and stores them.
func (h *ExchangeRateUpdateHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	today := time.Now().Format("2006-01-02")
	h.logger.Info("processing exchange rate update task", "date", today)

	result, err := h.exchangeRateSvc.FetchAndSaveRates(today)
	if err != nil {
		h.logger.Error("failed to fetch and save exchange rates", "date", today, "error", err)
		return fmt.Errorf("fetch and save rates: %w", err)
	}

	h.logger.Info("exchange rate update completed successfully",
		"date", today,
		"baseCurrency", result.BaseCurrency,
		"rateCount", result.RateCount,
	)

	h.sendRateUpdateNotification(result.BaseCurrency, result.RateCount)

	return nil
}

// sendRateUpdateNotification enqueues email:send tasks for each admin email
// with exchange rate update details. Failures are logged but never propagated.
func (h *ExchangeRateUpdateHandler) sendRateUpdateNotification(baseCurrency string, rateCount int) {
	if len(h.notifConfig.AdminEmails) == 0 {
		h.logger.Debug("no admin emails configured, skipping rate update notification")
		return
	}

	updatedAt := time.Now().Format(time.RFC3339)
	body := buildExchangeRateNotificationBody(
		h.notifConfig.Environment,
		updatedAt,
		rateCount,
		baseCurrency,
	)

	for _, adminEmail := range h.notifConfig.AdminEmails {
		emailTask, err := NewEmailSendTask(EmailSendPayload{
			To:      adminEmail,
			Subject: buildExchangeRateNotificationSubject(h.notifConfig.Environment),
			Body:    body,
		})
		if err != nil {
			h.logger.Error("failed to create email send task for rate update notification",
				"email", adminEmail,
				"error", err,
			)
			continue
		}

		if _, err := h.enqueuer.Enqueue(emailTask); err != nil {
			h.logger.Error("failed to enqueue rate update notification email",
				"email", adminEmail,
				"error", err,
			)
			continue
		}

		h.logger.Info("rate update notification email enqueued",
			"email", adminEmail,
		)
	}
}
