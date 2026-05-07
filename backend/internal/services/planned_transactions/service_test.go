package planned_transactions

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Mock repository ====================

type mockPlannedTxRepo struct {
	createFunc            func(tx *models.PlannedTransaction) (*models.PlannedTransaction, error)
	getByIDFunc           func(id int) (*models.PlannedTransaction, error)
	getByUserIDFunc       func(userID int, filters types.PlannedTxFilters) ([]*models.PlannedTransaction, error)
	getActiveByUserIDFunc func(userID int, includeInactive bool) ([]*models.PlannedTransaction, error)
	updateFunc            func(tx *models.PlannedTransaction) error
	deleteFunc            func(id int) error
}

func (m *mockPlannedTxRepo) Create(tx *models.PlannedTransaction) (*models.PlannedTransaction, error) {
	if m.createFunc != nil {
		return m.createFunc(tx)
	}
	return tx, nil
}

func (m *mockPlannedTxRepo) GetByID(id int) (*models.PlannedTransaction, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return &models.PlannedTransaction{ID: id, UserID: 1}, nil
}

func (m *mockPlannedTxRepo) GetByUserID(userID int, filters types.PlannedTxFilters) ([]*models.PlannedTransaction, error) {
	if m.getByUserIDFunc != nil {
		return m.getByUserIDFunc(userID, filters)
	}
	return []*models.PlannedTransaction{}, nil
}

func (m *mockPlannedTxRepo) GetActiveByUserID(userID int, includeInactive bool) ([]*models.PlannedTransaction, error) {
	if m.getActiveByUserIDFunc != nil {
		return m.getActiveByUserIDFunc(userID, includeInactive)
	}
	return []*models.PlannedTransaction{}, nil
}

func (m *mockPlannedTxRepo) Update(tx *models.PlannedTransaction) error {
	if m.updateFunc != nil {
		return m.updateFunc(tx)
	}
	return nil
}

func (m *mockPlannedTxRepo) Delete(id int) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func newTestService() (*Service, *mockPlannedTxRepo) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := &mockPlannedTxRepo{}
	svc := New(repo, logger)
	return svc, repo
}

// ==================== Create Tests ====================

func TestCreate_BaseCurrencyNotSet(t *testing.T) {
	svc, _ := newTestService()

	_, err := svc.Create(1, 0, CreateParams{
		Amount:      decimal.NewFromInt(100),
		PlannedDate: time.Now(),
	})
	assert.ErrorIs(t, err, ErrBaseCurrencyNotSet)
}

func TestCreate_Success(t *testing.T) {
	svc, repo := newTestService()

	var captured *models.PlannedTransaction
	repo.createFunc = func(tx *models.PlannedTransaction) (*models.PlannedTransaction, error) {
		captured = tx
		tx.ID = 1
		return tx, nil
	}

	notes := "test notes"
	accountID := 42
	params := CreateParams{
		AccountID:   &accountID,
		Amount:      decimal.NewFromFloat(99.50),
		Label:       "Test Label",
		Notes:       &notes,
		IsIncome:    true,
		PlannedDate: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		IsRecurring: false,
	}

	result, err := svc.Create(1, 5, params)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ID)
	assert.Equal(t, 1, captured.UserID)
	assert.Equal(t, 5, captured.CurrencyID)
	assert.True(t, captured.Amount.Equal(decimal.NewFromFloat(99.50)))
	assert.Equal(t, "Test Label", captured.Label)
	assert.Equal(t, &notes, captured.Notes)
	assert.True(t, captured.IsIncome)
	assert.Equal(t, time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), captured.PlannedDate)
	assert.False(t, captured.IsRecurring)
	assert.True(t, captured.IsActive)
	assert.Nil(t, captured.RecurrenceRule)
}

func TestCreate_CurrencyIDFromParam(t *testing.T) {
	svc, repo := newTestService()

	var capturedCurrencyID int
	repo.createFunc = func(tx *models.PlannedTransaction) (*models.PlannedTransaction, error) {
		capturedCurrencyID = tx.CurrencyID
		tx.ID = 1
		return tx, nil
	}

	_, err := svc.Create(1, 42, CreateParams{
		Amount:      decimal.NewFromInt(100),
		PlannedDate: time.Now(),
	})
	require.NoError(t, err)
	assert.Equal(t, 42, capturedCurrencyID)
}

func TestCreate_IsActiveAlwaysTrue(t *testing.T) {
	svc, repo := newTestService()

	var capturedIsActive bool
	repo.createFunc = func(tx *models.PlannedTransaction) (*models.PlannedTransaction, error) {
		capturedIsActive = tx.IsActive
		tx.ID = 1
		return tx, nil
	}

	_, err := svc.Create(1, 1, CreateParams{
		Amount:      decimal.NewFromInt(100),
		PlannedDate: time.Now(),
	})
	require.NoError(t, err)
	assert.True(t, capturedIsActive)
}

func TestCreate_RecurrenceRuleConversion(t *testing.T) {
	svc, repo := newTestService()

	var capturedRule *string
	repo.createFunc = func(tx *models.PlannedTransaction) (*models.PlannedTransaction, error) {
		capturedRule = tx.RecurrenceRule
		tx.ID = 1
		return tx, nil
	}

	endDate := time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC)
	dayOfMonth := 5
	params := CreateParams{
		Amount:      decimal.NewFromInt(100),
		PlannedDate: time.Now(),
		IsRecurring: true,
		RecurrenceRule: &RecurrenceRuleParams{
			Frequency:  "monthly",
			Interval:   1,
			EndDate:    &endDate,
			DayOfMonth: &dayOfMonth,
		},
	}

	_, err := svc.Create(1, 1, params)
	require.NoError(t, err)
	require.NotNil(t, capturedRule)

	// Verify snake_case JSON
	assert.Contains(t, *capturedRule, `"frequency"`)
	assert.Contains(t, *capturedRule, `"end_date"`)
	assert.Contains(t, *capturedRule, `"day_of_month"`)
	assert.NotContains(t, *capturedRule, `"endDate"`)
	assert.NotContains(t, *capturedRule, `"dayOfMonth"`)
}

func TestCreate_RecurrenceRuleNil(t *testing.T) {
	svc, repo := newTestService()

	var capturedRule *string
	repo.createFunc = func(tx *models.PlannedTransaction) (*models.PlannedTransaction, error) {
		capturedRule = tx.RecurrenceRule
		tx.ID = 1
		return tx, nil
	}

	_, err := svc.Create(1, 1, CreateParams{
		Amount:         decimal.NewFromInt(100),
		PlannedDate:    time.Now(),
		RecurrenceRule: nil,
	})
	require.NoError(t, err)
	assert.Nil(t, capturedRule)
}

func TestCreate_RepoError(t *testing.T) {
	svc, repo := newTestService()
	repo.createFunc = func(tx *models.PlannedTransaction) (*models.PlannedTransaction, error) {
		return nil, errors.New("db error")
	}

	_, err := svc.Create(1, 1, CreateParams{
		Amount:      decimal.NewFromInt(100),
		PlannedDate: time.Now(),
	})
	assert.Error(t, err)
	assert.Equal(t, "db error", err.Error())
}

// ==================== List Tests ====================

func TestList_Success(t *testing.T) {
	svc, repo := newTestService()

	var capturedFilters types.PlannedTxFilters
	repo.getByUserIDFunc = func(userID int, filters types.PlannedTxFilters) ([]*models.PlannedTransaction, error) {
		capturedFilters = filters
		return []*models.PlannedTransaction{
			{ID: 1, Amount: decimal.NewFromInt(100)},
			{ID: 2, Amount: decimal.NewFromInt(200)},
		}, nil
	}

	fromDate := "2024-01-01"
	toDate := "2024-12-31"
	filters := ListFilters{
		AccountIDs:      []int{1, 2},
		FromDate:        &fromDate,
		ToDate:          &toDate,
		IsRecurring:     "true",
		IsExecuted:      "false",
		IsActive:        "true",
		IncludeInactive: true,
	}

	txs, err := svc.List(1, filters)
	require.NoError(t, err)
	assert.Len(t, txs, 2)

	// Verify filters were correctly converted
	assert.Equal(t, []int{1, 2}, capturedFilters.AccountIDs)
	assert.Equal(t, &fromDate, capturedFilters.FromDate)
	assert.Equal(t, &toDate, capturedFilters.ToDate)
	assert.Equal(t, "true", capturedFilters.IsRecurring)
	assert.Equal(t, "false", capturedFilters.IsExecuted)
	assert.Equal(t, "true", capturedFilters.IsActive)
	assert.True(t, capturedFilters.IncludeInactive)
}

func TestList_RepoError(t *testing.T) {
	svc, repo := newTestService()
	repo.getByUserIDFunc = func(userID int, filters types.PlannedTxFilters) ([]*models.PlannedTransaction, error) {
		return nil, errors.New("db error")
	}

	_, err := svc.List(1, ListFilters{})
	assert.Error(t, err)
}

// ==================== GetUpcoming Tests ====================

func TestGetUpcoming_Success(t *testing.T) {
	svc, repo := newTestService()

	var capturedIncludeInactive bool
	repo.getActiveByUserIDFunc = func(userID int, includeInactive bool) ([]*models.PlannedTransaction, error) {
		capturedIncludeInactive = includeInactive
		return []*models.PlannedTransaction{
			{
				ID:          1,
				CurrencyID:  1,
				Amount:      decimal.NewFromInt(100),
				Label:       "Test",
				PlannedDate: time.Now().Add(24 * time.Hour), // Tomorrow
				IsActive:    true,
			},
		}, nil
	}

	occs, err := svc.GetUpcoming(1, 60, true)
	require.NoError(t, err)
	assert.Len(t, occs, 1)
	assert.True(t, capturedIncludeInactive)
}

func TestGetUpcoming_RepoError(t *testing.T) {
	svc, repo := newTestService()
	repo.getActiveByUserIDFunc = func(userID int, includeInactive bool) ([]*models.PlannedTransaction, error) {
		return nil, errors.New("db error")
	}

	_, err := svc.GetUpcoming(1, 30, false)
	assert.Error(t, err)
}

// ==================== GetByID Tests ====================

func TestGetByID_NotFound(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return nil, sql.ErrNoRows
	}

	_, err := svc.GetByID(1, 99)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetByID_DBError(t *testing.T) {
	svc, repo := newTestService()
	dbErr := errors.New("connection refused")
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return nil, dbErr
	}

	_, err := svc.GetByID(1, 99)
	assert.ErrorIs(t, err, dbErr)
	assert.False(t, errors.Is(err, ErrNotFound))
}

func TestGetByID_AccessDenied(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return &models.PlannedTransaction{ID: id, UserID: 999}, nil
	}

	_, err := svc.GetByID(1, 1)
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestGetByID_Success(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return &models.PlannedTransaction{
			ID:          id,
			UserID:      1,
			Amount:      decimal.NewFromInt(100),
			PlannedDate: now,
		}, nil
	}

	tx, err := svc.GetByID(1, 42)
	require.NoError(t, err)
	assert.Equal(t, 42, tx.ID)
	assert.Equal(t, 1, tx.UserID)
}

// ==================== Update Tests ====================

func TestUpdate_NotFound(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return nil, sql.ErrNoRows
	}

	_, err := svc.Update(1, 99, UpdateParams{
		Amount:      decimal.NewFromInt(100),
		PlannedDate: time.Now(),
	})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestUpdate_AccessDenied(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return &models.PlannedTransaction{ID: id, UserID: 999}, nil
	}

	_, err := svc.Update(1, 1, UpdateParams{
		Amount:      decimal.NewFromInt(100),
		PlannedDate: time.Now(),
	})
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestUpdate_Success(t *testing.T) {
	svc, repo := newTestService()
	now := time.Now()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return &models.PlannedTransaction{
			ID:         id,
			UserID:     1,
			CurrencyID: 5,
			Amount:     decimal.NewFromInt(100),
			Label:      "Old Label",
			IsActive:   true,
		}, nil
	}

	var captured *models.PlannedTransaction
	repo.updateFunc = func(tx *models.PlannedTransaction) error {
		captured = tx
		return nil
	}

	notes := "updated notes"
	isActive := false
	params := UpdateParams{
		Amount:      decimal.NewFromFloat(200.50),
		Label:       "New Label",
		Notes:       &notes,
		IsIncome:    true,
		PlannedDate: now,
		IsRecurring: true,
		IsActive:    &isActive,
	}

	result, err := svc.Update(1, 1, params)
	require.NoError(t, err)
	assert.True(t, captured.Amount.Equal(decimal.NewFromFloat(200.50)))
	assert.Equal(t, "New Label", captured.Label)
	assert.Equal(t, &notes, captured.Notes)
	assert.True(t, captured.IsIncome)
	assert.Equal(t, now, captured.PlannedDate)
	assert.True(t, captured.IsRecurring)
	assert.False(t, captured.IsActive)
	assert.Equal(t, captured.ID, result.ID)
	assert.Equal(t, captured.Label, result.Label)
	assert.True(t, captured.Amount.Equal(result.Amount))
}

func TestUpdate_CurrencyIDPreserved(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return &models.PlannedTransaction{
			ID:         id,
			UserID:     1,
			CurrencyID: 42,
		}, nil
	}

	var capturedCurrencyID int
	repo.updateFunc = func(tx *models.PlannedTransaction) error {
		capturedCurrencyID = tx.CurrencyID
		return nil
	}

	_, err := svc.Update(1, 1, UpdateParams{
		Amount:      decimal.NewFromInt(100),
		PlannedDate: time.Now(),
	})
	require.NoError(t, err)
	assert.Equal(t, 42, capturedCurrencyID)
}

func TestUpdate_IsActiveUpdatedWhenNonNil(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return &models.PlannedTransaction{
			ID:       id,
			UserID:   1,
			IsActive: true,
		}, nil
	}

	var capturedIsActive bool
	repo.updateFunc = func(tx *models.PlannedTransaction) error {
		capturedIsActive = tx.IsActive
		return nil
	}

	isActive := false
	_, err := svc.Update(1, 1, UpdateParams{
		Amount:      decimal.NewFromInt(100),
		PlannedDate: time.Now(),
		IsActive:    &isActive,
	})
	require.NoError(t, err)
	assert.False(t, capturedIsActive)
}

func TestUpdate_IsActiveNotUpdatedWhenNil(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return &models.PlannedTransaction{
			ID:       id,
			UserID:   1,
			IsActive: true,
		}, nil
	}

	var capturedIsActive bool
	repo.updateFunc = func(tx *models.PlannedTransaction) error {
		capturedIsActive = tx.IsActive
		return nil
	}

	_, err := svc.Update(1, 1, UpdateParams{
		Amount:      decimal.NewFromInt(100),
		PlannedDate: time.Now(),
		IsActive:    nil,
	})
	require.NoError(t, err)
	assert.True(t, capturedIsActive)
}

func TestUpdate_RecurrenceRuleConversion(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return &models.PlannedTransaction{ID: id, UserID: 1}, nil
	}

	var capturedRule *string
	repo.updateFunc = func(tx *models.PlannedTransaction) error {
		capturedRule = tx.RecurrenceRule
		return nil
	}

	endDate := time.Date(2027, 6, 15, 0, 0, 0, 0, time.UTC)
	_, err := svc.Update(1, 1, UpdateParams{
		Amount:      decimal.NewFromInt(100),
		PlannedDate: time.Now(),
		IsRecurring: true,
		RecurrenceRule: &RecurrenceRuleParams{
			Frequency: "weekly",
			Interval:  2,
			EndDate:   &endDate,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, capturedRule)
	assert.Contains(t, *capturedRule, `"end_date"`)
	assert.Contains(t, *capturedRule, `"2027-06-15T00:00:00"`)
}

func TestUpdate_RepoUpdateError(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return &models.PlannedTransaction{ID: id, UserID: 1}, nil
	}
	repo.updateFunc = func(tx *models.PlannedTransaction) error {
		return errors.New("db error")
	}

	_, err := svc.Update(1, 1, UpdateParams{
		Amount:      decimal.NewFromInt(100),
		PlannedDate: time.Now(),
	})
	assert.Error(t, err)
	assert.Equal(t, "db error", err.Error())
}

// ==================== Delete Tests ====================

func TestDelete_NotFound(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return nil, sql.ErrNoRows
	}

	err := svc.Delete(1, 99)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDelete_DBError(t *testing.T) {
	svc, repo := newTestService()
	dbErr := errors.New("connection refused")
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return nil, dbErr
	}

	err := svc.Delete(1, 99)
	assert.ErrorIs(t, err, dbErr)
	assert.False(t, errors.Is(err, ErrNotFound))
}

func TestDelete_AccessDenied(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return &models.PlannedTransaction{ID: id, UserID: 999}, nil
	}

	err := svc.Delete(1, 1)
	assert.ErrorIs(t, err, ErrAccessDenied)
}

func TestDelete_Success(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return &models.PlannedTransaction{ID: id, UserID: 1}, nil
	}

	var deletedID int
	repo.deleteFunc = func(id int) error {
		deletedID = id
		return nil
	}

	err := svc.Delete(1, 42)
	assert.NoError(t, err)
	assert.Equal(t, 42, deletedID)
}

func TestDelete_RepoDeleteError(t *testing.T) {
	svc, repo := newTestService()
	repo.getByIDFunc = func(id int) (*models.PlannedTransaction, error) {
		return &models.PlannedTransaction{ID: id, UserID: 1}, nil
	}
	repo.deleteFunc = func(id int) error {
		return errors.New("db error")
	}

	err := svc.Delete(1, 1)
	assert.Error(t, err)
	assert.Equal(t, "db error", err.Error())
}

// ==================== ruleToStorageJSON Tests ====================

func TestRuleToStorageJSON_Nil(t *testing.T) {
	result := ruleToStorageJSON(nil)
	assert.Nil(t, result)
}

func TestRuleToStorageJSON_Basic(t *testing.T) {
	rule := &RecurrenceRuleParams{
		Frequency: "monthly",
		Interval:  1,
	}
	result := ruleToStorageJSON(rule)
	require.NotNil(t, result)

	var stored models.RecurrenceRule
	err := json.Unmarshal([]byte(*result), &stored)
	require.NoError(t, err)
	assert.Equal(t, "monthly", stored.Frequency)
	assert.Equal(t, 1, stored.Interval)
	assert.Nil(t, stored.EndDate)
	assert.Nil(t, stored.Count)
	assert.Nil(t, stored.DayOfMonth)
}

func TestRuleToStorageJSON_WithEndDateAndDayOfMonth(t *testing.T) {
	endDate := time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC)
	dayOfMonth := 5
	count := 12
	rule := &RecurrenceRuleParams{
		Frequency:  "monthly",
		Interval:   1,
		EndDate:    &endDate,
		DayOfMonth: &dayOfMonth,
		Count:      &count,
	}
	result := ruleToStorageJSON(rule)
	require.NotNil(t, result)

	// Verify snake_case keys in JSON
	assert.Contains(t, *result, `"end_date"`)
	assert.Contains(t, *result, `"day_of_month"`)
	assert.Contains(t, *result, `"2027-12-31T00:00:00"`)
	assert.NotContains(t, *result, `"endDate"`)
	assert.NotContains(t, *result, `"dayOfMonth"`)

	var stored models.RecurrenceRule
	err := json.Unmarshal([]byte(*result), &stored)
	require.NoError(t, err)
	assert.Equal(t, 5, *stored.DayOfMonth)
	assert.Equal(t, 12, *stored.Count)
}

// ==================== StorageRuleToDTO Tests ====================

func TestStorageRuleToDTO_Basic(t *testing.T) {
	jsonStr := `{"frequency":"monthly","interval":1}`
	params := StorageRuleToDTO(jsonStr)
	require.NotNil(t, params)
	assert.Equal(t, "monthly", params.Frequency)
	assert.Equal(t, 1, params.Interval)
	assert.Nil(t, params.EndDate)
	assert.Nil(t, params.Count)
	assert.Nil(t, params.DayOfMonth)
}

func TestStorageRuleToDTO_WithEndDateAndDayOfMonth(t *testing.T) {
	jsonStr := `{"frequency":"monthly","interval":1,"end_date":"2027-12-31T00:00:00","day_of_month":5}`
	params := StorageRuleToDTO(jsonStr)
	require.NotNil(t, params)
	assert.Equal(t, "monthly", params.Frequency)
	assert.Equal(t, 1, params.Interval)
	require.NotNil(t, params.EndDate)
	assert.Equal(t, 2027, params.EndDate.Year())
	assert.Equal(t, time.December, params.EndDate.Month())
	assert.Equal(t, 31, params.EndDate.Day())
	require.NotNil(t, params.DayOfMonth)
	assert.Equal(t, 5, *params.DayOfMonth)
}

func TestStorageRuleToDTO_InvalidJSON(t *testing.T) {
	params := StorageRuleToDTO(`{invalid}`)
	assert.Nil(t, params)
}

func TestStorageRuleToDTO_EmptyEndDate(t *testing.T) {
	jsonStr := `{"frequency":"monthly","interval":1,"end_date":""}`
	params := StorageRuleToDTO(jsonStr)
	require.NotNil(t, params)
	assert.Nil(t, params.EndDate)
}

func TestStorageRuleToDTO_BareDateFormat(t *testing.T) {
	jsonStr := `{"frequency":"yearly","interval":1,"end_date":"2030-06-15"}`
	params := StorageRuleToDTO(jsonStr)
	require.NotNil(t, params)
	require.NotNil(t, params.EndDate)
	assert.Equal(t, 2030, params.EndDate.Year())
	assert.Equal(t, time.June, params.EndDate.Month())
	assert.Equal(t, 15, params.EndDate.Day())
}

func TestFullRoundTrip(t *testing.T) {
	// RecurrenceRuleParams -> ruleToStorageJSON -> StorageRuleToDTO -> equivalent params
	endDate := time.Date(2027, 12, 31, 0, 0, 0, 0, time.UTC)
	dayOfMonth := 5
	count := 12
	original := &RecurrenceRuleParams{
		Frequency:  "monthly",
		Interval:   1,
		EndDate:    &endDate,
		DayOfMonth: &dayOfMonth,
		Count:      &count,
	}

	stored := ruleToStorageJSON(original)
	require.NotNil(t, stored)

	restored := StorageRuleToDTO(*stored)
	require.NotNil(t, restored)

	assert.Equal(t, original.Frequency, restored.Frequency)
	assert.Equal(t, original.Interval, restored.Interval)
	require.NotNil(t, restored.EndDate)
	assert.Equal(t, original.EndDate.Year(), restored.EndDate.Year())
	assert.Equal(t, original.EndDate.Month(), restored.EndDate.Month())
	assert.Equal(t, original.EndDate.Day(), restored.EndDate.Day())
	require.NotNil(t, restored.DayOfMonth)
	assert.Equal(t, *original.DayOfMonth, *restored.DayOfMonth)
	require.NotNil(t, restored.Count)
	assert.Equal(t, *original.Count, *restored.Count)
}
