package payment

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/go-budget/backend/internal/models"
)

// WebhookHandler processes incoming payment provider webhook events. It verifies
// event signatures, routes events to the appropriate handler, and always returns
// a result (never propagating errors to the caller) so the provider receives a
// 200 response.
type WebhookHandler struct {
	provider      PaymentProvider
	subRepo       SubscriptionRepository
	planRepo      SubscriptionPlanRepository
	planPriceRepo PlanPriceRepository
	logger        *slog.Logger
}

// NewWebhookHandler creates a new WebhookHandler with the given dependencies.
func NewWebhookHandler(
	provider PaymentProvider,
	subRepo SubscriptionRepository,
	planRepo SubscriptionPlanRepository,
	planPriceRepo PlanPriceRepository,
	logger *slog.Logger,
) *WebhookHandler {
	return &WebhookHandler{
		provider:      provider,
		subRepo:       subRepo,
		planRepo:      planRepo,
		planPriceRepo: planPriceRepo,
		logger:        logger,
	}
}

// stripeEvent represents the top-level structure of a Stripe webhook event.
type stripeEvent struct {
	ID   string    `json:"id"`
	Type string    `json:"type"`
	Data eventData `json:"data"`
}

// eventData holds the raw object payload within a Stripe event.
type eventData struct {
	Object json.RawMessage `json:"object"`
}

// HandleWebhook verifies the webhook signature, parses the event, and routes
// it to the appropriate handler. It always returns a WebhookResult and never
// propagates errors — failures are logged internally.
func (h *WebhookHandler) HandleWebhook(payload []byte, signature string) *WebhookResult {
	// Step 1: Verify signature.
	_, err := h.provider.VerifyWebhookSignature(payload, signature)
	if err != nil {
		h.logger.Error("webhook signature verification failed", "error", err)
		return &WebhookResult{
			Success: false,
			Message: "signature verification failed",
		}
	}

	// Step 2: Parse the verified event to extract type and object data.
	var event stripeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		h.logger.Error("failed to parse webhook event JSON", "error", err)
		return &WebhookResult{
			Success: false,
			Message: "failed to parse event",
		}
	}

	h.logger.Info("processing webhook event", "eventID", event.ID, "eventType", event.Type)

	// Step 3: Route to the appropriate handler based on event type.
	var handlerErr error
	switch event.Type {
	case "checkout.session.completed":
		handlerErr = h.handleCheckoutSessionCompleted(event.Data.Object)
	case "customer.subscription.updated":
		handlerErr = h.handleSubscriptionUpdated(event.Data.Object)
	case "customer.subscription.deleted":
		handlerErr = h.handleSubscriptionDeleted(event.Data.Object)
	case "invoice.paid":
		handlerErr = h.handleInvoicePaid(event.Data.Object)
	case "invoice.payment_failed":
		handlerErr = h.handleInvoicePaymentFailed(event.Data.Object)
	case "subscription_schedule.completed":
		handlerErr = h.handleScheduleCompleted(event.Data.Object)
	case "subscription_schedule.released":
		handlerErr = h.handleScheduleReleased(event.Data.Object)
	default:
		h.logger.Info("unhandled webhook event type, acknowledging", "eventType", event.Type)
	}

	if handlerErr != nil {
		h.logger.Error("webhook handler error",
			"eventType", event.Type,
			"eventID", event.ID,
			"error", handlerErr,
		)
	}

	return &WebhookResult{
		EventType: event.Type,
		Success:   true,
		Message:   "event processed",
	}
}

// checkoutSessionObject holds the fields we need from a Stripe checkout.session object.
type checkoutSessionObject struct {
	Customer     string                  `json:"customer"`
	Subscription string                  `json:"subscription"`
	Metadata     checkoutSessionMetadata `json:"metadata"`
}

// checkoutSessionMetadata holds the metadata fields set during checkout creation.
type checkoutSessionMetadata struct {
	UserID string `json:"user_id"`
	PlanID string `json:"plan_id"`
}

// handleCheckoutSessionCompleted processes the checkout.session.completed event.
// It links the Stripe customer and subscription to our internal subscription,
// activates the target plan, and clears any trial or pending downgrade state.
func (h *WebhookHandler) handleCheckoutSessionCompleted(objectData []byte) error {
	var session checkoutSessionObject
	if err := json.Unmarshal(objectData, &session); err != nil {
		return fmt.Errorf("parse checkout session object: %w", err)
	}

	userID, err := strconv.Atoi(session.Metadata.UserID)
	if err != nil {
		return fmt.Errorf("invalid user_id in checkout session metadata: %w", err)
	}

	planID, err := strconv.Atoi(session.Metadata.PlanID)
	if err != nil {
		return fmt.Errorf("invalid plan_id in checkout session metadata: %w", err)
	}

	// Get the user's subscription.
	sub, err := h.subRepo.GetByUserID(userID)
	if err != nil {
		return fmt.Errorf("get subscription for user %d: %w", userID, err)
	}

	// Get the target plan.
	targetPlan, err := h.planRepo.GetByID(planID)
	if err != nil {
		return fmt.Errorf("get target plan %d: %w", planID, err)
	}

	// Get Stripe subscription details for current_period_end.
	stripeSub, err := h.provider.GetSubscription(session.Subscription)
	if err != nil {
		return fmt.Errorf("get stripe subscription details: %w", err)
	}

	// Get or create the provider subscription record.
	providerSub, err := h.subRepo.GetProviderSubscription(sub.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("get provider subscription: %w", err)
	}

	customerID := session.Customer
	subscriptionID := session.Subscription

	if providerSub != nil {
		// Update existing provider subscription with Stripe IDs.
		providerSub.ExternalCustomerID = &customerID
		providerSub.ExternalSubscriptionID = &subscriptionID
		if err := h.subRepo.UpdateProviderSubscription(providerSub); err != nil {
			return fmt.Errorf("update provider subscription: %w", err)
		}
	} else {
		// Create new provider subscription.
		pps := &models.PaymentProviderSubscription{
			SubscriptionID:         sub.ID,
			ProviderType:           ProviderTypeStripe,
			ExternalCustomerID:     &customerID,
			ExternalSubscriptionID: &subscriptionID,
		}
		if err := h.subRepo.CreateProviderSubscription(pps); err != nil {
			return fmt.Errorf("create provider subscription: %w", err)
		}
	}

	// Update the subscription with the new plan details.
	now := time.Now()
	sub.PlanID = planID
	sub.PlanType = targetPlan.PlanType
	sub.SubscribedAt = &now
	sub.ExpiresAt = &stripeSub.CurrentPeriodEnd
	sub.IsActive = true
	sub.AutoRenew = true
	sub.HasStripeSubscription = true

	// Clear trial fields.
	sub.TrialStartedAt = nil
	sub.TrialEndsAt = nil

	// Clear pending downgrade fields.
	sub.PendingPlanID = nil
	sub.PendingDowngradeAccountIDs = nil
	sub.PendingDowngradeBudgetID = nil

	// Set billing period if the target plan has one.
	if targetPlan.BillingPeriod != nil {
		bpID := targetPlan.BillingPeriod.ID
		sub.CurrentBillingPeriodID = &bpID
	}

	if err := h.subRepo.UpdateSubscriptionFull(sub); err != nil {
		return fmt.Errorf("update subscription after checkout: %w", err)
	}

	h.logger.Info("checkout session completed successfully",
		"userID", userID,
		"planID", planID,
		"stripeSubscriptionID", subscriptionID,
	)

	return nil
}

// subscriptionObject holds the fields we need from a Stripe subscription object.
type subscriptionObject struct {
	ID                string                `json:"id"`
	Status            string                `json:"status"`
	CurrentPeriodEnd  int64                 `json:"current_period_end"`
	CancelAtPeriodEnd bool                  `json:"cancel_at_period_end"`
	Items             subscriptionItemsData `json:"items"`
}

// subscriptionItemsData wraps the items array in a Stripe subscription object.
type subscriptionItemsData struct {
	Data []subscriptionItemData `json:"data"`
}

// subscriptionItemData holds a single item from a Stripe subscription.
type subscriptionItemData struct {
	Price subscriptionPriceData `json:"price"`
}

// subscriptionPriceData holds the price ID from a Stripe subscription item.
type subscriptionPriceData struct {
	ID string `json:"id"`
}

// handleSubscriptionUpdated processes the customer.subscription.updated event.
// It synchronizes plan changes, cancellation state, and period end from Stripe
// into our internal subscription records.
func (h *WebhookHandler) handleSubscriptionUpdated(objectData []byte) error {
	var subObj subscriptionObject
	if err := json.Unmarshal(objectData, &subObj); err != nil {
		return fmt.Errorf("parse subscription object: %w", err)
	}

	stripeSubID := subObj.ID

	// Find our provider subscription by the external subscription ID.
	providerSub, err := h.subRepo.GetProviderSubscriptionByExternalID(stripeSubID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Warn("received subscription.updated for unknown subscription",
				"stripeSubscriptionID", stripeSubID)
			return nil
		}
		return fmt.Errorf("lookup provider subscription by external ID: %w", err)
	}

	// Get our internal subscription using the provider subscription's reference.
	sub, err := h.subRepo.GetByID(providerSub.SubscriptionID)
	if err != nil {
		return fmt.Errorf("get subscription by ID: %w", err)
	}

	// Check if the price changed and update the plan accordingly.
	var currentPriceID string
	if len(subObj.Items.Data) > 0 {
		currentPriceID = subObj.Items.Data[0].Price.ID
	}

	if currentPriceID != "" {
		planPrice, err := h.planPriceRepo.GetByExternalPriceID(currentPriceID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				h.logger.Error("failed to lookup plan price by external price ID",
					"externalPriceID", currentPriceID, "error", err)
			}
			// If the price is not found in our system, we skip updating the plan
			// but continue with other field updates.
		} else {
			plan, err := h.planRepo.GetByID(planPrice.PlanID)
			if err != nil {
				h.logger.Error("failed to get plan for price",
					"planID", planPrice.PlanID, "error", err)
			} else {
				sub.PlanID = plan.ID
				sub.PlanType = plan.PlanType
				if plan.BillingPeriod != nil {
					bpID := plan.BillingPeriod.ID
					sub.CurrentBillingPeriodID = &bpID
				}
			}
		}
	}

	// Update expiration from Stripe's current_period_end.
	expiresAt := time.Unix(subObj.CurrentPeriodEnd, 0)
	sub.ExpiresAt = &expiresAt

	// Handle cancellation state.
	if subObj.CancelAtPeriodEnd {
		now := time.Now()
		sub.CanceledAt = &now
		sub.AutoRenew = false
	} else {
		sub.CanceledAt = nil
		sub.AutoRenew = true
	}

	// Map Stripe status to our IsActive flag.
	switch subObj.Status {
	case "active", "past_due":
		sub.IsActive = true
	case "canceled", "unpaid":
		sub.IsActive = false
	}

	if err := h.subRepo.UpdateSubscriptionFull(sub); err != nil {
		return fmt.Errorf("update subscription from subscription.updated event: %w", err)
	}

	h.logger.Info("subscription updated successfully",
		"subscriptionID", sub.ID,
		"stripeSubscriptionID", stripeSubID,
		"status", subObj.Status,
		"cancelAtPeriodEnd", subObj.CancelAtPeriodEnd,
	)

	return nil
}

// deletedSubscriptionObject holds the fields we need from a Stripe subscription
// object in a customer.subscription.deleted event.
type deletedSubscriptionObject struct {
	ID string `json:"id"`
}

// handleSubscriptionDeleted processes the customer.subscription.deleted event.
// It downgrades the user to the free plan and clears all Stripe-related data
// from the provider subscription record.
func (h *WebhookHandler) handleSubscriptionDeleted(objectData []byte) error {
	var subObj deletedSubscriptionObject
	if err := json.Unmarshal(objectData, &subObj); err != nil {
		return fmt.Errorf("parse subscription object: %w", err)
	}

	stripeSubID := subObj.ID

	// Find provider subscription by external Stripe subscription ID.
	providerSub, err := h.subRepo.GetProviderSubscriptionByExternalID(stripeSubID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Warn("received subscription.deleted for unknown subscription",
				"stripeSubscriptionID", stripeSubID)
			return nil
		}
		return fmt.Errorf("lookup provider subscription: %w", err)
	}

	// Get our internal subscription.
	sub, err := h.subRepo.GetByID(providerSub.SubscriptionID)
	if err != nil {
		return fmt.Errorf("get subscription by ID: %w", err)
	}

	// Get the free plan for downgrade.
	freePlan, err := h.planRepo.GetFreePlan()
	if err != nil {
		return fmt.Errorf("get free plan: %w", err)
	}

	// Downgrade subscription to free plan.
	now := time.Now()
	sub.PlanID = freePlan.ID
	sub.IsActive = true
	sub.AutoRenew = false
	sub.CanceledAt = &now
	sub.ExpiresAt = nil
	sub.CurrentBillingPeriodID = nil

	// Clear pending fields.
	sub.PendingPlanID = nil
	sub.PendingDowngradeAccountIDs = nil
	sub.PendingDowngradeBudgetID = nil

	if err := h.subRepo.UpdateSubscriptionFull(sub); err != nil {
		return fmt.Errorf("update subscription after deletion: %w", err)
	}

	// Clear Stripe data from provider subscription.
	providerSub.ExternalSubscriptionID = nil
	providerSub.ExternalScheduleID = nil

	if err := h.subRepo.UpdateProviderSubscription(providerSub); err != nil {
		return fmt.Errorf("update provider subscription after deletion: %w", err)
	}

	h.logger.Info("subscription deleted, downgraded to free plan",
		"subscriptionID", sub.ID,
		"stripeSubscriptionID", stripeSubID,
		"freePlanID", freePlan.ID,
	)

	return nil
}

// invoiceObject holds the fields we need from a Stripe invoice object.
type invoiceObject struct {
	Customer     string `json:"customer"`
	Subscription string `json:"subscription"`
}

// handleInvoicePaid processes the invoice.paid event.
// It clears the LastPaymentFailed flag on the provider subscription.
func (h *WebhookHandler) handleInvoicePaid(objectData []byte) error {
	var inv invoiceObject
	if err := json.Unmarshal(objectData, &inv); err != nil {
		return fmt.Errorf("parse invoice object: %w", err)
	}

	// If the invoice has no subscription (e.g. one-off payment), skip.
	if inv.Subscription == "" {
		h.logger.Info("invoice.paid has no subscription, skipping",
			"customerID", inv.Customer)
		return nil
	}

	// Find provider subscription by external subscription ID.
	providerSub, err := h.subRepo.GetProviderSubscriptionByExternalID(inv.Subscription)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Warn("received invoice.paid for unknown subscription",
				"stripeSubscriptionID", inv.Subscription,
				"customerID", inv.Customer)
			return nil
		}
		return fmt.Errorf("lookup provider subscription: %w", err)
	}

	providerSub.LastPaymentFailed = false
	if err := h.subRepo.UpdateProviderSubscription(providerSub); err != nil {
		return fmt.Errorf("update provider subscription after invoice.paid: %w", err)
	}

	h.logger.Info("invoice paid, cleared payment failure flag",
		"subscriptionID", providerSub.SubscriptionID,
		"stripeSubscriptionID", inv.Subscription,
	)

	return nil
}

// handleInvoicePaymentFailed processes the invoice.payment_failed event.
// It sets the LastPaymentFailed flag on the provider subscription.
func (h *WebhookHandler) handleInvoicePaymentFailed(objectData []byte) error {
	var inv invoiceObject
	if err := json.Unmarshal(objectData, &inv); err != nil {
		return fmt.Errorf("parse invoice object: %w", err)
	}

	// If the invoice has no subscription, skip.
	if inv.Subscription == "" {
		h.logger.Info("invoice.payment_failed has no subscription, skipping",
			"customerID", inv.Customer)
		return nil
	}

	// Find provider subscription by external subscription ID.
	providerSub, err := h.subRepo.GetProviderSubscriptionByExternalID(inv.Subscription)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.logger.Warn("received invoice.payment_failed for unknown subscription",
				"stripeSubscriptionID", inv.Subscription)
			return nil
		}
		return fmt.Errorf("lookup provider subscription: %w", err)
	}

	providerSub.LastPaymentFailed = true
	if err := h.subRepo.UpdateProviderSubscription(providerSub); err != nil {
		return fmt.Errorf("update provider subscription after invoice.payment_failed: %w", err)
	}

	h.logger.Info("invoice payment failed, set failure flag",
		"subscriptionID", providerSub.SubscriptionID,
		"stripeSubscriptionID", inv.Subscription,
	)

	return nil
}

// scheduleObject holds the fields we need from a Stripe subscription_schedule object.
type scheduleObject struct {
	ID string `json:"id"`
}

// handleScheduleCompleted processes the subscription_schedule.completed event.
// When a schedule completes, the pending plan transition takes effect: the
// subscription is moved to the pending plan and the schedule reference is cleared.
func (h *WebhookHandler) handleScheduleCompleted(objectData []byte) error {
	var schedObj scheduleObject
	if err := json.Unmarshal(objectData, &schedObj); err != nil {
		return fmt.Errorf("parse schedule object: %w", err)
	}

	scheduleID := schedObj.ID

	// Find provider subscription by the Stripe schedule ID.
	providerSub, err := h.subRepo.GetProviderSubscriptionByScheduleID(scheduleID)
	if err != nil {
		h.logger.Warn("received subscription_schedule.completed for unknown schedule",
			"stripeScheduleID", scheduleID, "error", err)
		return nil
	}

	// Get our internal subscription.
	sub, err := h.subRepo.GetByID(providerSub.SubscriptionID)
	if err != nil {
		return fmt.Errorf("get subscription by ID: %w", err)
	}

	// If the subscription has a pending plan, apply it.
	if sub.PendingPlanID != nil {
		pendingPlan, err := h.planRepo.GetByID(*sub.PendingPlanID)
		if err != nil {
			return fmt.Errorf("get pending plan: %w", err)
		}

		sub.PlanID = pendingPlan.ID
		if pendingPlan.BillingPeriod != nil {
			bpID := pendingPlan.BillingPeriod.ID
			sub.CurrentBillingPeriodID = &bpID
		}
		sub.PendingPlanID = nil
	}

	// Clear schedule ID from provider subscription.
	providerSub.ExternalScheduleID = nil

	if err := h.subRepo.UpdateSubscriptionFull(sub); err != nil {
		return fmt.Errorf("update subscription after schedule completed: %w", err)
	}

	if err := h.subRepo.UpdateProviderSubscription(providerSub); err != nil {
		return fmt.Errorf("update provider subscription after schedule completed: %w", err)
	}

	h.logger.Info("subscription schedule completed",
		"subscriptionID", sub.ID,
		"stripeScheduleID", scheduleID,
	)

	return nil
}

// handleScheduleReleased processes the subscription_schedule.released event.
// When a schedule is released (canceled before completion), we only clear
// the schedule reference from the provider subscription.
func (h *WebhookHandler) handleScheduleReleased(objectData []byte) error {
	var schedObj scheduleObject
	if err := json.Unmarshal(objectData, &schedObj); err != nil {
		return fmt.Errorf("parse schedule object: %w", err)
	}

	scheduleID := schedObj.ID

	// Find provider subscription by the Stripe schedule ID.
	providerSub, err := h.subRepo.GetProviderSubscriptionByScheduleID(scheduleID)
	if err != nil {
		h.logger.Warn("received subscription_schedule.released for unknown schedule",
			"stripeScheduleID", scheduleID, "error", err)
		return nil
	}

	// Clear schedule ID from provider subscription.
	providerSub.ExternalScheduleID = nil

	if err := h.subRepo.UpdateProviderSubscription(providerSub); err != nil {
		return fmt.Errorf("update provider subscription after schedule released: %w", err)
	}

	h.logger.Info("subscription schedule released",
		"subscriptionID", providerSub.SubscriptionID,
		"stripeScheduleID", scheduleID,
	)

	return nil
}
