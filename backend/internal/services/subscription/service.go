package subscription

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/go-budget/backend/internal/models"
)

// PlanLimits maps plan types to their resource limits.
// A nil value means no limits (unlimited).
var PlanLimits = map[string]*Limits{
	"free":    {Accounts: 2, Budgets: 1, PlanningDays: 14},
	"trial":   nil,
	"premium": nil,
}

// Service provides subscription management operations.
type Service struct {
	subscriptionRepo     SubscriptionRepository
	subscriptionPlanRepo SubscriptionPlanRepository
	planPriceRepo        PlanPriceRepository
	accountRepo          AccountCounter
	budgetRepo           BudgetCounter
	accountArchiver      AccountArchiver // optional; nil means archiving is skipped with a warning
	budgetArchiver       BudgetArchiver  // optional; nil means archiving is skipped with a warning
	logger               *slog.Logger
	trialDurationDays    int
}

// New creates a new subscription service with the given dependencies.
func New(
	subscriptionRepo SubscriptionRepository,
	subscriptionPlanRepo SubscriptionPlanRepository,
	planPriceRepo PlanPriceRepository,
	accountCounter AccountCounter,
	budgetCounter BudgetCounter,
	logger *slog.Logger,
	trialDurationDays int,
) *Service {
	return &Service{
		subscriptionRepo:     subscriptionRepo,
		subscriptionPlanRepo: subscriptionPlanRepo,
		planPriceRepo:        planPriceRepo,
		accountRepo:          accountCounter,
		budgetRepo:           budgetCounter,
		logger:               logger,
		trialDurationDays:    trialDurationDays,
	}
}

// SetArchivers sets the optional account and budget archiver dependencies.
// These are needed only for ApplyDowngradeEntitySelection. If not set, the
// method will log a warning and skip the archiving step.
func (s *Service) SetArchivers(accountArchiver AccountArchiver, budgetArchiver BudgetArchiver) {
	s.accountArchiver = accountArchiver
	s.budgetArchiver = budgetArchiver
}

// GetStatus returns the current subscription status for the given user.
// If no subscription exists, a default free plan status is returned.
// Expired trials are automatically downgraded to the free plan.
func (s *Service) GetStatus(userID int) (*SubscriptionStatus, error) {
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return s.defaultFreeStatus(), nil
		}
		return nil, err
	}

	// Auto-downgrade expired trials
	if sub.PlanType == "trial" && sub.TrialEndsAt != nil && sub.TrialEndsAt.Before(time.Now()) {
		if downgradeErr := s.downgradeToFree(sub); downgradeErr != nil {
			s.logger.Error("failed to auto-downgrade expired trial",
				"userID", userID,
				"subscriptionID", sub.ID,
				"error", downgradeErr,
			)
			return nil, downgradeErr
		}
		return s.buildFreeStatus(sub)
	}

	effectivePlanType := s.GetEffectivePlanType(sub)
	status := &SubscriptionStatus{
		PlanType:              effectivePlanType,
		IsActive:              sub.IsActive,
		PlanID:                sub.PlanID,
		Limits:                PlanLimits[effectivePlanType],
		SubscribedAt:          sub.SubscribedAt,
		ExpiresAt:             sub.ExpiresAt,
		PendingPlanID:         sub.PendingPlanID,
		HasStripeSubscription: sub.HasStripeSubscription,
	}

	// Check if free plan requires downgrade (user exceeds limits)
	if effectivePlanType == "free" {
		requiresDowngrade, err := s.checkRequiresDowngrade(userID)
		if err != nil {
			s.logger.Error("failed to check downgrade requirement",
				"userID", userID,
				"error", err,
			)
			// Non-fatal: continue with requiresDowngrade = false
		}
		status.RequiresDowngrade = requiresDowngrade
	}

	// Calculate trial days remaining
	if sub.PlanType == "trial" && sub.TrialEndsAt != nil {
		remaining := int(math.Ceil(time.Until(*sub.TrialEndsAt).Hours() / 24))
		if remaining < 0 {
			remaining = 0
		}
		status.TrialDaysRemaining = &remaining
	}

	// Look up pending plan name if there is a pending plan
	if sub.PendingPlanID != nil {
		pendingPlan, err := s.subscriptionPlanRepo.GetByID(*sub.PendingPlanID)
		if err != nil {
			s.logger.Error("failed to look up pending plan",
				"pendingPlanID", *sub.PendingPlanID,
				"error", err,
			)
			// Non-fatal: continue without the pending plan name
		} else {
			status.PendingPlanName = &pendingPlan.Name
		}
	}

	return status, nil
}

// GetPlans returns all active subscription plans.
func (s *Service) GetPlans() ([]*SubscriptionPlan, error) {
	plans, err := s.subscriptionPlanRepo.GetActivePlans()
	if err != nil {
		return nil, err
	}
	result := make([]*SubscriptionPlan, len(plans))
	for i, p := range plans {
		result[i] = toSubscriptionPlan(p)
	}
	return result, nil
}

// GetEffectivePlanType returns the effective plan type for a subscription,
// accounting for expired trials and expired premium subscriptions.
// This is exported because handlers need it to determine the actual plan state.
func (s *Service) GetEffectivePlanType(sub *models.Subscription) string {
	if sub.PlanType == "trial" && sub.TrialEndsAt != nil && sub.TrialEndsAt.Before(time.Now()) {
		return "free"
	}
	if sub.PlanType == "premium" && sub.ExpiresAt != nil && sub.ExpiresAt.Before(time.Now()) {
		return "free"
	}
	return sub.PlanType
}

// validatePlanTransition checks whether a plan transition is allowed.
func (s *Service) validatePlanTransition(currentPlanType, targetPlanType string) error {
	if currentPlanType == targetPlanType {
		return ErrSamePlan
	}
	if targetPlanType == "trial" {
		return ErrInvalidPlanTransition
	}
	if currentPlanType == "trial" && targetPlanType == "free" {
		return ErrInvalidPlanTransition
	}
	if currentPlanType == "premium" && targetPlanType == "free" {
		return ErrDowngradeNotAllowed
	}
	return nil
}

// defaultFreeStatus returns the default subscription status for users without a subscription record.
func (s *Service) defaultFreeStatus() *SubscriptionStatus {
	return &SubscriptionStatus{
		PlanType: "free",
		IsActive: true,
		Limits:   PlanLimits["free"],
	}
}

// downgradeToFree updates an expired trial subscription to the free plan in the database.
func (s *Service) downgradeToFree(sub *models.Subscription) error {
	freePlan, err := s.subscriptionPlanRepo.GetFreePlan()
	if err != nil {
		return err
	}

	sub.PlanID = freePlan.ID
	sub.PlanType = "free"
	sub.IsActive = true

	return s.subscriptionRepo.Update(sub)
}

// buildFreeStatus creates a SubscriptionStatus for a subscription that has been downgraded to free.
func (s *Service) buildFreeStatus(sub *models.Subscription) (*SubscriptionStatus, error) {
	requiresDowngrade, err := s.checkRequiresDowngrade(sub.UserID)
	if err != nil {
		s.logger.Error("failed to check downgrade requirement after trial expiry",
			"userID", sub.UserID,
			"error", err,
		)
		// Non-fatal: continue with requiresDowngrade = false
	}

	return &SubscriptionStatus{
		PlanType:              "free",
		IsActive:              true,
		PlanID:                sub.PlanID,
		RequiresDowngrade:     requiresDowngrade,
		Limits:                PlanLimits["free"],
		SubscribedAt:          sub.SubscribedAt,
		HasStripeSubscription: sub.HasStripeSubscription,
	}, nil
}

// checkRequiresDowngrade determines whether the user's active resources
// exceed the free plan limits.
func (s *Service) checkRequiresDowngrade(userID int) (bool, error) {
	limits := PlanLimits["free"]
	if limits == nil {
		return false, nil
	}

	accountCount, err := s.accountRepo.CountActiveByUserID(userID)
	if err != nil {
		return false, err
	}
	if accountCount > limits.Accounts {
		return true, nil
	}

	budgetCount, err := s.budgetRepo.CountActiveByUserID(userID)
	if err != nil {
		return false, err
	}
	if budgetCount > limits.Budgets {
		return true, nil
	}

	return false, nil
}

// Upgrade changes a user's subscription to the target plan. It handles both
// new subscriptions (no existing record) and transitions between plans.
//
// Returns an UpgradeResult indicating the type of change ("upgrade" or
// "plan_change") and the updated subscription. Validation errors are returned
// for invalid transitions (e.g., upgrading to the same plan, to "trial", or
// downgrade via this endpoint).
func (s *Service) Upgrade(userID int, planID int) (*UpgradeResult, error) {
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get subscription: %w", err)
		}

		// No existing subscription — treat as a new subscription.
		targetPlan, planErr := s.subscriptionPlanRepo.GetByID(planID)
		if planErr != nil {
			if errors.Is(planErr, sql.ErrNoRows) {
				return nil, ErrPlanNotFound
			}
			return nil, fmt.Errorf("get target plan: %w", planErr)
		}

		now := time.Now()
		newSub := &models.Subscription{
			UserID:   userID,
			PlanID:   targetPlan.ID,
			PlanType: targetPlan.PlanType,
			IsActive: true,
		}
		if targetPlan.BillingPeriod != nil {
			newSub.CurrentBillingPeriodID = &targetPlan.BillingPeriod.ID
			newSub.SubscribedAt = &now
			expiresAt := now.AddDate(0, 0, targetPlan.BillingPeriod.DurationDays)
			newSub.ExpiresAt = &expiresAt
		}

		if createErr := s.subscriptionRepo.Create(newSub); createErr != nil {
			return nil, fmt.Errorf("create subscription: %w", createErr)
		}

		return &UpgradeResult{
			ChangeType:   "upgrade",
			Subscription: toSubscription(newSub),
			Message:      fmt.Sprintf("Subscribed to %s plan", targetPlan.Name),
		}, nil
	}

	// Existing subscription — validate transition.
	targetPlan, err := s.subscriptionPlanRepo.GetByID(planID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get target plan: %w", err)
	}

	currentPlan, err := s.subscriptionPlanRepo.GetByID(sub.PlanID)
	if err != nil {
		return nil, fmt.Errorf("get current plan: %w", err)
	}

	// Use the effective plan type for validation (accounts for expired trials/premium).
	effectiveCurrentType := s.GetEffectivePlanType(sub)

	if err := s.validatePlanTransition(effectiveCurrentType, targetPlan.PlanType); err != nil {
		return nil, err
	}

	// Same plan ID is not allowed even if the plan type check passed
	// (e.g., two different premium plans could share the "premium" type).
	if sub.PlanID == planID {
		return nil, ErrSamePlan
	}

	// Determine change type based on billing period comparison.
	changeType := s.determineChangeType(currentPlan, targetPlan)

	// Apply the plan change.
	now := time.Now()
	sub.PlanID = targetPlan.ID
	sub.PlanType = targetPlan.PlanType
	sub.IsActive = true
	if targetPlan.BillingPeriod != nil {
		sub.CurrentBillingPeriodID = &targetPlan.BillingPeriod.ID
		sub.SubscribedAt = &now
		expiresAt := now.AddDate(0, 0, targetPlan.BillingPeriod.DurationDays)
		sub.ExpiresAt = &expiresAt
	}

	if updateErr := s.subscriptionRepo.Update(sub); updateErr != nil {
		return nil, fmt.Errorf("update subscription: %w", updateErr)
	}

	return &UpgradeResult{
		ChangeType:   changeType,
		Subscription: toSubscription(sub),
		Message:      fmt.Sprintf("Plan changed to %s", targetPlan.Name),
	}, nil
}

// determineChangeType compares the current and target plans to decide whether
// the transition is an "upgrade" (same or longer billing period) or a
// "plan_change" (shorter billing period, e.g., yearly to monthly within premium).
func (s *Service) determineChangeType(current, target *models.SubscriptionPlan) string {
	if current.BillingPeriod == nil || target.BillingPeriod == nil {
		return "upgrade"
	}
	if target.BillingPeriod.DurationDays >= current.BillingPeriod.DurationDays {
		return "upgrade"
	}
	return "plan_change"
}

// ScheduleDowngrade schedules (or immediately applies) a downgrade to the free plan.
// The accountIDs and budgetID parameters specify which entities the user wants
// to keep after the downgrade; all others will be archived.
//
// If the user has an active Stripe subscription the downgrade is scheduled for
// the end of the current billing period. Otherwise, it is applied immediately.
func (s *Service) ScheduleDowngrade(userID int, accountIDs []int, budgetID *int) (*DowngradeResult, error) {
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAlreadyOnFreePlan
		}
		return nil, fmt.Errorf("get subscription: %w", err)
	}

	effectiveType := s.GetEffectivePlanType(sub)
	if effectiveType == "free" {
		return nil, ErrAlreadyOnFreePlan
	}

	// Validate entity selection against free plan limits.
	freeLimits := PlanLimits["free"]
	if freeLimits != nil {
		if len(accountIDs) > freeLimits.Accounts {
			return nil, ErrInvalidEntitySelection
		}
	}

	freePlan, err := s.subscriptionPlanRepo.GetFreePlan()
	if err != nil {
		return nil, fmt.Errorf("get free plan: %w", err)
	}

	// Store the pending downgrade information on the subscription.
	sub.PendingPlanID = &freePlan.ID
	sub.PendingDowngradeAccountIDs = accountIDs
	sub.PendingDowngradeBudgetID = budgetID

	if !sub.HasStripeSubscription {
		// No Stripe subscription — apply the downgrade immediately.
		sub.PlanID = freePlan.ID
		sub.PlanType = freePlan.PlanType
		sub.PendingPlanID = nil
		sub.PendingDowngradeAccountIDs = nil
		sub.PendingDowngradeBudgetID = nil
		sub.IsActive = true

		if saveErr := s.subscriptionRepo.UpdateSubscriptionFull(sub); saveErr != nil {
			return nil, fmt.Errorf("save immediate downgrade: %w", saveErr)
		}

		if archiveErr := s.ApplyDowngradeEntitySelection(userID, accountIDs, budgetID); archiveErr != nil {
			s.logger.Error("failed to apply entity selection during immediate downgrade",
				"userID", userID,
				"error", archiveErr,
			)
			// Non-fatal: the subscription is already downgraded, but entities may
			// not have been archived. The user can still use the UI to manage them.
		}

		return &DowngradeResult{
			Scheduled: false,
			Message:   "Downgraded to free plan immediately",
		}, nil
	}

	// Has Stripe subscription — schedule the downgrade for the end of the billing period.
	if saveErr := s.subscriptionRepo.UpdateSubscriptionFull(sub); saveErr != nil {
		return nil, fmt.Errorf("save scheduled downgrade: %w", saveErr)
	}

	return &DowngradeResult{
		Scheduled: true,
		Message:   "Downgrade scheduled for end of billing period",
		ExpiresAt: sub.ExpiresAt,
	}, nil
}

// ApplyDowngradeEntitySelection archives the user's accounts and budgets that
// were not selected for retention during a downgrade to the free plan.
//
// accountIDs lists the account IDs to keep; all other active accounts belonging
// to the user will be archived. budgetID (if non-nil) specifies the single
// budget to keep; all others will be archived.
//
// If the archiver dependencies have not been set (via SetArchivers), the method
// logs a warning and returns nil without performing any archiving.
func (s *Service) ApplyDowngradeEntitySelection(userID int, accountIDs []int, budgetID *int) error {
	if s.accountArchiver == nil || s.budgetArchiver == nil {
		s.logger.Warn("archiver dependencies not set, skipping entity archiving",
			"userID", userID,
		)
		return nil
	}

	if err := s.accountArchiver.ArchiveByUserExcept(userID, accountIDs); err != nil {
		return fmt.Errorf("archive accounts for user %d: %w", userID, err)
	}

	if err := s.budgetArchiver.ArchiveByUserExcept(userID, budgetID); err != nil {
		return fmt.Errorf("archive budgets for user %d: %w", userID, err)
	}

	s.logger.Info("applied downgrade entity selection",
		"userID", userID,
		"keptAccounts", accountIDs,
		"keptBudget", budgetID,
	)

	return nil
}

// CreateTrialSubscription creates a new trial subscription for the given user.
// The trial duration is determined by the trialDurationDays configuration value.
// If no trial plan is found among the active plans, an error is returned.
func (s *Service) CreateTrialSubscription(userID int) error {
	trialPlan, err := s.findTrialPlan()
	if err != nil {
		return err
	}

	now := time.Now()
	trialEndsAt := now.AddDate(0, 0, s.trialDurationDays)

	sub := &models.Subscription{
		UserID:         userID,
		PlanID:         trialPlan.ID,
		PlanType:       trialPlan.PlanType,
		TrialStartedAt: &now,
		TrialEndsAt:    &trialEndsAt,
		IsActive:       true,
		AutoRenew:      false,
	}

	if err := s.subscriptionRepo.Create(sub); err != nil {
		return fmt.Errorf("create trial subscription for user %d: %w", userID, err)
	}

	s.logger.Info("created trial subscription",
		"userID", userID,
		"planID", trialPlan.ID,
		"trialEndsAt", trialEndsAt,
	)

	return nil
}

// ProcessRenewals handles expired subscriptions. For each expired subscription:
//   - trial subscriptions are downgraded to the free plan
//   - premium subscriptions without a Stripe subscription are deactivated
//   - premium subscriptions with a Stripe subscription are skipped (Stripe webhooks handle renewal)
//
// Individual subscription errors are logged and do not stop processing of
// remaining subscriptions.
func (s *Service) ProcessRenewals() error {
	expired, err := s.subscriptionRepo.GetExpiredSubscriptions()
	if err != nil {
		return fmt.Errorf("get expired subscriptions: %w", err)
	}

	s.logger.Info("processing expired subscriptions", "count", len(expired))

	for _, sub := range expired {
		switch sub.PlanType {
		case "trial":
			freePlan, err := s.subscriptionPlanRepo.GetFreePlan()
			if err != nil {
				s.logger.Error("failed to get free plan for trial downgrade",
					"subscriptionID", sub.ID,
					"userID", sub.UserID,
					"error", err,
				)
				continue
			}

			sub.PlanID = freePlan.ID
			sub.PlanType = freePlan.PlanType
			sub.TrialStartedAt = nil
			sub.TrialEndsAt = nil
			sub.IsActive = true

			if err := s.subscriptionRepo.Update(sub); err != nil {
				s.logger.Error("failed to downgrade expired trial to free",
					"subscriptionID", sub.ID,
					"userID", sub.UserID,
					"error", err,
				)
				continue
			}

			s.logger.Info("downgraded expired trial to free plan",
				"subscriptionID", sub.ID,
				"userID", sub.UserID,
			)

		case "premium":
			if sub.HasStripeSubscription {
				s.logger.Info("skipping premium subscription with Stripe (handled by webhooks)",
					"subscriptionID", sub.ID,
					"userID", sub.UserID,
				)
				continue
			}

			sub.IsActive = false

			if err := s.subscriptionRepo.Update(sub); err != nil {
				s.logger.Error("failed to deactivate expired premium subscription",
					"subscriptionID", sub.ID,
					"userID", sub.UserID,
					"error", err,
				)
				continue
			}

			s.logger.Info("deactivated expired premium subscription without Stripe",
				"subscriptionID", sub.ID,
				"userID", sub.UserID,
			)

		default:
			s.logger.Warn("unexpected plan type for expired subscription",
				"subscriptionID", sub.ID,
				"userID", sub.UserID,
				"planType", sub.PlanType,
			)
		}
	}

	return nil
}

// ProcessExpiredDowngrades applies pending plan downgrades whose billing period
// has ended (or has no expiration date). For each pending downgrade:
//   - the plan is switched to the pending plan
//   - pending fields are cleared
//   - if the new plan is "free", entity deactivation is applied using stored
//     PendingDowngradeAccountIDs and PendingDowngradeBudgetID
//
// Individual subscription errors are logged and do not stop processing of
// remaining subscriptions.
func (s *Service) ProcessExpiredDowngrades() error {
	pending, err := s.subscriptionRepo.GetPendingDowngrades()
	if err != nil {
		return fmt.Errorf("get pending downgrades: %w", err)
	}

	s.logger.Info("processing pending downgrades", "count", len(pending))

	for _, sub := range pending {
		if sub.PendingPlanID == nil {
			// Should not happen given the query, but guard defensively.
			continue
		}

		pendingPlan, err := s.subscriptionPlanRepo.GetByID(*sub.PendingPlanID)
		if err != nil {
			s.logger.Error("failed to get pending plan",
				"subscriptionID", sub.ID,
				"userID", sub.UserID,
				"pendingPlanID", *sub.PendingPlanID,
				"error", err,
			)
			continue
		}

		// Save the entity selection before clearing pending fields.
		accountIDs := sub.PendingDowngradeAccountIDs
		budgetID := sub.PendingDowngradeBudgetID

		// Apply the plan change.
		sub.PlanID = pendingPlan.ID
		sub.PlanType = pendingPlan.PlanType
		sub.PendingPlanID = nil
		sub.PendingDowngradeAccountIDs = nil
		sub.PendingDowngradeBudgetID = nil
		sub.IsActive = true

		if err := s.subscriptionRepo.UpdateSubscriptionFull(sub); err != nil {
			s.logger.Error("failed to apply pending downgrade",
				"subscriptionID", sub.ID,
				"userID", sub.UserID,
				"newPlanID", pendingPlan.ID,
				"error", err,
			)
			continue
		}

		// If the new plan is free, apply entity deactivation.
		if pendingPlan.PlanType == "free" {
			if archiveErr := s.ApplyDowngradeEntitySelection(sub.UserID, accountIDs, budgetID); archiveErr != nil {
				s.logger.Error("failed to apply entity selection during downgrade",
					"subscriptionID", sub.ID,
					"userID", sub.UserID,
					"error", archiveErr,
				)
				// Non-fatal: subscription is already downgraded.
			}
		}

		s.logger.Info("applied pending downgrade",
			"subscriptionID", sub.ID,
			"userID", sub.UserID,
			"newPlanType", pendingPlan.PlanType,
			"newPlanID", pendingPlan.ID,
		)
	}

	return nil
}

// findTrialPlan searches the active plans for one with PlanType "trial".
func (s *Service) findTrialPlan() (*models.SubscriptionPlan, error) {
	plans, err := s.subscriptionPlanRepo.GetActivePlans()
	if err != nil {
		return nil, fmt.Errorf("get active plans: %w", err)
	}

	for _, p := range plans {
		if p.PlanType == "trial" {
			return p, nil
		}
	}

	return nil, fmt.Errorf("no active trial plan found")
}

// toSubscriptionPlan converts a models.SubscriptionPlan to the service domain type.
func toSubscriptionPlan(m *models.SubscriptionPlan) *SubscriptionPlan {
	if m == nil {
		return nil
	}
	sp := &SubscriptionPlan{
		ID:             m.ID,
		Name:           m.Name,
		TranslationKey: m.TranslationKey,
		PlanType:       m.PlanType,
		Price:          m.Price,
		CurrencyCode:   m.CurrencyCode,
		IsFeatured:     m.IsFeatured,
		Description:    m.Description,
		SortOrder:      m.SortOrder,
	}
	if m.BillingPeriod != nil {
		sp.BillingPeriod = &BillingPeriod{
			ID:           m.BillingPeriod.ID,
			Code:         m.BillingPeriod.Code,
			Name:         m.BillingPeriod.Name,
			DurationDays: m.BillingPeriod.DurationDays,
		}
	}
	return sp
}

// toSubscription converts a models.Subscription to the service domain type.
func toSubscription(m *models.Subscription) *Subscription {
	if m == nil {
		return nil
	}
	return &Subscription{
		ID:                     m.ID,
		UserID:                 m.UserID,
		PlanID:                 m.PlanID,
		PlanType:               m.PlanType,
		CurrentBillingPeriodID: m.CurrentBillingPeriodID,
		TrialStartedAt:         m.TrialStartedAt,
		TrialEndsAt:            m.TrialEndsAt,
		SubscribedAt:           m.SubscribedAt,
		ExpiresAt:              m.ExpiresAt,
		AutoRenew:              m.AutoRenew,
		CanceledAt:             m.CanceledAt,
		PendingPlanID:          m.PendingPlanID,
		IsActive:               m.IsActive,
		HasStripeSubscription:  m.HasStripeSubscription,
		CreatedAt:              m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
	}
}
