package server

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/go-budget/backend/internal/auth"
	"github.com/go-budget/backend/internal/config"
	accountshandler "github.com/go-budget/backend/internal/handlers/accounts"
	analyticshandler "github.com/go-budget/backend/internal/handlers/analytics"
	authhandler "github.com/go-budget/backend/internal/handlers/auth"
	budgetshandler "github.com/go-budget/backend/internal/handlers/budgets"
	categorieshandler "github.com/go-budget/backend/internal/handlers/categories"
	currencieshandler "github.com/go-budget/backend/internal/handlers/currencies"
	exchangerateshandler "github.com/go-budget/backend/internal/handlers/exchange_rates"
	exporthandler "github.com/go-budget/backend/internal/handlers/export"
	financialplanninghandler "github.com/go-budget/backend/internal/handlers/financial_planning"
	internalapihandler "github.com/go-budget/backend/internal/handlers/internal_api"
	managementhandler "github.com/go-budget/backend/internal/handlers/management"
	paymentshandler "github.com/go-budget/backend/internal/handlers/payments"
	plannedtxhandler "github.com/go-budget/backend/internal/handlers/planned_transactions"
	reportshandler "github.com/go-budget/backend/internal/handlers/reports"
	settingshandler "github.com/go-budget/backend/internal/handlers/settings"
	subscriptionshandler "github.com/go-budget/backend/internal/handlers/subscriptions"
	txhandler "github.com/go-budget/backend/internal/handlers/transactions"
	"github.com/go-budget/backend/internal/middleware"
	"github.com/go-budget/backend/internal/repositories"
	accountsservice "github.com/go-budget/backend/internal/services/accounts"
	authservice "github.com/go-budget/backend/internal/services/auth"
	backupservice "github.com/go-budget/backend/internal/services/backup"
	backupnotifyservice "github.com/go-budget/backend/internal/services/backup_notify"
	budgetsservice "github.com/go-budget/backend/internal/services/budgets"
	categoriesservice "github.com/go-budget/backend/internal/services/categories"
	"github.com/go-budget/backend/internal/services/credit"
	currencyservice "github.com/go-budget/backend/internal/services/currency"
	emailservice "github.com/go-budget/backend/internal/services/email"
	exchangeratesservice "github.com/go-budget/backend/internal/services/exchange_rates"
	exportservice "github.com/go-budget/backend/internal/services/export"
	financialplanningservice "github.com/go-budget/backend/internal/services/financial_planning"
	paymentservice "github.com/go-budget/backend/internal/services/payment"
	plannedtxservice "github.com/go-budget/backend/internal/services/planned_transactions"
	reportsservice "github.com/go-budget/backend/internal/services/reports"
	settingsservice "github.com/go-budget/backend/internal/services/settings"
	subscriptionservice "github.com/go-budget/backend/internal/services/subscription"
	txservice "github.com/go-budget/backend/internal/services/transactions"
	"github.com/go-budget/backend/internal/workers"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

type Server struct {
	config        *config.Config
	logger        *slog.Logger
	db            *sql.DB
	workerManager *workers.Manager
}

func New(cfg *config.Config, logger *slog.Logger, db *sql.DB, workerManager *workers.Manager) (*Server, error) {
	return &Server{
		config:        cfg,
		logger:        logger,
		db:            db,
		workerManager: workerManager,
	}, nil
}

func (s *Server) Start() *http.Server {
	e := echo.New()
	e.HideBanner = true

	// Global middleware
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "auth-token"},
		ExposeHeaders:    []string{"new_access_token"},
		AllowCredentials: true,
	}))
	e.Validator = middleware.NewValidator()

	// Request logging middleware
	e.Use(echomiddleware.RequestLoggerWithConfig(echomiddleware.RequestLoggerConfig{
		Skipper: func(c echo.Context) bool {
			return c.Path() == "/health"
		},
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		LogRemoteIP: true,
		LogValuesFunc: func(c echo.Context, v echomiddleware.RequestLoggerValues) error {
			// `user_id` is set by the auth middleware for authenticated requests
			// and may be absent for public endpoints. We log nil in that case so
			// every request line carries the same key set, which simplifies log
			// querying.
			var userID any
			if uid := c.Get("user_id"); uid != nil {
				userID = uid
			}
			s.logger.Info("request",
				"method", v.Method,
				"uri", v.URI,
				"status", v.Status,
				"latency", v.Latency,
				"remote_ip", v.RemoteIP,
				"user_id", userID,
			)
			return nil
		},
	}))

	// Initialize repositories
	userRepo := repositories.NewUserRepository(s.db, s.logger)
	tokenRepo := repositories.NewActivationTokenRepository(s.db, s.logger)
	settingsRepo := repositories.NewUserSettingsRepository(s.db, s.logger)
	categoryRepo := repositories.NewCategoryRepository(s.db, s.logger)
	currencyRepo := repositories.NewCurrencyRepository(s.db, s.logger)
	accountRepo := repositories.NewAccountRepository(s.db, s.logger)
	accountTypeRepo := repositories.NewAccountTypeRepository(s.db, s.logger)
	transactionRepo := repositories.NewTransactionRepository(s.db, s.logger)
	templateRepo := repositories.NewTransactionTemplateRepository(s.db, s.logger)
	exchangeRateRepo := repositories.NewExchangeRateRepository(s.db, s.logger)
	languageRepo := repositories.NewLanguageRepository(s.db, s.logger)
	budgetRepo := repositories.NewBudgetRepository(s.db, s.logger)
	plannedTxRepo := repositories.NewPlannedTransactionRepository(s.db, s.logger)
	subscriptionRepo := repositories.NewSubscriptionRepository(s.db, s.logger)
	subscriptionPlanRepo := repositories.NewSubscriptionPlanRepository(s.db, s.logger)
	planPriceRepo := repositories.NewPlanPriceRepository(s.db, s.logger)

	// JWT service
	jwtService := auth.NewJWTService(s.config.JWTSecret, s.config.JWTExpirationMinutes)
	authMiddleware := middleware.NewAuthMiddleware(jwtService, s.logger)
	userActiveMiddleware := middleware.RequireActiveUser(userRepo, s.logger)

	// Initialize services
	currencySvc := currencyservice.NewWithCurrencyRepo(exchangeRateRepo, currencyRepo, s.logger)

	// Subscription service (used by subscriptions handler, workers, and auth for trial creation).
	// Default trial duration: 14 days.
	subscriptionSvc := subscriptionservice.New(
		subscriptionRepo,
		subscriptionPlanRepo,
		planPriceRepo,
		accountRepo,
		budgetRepo,
		s.logger,
		14,
	)

	authSvc := authservice.New(
		userRepo,
		tokenRepo,
		settingsRepo,
		categoryRepo,
		currencyRepo,
		jwtService,
		s.workerManager.Client(),
		subscriptionSvc,
		s.config,
		s.logger,
	)

	accountsSvc := accountsservice.New(
		accountRepo,
		accountTypeRepo,
		currencyRepo,
		userRepo,
		transactionRepo,
		currencySvc,
		s.logger,
	)

	txSvc := txservice.New(
		transactionRepo,
		accountRepo,
		categoryRepo,
		userRepo,
		templateRepo,
		currencySvc,
		s.workerManager.Client(),
		s.logger,
	)

	// Budget service for recalculation with currency conversion.
	// budgetRepo implements both BudgetRepository and TransactionRepository
	// interfaces needed by the budget service (GetExpenseTransactionsForBudget
	// is on BudgetRepositoryImpl because it is budget-specific query logic).
	budgetSvc := budgetsservice.New(budgetRepo, budgetRepo, currencyRepo, currencySvc, s.logger)

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Auth routes (mixed: some public, some protected)
	authGroup := e.Group("/auth")
	authHandler := authhandler.New(authSvc, s.config.GoogleClientID, s.logger)

	// Public auth routes
	authGroup.POST("/register/", authHandler.Register)
	authGroup.POST("/login/", authHandler.Login)
	authGroup.GET("/activate/:token", authHandler.Activate)
	authGroup.POST("/oauth/", authHandler.OAuth)

	// Protected auth routes
	protectedAuthGroup := authGroup.Group("")
	protectedAuthGroup.Use(authMiddleware.RequireAuth, userActiveMiddleware)
	protectedAuthGroup.GET("/profile/", authHandler.GetProfile)
	protectedAuthGroup.POST("/change-password/", authHandler.ChangePassword)

	// Accounts routes (all protected)
	accountsGroup := e.Group("/accounts")
	accountsGroup.Use(authMiddleware.RequireAuth, userActiveMiddleware)
	accountsHandler := accountshandler.New(accountsSvc, s.logger)
	accountsHandler.RegisterRoutes(accountsGroup)

	// Transactions routes (all protected)
	txGroup := e.Group("/transactions")
	txGroup.Use(authMiddleware.RequireAuth, userActiveMiddleware)
	transactionsHandler := txhandler.New(txSvc, s.logger)
	transactionsHandler.RegisterRoutes(txGroup)

	// Categories routes (all protected)
	categoriesGroup := e.Group("/categories")
	categoriesGroup.Use(authMiddleware.RequireAuth, userActiveMiddleware)
	categoriesSvc := categoriesservice.New(categoryRepo, s.logger)
	categoriesHandler := categorieshandler.New(categoriesSvc, s.logger)
	categoriesHandler.RegisterRoutes(categoriesGroup)

	// Currencies routes (all protected)
	currenciesGroup := e.Group("/currencies")
	currenciesGroup.Use(authMiddleware.RequireAuth, userActiveMiddleware)
	currenciesHandler := currencieshandler.New(currencySvc, s.logger)
	currenciesHandler.RegisterRoutes(currenciesGroup)

	// Settings routes (mixed: languages is public, rest protected)
	settingsGroup := e.Group("/settings")
	settingsSvc := settingsservice.New(settingsRepo, userRepo, currencyRepo, languageRepo, s.logger)
	settingsHandler := settingshandler.New(settingsSvc, s.logger)
	settingsGroup.GET("/languages", settingsHandler.GetLanguages)
	protectedSettingsGroup := settingsGroup.Group("")
	protectedSettingsGroup.Use(authMiddleware.RequireAuth, userActiveMiddleware)
	protectedSettingsGroup.GET("", settingsHandler.GetSettings)
	protectedSettingsGroup.POST("", settingsHandler.UpdateSettings)
	protectedSettingsGroup.GET("/base-currency/", settingsHandler.GetBaseCurrency)
	protectedSettingsGroup.PUT("/base-currency/", settingsHandler.UpdateBaseCurrency)

	// Budgets routes (all protected)
	budgetsGroup := e.Group("/budgets")
	budgetsGroup.Use(authMiddleware.RequireAuth, userActiveMiddleware)
	budgetsHandler := budgetshandler.New(budgetSvc, s.logger)
	budgetsHandler.RegisterRoutes(budgetsGroup)

	// Budget admin routes
	budgetsAdminGroup := budgetsGroup.Group("")
	budgetsAdminGroup.Use(middleware.RequireAdmin(s.config.AdminEmails))
	budgetsAdminGroup.GET("/daily-processing", budgetsHandler.DailyProcessing)

	// Planned Transactions routes (all protected)
	plannedTxGroup := e.Group("/planned-transactions")
	plannedTxGroup.Use(authMiddleware.RequireAuth, userActiveMiddleware)
	plannedTxSvc := plannedtxservice.New(plannedTxRepo, s.logger)
	plannedTxHandler := plannedtxhandler.New(plannedTxSvc, s.logger)
	plannedTxHandler.RegisterRoutes(plannedTxGroup)

	// Exchange Rates routes (all protected)
	exchangeRatesGroup := e.Group("/exchange-rates")
	exchangeRatesGroup.Use(authMiddleware.RequireAuth, userActiveMiddleware)
	exchangeRatesSvc := exchangeratesservice.New(exchangeRateRepo, s.config.CurrencyBeaconAPIURL, s.config.CurrencyBeaconAPIKey, s.logger)
	exchangeRatesHandler := exchangerateshandler.New(exchangeRatesSvc, s.workerManager.Client(), s.logger)
	exchangeRatesHandler.RegisterRoutes(exchangeRatesGroup)

	// Exchange rates admin routes
	exchangeRatesAdminGroup := exchangeRatesGroup.Group("")
	exchangeRatesAdminGroup.Use(middleware.RequireAdmin(s.config.AdminEmails))
	exchangeRatesAdminGroup.GET("/update", exchangeRatesHandler.UpdateRates)
	exchangeRatesAdminGroup.GET("/update/from/:start_date/to/:end_date", exchangeRatesHandler.UpdateRatesRange)

	// Reports routes (all protected)
	reportsGroup := e.Group("/reports")
	reportsGroup.Use(authMiddleware.RequireAuth, userActiveMiddleware)
	reportsSvc := reportsservice.New(transactionRepo, accountRepo, categoryRepo, currencyRepo, currencySvc, s.logger)
	reportsHandler := reportshandler.New(reportsSvc, s.logger)
	reportsHandler.RegisterRoutes(reportsGroup)

	// Financial Planning routes (all protected)
	financialPlanningGroup := e.Group("/financial-planning")
	financialPlanningGroup.Use(authMiddleware.RequireAuth, userActiveMiddleware)
	financialPlanningSvc := financialplanningservice.New(plannedTxSvc, accountRepo, currencyRepo, currencySvc, s.logger)
	financialPlanningHandler := financialplanninghandler.New(financialPlanningSvc, s.logger)
	financialPlanningHandler.RegisterRoutes(financialPlanningGroup)

	// Analytics routes (all protected)
	analyticsGroup := e.Group("/analytics")
	analyticsGroup.Use(authMiddleware.RequireAuth, userActiveMiddleware)
	analyticsHandler := analyticshandler.New(s.logger)
	analyticsHandler.RegisterRoutes(analyticsGroup)

	// Export routes (all protected)
	exportSvc := exportservice.New(transactionRepo, userRepo, currencySvc, s.logger)
	emailSvc := emailservice.New(s.config, s.logger)
	exportGroup := e.Group("/export")
	exportGroup.Use(authMiddleware.RequireAuth, userActiveMiddleware)
	exportHandler := exporthandler.New(exportSvc, emailSvc, s.workerManager.Client(), s.logger, s.config.MaxExportRangeDays)
	exportHandler.RegisterRoutes(exportGroup)

	// Management routes (admin only)
	managementGroup := e.Group("/management")
	managementGroup.Use(authMiddleware.RequireAuth, userActiveMiddleware, middleware.RequireAdmin(s.config.AdminEmails))
	var backupSvc managementhandler.BackupJobCreator
	if svc, err := backupservice.NewK8sBackupService(s.config.K8sNamespace, s.logger); err != nil {
		s.logger.Info("K8s backup service not available, backup endpoint will return 503", "error", err)
	} else {
		backupSvc = svc
	}
	managementHandler := managementhandler.New(backupSvc, s.logger)
	managementHandler.RegisterRoutes(managementGroup)

	// Internal API routes (token-based auth, no JWT)
	internalGroup := e.Group("/internal")
	internalGroup.Use(middleware.RequireInternalToken(s.config.InternalAPIToken))
	backupNotifySvc := backupnotifyservice.New(s.workerManager.Client(), s.config.AdminEmails, s.config.Environment, s.logger)
	internalAPIHandler := internalapihandler.New(backupNotifySvc, s.logger)
	internalAPIHandler.RegisterRoutes(internalGroup)

	// Subscriptions routes (mixed: plans is public, rest protected)
	subscriptionsGroup := e.Group("/subscriptions")
	subscriptionsHandler := subscriptionshandler.New(subscriptionSvc, s.logger)
	subscriptionsHandler.RegisterRoutes(subscriptionsGroup, authMiddleware.RequireAuth, userActiveMiddleware)

	// Payments routes (mixed: webhook is public, rest protected)
	stripeClient := paymentservice.NewStripeClient(s.config.StripeSecretKey, s.config.StripeWebhookSecret)
	creditCalc := credit.NewCreditCalculator()
	paymentSvc := paymentservice.NewService(stripeClient, subscriptionRepo, subscriptionPlanRepo, planPriceRepo, creditCalc, s.config, s.logger)
	webhookHandler := paymentservice.NewWebhookHandler(stripeClient, subscriptionRepo, subscriptionPlanRepo, planPriceRepo, s.logger)
	paymentsGroup := e.Group("/payments")
	paymentsHandler := paymentshandler.New(paymentSvc, webhookHandler, s.logger)
	paymentsHandler.RegisterRoutes(paymentsGroup, authMiddleware.RequireAuth, userActiveMiddleware)

	// Set worker dependencies now that services are initialized.
	// This must happen before workerManager.Start() is called.
	if s.workerManager != nil {
		s.workerManager.SetDependencies(&workers.Dependencies{
			ExportService:       exportSvc,
			EmailService:        emailSvc,
			ActivationTokenRepo: tokenRepo,
			ExchangeRateSvc:     exchangeRatesSvc,
			SubscriptionSvc:     subscriptionSvc,
			BudgetSvc:           budgetSvc,
		})
	}

	srv := &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: e,
	}

	return srv
}
