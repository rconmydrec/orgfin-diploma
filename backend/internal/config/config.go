package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	DatabaseURL string
	DBHost      string
	DBPort      int
	DBUser      string
	DBPassword  string
	DBName      string

	// Server
	Port        string
	Environment string

	// Security
	JWTSecret            string
	JWTExpirationMinutes int

	// Frontend
	FrontendURL string

	// Logging
	LogLevel string

	// SMTP (Email)
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	SMTPFromName string
	AppURL       string

	// MessageIDDomain is the FQDN used as the domain part of outbound email
	// Message-ID headers. It MUST contain at least one dot to satisfy RFC 5322
	// §3.6.4 and to avoid Gmail's bulk-mail spam heuristics. When empty, the
	// email service falls back to the local hostname / FQDN resolution chain.
	MessageIDDomain string

	// Stripe
	StripeSecretKey     string
	StripeWebhookSecret string
	StripeSuccessURL    string
	StripeCancelURL     string

	// OpenAI
	OpenAIAPIKey string
	OpenAIModel  string

	// Google OAuth
	GoogleClientID string

	// Exchange Rates API
	CurrencyBeaconAPIURL string
	CurrencyBeaconAPIKey string

	// Admin
	AdminEmails []string

	// Internal API
	InternalAPIToken string

	// Kubernetes
	K8sNamespace string

	// Redis / Async Workers
	RedisURL          string
	WorkerEnabled     bool
	WorkerConcurrency int

	// Worker Schedules (cron expressions)
	ScheduleExchangeRateUpdate    string
	ScheduleBudgetProcessing      string
	ScheduleTokenCleanup          string
	ScheduleSubscriptionRenewal   string
	ScheduleSubscriptionDowngrade string

	// Export
	MaxExportRangeDays int
}

func New() *Config {
	_ = godotenv.Load()

	// Database config
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnvInt("DB_PORT", 5432)
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "budget_tracker")

	// Build DATABASE_URL if not provided directly
	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL == "" {
		databaseURL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	return &Config{
		// Database
		DatabaseURL: databaseURL,
		DBHost:      dbHost,
		DBPort:      dbPort,
		DBUser:      dbUser,
		DBPassword:  dbPassword,
		DBName:      dbName,

		// Server
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),

		// Security
		JWTSecret:            getEnv("SECRET_KEY", "change-me-in-production"),
		JWTExpirationMinutes: getEnvInt("JWT_EXPIRATION_MINUTES", 30),

		// Frontend
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "info"),

		// SMTP
		SMTPHost:     getEnv("MAIL_SERVER", ""),
		SMTPPort:     getEnvInt("MAIL_PORT", 587),
		SMTPUser:     getEnv("MAIL_USERNAME", ""),
		SMTPPassword: getEnv("MAIL_PASSWORD", ""),
		SMTPFrom:     getEnv("MAIL_FROM", ""),
		SMTPFromName: getEnv("MAIL_FROM_NAME", ""),
		AppURL:       getEnv("APP_URL", "http://localhost:8080"),

		MessageIDDomain: getEnv("MESSAGE_ID_DOMAIN", ""),

		// Stripe
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripeSuccessURL:    getEnv("STRIPE_SUCCESS_URL", "/payment/success"),
		StripeCancelURL:     getEnv("STRIPE_CANCEL_URL", "/payment/cancel"),

		// OpenAI
		OpenAIAPIKey: getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:  getEnv("OPENAI_MODEL", "gpt-4"),

		// Google OAuth
		GoogleClientID: getEnv("GOOGLE_CLIENT_ID", ""),

		// Exchange Rates
		CurrencyBeaconAPIURL: getEnv("CURRENCYBEACON_API_URL", "https://api.currencybeacon.com"),
		CurrencyBeaconAPIKey: getEnv("CURRENCYBEACON_API_KEY", ""),

		// Admin
		AdminEmails: getEnvSlice("ADMINS_NOTIFICATION_EMAILS", []string{}),

		// Internal API
		InternalAPIToken: getEnv("INTERNAL_API_TOKEN", ""),

		// Kubernetes
		K8sNamespace: getEnv("K8S_NAMESPACE", ""),

		// Redis / Async Workers
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379"),
		WorkerEnabled:     getEnvBool("WORKER_ENABLED", true),
		WorkerConcurrency: getEnvInt("WORKER_CONCURRENCY", 10),

		// Worker Schedules
		ScheduleExchangeRateUpdate:    getEnv("SCHEDULE_EXCHANGE_RATE_UPDATE", "0 13 * * *"),
		ScheduleBudgetProcessing:      getEnv("SCHEDULE_BUDGET_PROCESSING", "1 0 * * *"),
		ScheduleTokenCleanup:          getEnv("SCHEDULE_TOKEN_CLEANUP", "0 2 * * *"),
		ScheduleSubscriptionRenewal:   getEnv("SCHEDULE_SUBSCRIPTION_RENEWAL", "0 1 * * *"),
		ScheduleSubscriptionDowngrade: getEnv("SCHEDULE_SUBSCRIPTION_DOWNGRADE", "0 * * * *"),

		// Export
		MaxExportRangeDays: 1096,
	}
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "dev" || c.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Environment == "prod" || c.Environment == "production"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		lower := strings.ToLower(value)
		return lower == "true" || lower == "1" || lower == "yes"
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		var result []string
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return defaultValue
}
