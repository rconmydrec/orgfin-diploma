package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/go-budget/backend/internal/models"
	exchangerates "github.com/go-budget/backend/internal/services/exchange_rates"
	"github.com/go-budget/backend/internal/types"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExchangeRateRepoFull implements exchange_rates.ExchangeRateRepository for testing.
type mockExchangeRateRepoFull struct {
	SaveRateFunc           func(rate *models.ExchangeRateHistory) error
	GetAllRatesForDateFunc func(date string) (*types.ExchangeRateSnapshot, error)
	SavedRates             []*models.ExchangeRateHistory
}

func (m *mockExchangeRateRepoFull) GetRate(baseCurrencyCode, targetCurrencyCode string) (*models.ExchangeRateHistory, error) {
	return nil, nil
}

func (m *mockExchangeRateRepoFull) GetRateForDate(baseCurrencyCode, targetCurrencyCode string, date string) (*models.ExchangeRateHistory, error) {
	return nil, nil
}

func (m *mockExchangeRateRepoFull) SaveRate(rate *models.ExchangeRateHistory) error {
	m.SavedRates = append(m.SavedRates, rate)
	if m.SaveRateFunc != nil {
		return m.SaveRateFunc(rate)
	}
	return nil
}

func (m *mockExchangeRateRepoFull) GetAllRatesForDate(date string) (*types.ExchangeRateSnapshot, error) {
	if m.GetAllRatesForDateFunc != nil {
		return m.GetAllRatesForDateFunc(date)
	}
	return &types.ExchangeRateSnapshot{}, nil
}

// mockHTTPClientForSvc implements exchangerates.HTTPClient for testing.
type mockHTTPClientForSvc struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClientForSvc) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, errors.New("no DoFunc configured")
}

// currencyBeaconTestResponse is used to build mock API responses.
type currencyBeaconTestResponse struct {
	Rates map[string]float64 `json:"rates"`
	Date  string             `json:"date"`
	Base  string             `json:"base"`
}

func makeTestAPIResponse(rates map[string]float64, date, base string) *http.Response {
	resp := currencyBeaconTestResponse{
		Rates: rates,
		Date:  date,
		Base:  base,
	}
	body, _ := json.Marshal(resp)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func newTestExchangeRateSvc(repo *mockExchangeRateRepoFull, httpClient exchangerates.HTTPClient) *exchangerates.Service {
	svc := exchangerates.New(repo, "https://api.currencybeacon.com", "test-key", testLogger())
	if httpClient != nil {
		svc.SetHTTPClient(httpClient)
	}
	return svc
}

// ==================== NewExchangeRateUpdateTask Tests ====================

func TestNewExchangeRateUpdateTask(t *testing.T) {
	task := NewExchangeRateUpdateTask()
	assert.Equal(t, TypeExchangeRateUpdate, task.Type())
	assert.Nil(t, task.Payload())
}

// ==================== ExchangeRateUpdateHandler ProcessTask Tests ====================

func TestExchangeRateUpdateHandler_ProcessTask_Success(t *testing.T) {
	rates := map[string]float64{"EUR": 0.85, "GBP": 0.73}
	httpClient := &mockHTTPClientForSvc{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.String(), "currencybeacon.com")
			assert.Contains(t, req.URL.String(), "api_key=test-key")
			return makeTestAPIResponse(rates, "2026-02-22", "USD"), nil
		},
	}

	repo := &mockExchangeRateRepoFull{}
	svc := newTestExchangeRateSvc(repo, httpClient)

	handler := NewExchangeRateUpdateHandler(svc, &mockTaskEnqueuer{}, NotificationConfig{}, testLogger())

	task := asynq.NewTask(TypeExchangeRateUpdate, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.NoError(t, err)

	require.Len(t, repo.SavedRates, 1)
	assert.Equal(t, "USD", repo.SavedRates[0].BaseCurrencyCode)
	assert.Equal(t, "CurrencyBeacon", repo.SavedRates[0].ServiceName)
	assert.Equal(t, rates, repo.SavedRates[0].Rates)
}

func TestExchangeRateUpdateHandler_ProcessTask_NoAPIKey(t *testing.T) {
	repo := &mockExchangeRateRepoFull{}
	svc := exchangerates.New(repo, "https://api.currencybeacon.com", "", testLogger())

	handler := NewExchangeRateUpdateHandler(svc, &mockTaskEnqueuer{}, NotificationConfig{}, testLogger())

	task := asynq.NewTask(TypeExchangeRateUpdate, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key is not configured")
}

func TestExchangeRateUpdateHandler_ProcessTask_APIError(t *testing.T) {
	httpClient := &mockHTTPClientForSvc{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("connection timeout")
		},
	}

	repo := &mockExchangeRateRepoFull{}
	svc := newTestExchangeRateSvc(repo, httpClient)

	handler := NewExchangeRateUpdateHandler(svc, &mockTaskEnqueuer{}, NotificationConfig{}, testLogger())

	task := asynq.NewTask(TypeExchangeRateUpdate, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetch rates")
}

func TestExchangeRateUpdateHandler_ProcessTask_APIBadStatus(t *testing.T) {
	httpClient := &mockHTTPClientForSvc{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(bytes.NewReader([]byte("rate limited"))),
			}, nil
		},
	}

	repo := &mockExchangeRateRepoFull{}
	svc := newTestExchangeRateSvc(repo, httpClient)

	handler := NewExchangeRateUpdateHandler(svc, &mockTaskEnqueuer{}, NotificationConfig{}, testLogger())

	task := asynq.NewTask(TypeExchangeRateUpdate, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 429")
}

func TestExchangeRateUpdateHandler_ProcessTask_APIBadJSON(t *testing.T) {
	httpClient := &mockHTTPClientForSvc{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte("not json"))),
			}, nil
		},
	}

	repo := &mockExchangeRateRepoFull{}
	svc := newTestExchangeRateSvc(repo, httpClient)

	handler := NewExchangeRateUpdateHandler(svc, &mockTaskEnqueuer{}, NotificationConfig{}, testLogger())

	task := asynq.NewTask(TypeExchangeRateUpdate, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse API response")
}

func TestExchangeRateUpdateHandler_ProcessTask_SaveError(t *testing.T) {
	httpClient := &mockHTTPClientForSvc{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return makeTestAPIResponse(map[string]float64{"EUR": 0.85}, "2026-02-22", "USD"), nil
		},
	}

	repo := &mockExchangeRateRepoFull{
		SaveRateFunc: func(rate *models.ExchangeRateHistory) error {
			return errors.New("unique constraint violation")
		},
	}
	svc := newTestExchangeRateSvc(repo, httpClient)

	handler := NewExchangeRateUpdateHandler(svc, &mockTaskEnqueuer{}, NotificationConfig{}, testLogger())

	task := asynq.NewTask(TypeExchangeRateUpdate, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save rates")
}

func TestExchangeRateUpdateHandler_ProcessTask_BadDateInResponse(t *testing.T) {
	resp := currencyBeaconTestResponse{
		Rates: map[string]float64{"EUR": 0.85},
		Date:  "not-a-date",
		Base:  "USD",
	}
	body, _ := json.Marshal(resp)
	httpClient := &mockHTTPClientForSvc{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
		},
	}

	repo := &mockExchangeRateRepoFull{}
	svc := newTestExchangeRateSvc(repo, httpClient)

	handler := NewExchangeRateUpdateHandler(svc, &mockTaskEnqueuer{}, NotificationConfig{}, testLogger())

	task := asynq.NewTask(TypeExchangeRateUpdate, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save rates")
}

// ==================== ExchangeRateUpdateHandler Notification Tests ====================

func makeSuccessHTTPClient(t *testing.T) *mockHTTPClientForSvc {
	return &mockHTTPClientForSvc{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return makeTestAPIResponse(
				map[string]float64{"EUR": 0.85, "GBP": 0.73},
				"2026-02-22",
				"USD",
			), nil
		},
	}
}

func TestExchangeRateUpdateHandler_ProcessTask_SuccessWithNotification(t *testing.T) {
	repo := &mockExchangeRateRepoFull{}
	httpClient := makeSuccessHTTPClient(t)
	svc := newTestExchangeRateSvc(repo, httpClient)

	notifCfg := NotificationConfig{
		AdminEmails: []string{"admin@example.com"},
		Environment: "production",
	}
	enqueuer := &mockTaskEnqueuer{}

	handler := NewExchangeRateUpdateHandler(svc, enqueuer, notifCfg, testLogger())

	task := asynq.NewTask(TypeExchangeRateUpdate, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.NoError(t, err)

	// Verify one email:send task was enqueued
	require.Len(t, enqueuer.Tasks, 1)
	assert.Equal(t, TypeEmailSend, enqueuer.Tasks[0].Type)

	var emailPayload EmailSendPayload
	err = json.Unmarshal(enqueuer.Tasks[0].Payload, &emailPayload)
	require.NoError(t, err)
	assert.Equal(t, "admin@example.com", emailPayload.To)
	assert.Equal(t, "[production] Currency rates updated", emailPayload.Subject)
	assert.Contains(t, emailPayload.Body, "production")
	assert.Contains(t, emailPayload.Body, "USD")
	assert.Contains(t, emailPayload.Body, "2")
	assert.Empty(t, emailPayload.AttachmentData)
	assert.Empty(t, emailPayload.AttachmentName)
}

func TestExchangeRateUpdateHandler_ProcessTask_SuccessWithMultipleAdmins(t *testing.T) {
	repo := &mockExchangeRateRepoFull{}
	httpClient := makeSuccessHTTPClient(t)
	svc := newTestExchangeRateSvc(repo, httpClient)

	notifCfg := NotificationConfig{
		AdminEmails: []string{"admin1@example.com", "admin2@example.com"},
		Environment: "staging",
	}
	enqueuer := &mockTaskEnqueuer{}

	handler := NewExchangeRateUpdateHandler(svc, enqueuer, notifCfg, testLogger())

	task := asynq.NewTask(TypeExchangeRateUpdate, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.NoError(t, err)

	// Verify two email:send tasks were enqueued
	require.Len(t, enqueuer.Tasks, 2)

	var payload1 EmailSendPayload
	err = json.Unmarshal(enqueuer.Tasks[0].Payload, &payload1)
	require.NoError(t, err)
	assert.Equal(t, "admin1@example.com", payload1.To)

	var payload2 EmailSendPayload
	err = json.Unmarshal(enqueuer.Tasks[1].Payload, &payload2)
	require.NoError(t, err)
	assert.Equal(t, "admin2@example.com", payload2.To)
}

func TestExchangeRateUpdateHandler_ProcessTask_SuccessWithNoAdmins(t *testing.T) {
	repo := &mockExchangeRateRepoFull{}
	httpClient := makeSuccessHTTPClient(t)
	svc := newTestExchangeRateSvc(repo, httpClient)

	notifCfg := NotificationConfig{
		AdminEmails: []string{},
		Environment: "development",
	}
	enqueuer := &mockTaskEnqueuer{}

	handler := NewExchangeRateUpdateHandler(svc, enqueuer, notifCfg, testLogger())

	task := asynq.NewTask(TypeExchangeRateUpdate, nil)
	err := handler.ProcessTask(context.Background(), task)
	assert.NoError(t, err)

	// Verify no tasks were enqueued
	assert.Empty(t, enqueuer.Tasks)
}

func TestExchangeRateUpdateHandler_ProcessTask_SuccessWithEnqueueFailure(t *testing.T) {
	repo := &mockExchangeRateRepoFull{}
	httpClient := makeSuccessHTTPClient(t)
	svc := newTestExchangeRateSvc(repo, httpClient)

	notifCfg := NotificationConfig{
		AdminEmails: []string{"admin@example.com"},
		Environment: "production",
	}
	enqueuer := &mockTaskEnqueuer{
		EnqueueFunc: func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
			return nil, errors.New("redis connection refused")
		},
	}

	handler := NewExchangeRateUpdateHandler(svc, enqueuer, notifCfg, testLogger())

	task := asynq.NewTask(TypeExchangeRateUpdate, nil)
	err := handler.ProcessTask(context.Background(), task)
	// Rate update should still succeed even though email enqueue failed
	assert.NoError(t, err)

	// Verify enqueue was actually attempted
	require.Len(t, enqueuer.Tasks, 1)
	assert.Equal(t, TypeEmailSend, enqueuer.Tasks[0].Type)
}
