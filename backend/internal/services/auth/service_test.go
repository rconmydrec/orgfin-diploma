package auth

import (
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-budget/backend/internal/auth"
	"github.com/go-budget/backend/internal/config"
	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/workers/tasks"
)

// --- Mock implementations for auth service dependencies ---

type mockUserRepo struct {
	getByEmailFn func(email string) (*models.User, error)
	createFn     func(user *models.User) (*models.User, error)
}

func (m *mockUserRepo) GetByEmail(email string) (*models.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(email)
	}
	return nil, sql.ErrNoRows
}

func (m *mockUserRepo) Create(user *models.User) (*models.User, error) {
	if m.createFn != nil {
		return m.createFn(user)
	}
	created := *user
	created.ID = 42
	return &created, nil
}

func (m *mockUserRepo) GetByID(int) (*models.User, error) { return nil, sql.ErrNoRows }
func (m *mockUserRepo) Update(*models.User) error         { return nil }
func (m *mockUserRepo) Activate(int) error                { return nil }
func (m *mockUserRepo) UpdatePassword(int, string) error  { return nil }

type mockActivationTokenRepo struct {
	createFn func(token *models.ActivationToken) error
}

func (m *mockActivationTokenRepo) Create(token *models.ActivationToken) error {
	if m.createFn != nil {
		return m.createFn(token)
	}
	return nil
}

func (m *mockActivationTokenRepo) GetByToken(string) (*models.ActivationToken, error) {
	return nil, sql.ErrNoRows
}

func (m *mockActivationTokenRepo) Delete(int) error     { return nil }
func (m *mockActivationTokenRepo) DeleteExpired() error { return nil }

type mockSettingsRepo struct{}

func (m *mockSettingsRepo) CreateDefault(int) error                       { return nil }
func (m *mockSettingsRepo) GetByUserID(int) (*models.UserSettings, error) { return nil, nil }
func (m *mockSettingsRepo) Update(*models.UserSettings) error             { return nil }

type mockCategoryRepo struct{}

func (m *mockCategoryRepo) GetByUserID(int) ([]*models.UserCategory, error)        { return nil, nil }
func (m *mockCategoryRepo) GetByUserIDGrouped(int) ([]*models.UserCategory, error) { return nil, nil }
func (m *mockCategoryRepo) GetByID(int) (*models.UserCategory, error)              { return nil, nil }
func (m *mockCategoryRepo) Create(*models.UserCategory) (*models.UserCategory, error) {
	return nil, nil
}
func (m *mockCategoryRepo) Update(*models.UserCategory) error { return nil }
func (m *mockCategoryRepo) Delete(int) error                  { return nil }
func (m *mockCategoryRepo) CopyDefaultCategories(int) error   { return nil }

type mockCurrencyRepo struct{}

func (m *mockCurrencyRepo) GetAll() ([]*models.Currency, error) { return nil, nil }
func (m *mockCurrencyRepo) GetByCode(code string) (*models.Currency, error) {
	return &models.Currency{ID: 1, Code: code, Name: "US Dollar"}, nil
}
func (m *mockCurrencyRepo) GetByID(int) (*models.Currency, error) { return nil, nil }

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

// mockSubscriptionTrialCreator records calls and optionally returns an error.
type mockSubscriptionTrialCreator struct {
	calledWithUserID int
	createErr        error
}

func (m *mockSubscriptionTrialCreator) CreateTrialSubscription(userID int) error {
	m.calledWithUserID = userID
	return m.createErr
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
}

func testConfig() *config.Config {
	return &config.Config{
		JWTSecret:            "test-secret",
		JWTExpirationMinutes: 1440,
		AppURL:               "http://localhost:8080",
		FrontendURL:          "http://localhost:5173",
	}
}

// --- Tests ---

func TestRegister_EnqueueActivationEmail(t *testing.T) {
	t.Run("enqueues activation email task on successful registration", func(t *testing.T) {
		enqueuer := &mockTaskEnqueuer{}
		svc := New(
			&mockUserRepo{},
			&mockActivationTokenRepo{},
			&mockSettingsRepo{},
			&mockCategoryRepo{},
			&mockCurrencyRepo{},
			auth.NewJWTService(testConfig().JWTSecret, testConfig().JWTExpirationMinutes),
			enqueuer,
			nil, // subscriptionSvc
			testConfig(),
			testLogger(),
		)

		result, err := svc.Register("test@example.com", "password123", "John", "Doe")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, "test@example.com", result.User.Email)
		assert.NotEmpty(t, result.ActivationToken)

		// Verify the enqueuer was called exactly once with the correct task type.
		require.Len(t, enqueuer.tasks, 1, "expected exactly one enqueued task")
		assert.Equal(t, tasks.TypeEmailActivation, enqueuer.tasks[0].taskType)

		// Verify payload contains the user email.
		assert.Contains(t, string(enqueuer.tasks[0].payload), "test@example.com")
	})

	t.Run("registration succeeds even when enqueue fails", func(t *testing.T) {
		enqueuer := &mockTaskEnqueuer{
			err: errors.New("redis connection refused"),
		}
		svc := New(
			&mockUserRepo{},
			&mockActivationTokenRepo{},
			&mockSettingsRepo{},
			&mockCategoryRepo{},
			&mockCurrencyRepo{},
			auth.NewJWTService(testConfig().JWTSecret, testConfig().JWTExpirationMinutes),
			enqueuer,
			nil, // subscriptionSvc
			testConfig(),
			testLogger(),
		)

		result, err := svc.Register("test@example.com", "password123", "Jane", "Doe")
		require.NoError(t, err, "registration must succeed even if enqueue fails")
		require.NotNil(t, result)
		assert.Equal(t, 42, result.User.ID)

		// The enqueuer was called, but it returned an error. The task is still recorded.
		require.Len(t, enqueuer.tasks, 1)
		assert.Equal(t, tasks.TypeEmailActivation, enqueuer.tasks[0].taskType)
	})

	t.Run("registration succeeds with nil enqueuer", func(t *testing.T) {
		svc := New(
			&mockUserRepo{},
			&mockActivationTokenRepo{},
			&mockSettingsRepo{},
			&mockCategoryRepo{},
			&mockCurrencyRepo{},
			auth.NewJWTService(testConfig().JWTSecret, testConfig().JWTExpirationMinutes),
			nil, // nil enqueuer
			nil, // nil subscriptionSvc
			testConfig(),
			testLogger(),
		)

		result, err := svc.Register("test@example.com", "password123", "No", "Queue")
		require.NoError(t, err, "registration must succeed with nil enqueuer")
		require.NotNil(t, result)
		assert.Equal(t, 42, result.User.ID)
		assert.NotEmpty(t, result.ActivationToken)
	})
}

func TestRegister_TrialSubscription(t *testing.T) {
	t.Run("registration creates trial subscription", func(t *testing.T) {
		subMock := &mockSubscriptionTrialCreator{}
		svc := New(
			&mockUserRepo{},
			&mockActivationTokenRepo{},
			&mockSettingsRepo{},
			&mockCategoryRepo{},
			&mockCurrencyRepo{},
			auth.NewJWTService(testConfig().JWTSecret, testConfig().JWTExpirationMinutes),
			nil, // enqueuer
			subMock,
			testConfig(),
			testLogger(),
		)

		result, err := svc.Register("trial@example.com", "password123", "Trial", "User")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 42, subMock.calledWithUserID, "trial should be created for the new user")
	})

	t.Run("registration succeeds even when trial creation fails", func(t *testing.T) {
		subMock := &mockSubscriptionTrialCreator{
			createErr: errors.New("no active trial plan found"),
		}
		svc := New(
			&mockUserRepo{},
			&mockActivationTokenRepo{},
			&mockSettingsRepo{},
			&mockCategoryRepo{},
			&mockCurrencyRepo{},
			auth.NewJWTService(testConfig().JWTSecret, testConfig().JWTExpirationMinutes),
			nil, // enqueuer
			subMock,
			testConfig(),
			testLogger(),
		)

		result, err := svc.Register("trial-fail@example.com", "password123", "Fail", "Trial")
		require.NoError(t, err, "registration must succeed even if trial creation fails")
		require.NotNil(t, result)
		assert.Equal(t, 42, result.User.ID)
	})

	t.Run("nil subscriptionSvc handled gracefully during registration", func(t *testing.T) {
		svc := New(
			&mockUserRepo{},
			&mockActivationTokenRepo{},
			&mockSettingsRepo{},
			&mockCategoryRepo{},
			&mockCurrencyRepo{},
			auth.NewJWTService(testConfig().JWTSecret, testConfig().JWTExpirationMinutes),
			nil, // enqueuer
			nil, // subscriptionSvc
			testConfig(),
			testLogger(),
		)

		result, err := svc.Register("no-sub@example.com", "password123", "No", "Sub")
		require.NoError(t, err, "registration must succeed with nil subscriptionSvc")
		require.NotNil(t, result)
		assert.Equal(t, 42, result.User.ID)
	})
}

func TestLoginOrRegisterOAuth_TrialSubscription(t *testing.T) {
	t.Run("OAuth registration creates trial subscription for new user", func(t *testing.T) {
		subMock := &mockSubscriptionTrialCreator{}
		svc := New(
			&mockUserRepo{}, // getByEmailFn defaults to sql.ErrNoRows (new user)
			&mockActivationTokenRepo{},
			&mockSettingsRepo{},
			&mockCategoryRepo{},
			&mockCurrencyRepo{},
			auth.NewJWTService(testConfig().JWTSecret, testConfig().JWTExpirationMinutes),
			nil, // enqueuer
			subMock,
			testConfig(),
			testLogger(),
		)

		token, err := svc.LoginOrRegisterOAuth("oauth@example.com", "OAuth", "User")
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Equal(t, 42, subMock.calledWithUserID, "trial should be created for new OAuth user")
	})

	t.Run("OAuth login does not create trial subscription for existing user", func(t *testing.T) {
		subMock := &mockSubscriptionTrialCreator{}
		svc := New(
			&mockUserRepo{
				getByEmailFn: func(email string) (*models.User, error) {
					return &models.User{ID: 99, Email: email, IsActive: true}, nil
				},
			},
			&mockActivationTokenRepo{},
			&mockSettingsRepo{},
			&mockCategoryRepo{},
			&mockCurrencyRepo{},
			auth.NewJWTService(testConfig().JWTSecret, testConfig().JWTExpirationMinutes),
			nil, // enqueuer
			subMock,
			testConfig(),
			testLogger(),
		)

		token, err := svc.LoginOrRegisterOAuth("existing@example.com", "Existing", "User")
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Equal(t, 0, subMock.calledWithUserID, "trial must NOT be created for existing user")
	})

	t.Run("OAuth registration succeeds even when trial creation fails", func(t *testing.T) {
		subMock := &mockSubscriptionTrialCreator{
			createErr: errors.New("db connection lost"),
		}
		svc := New(
			&mockUserRepo{},
			&mockActivationTokenRepo{},
			&mockSettingsRepo{},
			&mockCategoryRepo{},
			&mockCurrencyRepo{},
			auth.NewJWTService(testConfig().JWTSecret, testConfig().JWTExpirationMinutes),
			nil, // enqueuer
			subMock,
			testConfig(),
			testLogger(),
		)

		token, err := svc.LoginOrRegisterOAuth("fail-trial@example.com", "Fail", "Trial")
		require.NoError(t, err, "OAuth registration must succeed even if trial creation fails")
		assert.NotEmpty(t, token)
	})
}
