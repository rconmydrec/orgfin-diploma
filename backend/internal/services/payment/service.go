package payment

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/go-budget/backend/internal/config"
	"github.com/go-budget/backend/internal/models"
	"github.com/shopspring/decimal"
)

// Service orchestrates payment-related business logic, coordinating between
// the PaymentProvider (e.g. Stripe), subscription repositories, and the
// credit calculator.
type Service struct {
	provider         PaymentProvider
	subscriptionRepo SubscriptionRepository
	planRepo         SubscriptionPlanRepository
	planPriceRepo    PlanPriceRepository
	creditCalc       CreditCalculator
	config           *config.Config
	logger           *slog.Logger
}

// NewService creates a new payment Service with the given dependencies.
func NewService(
	provider PaymentProvider,
	subscriptionRepo SubscriptionRepository,
	planRepo SubscriptionPlanRepository,
	planPriceRepo PlanPriceRepository,
	creditCalc CreditCalculator,
	cfg *config.Config,
	logger *slog.Logger,
) *Service {
	return &Service{
		provider:         provider,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		planPriceRepo:    planPriceRepo,
		creditCalc:       creditCalc,
		config:           cfg,
		logger:           logger,
	}
}

// GetOrCreateCustomer ensures the user has a Stripe customer record. If a
// provider subscription already exists with an external customer ID it is
// returned directly. Otherwise a new customer is created in Stripe and the
// provider subscription record is created or updated accordingly.
func (s *Service) GetOrCreateCustomer(userID int, email string, name string) (string, error) {
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("get subscription: %w", err)
	}

	var providerSub *models.PaymentProviderSubscription
	if sub != nil {
		providerSub, err = s.subscriptionRepo.GetProviderSubscription(sub.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("get provider subscription: %w", err)
		}
	}

	// If we already have a customer ID, return it.
	if providerSub != nil && providerSub.ExternalCustomerID != nil && *providerSub.ExternalCustomerID != "" {
		return *providerSub.ExternalCustomerID, nil
	}

	// Create a new customer in Stripe.
	metadata := map[string]string{
		"user_id": strconv.Itoa(userID),
	}
	customerID, err := s.provider.CreateCustomer(email, name, metadata)
	if err != nil {
		return "", fmt.Errorf("create stripe customer: %w", err)
	}

	if providerSub != nil {
		// Update existing provider subscription with the new customer ID.
		providerSub.ExternalCustomerID = &customerID
		if err := s.subscriptionRepo.UpdateProviderSubscription(providerSub); err != nil {
			return "", fmt.Errorf("update provider subscription: %w", err)
		}
	} else if sub != nil {
		// Create a new provider subscription record.
		pps := &models.PaymentProviderSubscription{
			SubscriptionID:     sub.ID,
			ProviderType:       ProviderTypeStripe,
			ExternalCustomerID: &customerID,
		}
		if err := s.subscriptionRepo.CreateProviderSubscription(pps); err != nil {
			return "", fmt.Errorf("create provider subscription: %w", err)
		}
	}

	return customerID, nil
}

// CreateCheckout creates a Stripe Checkout session for the given user and plan.
// If the user has remaining value on their current subscription, it is applied
// as a credit to their Stripe customer balance before creating the session.
func (s *Service) CreateCheckout(userID int, planID int, email string, name string) (*CheckoutResult, error) {
	// Look up the target plan.
	targetPlan, err := s.planRepo.GetByID(planID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get plan: %w", err)
	}

	// Look up the Stripe price for this plan.
	planPrice, err := s.planPriceRepo.GetByPlanAndProvider(planID, ProviderTypeStripe)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidPrice
		}
		return nil, fmt.Errorf("get plan price: %w", err)
	}

	// Ensure we have a Stripe customer.
	customerID, err := s.GetOrCreateCustomer(userID, email, name)
	if err != nil {
		return nil, err
	}

	// Calculate prorated credit from the current subscription if applicable.
	creditCents, creditFormatted, err := s.calculateCurrentCredit(userID)
	if err != nil {
		return nil, err
	}

	// Apply credit to customer balance if applicable.
	if creditCents > 0 {
		if err := s.provider.UpdateCustomerBalance(customerID, creditCents); err != nil {
			s.logger.Error("failed to apply credit to customer balance",
				"userID", userID,
				"customerID", customerID,
				"creditCents", creditCents,
				"error", err,
			)
			return nil, fmt.Errorf("apply credit: %w", err)
		}
		s.logger.Info("applied prorated credit to customer balance",
			"userID", userID,
			"customerID", customerID,
			"creditCents", creditCents,
		)
	}

	// Build success and cancel URLs.
	successURL := s.config.FrontendURL + s.config.StripeSuccessURL + "?session_id={CHECKOUT_SESSION_ID}"
	cancelURL := s.config.FrontendURL + s.config.StripeCancelURL

	// Create the Stripe Checkout session.
	session, err := s.provider.CreateCheckoutSession(&CheckoutParams{
		CustomerID: customerID,
		PriceID:    planPrice.ExternalPriceID,
		SuccessURL: successURL,
		CancelURL:  cancelURL,
		Metadata: map[string]string{
			"user_id": strconv.Itoa(userID),
			"plan_id": strconv.Itoa(targetPlan.ID),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create checkout session: %w", err)
	}

	return &CheckoutResult{
		CheckoutURL:            session.URL,
		SessionID:              session.ID,
		CreditAppliedCents:     creditCents,
		CreditAppliedFormatted: creditFormatted,
	}, nil
}

// CreatePortal creates a Stripe Billing Portal session for the given user.
// An optional returnURL overrides the default (FrontendURL).
func (s *Service) CreatePortal(userID int, returnURL *string) (*PortalResult, error) {
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoStripeCustomer
		}
		return nil, fmt.Errorf("get subscription: %w", err)
	}

	providerSub, err := s.subscriptionRepo.GetProviderSubscription(sub.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoStripeCustomer
		}
		return nil, fmt.Errorf("get provider subscription: %w", err)
	}

	if providerSub.ExternalCustomerID == nil || *providerSub.ExternalCustomerID == "" {
		return nil, ErrNoStripeCustomer
	}

	portalReturnURL := s.config.FrontendURL
	if returnURL != nil && *returnURL != "" {
		// Only allow return URLs that start with our frontend URL (prevent open redirect).
		if strings.HasPrefix(*returnURL, s.config.FrontendURL) {
			portalReturnURL = *returnURL
		}
	}

	portalURL, err := s.provider.CreatePortalSession(*providerSub.ExternalCustomerID, portalReturnURL)
	if err != nil {
		return nil, fmt.Errorf("create portal session: %w", err)
	}

	return &PortalResult{
		PortalURL: portalURL,
	}, nil
}

// GetUpgradePrice calculates the price breakdown for upgrading to the target
// plan, including any prorated credit from the user's current subscription.
func (s *Service) GetUpgradePrice(userID int, targetPlanID int) (*UpgradePriceInfo, error) {
	// Get the target plan.
	targetPlan, err := s.planRepo.GetByID(targetPlanID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get target plan: %w", err)
	}

	// Calculate credit from the current subscription if one exists.
	creditCents, creditFormatted, err := s.calculateCurrentCredit(userID)
	if err != nil {
		return nil, err
	}

	// Convert the target plan price to cents.
	originalCents := targetPlan.Price.Mul(decimal.NewFromInt(100)).IntPart()

	// Final price is the original minus credit, clamped to zero.
	finalCents := max(0, originalCents-creditCents)

	// Format prices for display.
	originalFormatted := s.creditCalc.FormatCreditForDisplay(targetPlan.Price, targetPlan.CurrencyCode)
	finalFormatted := s.creditCalc.FormatCreditForDisplay(
		decimal.NewFromInt(finalCents).Div(decimal.NewFromInt(100)),
		targetPlan.CurrencyCode,
	)

	return &UpgradePriceInfo{
		PlanID:                 targetPlan.ID,
		PlanName:               targetPlan.Name,
		PlanType:               targetPlan.PlanType,
		OriginalPriceCents:     originalCents,
		CreditCents:            creditCents,
		FinalPriceCents:        finalCents,
		OriginalPriceFormatted: originalFormatted,
		CreditFormatted:        creditFormatted,
		FinalPriceFormatted:    finalFormatted,
		CurrencyCode:           targetPlan.CurrencyCode,
	}, nil
}

// ChangePlan switches the user's subscription to the target plan. Upgrades
// (same or longer billing period) are applied immediately with prorations.
// Downgrades (shorter billing period) are scheduled to take effect at the
// end of the current billing period.
func (s *Service) ChangePlan(userID int, targetPlanID int) (*ChangePlanResult, error) {
	// Get the user's subscription.
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get subscription: %w", err)
		}
		return nil, fmt.Errorf("get subscription: %w", err)
	}

	// Get the provider subscription (must have an active Stripe subscription).
	providerSub, err := s.subscriptionRepo.GetProviderSubscription(sub.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoStripeSubscription
		}
		return nil, fmt.Errorf("get provider subscription: %w", err)
	}
	if providerSub.ExternalSubscriptionID == nil || *providerSub.ExternalSubscriptionID == "" {
		return nil, ErrNoStripeSubscription
	}

	// Get the target plan.
	targetPlan, err := s.planRepo.GetByID(targetPlanID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlanNotFound
		}
		return nil, fmt.Errorf("get target plan: %w", err)
	}

	// Get the Stripe price for the target plan.
	targetPlanPrice, err := s.planPriceRepo.GetByPlanAndProvider(targetPlanID, ProviderTypeStripe)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidPrice
		}
		return nil, fmt.Errorf("get target plan price: %w", err)
	}

	// Get the current plan for comparison.
	currentPlan, err := s.planRepo.GetByID(sub.PlanID)
	if err != nil {
		return nil, fmt.Errorf("get current plan: %w", err)
	}

	// Determine whether this is an upgrade or downgrade based on billing period duration.
	isUpgrade := true
	if targetPlan.BillingPeriod != nil && currentPlan.BillingPeriod != nil {
		if targetPlan.BillingPeriod.DurationDays < currentPlan.BillingPeriod.DurationDays {
			isUpgrade = false
		}
	}

	externalSubID := *providerSub.ExternalSubscriptionID

	if isUpgrade {
		return s.handleUpgrade(sub, providerSub, targetPlan, targetPlanPrice.ExternalPriceID, externalSubID)
	}

	return s.handleDowngrade(sub, providerSub, currentPlan, targetPlan, targetPlanPrice.ExternalPriceID, externalSubID)
}

// handleUpgrade performs an immediate plan upgrade with prorations.
func (s *Service) handleUpgrade(
	sub *models.Subscription,
	_ *models.PaymentProviderSubscription,
	targetPlan *models.SubscriptionPlan,
	stripePriceID string,
	externalSubID string,
) (*ChangePlanResult, error) {
	_, err := s.provider.UpdateSubscription(externalSubID, &UpdateSubscriptionParams{
		PriceID:           stripePriceID,
		ProrationBehavior: "create_prorations",
	})
	if err != nil {
		return nil, fmt.Errorf("update stripe subscription: %w", err)
	}

	// Update the local subscription record.
	sub.PlanID = targetPlan.ID
	if targetPlan.BillingPeriod != nil {
		sub.CurrentBillingPeriodID = &targetPlan.BillingPeriod.ID
	}
	if err := s.subscriptionRepo.UpdateSubscriptionFull(sub); err != nil {
		return nil, fmt.Errorf("update subscription: %w", err)
	}

	return &ChangePlanResult{
		Success:       true,
		IsUpgrade:     true,
		NewPlanName:   targetPlan.Name,
		EffectiveDate: time.Now().Format(time.RFC3339),
		Message:       "Plan upgraded successfully",
	}, nil
}

// handleDowngrade schedules a plan downgrade to take effect at the end of the
// current billing period using a Stripe subscription schedule.
func (s *Service) handleDowngrade(
	sub *models.Subscription,
	providerSub *models.PaymentProviderSubscription,
	currentPlan *models.SubscriptionPlan,
	targetPlan *models.SubscriptionPlan,
	stripePriceID string,
	externalSubID string,
) (*ChangePlanResult, error) {
	// Get the current Stripe subscription to find period end.
	stripeSubscription, err := s.provider.GetSubscription(externalSubID)
	if err != nil {
		return nil, fmt.Errorf("get stripe subscription: %w", err)
	}

	// Get the current plan's Stripe price ID for the first phase.
	currentPlanPrice, err := s.planPriceRepo.GetByPlanAndProvider(currentPlan.ID, ProviderTypeStripe)
	if err != nil {
		return nil, fmt.Errorf("get current plan price: %w", err)
	}

	// Build the schedule phases: keep current plan until period end, then switch.
	periodEnd := stripeSubscription.CurrentPeriodEnd.Unix()
	phases := []SchedulePhase{
		{
			StartDate: time.Now().Unix(),
			EndDate:   periodEnd,
			PriceID:   currentPlanPrice.ExternalPriceID,
		},
		{
			StartDate: periodEnd,
			EndDate:   0, // open-ended
			PriceID:   stripePriceID,
		},
	}

	scheduleID, err := s.provider.CreateSubscriptionSchedule(externalSubID, phases)
	if err != nil {
		return nil, fmt.Errorf("create subscription schedule: %w", err)
	}

	// Store the schedule ID and pending plan in our records.
	providerSub.ExternalScheduleID = &scheduleID
	if err := s.subscriptionRepo.UpdateProviderSubscription(providerSub); err != nil {
		return nil, fmt.Errorf("update provider subscription: %w", err)
	}

	sub.PendingPlanID = &targetPlan.ID
	if err := s.subscriptionRepo.UpdateSubscriptionFull(sub); err != nil {
		return nil, fmt.Errorf("update subscription: %w", err)
	}

	effectiveDate := stripeSubscription.CurrentPeriodEnd.Format(time.RFC3339)

	return &ChangePlanResult{
		Success:       true,
		IsUpgrade:     false,
		NewPlanName:   targetPlan.Name,
		EffectiveDate: effectiveDate,
		Message:       "Plan downgrade scheduled",
		ScheduleID:    &scheduleID,
	}, nil
}

// CancelScheduledChange cancels any pending plan change (downgrade schedule
// or pending cancellation) for the given user.
func (s *Service) CancelScheduledChange(userID int) error {
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("get subscription: %w", err)
	}

	providerSub, err := s.subscriptionRepo.GetProviderSubscription(sub.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrScheduleNotFound
		}
		return fmt.Errorf("get provider subscription: %w", err)
	}

	if providerSub.ExternalScheduleID == nil || *providerSub.ExternalScheduleID == "" {
		return ErrScheduleNotFound
	}

	// Cancel the schedule in Stripe.
	if err := s.provider.CancelSubscriptionSchedule(*providerSub.ExternalScheduleID); err != nil {
		return fmt.Errorf("cancel subscription schedule: %w", err)
	}

	// Clear the schedule ID.
	providerSub.ExternalScheduleID = nil

	// Clear pending downgrade fields.
	sub.PendingPlanID = nil
	sub.PendingDowngradeAccountIDs = nil
	sub.PendingDowngradeBudgetID = nil

	// If the subscription was being canceled (downgrade to free), re-enable it.
	if sub.CanceledAt != nil {
		if providerSub.ExternalSubscriptionID != nil && *providerSub.ExternalSubscriptionID != "" {
			if err := s.provider.ReenableSubscription(*providerSub.ExternalSubscriptionID); err != nil {
				return fmt.Errorf("reenable subscription: %w", err)
			}
		}
		sub.CanceledAt = nil
		sub.AutoRenew = true
	}

	// Persist changes.
	if err := s.subscriptionRepo.UpdateProviderSubscription(providerSub); err != nil {
		return fmt.Errorf("update provider subscription: %w", err)
	}
	if err := s.subscriptionRepo.UpdateSubscriptionFull(sub); err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}

	return nil
}

// GetPaymentStatus returns the payment and subscription status for the given user.
func (s *Service) GetPaymentStatus(userID int) (*PaymentStatus, error) {
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &PaymentStatus{
				CurrentPlanType: "free",
			}, nil
		}
		return nil, fmt.Errorf("get subscription: %w", err)
	}

	providerSub, err := s.subscriptionRepo.GetProviderSubscription(sub.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &PaymentStatus{
				CurrentPlanType: sub.PlanType,
			}, nil
		}
		return nil, fmt.Errorf("get provider subscription: %w", err)
	}

	hasCustomer := providerSub.ExternalCustomerID != nil && *providerSub.ExternalCustomerID != ""
	subActive := providerSub.ExternalSubscriptionID != nil && *providerSub.ExternalSubscriptionID != ""

	return &PaymentStatus{
		HasStripeCustomer:        hasCustomer,
		StripeSubscriptionActive: subActive,
		StripeSubscriptionID:     providerSub.ExternalSubscriptionID,
		CurrentPlanType:          sub.PlanType,
		LastPaymentFailed:        providerSub.LastPaymentFailed,
	}, nil
}

// GetSessionStatus retrieves the status of a Stripe Checkout session.
// It verifies that the session belongs to the given user by checking the
// user_id metadata field stored during checkout creation.
func (s *Service) GetSessionStatus(userID int, sessionID string) (*SessionStatus, error) {
	session, err := s.provider.GetCheckoutSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("get checkout session: %w", err)
	}

	// Verify the session belongs to the authenticated user via metadata.
	if metaUserID, ok := session.Metadata["user_id"]; !ok || metaUserID != strconv.Itoa(userID) {
		s.logger.Warn("session ownership mismatch",
			"sessionID", sessionID,
			"requestUserID", userID,
			"metadataUserID", session.Metadata["user_id"],
		)
		return nil, ErrSessionNotFound
	}

	return &SessionStatus{
		SessionID:      session.ID,
		Status:         session.Status,
		PaymentStatus:  session.PaymentStatus,
		CustomerID:     session.CustomerID,
		SubscriptionID: session.SubscriptionID,
		AmountTotal:    session.AmountTotal,
		Currency:       session.Currency,
	}, nil
}

// calculateCurrentCredit computes the prorated credit (in cents) from the
// user's current subscription. Returns zero values when no credit applies.
func (s *Service) calculateCurrentCredit(userID int) (creditCents int64, creditFormatted string, err error) {
	sub, err := s.subscriptionRepo.GetByUserID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", nil
		}
		return 0, "", fmt.Errorf("get subscription for credit: %w", err)
	}

	if sub.CurrentBillingPeriodID == nil || sub.ExpiresAt == nil || !sub.ExpiresAt.After(time.Now()) {
		return 0, "", nil
	}

	currentPlan, err := s.planRepo.GetByID(sub.PlanID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", nil
		}
		return 0, "", fmt.Errorf("get current plan: %w", err)
	}

	if currentPlan.BillingPeriod == nil {
		return 0, "", nil
	}

	remainingCredit := s.creditCalc.CalculateRemainingCredit(
		currentPlan.Price,
		currentPlan.BillingPeriod.DurationDays,
		*sub.ExpiresAt,
	)

	creditCents = remainingCredit.Mul(decimal.NewFromInt(100)).IntPart()
	if creditCents > 0 {
		creditFormatted = s.creditCalc.FormatCreditForDisplay(remainingCredit, currentPlan.CurrencyCode)
	}

	return creditCents, creditFormatted, nil
}
