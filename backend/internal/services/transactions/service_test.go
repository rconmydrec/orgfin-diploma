package transactions

import (
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/go-budget/backend/internal/workers/tasks"
)

// --- Mock implementations for transaction service dependencies ---

type mockUserRepo struct {
	users map[int]*models.User
}

func newMockUserRepo() *mockUserRepo {
	firstName := "Test"
	lastName := "User"
	return &mockUserRepo{
		users: map[int]*models.User{
			1: {
				ID:             1,
				Email:          "test@example.com",
				IsActive:       true,
				FirstName:      &firstName,
				LastName:       &lastName,
				BaseCurrencyID: 1,
			},
		},
	}
}

func (m *mockUserRepo) GetByID(id int) (*models.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return u, nil
}

func (m *mockUserRepo) GetByEmail(string) (*models.User, error)     { return nil, sql.ErrNoRows }
func (m *mockUserRepo) Create(u *models.User) (*models.User, error) { return u, nil }
func (m *mockUserRepo) Update(*models.User) error                   { return nil }
func (m *mockUserRepo) Activate(int) error                          { return nil }
func (m *mockUserRepo) UpdatePassword(int, string) error            { return nil }

type mockAccountRepo struct {
	accounts map[int]*models.Account
}

func newMockAccountRepo() *mockAccountRepo {
	return &mockAccountRepo{
		accounts: map[int]*models.Account{
			10: {
				ID:             10,
				UserID:         1,
				Name:           "Test Account",
				Balance:        decimal.NewFromInt(1000),
				InitialBalance: decimal.NewFromInt(0),
				CurrencyID:     1,
			},
		},
	}
}

func (m *mockAccountRepo) GetByID(id int) (*models.Account, error) {
	a, ok := m.accounts[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return a, nil
}

func (m *mockAccountRepo) Create(a *models.Account) (*models.Account, error) { return a, nil }
func (m *mockAccountRepo) GetByUserID(int, types.AccountFilters) ([]*models.Account, error) {
	return nil, nil
}
func (m *mockAccountRepo) Update(*models.Account) error             { return nil }
func (m *mockAccountRepo) SoftDelete(int) error                     { return nil }
func (m *mockAccountRepo) UpdateBalance(int, decimal.Decimal) error { return nil }
func (m *mockAccountRepo) SetArchiveStatus(int, bool) error         { return nil }
func (m *mockAccountRepo) CountActiveByUserID(int) (int, error)     { return 0, nil }

type mockTransactionRepo struct {
	nextID int
}

func newMockTransactionRepo() *mockTransactionRepo {
	return &mockTransactionRepo{nextID: 100}
}

func (m *mockTransactionRepo) Create(tx *models.Transaction) (*models.Transaction, error) {
	created := *tx
	created.ID = m.nextID
	m.nextID++
	return &created, nil
}

func (m *mockTransactionRepo) GetByID(int) (*models.Transaction, error) {
	return nil, sql.ErrNoRows
}

func (m *mockTransactionRepo) GetByAccountID(int, int, int) ([]*models.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionRepo) GetByAccountIDForRecalc(int) ([]*models.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionRepo) GetByUserID(int, types.TransactionFilters) ([]*models.Transaction, int, error) {
	return nil, 0, nil
}

func (m *mockTransactionRepo) GetForExport(int, string, string) ([]*models.Transaction, error) {
	return nil, nil
}

func (m *mockTransactionRepo) Update(*models.Transaction) error         { return nil }
func (m *mockTransactionRepo) UpdateLinkedID(int, int) error            { return nil }
func (m *mockTransactionRepo) UpdateBalance(int, decimal.Decimal) error { return nil }
func (m *mockTransactionRepo) Delete(int) error                         { return nil }

type mockCategoryRepo struct{}

func (m *mockCategoryRepo) GetByUserID(int) ([]*models.UserCategory, error)        { return nil, nil }
func (m *mockCategoryRepo) GetByUserIDGrouped(int) ([]*models.UserCategory, error) { return nil, nil }
func (m *mockCategoryRepo) GetByID(int) (*models.UserCategory, error)              { return nil, sql.ErrNoRows }
func (m *mockCategoryRepo) Create(*models.UserCategory) (*models.UserCategory, error) {
	return nil, nil
}
func (m *mockCategoryRepo) Update(*models.UserCategory) error { return nil }
func (m *mockCategoryRepo) Delete(int) error                  { return nil }
func (m *mockCategoryRepo) CopyDefaultCategories(int) error   { return nil }

type mockTemplateRepo struct{}

func (m *mockTemplateRepo) Create(*models.TransactionTemplate) error { return nil }
func (m *mockTemplateRepo) GetByID(int) (*models.TransactionTemplate, error) {
	return nil, sql.ErrNoRows
}
func (m *mockTemplateRepo) GetByUserID(int) ([]*models.TransactionTemplate, error) { return nil, nil }
func (m *mockTemplateRepo) Update(*models.TransactionTemplate) error               { return nil }
func (m *mockTemplateRepo) Delete(int) error                                       { return nil }

// mockTaskEnqueuer records enqueued tasks and optionally returns an error.
type mockTaskEnqueuer struct {
	tasks []enqueuedTask
	err   error
}

type enqueuedTask struct {
	taskType string
	payload  []byte
}

func (m *mockTaskEnqueuer) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	m.tasks = append(m.tasks, enqueuedTask{
		taskType: task.Type(),
		payload:  task.Payload(),
	})
	if m.err != nil {
		return nil, m.err
	}
	return &asynq.TaskInfo{}, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
}

// --- Tests ---

func TestCreateTransaction_EnqueueBudgetUpdate(t *testing.T) {
	t.Run("enqueues budget update task after successful create", func(t *testing.T) {
		enqueuer := &mockTaskEnqueuer{}
		svc := New(
			newMockTransactionRepo(),
			newMockAccountRepo(),
			&mockCategoryRepo{},
			newMockUserRepo(),
			&mockTemplateRepo{},
			nil, // currencyService
			enqueuer,
			testLogger(),
		)

		tx, err := svc.CreateTransaction(1, CreateTransactionInput{
			AccountID: 10,
			Amount:    decimal.NewFromInt(50),
			Label:     "Groceries",
		})
		require.NoError(t, err)
		require.NotNil(t, tx)

		// Verify the budget update task was enqueued.
		require.Len(t, enqueuer.tasks, 1, "expected exactly one enqueued task")
		assert.Equal(t, tasks.TypeBudgetUserUpdate, enqueuer.tasks[0].taskType)

		// Verify payload contains userID and accountID.
		assert.Contains(t, string(enqueuer.tasks[0].payload), `"user_id":1`)
		assert.Contains(t, string(enqueuer.tasks[0].payload), `"account_id":10`)
	})

	t.Run("create succeeds with nil enqueuer", func(t *testing.T) {
		svc := New(
			newMockTransactionRepo(),
			newMockAccountRepo(),
			&mockCategoryRepo{},
			newMockUserRepo(),
			&mockTemplateRepo{},
			nil, // currencyService
			nil, // nil enqueuer
			testLogger(),
		)

		tx, err := svc.CreateTransaction(1, CreateTransactionInput{
			AccountID: 10,
			Amount:    decimal.NewFromInt(50),
			Label:     "Coffee",
		})
		require.NoError(t, err, "create must succeed with nil enqueuer")
		require.NotNil(t, tx)
	})

	t.Run("create succeeds even when enqueue fails", func(t *testing.T) {
		enqueuer := &mockTaskEnqueuer{
			err: errors.New("redis unavailable"),
		}
		svc := New(
			newMockTransactionRepo(),
			newMockAccountRepo(),
			&mockCategoryRepo{},
			newMockUserRepo(),
			&mockTemplateRepo{},
			nil, // currencyService
			enqueuer,
			testLogger(),
		)

		tx, err := svc.CreateTransaction(1, CreateTransactionInput{
			AccountID: 10,
			Amount:    decimal.NewFromInt(25),
			Label:     "Lunch",
		})
		require.NoError(t, err, "create must succeed even if enqueue fails")
		require.NotNil(t, tx)

		// The enqueuer was called but returned an error.
		require.Len(t, enqueuer.tasks, 1)
		assert.Equal(t, tasks.TypeBudgetUserUpdate, enqueuer.tasks[0].taskType)
	})
}

func TestEnqueueBudgetUpdate_Directly(t *testing.T) {
	t.Run("nil enqueuer is a no-op", func(t *testing.T) {
		svc := &Service{
			enqueuer: nil,
			logger:   testLogger(),
		}
		// Should not panic.
		svc.enqueueBudgetUpdate(1, 10)
	})

	t.Run("enqueue error is logged but not propagated", func(t *testing.T) {
		enqueuer := &mockTaskEnqueuer{
			err: errors.New("connection refused"),
		}
		svc := &Service{
			enqueuer: enqueuer,
			logger:   testLogger(),
		}

		// Should not panic; errors are logged internally.
		svc.enqueueBudgetUpdate(1, 10)

		require.Len(t, enqueuer.tasks, 1)
		assert.Equal(t, tasks.TypeBudgetUserUpdate, enqueuer.tasks[0].taskType)
	})

	t.Run("successful enqueue records the correct task", func(t *testing.T) {
		enqueuer := &mockTaskEnqueuer{}
		svc := &Service{
			enqueuer: enqueuer,
			logger:   testLogger(),
		}

		svc.enqueueBudgetUpdate(99, 200)

		require.Len(t, enqueuer.tasks, 1)
		assert.Equal(t, tasks.TypeBudgetUserUpdate, enqueuer.tasks[0].taskType)
		assert.Contains(t, string(enqueuer.tasks[0].payload), `"user_id":99`)
		assert.Contains(t, string(enqueuer.tasks[0].payload), `"account_id":200`)
	})
}
