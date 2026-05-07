package exchange_rates

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-budget/backend/internal/models"
	exchangerates "github.com/go-budget/backend/internal/services/exchange_rates"
	"github.com/go-budget/backend/internal/types"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEnqueuer implements tasks.TaskEnqueuer for unit tests.
type MockEnqueuer struct {
	EnqueueFunc func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

func (m *MockEnqueuer) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	if m.EnqueueFunc != nil {
		return m.EnqueueFunc(task, opts...)
	}
	return &asynq.TaskInfo{}, nil
}

// ==================== Mocks ====================

// MockExchangeRateRepo mocks the ExchangeRateRepository interface.
type MockExchangeRateRepo struct {
	GetAllRatesForDateFunc func(date string) (*types.ExchangeRateSnapshot, error)
	SaveRateFunc           func(rate *models.ExchangeRateHistory) error
}

func (m *MockExchangeRateRepo) GetRate(baseCurrencyCode, targetCurrencyCode string) (*models.ExchangeRateHistory, error) {
	return nil, nil
}

func (m *MockExchangeRateRepo) GetRateForDate(baseCurrencyCode, targetCurrencyCode string, date string) (*models.ExchangeRateHistory, error) {
	return nil, nil
}

func (m *MockExchangeRateRepo) SaveRate(rate *models.ExchangeRateHistory) error {
	if m.SaveRateFunc != nil {
		return m.SaveRateFunc(rate)
	}
	rate.ID = 42
	return nil
}

func (m *MockExchangeRateRepo) GetAllRatesForDate(date string) (*types.ExchangeRateSnapshot, error) {
	if m.GetAllRatesForDateFunc != nil {
		return m.GetAllRatesForDateFunc(date)
	}
	return &types.ExchangeRateSnapshot{
		ID:               1,
		Rates:            map[string]float64{"EUR": 0.92, "GBP": 0.79},
		ActualDate:       date,
		BaseCurrencyCode: "USD",
	}, nil
}

// MockHTTPClient mocks the exchangerates.HTTPClient interface for testing API calls.
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// ==================== Helpers ====================

// currencyBeaconResponse mirrors the API response shape for building mock responses.
type currencyBeaconResponse struct {
	Rates map[string]float64 `json:"rates"`
	Date  string             `json:"date"`
	Base  string             `json:"base"`
}

func setupTestHandler() (*Handler, *echo.Echo, *MockExchangeRateRepo, *MockEnqueuer, *exchangerates.Service) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := &MockExchangeRateRepo{}
	enqueuer := &MockEnqueuer{}
	svc := exchangerates.New(repo, "https://api.currencybeacon.com", "test-api-key", logger)
	handler := New(svc, enqueuer, logger)
	e := echo.New()
	return handler, e, repo, enqueuer, svc
}

func setUserContext(c echo.Context, userID int) {
	c.Set("user_id", userID)
	c.Set("active_user_email", "test@example.com")
	c.Set("active_user_base_currency_id", 1)
	c.Set("active_user_display_name", "Test User")
}

// mockAPIResponse builds a mock HTTP response with the given CurrencyBeacon-shaped body.
func mockAPIResponse(statusCode int, body interface{}) *http.Response {
	data, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(data)),
		Header:     make(http.Header),
	}
}

// ==================== GetRates Tests ====================

func TestGetRatesDBError(t *testing.T) {
	handler, e, repo, _, _ := setupTestHandler()
	repo.GetAllRatesForDateFunc = func(date string) (*types.ExchangeRateSnapshot, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetRates(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetRatesSuccess(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.GetRates(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "EUR")
}

// ==================== UpdateRates Tests ====================

func TestUpdateRatesSuccess(t *testing.T) {
	handler, e, _, enqueuer, _ := setupTestHandler()

	var enqueuedTask *asynq.Task
	enqueuer.EnqueueFunc = func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
		enqueuedTask = task
		return &asynq.TaskInfo{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/update", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateRates(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Exchange rate update task enqueued")
	assert.NotNil(t, enqueuedTask)
	assert.Equal(t, "exchange_rate:update", enqueuedTask.Type())
}

func TestUpdateRatesEnqueueError(t *testing.T) {
	handler, e, _, enqueuer, _ := setupTestHandler()

	enqueuer.EnqueueFunc = func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
		return nil, errors.New("redis down")
	}

	req := httptest.NewRequest(http.MethodGet, "/update", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, 1)

	err := handler.UpdateRates(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to start exchange rate update")
}

// ==================== UpdateRatesRange Tests ====================

func TestUpdateRatesRangeInvalidStartDate(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/update/from/invalid/to/2024-01-31", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("start_date", "end_date")
	c.SetParamValues("invalid", "2024-01-31")
	setUserContext(c, 1)

	err := handler.UpdateRatesRange(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid start_date")
}

func TestUpdateRatesRangeInvalidEndDate(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/update/from/2024-01-01/to/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("start_date", "end_date")
	c.SetParamValues("2024-01-01", "invalid")
	setUserContext(c, 1)

	err := handler.UpdateRatesRange(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid end_date")
}

func TestUpdateRatesRangeEndBeforeStart(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/update/from/2024-01-31/to/2024-01-01", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("start_date", "end_date")
	c.SetParamValues("2024-01-31", "2024-01-01")
	setUserContext(c, 1)

	err := handler.UpdateRatesRange(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "end_date must not be before start_date")
}

func TestUpdateRatesRangeSuccess(t *testing.T) {
	handler, e, _, _, svc := setupTestHandler()

	apiCallCount := 0
	svc.SetHTTPClient(&MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			apiCallCount++
			return mockAPIResponse(http.StatusOK, currencyBeaconResponse{
				Rates: map[string]float64{"EUR": 0.92, "GBP": 0.79},
				Date:  "2024-01-01", // simplified, actual date varies
				Base:  "USD",
			}), nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/update/from/2024-01-01/to/2024-01-03", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("start_date", "end_date")
	c.SetParamValues("2024-01-01", "2024-01-03")
	setUserContext(c, 1)

	err := handler.UpdateRatesRange(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response UpdateRatesRangeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, 3, response.DatesProcessed)
	assert.Equal(t, "2024-01-01", response.StartDate)
	assert.Equal(t, "2024-01-03", response.EndDate)

	// 3 days => 3 API calls
	assert.Equal(t, 3, apiCallCount)
}

func TestUpdateRatesRangeSingleDay(t *testing.T) {
	handler, e, _, _, svc := setupTestHandler()

	svc.SetHTTPClient(&MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return mockAPIResponse(http.StatusOK, currencyBeaconResponse{
				Rates: map[string]float64{"EUR": 0.92},
				Date:  "2024-01-15",
				Base:  "USD",
			}), nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/update/from/2024-01-15/to/2024-01-15", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("start_date", "end_date")
	c.SetParamValues("2024-01-15", "2024-01-15")
	setUserContext(c, 1)

	err := handler.UpdateRatesRange(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response UpdateRatesRangeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, 1, response.DatesProcessed)
}

func TestUpdateRatesRangeAPIFailure(t *testing.T) {
	handler, e, _, _, svc := setupTestHandler()

	callCount := 0
	svc.SetHTTPClient(&MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 2 {
				return nil, errors.New("network error")
			}
			return mockAPIResponse(http.StatusOK, currencyBeaconResponse{
				Rates: map[string]float64{"EUR": 0.92},
				Date:  "2024-01-01",
				Base:  "USD",
			}), nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/update/from/2024-01-01/to/2024-01-03", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("start_date", "end_date")
	c.SetParamValues("2024-01-01", "2024-01-03")
	setUserContext(c, 1)

	err := handler.UpdateRatesRange(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unable to fetch exchange rates for 2024-01-02")
}

func TestUpdateRatesRangeDBSaveFailure(t *testing.T) {
	handler, e, repo, _, svc := setupTestHandler()

	callCount := 0
	repo.SaveRateFunc = func(rate *models.ExchangeRateHistory) error {
		callCount++
		if callCount == 2 {
			return errors.New("db write error")
		}
		rate.ID = callCount
		return nil
	}

	svc.SetHTTPClient(&MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return mockAPIResponse(http.StatusOK, currencyBeaconResponse{
				Rates: map[string]float64{"EUR": 0.92},
				Date:  "2024-01-01",
				Base:  "USD",
			}), nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/update/from/2024-01-01/to/2024-01-03", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("start_date", "end_date")
	c.SetParamValues("2024-01-01", "2024-01-03")
	setUserContext(c, 1)

	err := handler.UpdateRatesRange(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Unable to save exchange rates for 2024-01-02")
}

func TestUpdateRatesRangeAPIKeyNotConfigured(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	repo := &MockExchangeRateRepo{}
	enqueuer := &MockEnqueuer{}
	svc := exchangerates.New(repo, "https://api.currencybeacon.com", "", logger)
	handler := New(svc, enqueuer, logger)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/update/from/2024-01-01/to/2024-01-03", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("start_date", "end_date")
	c.SetParamValues("2024-01-01", "2024-01-03")
	setUserContext(c, 1)

	err := handler.UpdateRatesRange(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ==================== Date Range Limit Tests ====================

func TestUpdateRatesRangeExceedsMaxDays(t *testing.T) {
	handler, e, _, _, _ := setupTestHandler()

	// 366 days exceeds the 365-day limit
	req := httptest.NewRequest(http.MethodGet, "/update/from/2023-01-01/to/2024-01-01", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("start_date", "end_date")
	c.SetParamValues("2023-01-01", "2024-01-01")
	setUserContext(c, 1)

	err := handler.UpdateRatesRange(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Date range must not exceed 365 days")
}

func TestUpdateRatesRangeExactlyMaxDays(t *testing.T) {
	handler, e, _, _, svc := setupTestHandler()

	// Exactly 365 days should be accepted
	svc.SetHTTPClient(&MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return mockAPIResponse(http.StatusOK, currencyBeaconResponse{
				Rates: map[string]float64{"EUR": 0.92},
				Date:  "2024-01-01",
				Base:  "USD",
			}), nil
		},
	})

	// 2024-01-01 to 2024-12-30 = 365 days
	req := httptest.NewRequest(http.MethodGet, "/update/from/2024-01-01/to/2024-12-30", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("start_date", "end_date")
	c.SetParamValues("2024-01-01", "2024-12-30")
	setUserContext(c, 1)

	err := handler.UpdateRatesRange(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}
