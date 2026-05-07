package planned_transactions

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/go-budget/backend/internal/dateutil"
	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
)

// Service encapsulates all planned transactions business logic.
type Service struct {
	plannedTxRepo PlannedTransactionRepository
	logger        *slog.Logger
}

// New creates a new planned transactions Service with the given dependencies.
func New(
	plannedTxRepo PlannedTransactionRepository,
	logger *slog.Logger,
) *Service {
	return &Service{
		plannedTxRepo: plannedTxRepo,
		logger:        logger,
	}
}

// Create creates a new planned transaction for the given user.
func (s *Service) Create(userID int, baseCurrencyID int, params CreateParams) (*PlannedTransaction, error) {
	if baseCurrencyID == 0 {
		return nil, ErrBaseCurrencyNotSet
	}

	plannedTx := &models.PlannedTransaction{
		UserID:         userID,
		CurrencyID:     baseCurrencyID,
		Amount:         params.Amount,
		Label:          params.Label,
		Notes:          params.Notes,
		IsIncome:       params.IsIncome,
		PlannedDate:    params.PlannedDate,
		IsRecurring:    params.IsRecurring,
		RecurrenceRule: ruleToStorageJSON(params.RecurrenceRule),
		IsActive:       true,
	}

	created, err := s.plannedTxRepo.Create(plannedTx)
	if err != nil {
		s.logger.Error("create planned transaction failed", "error", err)
		return nil, err
	}

	return toPlannedTransaction(created), nil
}

// List returns planned transactions for a user matching the given filters.
func (s *Service) List(userID int, filters ListFilters) ([]*PlannedTransaction, error) {
	repoFilters := types.PlannedTxFilters{
		AccountIDs:      filters.AccountIDs,
		FromDate:        filters.FromDate,
		ToDate:          filters.ToDate,
		IsRecurring:     filters.IsRecurring,
		IsExecuted:      filters.IsExecuted,
		IsActive:        filters.IsActive,
		IncludeInactive: filters.IncludeInactive,
	}

	txs, err := s.plannedTxRepo.GetByUserID(userID, repoFilters)
	if err != nil {
		s.logger.Error("list planned transactions failed", "error", err)
		return nil, err
	}

	result := make([]*PlannedTransaction, len(txs))
	for i, tx := range txs {
		result[i] = toPlannedTransaction(tx)
	}
	return result, nil
}

// GetUpcoming returns upcoming planned transaction occurrences for a user.
// It fetches active planned transactions from the repository and generates
// occurrences using the business logic in occurrences.go.
func (s *Service) GetUpcoming(userID int, days int, includeInactive bool) ([]Occurrence, error) {
	txs, err := s.plannedTxRepo.GetActiveByUserID(userID, includeInactive)
	if err != nil {
		s.logger.Error("get active planned transactions failed", "error", err)
		return nil, err
	}

	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endDate := startDate.AddDate(0, 0, days)

	occurrences := GenerateOccurrencesForTxs(txs, startDate, endDate)

	return occurrences, nil
}

// GetByID returns a planned transaction by ID, verifying ownership.
func (s *Service) GetByID(userID, id int) (*PlannedTransaction, error) {
	tx, err := s.plannedTxRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if tx.UserID != userID {
		return nil, ErrAccessDenied
	}

	return toPlannedTransaction(tx), nil
}

// Update modifies an existing planned transaction, verifying ownership.
func (s *Service) Update(userID, id int, params UpdateParams) (*PlannedTransaction, error) {
	existing, err := s.plannedTxRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if existing.UserID != userID {
		return nil, ErrAccessDenied
	}

	existing.Amount = params.Amount
	existing.Label = params.Label
	existing.Notes = params.Notes
	existing.IsIncome = params.IsIncome
	existing.PlannedDate = params.PlannedDate
	existing.IsRecurring = params.IsRecurring
	existing.RecurrenceRule = ruleToStorageJSON(params.RecurrenceRule)
	if params.IsActive != nil {
		existing.IsActive = *params.IsActive
	}

	if err := s.plannedTxRepo.Update(existing); err != nil {
		s.logger.Error("update planned transaction failed", "error", err)
		return nil, err
	}

	return toPlannedTransaction(existing), nil
}

// Delete removes a planned transaction, verifying ownership.
func (s *Service) Delete(userID, id int) error {
	existing, err := s.plannedTxRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	if existing.UserID != userID {
		return ErrAccessDenied
	}

	if err := s.plannedTxRepo.Delete(id); err != nil {
		s.logger.Error("delete planned transaction failed", "error", err)
		return err
	}

	return nil
}

// ruleToStorageJSON converts a RecurrenceRuleParams to a JSON string using
// models.RecurrenceRule (snake_case tags) so that the DB stores field names
// consistent with what generateOccurrences expects.
func ruleToStorageJSON(rule *RecurrenceRuleParams) *string {
	if rule == nil {
		return nil
	}
	storageRule := models.RecurrenceRule{
		Frequency:  rule.Frequency,
		Interval:   rule.Interval,
		Count:      rule.Count,
		DayOfMonth: rule.DayOfMonth,
	}
	if rule.EndDate != nil {
		s := rule.EndDate.Format("2006-01-02T15:04:05")
		storageRule.EndDate = &s
	}
	b, _ := json.Marshal(storageRule)
	str := string(b)
	return &str
}

// toPlannedTransaction converts a models.PlannedTransaction to the service domain type.
func toPlannedTransaction(m *models.PlannedTransaction) *PlannedTransaction {
	if m == nil {
		return nil
	}
	pt := &PlannedTransaction{
		ID:                    m.ID,
		UserID:                m.UserID,
		CurrencyID:            m.CurrencyID,
		Amount:                m.Amount,
		Label:                 m.Label,
		Notes:                 m.Notes,
		IsIncome:              m.IsIncome,
		PlannedDate:           m.PlannedDate,
		IsRecurring:           m.IsRecurring,
		RecurrenceRule:        m.RecurrenceRule,
		IsExecuted:            m.IsExecuted,
		ExecutedTransactionID: m.ExecutedTransactionID,
		ExecutionDate:         m.ExecutionDate,
		IsActive:              m.IsActive,
		IsDeleted:             m.IsDeleted,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}
	if m.Currency != nil {
		pt.Currency = &Currency{
			ID:   m.Currency.ID,
			Code: m.Currency.Code,
			Name: m.Currency.Name,
		}
	}
	return pt
}

// StorageRuleToDTO converts a DB JSON string (snake_case fields) into a
// RecurrenceRuleParams for the caller to use.
func StorageRuleToDTO(jsonStr string) *RecurrenceRuleParams {
	var rule models.RecurrenceRule
	if err := json.Unmarshal([]byte(jsonStr), &rule); err != nil {
		return nil
	}
	params := &RecurrenceRuleParams{
		Frequency:  rule.Frequency,
		Interval:   rule.Interval,
		Count:      rule.Count,
		DayOfMonth: rule.DayOfMonth,
	}
	if rule.EndDate != nil && *rule.EndDate != "" {
		if t, err := dateutil.ParseDate(*rule.EndDate); err == nil {
			params.EndDate = &t
		}
	}
	return params
}
