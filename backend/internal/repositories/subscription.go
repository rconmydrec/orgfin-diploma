package repositories

import (
	"database/sql"
	"encoding/json"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
)

type SubscriptionRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewSubscriptionRepository(db *sql.DB, logger *slog.Logger) *SubscriptionRepositoryImpl {
	return &SubscriptionRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

// GetByID returns a subscription by its primary key ID.
func (r *SubscriptionRepositoryImpl) GetByID(id int) (*models.Subscription, error) {
	query := `
		SELECT s.id, s.user_id, s.plan_id, LOWER(sp.plan_type::text),
		       s.current_billing_period_id,
		       s.trial_started_at, s.trial_ends_at, s.subscribed_at, s.expires_at,
		       s.auto_renew, s.canceled_at,
		       s.is_active, s.pending_plan_id,
		       s.pending_downgrade_account_ids, s.pending_downgrade_budget_id,
		       CASE WHEN pps.external_subscription_id IS NOT NULL THEN true ELSE false END as has_stripe,
		       s.created_at, s.updated_at
		FROM subscriptions s
		JOIN subscription_plans sp ON s.plan_id = sp.id
		LEFT JOIN payment_provider_subscriptions pps ON s.id = pps.subscription_id
		WHERE s.id = $1
	`

	sub := &models.Subscription{}
	var pendingAccountIDsJSON []byte
	err := r.db.QueryRow(query, id).Scan(
		&sub.ID, &sub.UserID, &sub.PlanID, &sub.PlanType,
		&sub.CurrentBillingPeriodID,
		&sub.TrialStartedAt, &sub.TrialEndsAt, &sub.SubscribedAt, &sub.ExpiresAt,
		&sub.AutoRenew, &sub.CanceledAt,
		&sub.IsActive, &sub.PendingPlanID,
		&pendingAccountIDsJSON, &sub.PendingDowngradeBudgetID,
		&sub.HasStripeSubscription,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(pendingAccountIDsJSON) > 0 {
		if err := json.Unmarshal(pendingAccountIDsJSON, &sub.PendingDowngradeAccountIDs); err != nil {
			r.logger.Error("failed to unmarshal pending_downgrade_account_ids", "error", err)
		}
	}

	return sub, nil
}

func (r *SubscriptionRepositoryImpl) GetByUserID(userID int) (*models.Subscription, error) {
	query := `
		SELECT s.id, s.user_id, s.plan_id, LOWER(sp.plan_type::text),
		       s.current_billing_period_id,
		       s.trial_started_at, s.trial_ends_at, s.subscribed_at, s.expires_at,
		       s.auto_renew, s.canceled_at,
		       s.is_active, s.pending_plan_id,
		       s.pending_downgrade_account_ids, s.pending_downgrade_budget_id,
		       CASE WHEN pps.external_subscription_id IS NOT NULL THEN true ELSE false END as has_stripe,
		       s.created_at, s.updated_at
		FROM subscriptions s
		JOIN subscription_plans sp ON s.plan_id = sp.id
		LEFT JOIN payment_provider_subscriptions pps ON s.id = pps.subscription_id
		WHERE s.user_id = $1
	`

	sub := &models.Subscription{}
	var pendingAccountIDsJSON []byte
	err := r.db.QueryRow(query, userID).Scan(
		&sub.ID, &sub.UserID, &sub.PlanID, &sub.PlanType,
		&sub.CurrentBillingPeriodID,
		&sub.TrialStartedAt, &sub.TrialEndsAt, &sub.SubscribedAt, &sub.ExpiresAt,
		&sub.AutoRenew, &sub.CanceledAt,
		&sub.IsActive, &sub.PendingPlanID,
		&pendingAccountIDsJSON, &sub.PendingDowngradeBudgetID,
		&sub.HasStripeSubscription,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(pendingAccountIDsJSON) > 0 {
		if err := json.Unmarshal(pendingAccountIDsJSON, &sub.PendingDowngradeAccountIDs); err != nil {
			r.logger.Error("failed to unmarshal pending_downgrade_account_ids", "error", err)
		}
	}

	return sub, nil
}

func (r *SubscriptionRepositoryImpl) Create(sub *models.Subscription) error {
	pendingAccountIDsJSON, err := marshalJSONBIntSlice(sub.PendingDowngradeAccountIDs)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO subscriptions (user_id, plan_id, current_billing_period_id,
		                          trial_started_at, trial_ends_at,
		                          subscribed_at, expires_at,
		                          auto_renew, canceled_at,
		                          is_active, pending_plan_id,
		                          pending_downgrade_account_ids, pending_downgrade_budget_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at
	`

	err = r.db.QueryRow(
		query,
		sub.UserID, sub.PlanID, sub.CurrentBillingPeriodID,
		sub.TrialStartedAt, sub.TrialEndsAt,
		sub.SubscribedAt, sub.ExpiresAt,
		sub.AutoRenew, sub.CanceledAt,
		sub.IsActive, sub.PendingPlanID,
		pendingAccountIDsJSON, sub.PendingDowngradeBudgetID,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)

	return err
}

func (r *SubscriptionRepositoryImpl) Update(sub *models.Subscription) error {
	query := `
		UPDATE subscriptions
		SET plan_id = $1, subscribed_at = $2, expires_at = $3,
		    is_active = $4, pending_plan_id = $5, updated_at = NOW()
		WHERE id = $6
	`

	_, err := r.db.Exec(query,
		sub.PlanID, sub.SubscribedAt, sub.ExpiresAt,
		sub.IsActive, sub.PendingPlanID, sub.ID,
	)
	return err
}

// UpdateSubscriptionFull updates ALL mutable subscription fields including billing period,
// auto-renew, canceled_at, and pending downgrade fields.
// Unlike Update, which only modifies a subset, this method persists every mutable column.
// Note: HasStripeSubscription is computed from the payment_provider_subscriptions table
// via a JOIN and is not stored directly on the subscriptions table.
func (r *SubscriptionRepositoryImpl) UpdateSubscriptionFull(sub *models.Subscription) error {
	pendingAccountIDsJSON, err := marshalJSONBIntSlice(sub.PendingDowngradeAccountIDs)
	if err != nil {
		return err
	}

	query := `
		UPDATE subscriptions
		SET plan_id = $1, current_billing_period_id = $2,
		    trial_started_at = $3, trial_ends_at = $4,
		    subscribed_at = $5, expires_at = $6,
		    auto_renew = $7, canceled_at = $8,
		    is_active = $9, pending_plan_id = $10,
		    pending_downgrade_account_ids = $11, pending_downgrade_budget_id = $12,
		    updated_at = NOW()
		WHERE id = $13
	`

	_, err = r.db.Exec(query,
		sub.PlanID, sub.CurrentBillingPeriodID,
		sub.TrialStartedAt, sub.TrialEndsAt,
		sub.SubscribedAt, sub.ExpiresAt,
		sub.AutoRenew, sub.CanceledAt,
		sub.IsActive, sub.PendingPlanID,
		pendingAccountIDsJSON, sub.PendingDowngradeBudgetID,
		sub.ID,
	)
	return err
}

// GetExpiredSubscriptions returns all active subscriptions that have passed their expiration date.
func (r *SubscriptionRepositoryImpl) GetExpiredSubscriptions() ([]*models.Subscription, error) {
	query := `
		SELECT s.id, s.user_id, s.plan_id, LOWER(sp.plan_type::text),
		       s.current_billing_period_id,
		       s.trial_started_at, s.trial_ends_at, s.subscribed_at, s.expires_at,
		       s.auto_renew, s.canceled_at,
		       s.is_active, s.pending_plan_id,
		       s.pending_downgrade_account_ids, s.pending_downgrade_budget_id,
		       CASE WHEN pps.external_subscription_id IS NOT NULL THEN true ELSE false END as has_stripe,
		       s.created_at, s.updated_at
		FROM subscriptions s
		JOIN subscription_plans sp ON s.plan_id = sp.id
		LEFT JOIN payment_provider_subscriptions pps ON s.id = pps.subscription_id
		WHERE s.is_active = true AND s.expires_at IS NOT NULL AND s.expires_at < NOW()
	`

	return r.scanSubscriptionRows(query)
}

// GetPendingDowngrades returns all subscriptions that have a pending downgrade plan.
func (r *SubscriptionRepositoryImpl) GetPendingDowngrades() ([]*models.Subscription, error) {
	query := `
		SELECT s.id, s.user_id, s.plan_id, LOWER(sp.plan_type::text),
		       s.current_billing_period_id,
		       s.trial_started_at, s.trial_ends_at, s.subscribed_at, s.expires_at,
		       s.auto_renew, s.canceled_at,
		       s.is_active, s.pending_plan_id,
		       s.pending_downgrade_account_ids, s.pending_downgrade_budget_id,
		       CASE WHEN pps.external_subscription_id IS NOT NULL THEN true ELSE false END as has_stripe,
		       s.created_at, s.updated_at
		FROM subscriptions s
		JOIN subscription_plans sp ON s.plan_id = sp.id
		LEFT JOIN payment_provider_subscriptions pps ON s.id = pps.subscription_id
		WHERE s.pending_plan_id IS NOT NULL
		  AND (s.expires_at IS NULL OR s.expires_at <= NOW())
	`

	return r.scanSubscriptionRows(query)
}

// GetByPlanType returns all subscriptions with the given plan type
// (joins subscription_plans to filter by plan_type).
func (r *SubscriptionRepositoryImpl) GetByPlanType(planType string) ([]*models.Subscription, error) {
	query := `
		SELECT s.id, s.user_id, s.plan_id, LOWER(sp.plan_type::text),
		       s.current_billing_period_id,
		       s.trial_started_at, s.trial_ends_at, s.subscribed_at, s.expires_at,
		       s.auto_renew, s.canceled_at,
		       s.is_active, s.pending_plan_id,
		       s.pending_downgrade_account_ids, s.pending_downgrade_budget_id,
		       CASE WHEN pps.external_subscription_id IS NOT NULL THEN true ELSE false END as has_stripe,
		       s.created_at, s.updated_at
		FROM subscriptions s
		JOIN subscription_plans sp ON s.plan_id = sp.id
		LEFT JOIN payment_provider_subscriptions pps ON s.id = pps.subscription_id
		WHERE LOWER(sp.plan_type) = LOWER($1)
	`

	return r.scanSubscriptionRowsWithArgs(query, planType)
}

// scanSubscriptionRows executes a query that returns subscription rows (no args)
// and scans the results into a Subscription slice.
func (r *SubscriptionRepositoryImpl) scanSubscriptionRows(query string) ([]*models.Subscription, error) {
	return r.scanSubscriptionRowsWithArgs(query)
}

// scanSubscriptionRowsWithArgs executes a query with arguments that returns subscription rows
// and scans the results into a Subscription slice.
func (r *SubscriptionRepositoryImpl) scanSubscriptionRowsWithArgs(query string, args ...any) ([]*models.Subscription, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*models.Subscription
	for rows.Next() {
		sub := &models.Subscription{}
		var pendingAccountIDsJSON []byte
		err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.PlanID, &sub.PlanType,
			&sub.CurrentBillingPeriodID,
			&sub.TrialStartedAt, &sub.TrialEndsAt, &sub.SubscribedAt, &sub.ExpiresAt,
			&sub.AutoRenew, &sub.CanceledAt,
			&sub.IsActive, &sub.PendingPlanID,
			&pendingAccountIDsJSON, &sub.PendingDowngradeBudgetID,
			&sub.HasStripeSubscription,
			&sub.CreatedAt, &sub.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if len(pendingAccountIDsJSON) > 0 {
			if err := json.Unmarshal(pendingAccountIDsJSON, &sub.PendingDowngradeAccountIDs); err != nil {
				r.logger.Error("failed to unmarshal pending_downgrade_account_ids", "error", err)
			}
		}

		subs = append(subs, sub)
	}

	return subs, rows.Err()
}

// GetProviderSubscription returns the payment provider subscription for the given subscription ID.
func (r *SubscriptionRepositoryImpl) GetProviderSubscription(subscriptionID int) (*models.PaymentProviderSubscription, error) {
	query := `
		SELECT id, subscription_id, provider_type,
		       external_customer_id, external_subscription_id, external_schedule_id,
		       payment_method_id, last_payment_failed, provider_metadata,
		       created_at, updated_at
		FROM payment_provider_subscriptions
		WHERE subscription_id = $1
	`

	return r.scanProviderSubscription(query, subscriptionID)
}

// GetProviderSubscriptionByExternalID looks up a payment provider subscription
// by the external (e.g. Stripe) subscription ID.
func (r *SubscriptionRepositoryImpl) GetProviderSubscriptionByExternalID(externalSubscriptionID string) (*models.PaymentProviderSubscription, error) {
	query := `
		SELECT id, subscription_id, provider_type,
		       external_customer_id, external_subscription_id, external_schedule_id,
		       payment_method_id, last_payment_failed, provider_metadata,
		       created_at, updated_at
		FROM payment_provider_subscriptions
		WHERE external_subscription_id = $1
	`

	return r.scanProviderSubscription(query, externalSubscriptionID)
}

// GetProviderSubscriptionByScheduleID looks up a payment provider subscription
// by the external (e.g. Stripe) schedule ID.
func (r *SubscriptionRepositoryImpl) GetProviderSubscriptionByScheduleID(externalScheduleID string) (*models.PaymentProviderSubscription, error) {
	query := `
		SELECT id, subscription_id, provider_type,
		       external_customer_id, external_subscription_id, external_schedule_id,
		       payment_method_id, last_payment_failed, provider_metadata,
		       created_at, updated_at
		FROM payment_provider_subscriptions
		WHERE external_schedule_id = $1
	`

	return r.scanProviderSubscription(query, externalScheduleID)
}

// scanProviderSubscription executes a single-row query and scans the result
// into a PaymentProviderSubscription, including JSONB deserialization for provider_metadata.
func (r *SubscriptionRepositoryImpl) scanProviderSubscription(query string, arg any) (*models.PaymentProviderSubscription, error) {
	pps := &models.PaymentProviderSubscription{}
	var metadataJSON []byte

	err := r.db.QueryRow(query, arg).Scan(
		&pps.ID, &pps.SubscriptionID, &pps.ProviderType,
		&pps.ExternalCustomerID, &pps.ExternalSubscriptionID, &pps.ExternalScheduleID,
		&pps.PaymentMethodID, &pps.LastPaymentFailed, &metadataJSON,
		&pps.CreatedAt, &pps.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &pps.ProviderMetadata); err != nil {
			r.logger.Error("failed to unmarshal provider_metadata", "error", err)
		}
	}

	return pps, nil
}

// CreateProviderSubscription inserts a new payment provider subscription record.
func (r *SubscriptionRepositoryImpl) CreateProviderSubscription(pps *models.PaymentProviderSubscription) error {
	metadataJSON, err := marshalJSONBMap(pps.ProviderMetadata)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO payment_provider_subscriptions
		    (subscription_id, provider_type,
		     external_customer_id, external_subscription_id, external_schedule_id,
		     payment_method_id, last_payment_failed, provider_metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	err = r.db.QueryRow(
		query,
		pps.SubscriptionID, pps.ProviderType,
		pps.ExternalCustomerID, pps.ExternalSubscriptionID, pps.ExternalScheduleID,
		pps.PaymentMethodID, pps.LastPaymentFailed, metadataJSON,
	).Scan(&pps.ID, &pps.CreatedAt, &pps.UpdatedAt)

	return err
}

// UpdateProviderSubscription updates all mutable fields of a payment provider subscription.
func (r *SubscriptionRepositoryImpl) UpdateProviderSubscription(pps *models.PaymentProviderSubscription) error {
	metadataJSON, err := marshalJSONBMap(pps.ProviderMetadata)
	if err != nil {
		return err
	}

	query := `
		UPDATE payment_provider_subscriptions
		SET external_customer_id = $1, external_subscription_id = $2,
		    external_schedule_id = $3, payment_method_id = $4,
		    last_payment_failed = $5, provider_metadata = $6,
		    updated_at = NOW()
		WHERE id = $7
	`

	_, err = r.db.Exec(query,
		pps.ExternalCustomerID, pps.ExternalSubscriptionID,
		pps.ExternalScheduleID, pps.PaymentMethodID,
		pps.LastPaymentFailed, metadataJSON,
		pps.ID,
	)
	return err
}

// marshalJSONBIntSlice serializes an int slice to JSONB-compatible bytes.
// Returns nil (SQL NULL) if the slice is nil or empty.
func marshalJSONBIntSlice(ids []int) ([]byte, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return json.Marshal(ids)
}

// marshalJSONBMap serializes a map to JSONB-compatible bytes.
// Returns []byte("{}") if the map is nil to ensure a valid JSONB value.
func marshalJSONBMap(m map[string]any) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

type SubscriptionPlanRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewSubscriptionPlanRepository(db *sql.DB, logger *slog.Logger) *SubscriptionPlanRepositoryImpl {
	return &SubscriptionPlanRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *SubscriptionPlanRepositoryImpl) GetActivePlans() ([]*models.SubscriptionPlan, error) {
	query := `
		SELECT sp.id, sp.name, sp.translation_key, LOWER(sp.plan_type::text), sp.price,
		       c.code as currency_code, sp.is_featured, sp.description, sp.sort_order,
		       bp.id, bp.code, bp.name, bp.duration_days
		FROM subscription_plans sp
		JOIN currencies c ON sp.currency_id = c.id
		LEFT JOIN billing_periods bp ON sp.billing_period_id = bp.id
		WHERE sp.is_active = true AND sp.is_deleted = false
		ORDER BY sp.sort_order
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []*models.SubscriptionPlan
	for rows.Next() {
		plan := &models.SubscriptionPlan{}
		var bpID sql.NullInt64
		var bpCode, bpName sql.NullString
		var bpDuration sql.NullInt64

		err := rows.Scan(
			&plan.ID, &plan.Name, &plan.TranslationKey, &plan.PlanType, &plan.Price,
			&plan.CurrencyCode, &plan.IsFeatured, &plan.Description, &plan.SortOrder,
			&bpID, &bpCode, &bpName, &bpDuration,
		)
		if err != nil {
			return nil, err
		}

		if bpID.Valid {
			plan.BillingPeriod = &models.BillingPeriod{
				ID:           int(bpID.Int64),
				Code:         bpCode.String,
				Name:         bpName.String,
				DurationDays: int(bpDuration.Int64),
			}
		}

		plans = append(plans, plan)
	}

	return plans, rows.Err()
}

func (r *SubscriptionPlanRepositoryImpl) GetByID(id int) (*models.SubscriptionPlan, error) {
	query := `
		SELECT sp.id, sp.name, sp.translation_key, LOWER(sp.plan_type::text), sp.price,
		       c.code as currency_code, sp.is_featured, sp.description, sp.sort_order,
		       bp.id, bp.code, bp.name, bp.duration_days
		FROM subscription_plans sp
		JOIN currencies c ON sp.currency_id = c.id
		LEFT JOIN billing_periods bp ON sp.billing_period_id = bp.id
		WHERE sp.id = $1 AND sp.is_deleted = false
	`

	plan := &models.SubscriptionPlan{}
	var bpID sql.NullInt64
	var bpCode, bpName sql.NullString
	var bpDuration sql.NullInt64

	err := r.db.QueryRow(query, id).Scan(
		&plan.ID, &plan.Name, &plan.TranslationKey, &plan.PlanType, &plan.Price,
		&plan.CurrencyCode, &plan.IsFeatured, &plan.Description, &plan.SortOrder,
		&bpID, &bpCode, &bpName, &bpDuration,
	)
	if err != nil {
		return nil, err
	}

	if bpID.Valid {
		plan.BillingPeriod = &models.BillingPeriod{
			ID:           int(bpID.Int64),
			Code:         bpCode.String,
			Name:         bpName.String,
			DurationDays: int(bpDuration.Int64),
		}
	}

	return plan, nil
}

func (r *SubscriptionPlanRepositoryImpl) GetFreePlan() (*models.SubscriptionPlan, error) {
	query := `
		SELECT sp.id, sp.name, sp.translation_key, LOWER(sp.plan_type::text), sp.price,
		       c.code as currency_code, sp.is_featured, sp.description, sp.sort_order
		FROM subscription_plans sp
		JOIN currencies c ON sp.currency_id = c.id
		WHERE LOWER(sp.plan_type::text) = 'free' AND sp.is_active = true AND sp.is_deleted = false
		LIMIT 1
	`

	plan := &models.SubscriptionPlan{}
	err := r.db.QueryRow(query).Scan(
		&plan.ID, &plan.Name, &plan.TranslationKey, &plan.PlanType, &plan.Price,
		&plan.CurrencyCode, &plan.IsFeatured, &plan.Description, &plan.SortOrder,
	)
	if err != nil {
		return nil, err
	}

	return plan, nil
}
