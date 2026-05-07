// Package subscription provides subscription lifecycle management including
// status retrieval, plan listing, upgrade/downgrade, trial creation, renewal
// processing, and entity archiving on downgrade.
//
// Invariants:
//   - Expired trials are auto-downgraded to free on status check.
//   - Plan transitions are validated: trial->free is blocked, premium->free
//     requires the downgrade endpoint, same plan is rejected.
//   - Entity archiving is optional: if archivers are not set, a warning is logged.
//   - Downgrade with Stripe subscription is scheduled for end of billing period.
package subscription

import (
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/shopspring/decimal"
)

// ServiceInterface defines the public API of the subscription service.
type ServiceInterface interface {
	GetStatus(userID int) (*SubscriptionStatus, error)
	GetPlans() ([]*SubscriptionPlan, error)
	Upgrade(userID int, planID int) (*UpgradeResult, error)
	ScheduleDowngrade(userID int, accountIDs []int, budgetID *int) (*DowngradeResult, error)
	ApplyDowngradeEntitySelection(userID int, accountIDs []int, budgetID *int) error
	CreateTrialSubscription(userID int) error
	ProcessRenewals() error
	ProcessExpiredDowngrades() error
	SetArchivers(accountArchiver AccountArchiver, budgetArchiver BudgetArchiver)
}

// SubscriptionPlan represents a subscription plan in the service domain.
type SubscriptionPlan struct {
	ID             int
	Name           string
	TranslationKey *string
	PlanType       string
	Price          decimal.Decimal
	CurrencyCode   string
	IsFeatured     bool
	Description    *string
	SortOrder      int
	BillingPeriod  *BillingPeriod
}

// BillingPeriod represents a billing period in the service domain.
type BillingPeriod struct {
	ID           int
	Code         string
	Name         string
	DurationDays int
}

// Subscription represents a subscription in the service domain.
type Subscription struct {
	ID                     int
	UserID                 int
	PlanID                 int
	PlanType               string
	CurrentBillingPeriodID *int
	TrialStartedAt         *time.Time
	TrialEndsAt            *time.Time
	SubscribedAt           *time.Time
	ExpiresAt              *time.Time
	AutoRenew              bool
	CanceledAt             *time.Time
	PendingPlanID          *int
	IsActive               bool
	HasStripeSubscription  bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// SubscriptionRepository defines subscription data access methods needed by the service.
type SubscriptionRepository interface {
	GetByID(id int) (*models.Subscription, error)
	GetByUserID(userID int) (*models.Subscription, error)
	Create(sub *models.Subscription) error
	Update(sub *models.Subscription) error
	UpdateSubscriptionFull(sub *models.Subscription) error
	GetExpiredSubscriptions() ([]*models.Subscription, error)
	GetPendingDowngrades() ([]*models.Subscription, error)
}

// SubscriptionPlanRepository defines plan data access methods needed by the service.
type SubscriptionPlanRepository interface {
	GetActivePlans() ([]*models.SubscriptionPlan, error)
	GetByID(id int) (*models.SubscriptionPlan, error)
	GetFreePlan() (*models.SubscriptionPlan, error)
}

// PlanPriceRepository defines plan price data access methods needed by the service.
type PlanPriceRepository interface {
	GetByPlanAndProvider(planID int, providerType string) (*models.PlanPrice, error)
}

// AccountCounter provides a minimal interface for counting active accounts.
// Defined here to avoid importing the full account repository.
type AccountCounter interface {
	CountActiveByUserID(userID int) (int, error)
}

// BudgetCounter provides a minimal interface for counting active budgets.
// Defined here to avoid importing the full budget repository.
type BudgetCounter interface {
	CountActiveByUserID(userID int) (int, error)
}

// AccountArchiver provides the ability to archive accounts that a user
// no longer needs when downgrading to the free plan.
type AccountArchiver interface {
	ArchiveByUserExcept(userID int, keepIDs []int) error
}

// BudgetArchiver provides the ability to archive budgets that a user
// no longer needs when downgrading to the free plan.
type BudgetArchiver interface {
	ArchiveByUserExcept(userID int, keepID *int) error
}

// UpgradeResult describes the outcome of an Upgrade operation.
type UpgradeResult struct {
	ChangeType   string // "upgrade", "plan_change"
	Subscription *Subscription
	Message      string
}

// DowngradeResult describes the outcome of a ScheduleDowngrade operation.
type DowngradeResult struct {
	Scheduled bool
	Message   string
	ExpiresAt *time.Time
}

// Limits defines the resource limits for a subscription plan.
type Limits struct {
	Accounts     int
	Budgets      int
	PlanningDays int
}

// SubscriptionStatus represents the current subscription state for a user.
type SubscriptionStatus struct {
	PlanType              string
	IsActive              bool
	PlanID                int
	TrialDaysRemaining    *int
	RequiresDowngrade     bool
	Limits                *Limits
	SubscribedAt          *time.Time
	ExpiresAt             *time.Time
	PendingPlanID         *int
	PendingPlanName       *string
	HasStripeSubscription bool
}
