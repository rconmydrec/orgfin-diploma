package tasks

// Task type constants define all async task types in the system.
// These are used as the type argument when creating asynq.Task instances
// and when registering handlers on the asynq.ServeMux.
const (
	// Scheduled tasks
	TypeExchangeRateUpdate    = "exchange_rate:update"
	TypeBudgetProcessing      = "budget:processing"
	TypeTokenCleanup          = "token:cleanup"
	TypeSubscriptionRenewal   = "subscription:renewal"
	TypeSubscriptionDowngrade = "subscription:downgrade"

	// On-demand tasks
	TypeEmailSend        = "email:send"
	TypeEmailActivation  = "email:activation"
	TypeBudgetUserUpdate = "budget:user_update"
	TypeExportEmail      = "export:email"
)
