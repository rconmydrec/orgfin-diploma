// Package testutil provides test utilities and helpers for integration tests.
package testutil

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

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
	managementhandler "github.com/go-budget/backend/internal/handlers/management"
	paymentshandler "github.com/go-budget/backend/internal/handlers/payments"
	plannedtxhandler "github.com/go-budget/backend/internal/handlers/planned_transactions"
	reportshandler "github.com/go-budget/backend/internal/handlers/reports"
	settingshandler "github.com/go-budget/backend/internal/handlers/settings"
	subscriptionshandler "github.com/go-budget/backend/internal/handlers/subscriptions"
	transactionshandler "github.com/go-budget/backend/internal/handlers/transactions"
	"github.com/go-budget/backend/internal/middleware"
	"github.com/go-budget/backend/internal/repositories"
	accountsservice "github.com/go-budget/backend/internal/services/accounts"
	authservice "github.com/go-budget/backend/internal/services/auth"
	budgetsservice "github.com/go-budget/backend/internal/services/budgets"
	categoriesservice "github.com/go-budget/backend/internal/services/categories"
	"github.com/go-budget/backend/internal/services/credit"
	currencyservice "github.com/go-budget/backend/internal/services/currency"
	exchangeratesservice "github.com/go-budget/backend/internal/services/exchange_rates"
	exportservice "github.com/go-budget/backend/internal/services/export"
	financialplanningservice "github.com/go-budget/backend/internal/services/financial_planning"
	paymentservice "github.com/go-budget/backend/internal/services/payment"
	plannedtxservice "github.com/go-budget/backend/internal/services/planned_transactions"
	reportsservice "github.com/go-budget/backend/internal/services/reports"
	settingsservice "github.com/go-budget/backend/internal/services/settings"
	subscriptionservice "github.com/go-budget/backend/internal/services/subscription"
	transactionsservice "github.com/go-budget/backend/internal/services/transactions"
	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
)

// TestDB holds the test database connection
var TestDB *sql.DB

// AdminTestEmail is a well-known admin email used in integration tests that
// require admin access. Test users created with this email will pass admin
// authorization checks.
const AdminTestEmail = "admin-integration-test@example.com"

func init() {
	// Load .env.test from backend/ directory (two levels up from this file)
	_, thisFile, _, _ := runtime.Caller(0)
	backendDir := filepath.Join(filepath.Dir(thisFile), "..", "..")
	_ = godotenv.Load(filepath.Join(backendDir, ".env.test"))
}

// envOrDefault reads an environment variable or returns a default value.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// envOrDefaultInt reads an environment variable as int or returns a default.
func envOrDefaultInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

// TestConfig for tests — values can be overridden via environment variables or .env.test
var TestConfig = &config.Config{
	DBHost:               envOrDefault("TEST_DB_HOST", "localhost"),
	DBPort:               envOrDefaultInt("TEST_DB_PORT", 5432),
	DBUser:               envOrDefault("TEST_DB_USER", "postgres"),
	DBPassword:           envOrDefault("TEST_DB_PASSWORD", "123"),
	DBName:               envOrDefault("TEST_DB_NAME", "budgeter_test"),
	JWTSecret:            "test-secret-key-for-testing",
	JWTExpirationMinutes: 1440,
	GoogleClientID:       "test-google-client-id",
	Environment:          "test",
	AdminEmails:          []string{AdminTestEmail},
	MaxExportRangeDays:   1096,
}

// SetupTestDB initializes the test database connection
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	if TestDB != nil {
		return TestDB
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		TestConfig.DBUser,
		TestConfig.DBPassword,
		TestConfig.DBHost,
		TestConfig.DBPort,
		TestConfig.DBName,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	TestDB = db
	return db
}

// CleanupTestDB closes the test database connection
func CleanupTestDB() {
	if TestDB != nil {
		TestDB.Close()
		TestDB = nil
	}
}

// TestLogger returns a logger for tests
func TestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Only show errors in tests
	}))
}

// TestApp holds all test application dependencies
type TestApp struct {
	DB          *sql.DB
	Echo        *echo.Echo
	Logger      *slog.Logger
	JWTService  *auth.JWTService
	AuthService *authservice.Service
	AuthHandler *authhandler.Handler

	// Accounts
	AccountsService *accountsservice.Service
	AccountsHandler *accountshandler.Handler

	// Transactions
	TransactionsService *transactionsservice.Service
	TransactionsHandler *transactionshandler.Handler

	// Categories
	CategoriesHandler *categorieshandler.Handler

	// Currencies
	CurrenciesHandler *currencieshandler.Handler

	// Settings
	SettingsHandler *settingshandler.Handler

	// Budgets
	BudgetsHandler *budgetshandler.Handler

	// Reports
	ReportsHandler *reportshandler.Handler

	// Exchange Rates
	ExchangeRatesHandler *exchangerateshandler.Handler

	// Planned Transactions
	PlannedTxHandler *plannedtxhandler.Handler

	// Financial Planning
	FinancialPlanningHandler *financialplanninghandler.Handler

	// Subscriptions
	SubscriptionsHandler *subscriptionshandler.Handler

	// Payments
	PaymentsHandler *paymentshandler.Handler
	StubPaymentProv *StubPaymentProvider

	// Management
	ManagementHandler *managementhandler.Handler

	// Analytics
	AnalyticsHandler *analyticshandler.Handler

	// Export
	ExportHandler *exporthandler.Handler

	// MockEmailService is exposed so tests can inspect email send calls
	MockEmailSvc *MockEmailService

	// MockEnqueuer is exposed so tests can inspect enqueued tasks
	MockEnqueuer *MockTaskEnqueuer

	// Repositories
	UserRepo         repositories.UserRepository
	TokenRepo        repositories.ActivationTokenRepository
	SettingsRepo     repositories.UserSettingsRepository
	CategoryRepo     repositories.CategoryRepository
	CurrencyRepo     repositories.CurrencyRepository
	AccountRepo      repositories.AccountRepository
	AccountTypeRepo  repositories.AccountTypeRepository
	TransactionRepo  repositories.TransactionRepository
	ExchangeRateRepo repositories.ExchangeRateRepository
	TemplateRepo     repositories.TransactionTemplateRepository
	LanguageRepo     repositories.LanguageRepository
	BudgetRepo       repositories.BudgetRepository
	PlannedTxRepo    repositories.PlannedTransactionRepository
	SubscriptionRepo repositories.SubscriptionRepository
	SubPlanRepo      repositories.SubscriptionPlanRepository
}

// MockEmailService implements the export.EmailService interface for testing.
// It records calls so tests can verify email sending behaviour without SMTP.
// Access to the Calls slice is protected by a mutex because the handler fires
// email sends in a background goroutine.
type MockEmailService struct {
	mu   sync.Mutex
	done chan struct{} // closed after each SendWithAttachment call

	Calls []MockEmailCall
}

// MockEmailCall records a single SendWithAttachment invocation.
type MockEmailCall struct {
	To, Subject, Body string
	Attachment        []byte
	Filename          string
}

// NewMockEmailService creates a MockEmailService with the signaling channel
// initialised. Tests should call WaitForCall instead of time.Sleep.
func NewMockEmailService() *MockEmailService {
	return &MockEmailService{
		done: make(chan struct{}, 1),
	}
}

func (m *MockEmailService) Send(to, subject, body string) error {
	return nil
}

func (m *MockEmailService) SendWithAttachment(to, subject, body string, attachment []byte, filename string) error {
	m.mu.Lock()
	m.Calls = append(m.Calls, MockEmailCall{
		To: to, Subject: subject, Body: body,
		Attachment: attachment, Filename: filename,
	})
	m.mu.Unlock()

	// Signal that a call was recorded so tests can proceed without sleeping.
	select {
	case m.done <- struct{}{}:
	default:
	}

	return nil
}

// WaitForCall blocks until SendWithAttachment is invoked or the timeout
// expires. Returns true if a call was received, false on timeout.
func (m *MockEmailService) WaitForCall(timeout time.Duration) bool {
	select {
	case <-m.done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// GetCalls returns a snapshot of recorded calls, safe for concurrent use.
func (m *MockEmailService) GetCalls() []MockEmailCall {
	m.mu.Lock()
	defer m.mu.Unlock()

	cp := make([]MockEmailCall, len(m.Calls))
	copy(cp, m.Calls)
	return cp
}

// MockTaskEnqueuer implements workers.TaskEnqueuer for testing.
// It records enqueued tasks so tests can verify enqueueing behaviour
// without a real Redis connection.
type MockTaskEnqueuer struct {
	mu    sync.Mutex
	done  chan struct{}
	Tasks []MockEnqueuedTask
}

// MockEnqueuedTask records a single Enqueue invocation.
type MockEnqueuedTask struct {
	Type    string
	Payload []byte
}

// NewMockTaskEnqueuer creates a MockTaskEnqueuer with the signaling channel
// initialised. Tests should call WaitForEnqueue instead of time.Sleep.
func NewMockTaskEnqueuer() *MockTaskEnqueuer {
	return &MockTaskEnqueuer{
		done: make(chan struct{}, 1),
	}
}

func (m *MockTaskEnqueuer) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	m.mu.Lock()
	m.Tasks = append(m.Tasks, MockEnqueuedTask{
		Type:    task.Type(),
		Payload: task.Payload(),
	})
	m.mu.Unlock()

	select {
	case m.done <- struct{}{}:
	default:
	}

	return &asynq.TaskInfo{}, nil
}

// WaitForEnqueue blocks until Enqueue is invoked or the timeout expires.
// Returns true if a task was enqueued, false on timeout.
func (m *MockTaskEnqueuer) WaitForEnqueue(timeout time.Duration) bool {
	select {
	case <-m.done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// GetTasks returns a snapshot of enqueued tasks, safe for concurrent use.
func (m *MockTaskEnqueuer) GetTasks() []MockEnqueuedTask {
	m.mu.Lock()
	defer m.mu.Unlock()

	cp := make([]MockEnqueuedTask, len(m.Tasks))
	copy(cp, m.Tasks)
	return cp
}

// NewTestApp creates a new test application with all dependencies
func NewTestApp(t *testing.T) *TestApp {
	t.Helper()

	db := SetupTestDB(t)
	logger := TestLogger()

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db, logger)
	tokenRepo := repositories.NewActivationTokenRepository(db, logger)
	settingsRepo := repositories.NewUserSettingsRepository(db, logger)
	categoryRepo := repositories.NewCategoryRepository(db, logger)
	currencyRepo := repositories.NewCurrencyRepository(db, logger)
	accountRepo := repositories.NewAccountRepository(db, logger)
	accountTypeRepo := repositories.NewAccountTypeRepository(db, logger)
	transactionRepo := repositories.NewTransactionRepository(db, logger)
	exchangeRateRepo := repositories.NewExchangeRateRepository(db, logger)
	templateRepo := repositories.NewTransactionTemplateRepository(db, logger)
	languageRepo := repositories.NewLanguageRepository(db, logger)
	budgetRepo := repositories.NewBudgetRepository(db, logger)
	plannedTxRepo := repositories.NewPlannedTransactionRepository(db, logger)
	subscriptionRepo := repositories.NewSubscriptionRepository(db, logger)
	subPlanRepo := repositories.NewSubscriptionPlanRepository(db, logger)

	// JWT service
	jwtService := auth.NewJWTService(TestConfig.JWTSecret, TestConfig.JWTExpirationMinutes)

	// Currency service
	currencySvc := currencyservice.NewWithCurrencyRepo(exchangeRateRepo, currencyRepo, logger)

	// Auth service uses its own mock enqueuer (separate from the export handler's enqueuer)
	// so that activation email tasks don't interfere with export handler test assertions.
	authMockEnqueuer := NewMockTaskEnqueuer()

	// Auth service
	authSvc := authservice.New(
		userRepo,
		tokenRepo,
		settingsRepo,
		categoryRepo,
		currencyRepo,
		jwtService,
		authMockEnqueuer,
		nil, // subscriptionSvc: not needed in integration tests
		TestConfig,
		logger,
	)

	// Accounts service
	accountsSvc := accountsservice.New(
		accountRepo,
		accountTypeRepo,
		currencyRepo,
		userRepo,
		transactionRepo,
		currencySvc,
		logger,
	)

	// Transactions service (nil enqueuer: budget update tasks are not needed in tests)
	transactionsSvc := transactionsservice.New(
		transactionRepo,
		accountRepo,
		categoryRepo,
		userRepo,
		templateRepo,
		nil, // currencyService
		nil, // enqueuer
		logger,
	)

	// Echo instance
	e := echo.New()
	e.Validator = middleware.NewValidator()

	// Handlers
	authHandler := authhandler.New(authSvc, TestConfig.GoogleClientID, logger)
	accountsHandler := accountshandler.New(accountsSvc, logger)
	transactionsHandler := transactionshandler.New(transactionsSvc, logger)
	categoriesSvc := categoriesservice.New(categoryRepo, logger)
	categoriesHandler := categorieshandler.New(categoriesSvc, logger)
	currenciesHandler := currencieshandler.New(currencySvc, logger)
	settingsSvc := settingsservice.New(settingsRepo, userRepo, currencyRepo, languageRepo, logger)
	settingsHandler := settingshandler.New(settingsSvc, logger)
	budgetSvc := budgetsservice.New(budgetRepo, budgetRepo, currencyRepo, currencySvc, logger)
	budgetsHandler := budgetshandler.New(budgetSvc, logger)
	reportsSvc := reportsservice.New(transactionRepo, accountRepo, categoryRepo, currencyRepo, currencySvc, logger)
	reportsHandler := reportshandler.New(reportsSvc, logger)
	plannedTxSvc := plannedtxservice.New(plannedTxRepo, logger)
	plannedTxHandler := plannedtxhandler.New(plannedTxSvc, logger)
	financialPlanningSvc := financialplanningservice.New(plannedTxSvc, accountRepo, currencyRepo, currencySvc, logger)
	financialPlanningHandler := financialplanninghandler.New(financialPlanningSvc, logger)
	planPriceRepo := repositories.NewPlanPriceRepository(db, logger)
	subscriptionSvc := subscriptionservice.New(
		subscriptionRepo,
		subPlanRepo,
		planPriceRepo,
		accountRepo,
		budgetRepo,
		logger,
		14,
	)
	subscriptionsHandler := subscriptionshandler.New(subscriptionSvc, logger)
	stubPaymentProvider := &StubPaymentProvider{}
	creditCalc := credit.NewCreditCalculator()
	paymentSvc := paymentservice.NewService(stubPaymentProvider, subscriptionRepo, subPlanRepo, planPriceRepo, creditCalc, TestConfig, logger)
	webhookHandler := paymentservice.NewWebhookHandler(stubPaymentProvider, subscriptionRepo, subPlanRepo, planPriceRepo, logger)
	paymentsHandler := paymentshandler.New(paymentSvc, webhookHandler, logger)
	analyticsHandler := analyticshandler.New(logger)

	// Mock services for export, management, and exchange rates handlers
	exportSvc := exportservice.New(transactionRepo, userRepo, currencySvc, logger)
	mockEmailSvc := NewMockEmailService()
	mockEnqueuer := NewMockTaskEnqueuer()
	exchangeRatesSvc := exchangeratesservice.New(exchangeRateRepo, TestConfig.CurrencyBeaconAPIURL, TestConfig.CurrencyBeaconAPIKey, logger)
	exchangeRatesHandler := exchangerateshandler.New(exchangeRatesSvc, mockEnqueuer, logger)
	managementHandler := managementhandler.New(nil, logger)
	exportHandler := exporthandler.New(exportSvc, mockEmailSvc, mockEnqueuer, logger, TestConfig.MaxExportRangeDays)

	return &TestApp{
		DB:          db,
		Echo:        e,
		Logger:      logger,
		JWTService:  jwtService,
		AuthService: authSvc,
		AuthHandler: authHandler,

		AccountsService: accountsSvc,
		AccountsHandler: accountsHandler,

		TransactionsService: transactionsSvc,
		TransactionsHandler: transactionsHandler,

		CategoriesHandler: categoriesHandler,

		CurrenciesHandler: currenciesHandler,

		SettingsHandler: settingsHandler,

		BudgetsHandler: budgetsHandler,

		ReportsHandler: reportsHandler,

		ExchangeRatesHandler:     exchangeRatesHandler,
		PlannedTxHandler:         plannedTxHandler,
		FinancialPlanningHandler: financialPlanningHandler,
		SubscriptionsHandler:     subscriptionsHandler,
		PaymentsHandler:          paymentsHandler,
		StubPaymentProv:          stubPaymentProvider,
		ManagementHandler:        managementHandler,
		AnalyticsHandler:         analyticsHandler,
		ExportHandler:            exportHandler,
		MockEmailSvc:             mockEmailSvc,
		MockEnqueuer:             mockEnqueuer,

		UserRepo:         userRepo,
		TokenRepo:        tokenRepo,
		SettingsRepo:     settingsRepo,
		CategoryRepo:     categoryRepo,
		CurrencyRepo:     currencyRepo,
		AccountRepo:      accountRepo,
		AccountTypeRepo:  accountTypeRepo,
		TransactionRepo:  transactionRepo,
		ExchangeRateRepo: exchangeRateRepo,
		TemplateRepo:     templateRepo,
		LanguageRepo:     languageRepo,
		BudgetRepo:       budgetRepo,
		PlannedTxRepo:    plannedTxRepo,
		SubscriptionRepo: subscriptionRepo,
		SubPlanRepo:      subPlanRepo,
	}
}

// SetupAuthRoutes registers auth routes on the Echo instance
func (app *TestApp) SetupAuthRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	authGroup := app.Echo.Group("/auth")

	// Public auth routes
	authGroup.POST("/register/", app.AuthHandler.Register)
	authGroup.POST("/login/", app.AuthHandler.Login)
	authGroup.GET("/activate/:token", app.AuthHandler.Activate)
	authGroup.POST("/oauth/", app.AuthHandler.OAuth)

	// Protected auth routes
	protectedAuthGroup := authGroup.Group("")
	protectedAuthGroup.Use(authMiddleware.RequireAuth, userActiveMW)
	protectedAuthGroup.GET("/profile/", app.AuthHandler.GetProfile)
	protectedAuthGroup.POST("/change-password/", app.AuthHandler.ChangePassword)
}

// SetupAccountsRoutes registers accounts routes on the Echo instance
func (app *TestApp) SetupAccountsRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	accountsGroup := app.Echo.Group("/accounts")
	accountsGroup.Use(authMiddleware.RequireAuth, userActiveMW)
	app.AccountsHandler.RegisterRoutes(accountsGroup)
}

// SetupTransactionsRoutes registers transactions routes on the Echo instance
func (app *TestApp) SetupTransactionsRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	transactionsGroup := app.Echo.Group("/transactions")
	transactionsGroup.Use(authMiddleware.RequireAuth, userActiveMW)
	app.TransactionsHandler.RegisterRoutes(transactionsGroup)
}

// SetupCategoriesRoutes registers categories routes on the Echo instance
func (app *TestApp) SetupCategoriesRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	categoriesGroup := app.Echo.Group("/categories")
	categoriesGroup.Use(authMiddleware.RequireAuth, userActiveMW)
	app.CategoriesHandler.RegisterRoutes(categoriesGroup)
}

// SetupCurrenciesRoutes registers currencies routes on the Echo instance
func (app *TestApp) SetupCurrenciesRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	currenciesGroup := app.Echo.Group("/currencies")
	currenciesGroup.Use(authMiddleware.RequireAuth, userActiveMW)
	app.CurrenciesHandler.RegisterRoutes(currenciesGroup)
}

// SetupSettingsRoutes registers settings routes on the Echo instance
func (app *TestApp) SetupSettingsRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	settingsGroup := app.Echo.Group("/settings")
	// Languages is public
	settingsGroup.GET("/languages", app.SettingsHandler.GetLanguages)
	// Other routes need auth
	protectedSettings := settingsGroup.Group("")
	protectedSettings.Use(authMiddleware.RequireAuth, userActiveMW)
	protectedSettings.GET("", app.SettingsHandler.GetSettings)
	protectedSettings.POST("", app.SettingsHandler.UpdateSettings)
	protectedSettings.GET("/base-currency/", app.SettingsHandler.GetBaseCurrency)
	protectedSettings.PUT("/base-currency/", app.SettingsHandler.UpdateBaseCurrency)
}

// SetupBudgetsRoutes registers budgets routes on the Echo instance
func (app *TestApp) SetupBudgetsRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	budgetsGroup := app.Echo.Group("/budgets")
	budgetsGroup.Use(authMiddleware.RequireAuth, userActiveMW)
	app.BudgetsHandler.RegisterRoutes(budgetsGroup)

	// Budget admin routes
	budgetsAdminGroup := budgetsGroup.Group("")
	budgetsAdminGroup.Use(middleware.RequireAdmin(TestConfig.AdminEmails))
	budgetsAdminGroup.GET("/daily-processing", app.BudgetsHandler.DailyProcessing)
}

// SetupReportsRoutes registers reports routes on the Echo instance
func (app *TestApp) SetupReportsRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	reportsGroup := app.Echo.Group("/reports")
	reportsGroup.Use(authMiddleware.RequireAuth, userActiveMW)
	app.ReportsHandler.RegisterRoutes(reportsGroup)
}

// SetupExchangeRatesRoutes registers exchange rates routes on the Echo instance
func (app *TestApp) SetupExchangeRatesRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	exchangeRatesGroup := app.Echo.Group("/exchange-rates")
	exchangeRatesGroup.Use(authMiddleware.RequireAuth, userActiveMW)
	app.ExchangeRatesHandler.RegisterRoutes(exchangeRatesGroup)

	// Exchange rates admin routes
	exchangeRatesAdminGroup := exchangeRatesGroup.Group("")
	exchangeRatesAdminGroup.Use(middleware.RequireAdmin(TestConfig.AdminEmails))
	exchangeRatesAdminGroup.GET("/update", app.ExchangeRatesHandler.UpdateRates)
	exchangeRatesAdminGroup.GET("/update/from/:start_date/to/:end_date", app.ExchangeRatesHandler.UpdateRatesRange)
}

// SetupPlannedTransactionsRoutes registers planned transactions routes on the Echo instance
func (app *TestApp) SetupPlannedTransactionsRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	plannedTxGroup := app.Echo.Group("/planned-transactions")
	plannedTxGroup.Use(authMiddleware.RequireAuth, userActiveMW)
	app.PlannedTxHandler.RegisterRoutes(plannedTxGroup)
}

// SetupFinancialPlanningRoutes registers financial planning routes on the Echo instance
func (app *TestApp) SetupFinancialPlanningRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	financialPlanningGroup := app.Echo.Group("/financial-planning")
	financialPlanningGroup.Use(authMiddleware.RequireAuth, userActiveMW)
	app.FinancialPlanningHandler.RegisterRoutes(financialPlanningGroup)
}

// SetupSubscriptionsRoutes registers subscriptions routes on the Echo instance
func (app *TestApp) SetupSubscriptionsRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	subscriptionsGroup := app.Echo.Group("/subscriptions")
	app.SubscriptionsHandler.RegisterRoutes(subscriptionsGroup, authMiddleware.RequireAuth, userActiveMW)
}

// SetupPaymentsRoutes registers payments routes on the Echo instance
func (app *TestApp) SetupPaymentsRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	paymentsGroup := app.Echo.Group("/payments")
	app.PaymentsHandler.RegisterRoutes(paymentsGroup, authMiddleware.RequireAuth, userActiveMW)
}

// SetupManagementRoutes registers management routes on the Echo instance
func (app *TestApp) SetupManagementRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	managementGroup := app.Echo.Group("/management")
	managementGroup.Use(authMiddleware.RequireAuth, userActiveMW, middleware.RequireAdmin(TestConfig.AdminEmails))
	app.ManagementHandler.RegisterRoutes(managementGroup)
}

// SetupAnalyticsRoutes registers analytics routes on the Echo instance
func (app *TestApp) SetupAnalyticsRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	analyticsGroup := app.Echo.Group("/analytics")
	analyticsGroup.Use(authMiddleware.RequireAuth, userActiveMW)
	app.AnalyticsHandler.RegisterRoutes(analyticsGroup)
}

// SetupExportRoutes registers export routes on the Echo instance
func (app *TestApp) SetupExportRoutes() {
	authMiddleware := middleware.NewAuthMiddleware(app.JWTService, app.Logger)
	userActiveMW := middleware.RequireActiveUser(app.UserRepo, app.Logger)

	exportGroup := app.Echo.Group("/export")
	exportGroup.Use(authMiddleware.RequireAuth, userActiveMW)
	app.ExportHandler.RegisterRoutes(exportGroup)
}

// SetupAllRoutes sets up all routes
func (app *TestApp) SetupAllRoutes() {
	app.SetupAuthRoutes()
	app.SetupAccountsRoutes()
	app.SetupTransactionsRoutes()
	app.SetupCategoriesRoutes()
	app.SetupCurrenciesRoutes()
	app.SetupSettingsRoutes()
	app.SetupBudgetsRoutes()
	app.SetupReportsRoutes()
	app.SetupExchangeRatesRoutes()
	app.SetupPlannedTransactionsRoutes()
	app.SetupFinancialPlanningRoutes()
	app.SetupSubscriptionsRoutes()
	app.SetupPaymentsRoutes()
	app.SetupManagementRoutes()
	app.SetupAnalyticsRoutes()
	app.SetupExportRoutes()
}

// NewRequest creates a new HTTP request for testing
func NewRequest(method, path string, body string) (*httptest.ResponseRecorder, echo.Context) {
	e := echo.New()
	e.Validator = middleware.NewValidator()

	req := httptest.NewRequest(method, path, nil)
	if body != "" {
		req = httptest.NewRequest(method, path, nil)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	return rec, c
}

// CleanupUser removes a user and all related data from the test database
func CleanupUser(t *testing.T, db *sql.DB, userID int) {
	t.Helper()

	// Delete in order of dependencies
	queries := []string{
		"DELETE FROM transaction_templates WHERE user_id = $1",
		"DELETE FROM transactions WHERE user_id = $1",
		"DELETE FROM planned_transactions WHERE user_id = $1",
		"DELETE FROM budgets WHERE user_id = $1",
		"DELETE FROM accounts WHERE user_id = $1",
		"DELETE FROM user_settings WHERE user_id = $1",
		"DELETE FROM user_categories WHERE user_id = $1",
		"DELETE FROM activation_tokens WHERE user_id = $1",
		"DELETE FROM subscriptions WHERE user_id = $1",
		"DELETE FROM users WHERE id = $1",
	}

	for _, q := range queries {
		_, _ = db.Exec(q, userID)
	}
}

// CleanupUserByEmail removes a user by email
func CleanupUserByEmail(t *testing.T, db *sql.DB, email string) {
	t.Helper()

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		return // User doesn't exist
	}

	CleanupUser(t, db, userID)
}

// CreateTestUser creates a test user and returns their ID
func CreateTestUser(t *testing.T, app *TestApp, email, password string) int {
	t.Helper()

	result, err := app.AuthService.Register(email, password, "", "")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Activate user
	_, err = app.DB.Exec("UPDATE users SET is_active = true WHERE id = $1", result.User.ID)
	if err != nil {
		t.Fatalf("Failed to activate test user: %v", err)
	}

	return result.User.ID
}

// GetAuthToken gets a JWT token for a user
func GetAuthToken(t *testing.T, app *TestApp, email, password string) string {
	t.Helper()

	token, err := app.AuthService.Login(email, password)
	if err != nil {
		t.Fatalf("Failed to get auth token: %v", err)
	}

	return token
}

// AuthHeaders returns headers with auth token
func AuthHeaders(token string) map[string]string {
	return map[string]string{
		"auth-token": token,
	}
}

// GetCurrencyID returns the ID of a currency by code (e.g., "USD")
func GetCurrencyID(t *testing.T, db *sql.DB, code string) int {
	t.Helper()

	var id int
	err := db.QueryRow("SELECT id FROM currencies WHERE code = $1", code).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to get currency %s: %v", code, err)
	}
	return id
}

// GetAccountTypeID returns the ID of an account type
func GetAccountTypeID(t *testing.T, db *sql.DB, isCredit bool) int {
	t.Helper()

	var id int
	err := db.QueryRow("SELECT id FROM account_types WHERE is_credit = $1 LIMIT 1", isCredit).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to get account type (is_credit=%v): %v", isCredit, err)
	}
	return id
}

// CreateTestAccount creates a test account and returns its ID
func CreateTestAccount(t *testing.T, db *sql.DB, userID int, name string, currencyID, accountTypeID int, balance float64) int {
	t.Helper()

	var id int
	err := db.QueryRow(`
		INSERT INTO accounts (user_id, name, currency_id, account_type_id, initial_balance, balance, is_hidden, show_in_reports, is_deleted, is_archived, opening_date)
		VALUES ($1, $2, $3, $4, $5, $5, false, true, false, false, NOW())
		RETURNING id
	`, userID, name, currencyID, accountTypeID, balance).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test account: %v", err)
	}
	return id
}

// DeleteTestAccount removes a test account
func DeleteTestAccount(t *testing.T, db *sql.DB, accountID int) {
	t.Helper()

	// Delete related transactions first
	_, _ = db.Exec("DELETE FROM transactions WHERE account_id = $1", accountID)
	_, _ = db.Exec("DELETE FROM accounts WHERE id = $1", accountID)
}

// CreateTestCategory creates a test category and returns its ID
func CreateTestCategory(t *testing.T, db *sql.DB, userID int, name string, isIncome bool) int {
	t.Helper()

	var id int
	err := db.QueryRow(`
		INSERT INTO user_categories (user_id, name, is_income)
		VALUES ($1, $2, $3)
		RETURNING id
	`, userID, name, isIncome).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test category: %v", err)
	}
	return id
}

// DeleteTestCategory removes a test category
func DeleteTestCategory(t *testing.T, db *sql.DB, categoryID int) {
	t.Helper()
	_, _ = db.Exec("DELETE FROM user_categories WHERE id = $1", categoryID)
}

// CreateTestTransaction creates a test transaction and returns its ID
func CreateTestTransaction(t *testing.T, db *sql.DB, userID, accountID int, categoryID *int, amount float64, isIncome bool) int {
	t.Helper()

	var id int
	err := db.QueryRow(`
		INSERT INTO transactions (user_id, account_id, category_id, amount, is_income, is_transfer, is_deleted, date_time, new_balance)
		VALUES ($1, $2, $3, $4, $5, false, false, NOW(), $4)
		RETURNING id
	`, userID, accountID, categoryID, amount, isIncome).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test transaction: %v", err)
	}
	return id
}

// DeleteTestTransaction removes a test transaction
func DeleteTestTransaction(t *testing.T, db *sql.DB, transactionID int) {
	t.Helper()
	_, _ = db.Exec("DELETE FROM transactions WHERE id = $1", transactionID)
}

// CreateTestTemplate creates a test transaction template and returns its ID
func CreateTestTemplate(t *testing.T, db *sql.DB, userID int, label string, categoryID *int) int {
	t.Helper()

	var id int
	err := db.QueryRow(`
		INSERT INTO transaction_templates (user_id, label, category_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`, userID, label, categoryID).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}
	return id
}

// CreateTestTransferTemplate creates a transaction template with a target account for testing
func CreateTestTransferTemplate(t *testing.T, db *sql.DB, userID int, label string, targetAccountID int) int {
	t.Helper()

	var id int
	err := db.QueryRow(`
		INSERT INTO transaction_templates (user_id, label, target_account_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`, userID, label, targetAccountID).Scan(&id)
	if err != nil {
		t.Fatalf("Failed to create test transfer template: %v", err)
	}
	return id
}

// DeleteTestTemplate removes a test transaction template
func DeleteTestTemplate(t *testing.T, db *sql.DB, templateID int) {
	t.Helper()
	_, _ = db.Exec("DELETE FROM transaction_templates WHERE id = $1", templateID)
}

// StubPaymentProvider implements payment.PaymentProvider for integration tests.
// It provides minimal stub responses so that payment handler integration tests
// pass without a real Stripe connection.
type StubPaymentProvider struct {
	// SessionUserID is returned in checkout session metadata as "user_id".
	// Set this before calling GetSessionStatus so ownership checks pass.
	SessionUserID string
}

func (s *StubPaymentProvider) CreateCustomer(email string, name string, metadata map[string]string) (string, error) {
	return "cus_stub_" + email, nil
}

func (s *StubPaymentProvider) GetCustomer(customerID string) (*paymentservice.Customer, error) {
	return &paymentservice.Customer{ID: customerID, Email: "stub@example.com"}, nil
}

func (s *StubPaymentProvider) UpdateCustomerBalance(customerID string, balanceCents int64) error {
	return nil
}

func (s *StubPaymentProvider) CreateCheckoutSession(params *paymentservice.CheckoutParams) (*paymentservice.CheckoutSession, error) {
	return &paymentservice.CheckoutSession{
		ID:  "cs_stub_123",
		URL: "https://checkout.stripe.com/stub",
	}, nil
}

func (s *StubPaymentProvider) GetCheckoutSession(sessionID string) (*paymentservice.CheckoutSession, error) {
	metadata := map[string]string{}
	if s.SessionUserID != "" {
		metadata["user_id"] = s.SessionUserID
	}
	return &paymentservice.CheckoutSession{
		ID:            sessionID,
		Status:        "complete",
		PaymentStatus: "paid",
		Metadata:      metadata,
	}, nil
}

func (s *StubPaymentProvider) CreatePortalSession(customerID string, returnURL string) (string, error) {
	return "https://billing.stripe.com/stub", nil
}

func (s *StubPaymentProvider) GetSubscription(subscriptionID string) (*paymentservice.ProviderSubscription, error) {
	now := time.Now()
	return &paymentservice.ProviderSubscription{
		ID:                subscriptionID,
		Status:            "active",
		CurrentPeriodEnd:  now.Add(30 * 24 * time.Hour),
		CancelAtPeriodEnd: false,
	}, nil
}

func (s *StubPaymentProvider) CancelSubscription(subscriptionID string, cancelAtPeriodEnd bool) error {
	return nil
}

func (s *StubPaymentProvider) ReenableSubscription(subscriptionID string) error {
	return nil
}

func (s *StubPaymentProvider) UpdateSubscription(subscriptionID string, params *paymentservice.UpdateSubscriptionParams) (*paymentservice.ProviderSubscription, error) {
	now := time.Now()
	return &paymentservice.ProviderSubscription{
		ID:               subscriptionID,
		Status:           "active",
		CurrentPeriodEnd: now.Add(30 * 24 * time.Hour),
	}, nil
}

func (s *StubPaymentProvider) CreateSubscriptionSchedule(subscriptionID string, phases []paymentservice.SchedulePhase) (string, error) {
	return "sub_sched_stub_123", nil
}

func (s *StubPaymentProvider) ModifySubscriptionSchedule(scheduleID string, phases []paymentservice.SchedulePhase) error {
	return nil
}

func (s *StubPaymentProvider) CancelSubscriptionSchedule(scheduleID string) error {
	return nil
}

func (s *StubPaymentProvider) VerifyWebhookSignature(payload []byte, signature string) ([]byte, error) {
	return payload, nil
}
