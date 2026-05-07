package workers

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/go-budget/backend/internal/config"
	exchangerates "github.com/go-budget/backend/internal/services/exchange_rates"
	"github.com/go-budget/backend/internal/workers/tasks"
)

// Dependencies holds all the repositories and services that task handlers
// may need. Fields are added as task handlers are implemented across phases.
type Dependencies struct {
	// Phase 2: Export email task
	ExportService tasks.ExportService
	EmailService  tasks.EmailService

	// Phase 4: Scheduled task dependencies
	ActivationTokenRepo tasks.ActivationTokenRepository
	ExchangeRateSvc     *exchangerates.Service
	SubscriptionSvc     tasks.SubscriptionService

	// Budget service: used by both budget:processing (daily processing)
	// and budget:user_update (on-demand recalculation).
	BudgetSvc tasks.BudgetService
}

// Manager orchestrates the asynq client, server, and scheduler.
// The client is always available for enqueueing tasks. The server
// and scheduler only start when WorkerEnabled is true in config.
type Manager struct {
	client    *asynq.Client
	server    *asynq.Server
	scheduler *asynq.Scheduler
	mux       *asynq.ServeMux
	cfg       *config.Config
	logger    *slog.Logger
	deps      *Dependencies
}

// NewManager creates a new Manager with all asynq components initialized.
// The client is always created (needed for enqueueing). The server and
// scheduler are created only if WorkerEnabled is true.
func NewManager(cfg *config.Config, logger *slog.Logger, deps *Dependencies) (*Manager, error) {
	redisOpt, err := ParseRedisURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis URL: %w", err)
	}

	client := asynq.NewClient(redisOpt)

	m := &Manager{
		client: client,
		cfg:    cfg,
		logger: logger,
		deps:   deps,
	}

	if cfg.WorkerEnabled {
		m.server = asynq.NewServer(redisOpt, asynq.Config{
			Concurrency: cfg.WorkerConcurrency,
			Queues:      QueueConfig(),
			Logger:      newAsynqLogger(logger),
		})

		m.mux = asynq.NewServeMux()

		m.scheduler = asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{
			Logger: newAsynqLogger(logger),
		})
	}

	return m, nil
}

// SetDependencies updates the dependencies used by task handlers.
// Must be called before Start() so that handlers are registered with the
// correct service implementations.
func (m *Manager) SetDependencies(deps *Dependencies) {
	m.deps = deps
}

// Client returns the TaskEnqueuer interface backed by the asynq client.
// This is safe to call regardless of whether workers are enabled.
func (m *Manager) Client() TaskEnqueuer {
	return m.client
}

// Start begins processing tasks and running scheduled jobs.
// If WorkerEnabled is false, this is a no-op.
func (m *Manager) Start() error {
	if !m.cfg.WorkerEnabled {
		m.logger.Info("worker processing is disabled, skipping server and scheduler start")
		return nil
	}

	m.registerHandlers()

	if err := m.registerSchedules(); err != nil {
		return fmt.Errorf("register schedules: %w", err)
	}

	// Start the server (task processor) in a goroutine.
	// Use error channels to detect immediate startup failures.
	serverErrCh := make(chan error, 1)
	go func() {
		if err := m.server.Start(m.mux); err != nil {
			m.logger.Error("asynq server stopped", "error", err)
			serverErrCh <- err
		}
	}()

	schedulerErrCh := make(chan error, 1)
	go func() {
		if err := m.scheduler.Start(); err != nil {
			m.logger.Error("asynq scheduler stopped", "error", err)
			schedulerErrCh <- err
		}
	}()

	// Brief startup check -- detect immediate failures (e.g., bad Redis connection).
	select {
	case err := <-serverErrCh:
		return fmt.Errorf("asynq server failed to start: %w", err)
	case err := <-schedulerErrCh:
		return fmt.Errorf("asynq scheduler failed to start: %w", err)
	case <-time.After(100 * time.Millisecond):
		// No immediate failure, continue
	}

	m.logger.Info("async worker manager started",
		"concurrency", m.cfg.WorkerConcurrency,
		"queues", QueueConfig(),
	)

	return nil
}

// Stop gracefully shuts down the server and scheduler, then closes the client.
// Must be called before HTTP server shutdown to allow in-flight tasks to complete.
func (m *Manager) Stop() {
	if m.cfg.WorkerEnabled && m.scheduler != nil {
		m.scheduler.Shutdown()
		m.logger.Info("asynq scheduler stopped")
	}

	if m.cfg.WorkerEnabled && m.server != nil {
		m.server.Shutdown()
		m.logger.Info("asynq server stopped")
	}

	if err := m.client.Close(); err != nil {
		m.logger.Error("failed to close asynq client", "error", err)
	} else {
		m.logger.Info("asynq client closed")
	}
}

// registerHandlers registers all task handlers on the ServeMux.
func (m *Manager) registerHandlers() {
	// Phase 2: Export email task
	if m.deps.ExportService != nil && m.deps.EmailService != nil {
		exportEmailHandler := tasks.NewExportEmailHandler(m.deps.ExportService, m.deps.EmailService, m.logger)
		m.mux.HandleFunc(tasks.TypeExportEmail, exportEmailHandler.ProcessTask)
		m.logger.Info("registered task handler", "type", tasks.TypeExportEmail)
	}

	// Phase 3: Email tasks
	if m.deps.EmailService != nil {
		emailSendHandler := tasks.NewEmailSendHandler(m.deps.EmailService, m.logger)
		m.mux.HandleFunc(tasks.TypeEmailSend, emailSendHandler.ProcessTask)
		m.logger.Info("registered task handler", "type", tasks.TypeEmailSend)
	}

	// Activation email handler uses the client for task chaining (enqueues email:send)
	activationEmailHandler := tasks.NewActivationEmailHandler(m.client, m.logger)
	m.mux.HandleFunc(tasks.TypeEmailActivation, activationEmailHandler.ProcessTask)
	m.logger.Info("registered task handler", "type", tasks.TypeEmailActivation)

	// Phase 4: Scheduled task handlers
	if m.deps.ActivationTokenRepo != nil {
		tokenCleanupHandler := tasks.NewTokenCleanupHandler(m.deps.ActivationTokenRepo, m.logger)
		m.mux.HandleFunc(tasks.TypeTokenCleanup, tokenCleanupHandler.ProcessTask)
		m.logger.Info("registered task handler", "type", tasks.TypeTokenCleanup)
	}

	// Budget service handlers: daily processing (scheduled) and user update (on-demand).
	// Both depend on BudgetService.
	if m.deps.BudgetSvc != nil {
		budgetProcessingHandler := tasks.NewBudgetProcessingHandler(m.deps.BudgetSvc, m.logger)
		m.mux.HandleFunc(tasks.TypeBudgetProcessing, budgetProcessingHandler.ProcessTask)
		m.logger.Info("registered task handler", "type", tasks.TypeBudgetProcessing)

		budgetUserUpdateHandler := tasks.NewBudgetUserUpdateHandler(m.deps.BudgetSvc, m.logger)
		m.mux.HandleFunc(tasks.TypeBudgetUserUpdate, budgetUserUpdateHandler.ProcessTask)
		m.logger.Info("registered task handler", "type", tasks.TypeBudgetUserUpdate)
	}

	notifCfg := tasks.NotificationConfig{
		AdminEmails: m.cfg.AdminEmails,
		Environment: m.cfg.Environment,
	}

	if m.deps.ExchangeRateSvc != nil {
		exchangeRateHandler := tasks.NewExchangeRateUpdateHandler(
			m.deps.ExchangeRateSvc,
			m.client,
			notifCfg,
			m.logger,
		)
		m.mux.HandleFunc(tasks.TypeExchangeRateUpdate, exchangeRateHandler.ProcessTask)
		m.logger.Info("registered task handler", "type", tasks.TypeExchangeRateUpdate)
	}

	if m.deps.SubscriptionSvc != nil {
		renewalHandler := tasks.NewSubscriptionRenewalHandler(m.deps.SubscriptionSvc, m.logger)
		m.mux.HandleFunc(tasks.TypeSubscriptionRenewal, renewalHandler.ProcessTask)
		m.logger.Info("registered task handler", "type", tasks.TypeSubscriptionRenewal)

		downgradeHandler := tasks.NewSubscriptionDowngradeHandler(m.deps.SubscriptionSvc, m.logger)
		m.mux.HandleFunc(tasks.TypeSubscriptionDowngrade, downgradeHandler.ProcessTask)
		m.logger.Info("registered task handler", "type", tasks.TypeSubscriptionDowngrade)
	}
}

// registerSchedules registers all cron-scheduled tasks with the scheduler.
func (m *Manager) registerSchedules() error {
	schedules := []struct {
		cronExpr string
		task     *asynq.Task
	}{
		{m.cfg.ScheduleExchangeRateUpdate, asynq.NewTask(tasks.TypeExchangeRateUpdate, nil)},
		{m.cfg.ScheduleBudgetProcessing, asynq.NewTask(tasks.TypeBudgetProcessing, nil)},
		{m.cfg.ScheduleTokenCleanup, asynq.NewTask(tasks.TypeTokenCleanup, nil)},
		{m.cfg.ScheduleSubscriptionRenewal, asynq.NewTask(tasks.TypeSubscriptionRenewal, nil)},
		{m.cfg.ScheduleSubscriptionDowngrade, asynq.NewTask(tasks.TypeSubscriptionDowngrade, nil)},
	}

	for _, s := range schedules {
		entryID, err := m.scheduler.Register(s.cronExpr, s.task)
		if err != nil {
			return fmt.Errorf("register schedule %q: %w", s.task.Type(), err)
		}
		m.logger.Info("registered scheduled task",
			"type", s.task.Type(),
			"cron", s.cronExpr,
			"entryID", entryID,
		)
	}

	return nil
}

// asynqLogger adapts slog.Logger to the asynq.Logger interface.
type asynqLogger struct {
	logger *slog.Logger
}

func newAsynqLogger(logger *slog.Logger) *asynqLogger {
	return &asynqLogger{logger: logger.With("component", "asynq")}
}

func (l *asynqLogger) Debug(args ...any) {
	l.logger.Debug(fmt.Sprint(args...))
}

func (l *asynqLogger) Info(args ...any) {
	l.logger.Info(fmt.Sprint(args...))
}

func (l *asynqLogger) Warn(args ...any) {
	l.logger.Warn(fmt.Sprint(args...))
}

func (l *asynqLogger) Error(args ...any) {
	l.logger.Error(fmt.Sprint(args...))
}

func (l *asynqLogger) Fatal(args ...any) {
	l.logger.Error(fmt.Sprint(args...))
}
