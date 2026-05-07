package exchange_rates

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Mocks ====================

type mockExchangeRateRepo struct {
	GetAllRatesForDateFunc func(date string) (*types.ExchangeRateSnapshot, error)
	SaveRateFunc           func(rate *models.ExchangeRateHistory) error
	SavedRates             []*models.ExchangeRateHistory
}

func (m *mockExchangeRateRepo) GetRate(baseCurrencyCode, targetCurrencyCode string) (*models.ExchangeRateHistory, error) {
	return nil, nil
}

func (m *mockExchangeRateRepo) GetRateForDate(baseCurrencyCode, targetCurrencyCode string, date string) (*models.ExchangeRateHistory, error) {
	return nil, nil
}

func (m *mockExchangeRateRepo) SaveRate(rate *models.ExchangeRateHistory) error {
	m.SavedRates = append(m.SavedRates, rate)
	if m.SaveRateFunc != nil {
		return m.SaveRateFunc(rate)
	}
	rate.ID = 42
	return nil
}

func (m *mockExchangeRateRepo) GetAllRatesForDate(date string) (*types.ExchangeRateSnapshot, error) {
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

type mockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, errors.New("no DoFunc configured")
}

// ==================== Helpers ====================

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func newTestService(repo *mockExchangeRateRepo, httpClient HTTPClient) *Service {
	svc := New(repo, "https://api.currencybeacon.com", "test-api-key", testLogger())
	if httpClient != nil {
		svc.SetHTTPClient(httpClient)
	}
	return svc
}

func makeAPIResponse(rates map[string]float64, date, base string) *http.Response {
	resp := currencyBeaconResponse{
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

// ==================== GetRatesForDate Tests ====================

func TestGetRatesForDate_Success(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	svc := newTestService(repo, nil)

	result, err := svc.GetRatesForDate("2024-06-15")
	require.NoError(t, err)
	assert.Equal(t, 1, result.ID)
	assert.Equal(t, "USD", result.BaseCurrencyCode)
	assert.Equal(t, "2024-06-15", result.ActualDate)
	assert.InDelta(t, 0.92, result.Rates["EUR"], 0.001)
}

func TestGetRatesForDate_RepoError(t *testing.T) {
	repo := &mockExchangeRateRepo{
		GetAllRatesForDateFunc: func(date string) (*types.ExchangeRateSnapshot, error) {
			return nil, errors.New("db connection error")
		},
	}
	svc := newTestService(repo, nil)

	result, err := svc.GetRatesForDate("2024-06-15")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "db connection error")
}

// ==================== FetchAndSaveRates Tests ====================

func TestFetchAndSaveRates_Success(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	rates := map[string]float64{"EUR": 0.85, "GBP": 0.73}
	httpClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return makeAPIResponse(rates, "2024-06-15", "USD"), nil
		},
	}
	svc := newTestService(repo, httpClient)

	result, err := svc.FetchAndSaveRates("2024-06-15")
	require.NoError(t, err)
	assert.Equal(t, "USD", result.BaseCurrency)
	assert.Equal(t, 2, result.RateCount)

	require.Len(t, repo.SavedRates, 1)
	assert.Equal(t, "USD", repo.SavedRates[0].BaseCurrencyCode)
	assert.Equal(t, "CurrencyBeacon", repo.SavedRates[0].ServiceName)
	assert.Equal(t, rates, repo.SavedRates[0].Rates)
}

func TestFetchAndSaveRates_APIKeyNotConfigured(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	svc := New(repo, "https://api.currencybeacon.com", "", testLogger())

	result, err := svc.FetchAndSaveRates("2024-06-15")
	assert.ErrorIs(t, err, ErrAPIKeyNotConfigured)
	assert.Nil(t, result)
}

func TestFetchAndSaveRates_HTTPClientDoError(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	httpClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("connection timeout")
		},
	}
	svc := newTestService(repo, httpClient)

	result, err := svc.FetchAndSaveRates("2024-06-15")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to fetch rates")
}

func TestFetchAndSaveRates_Non200Status(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	httpClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(bytes.NewReader([]byte("rate limited"))),
			}, nil
		},
	}
	svc := newTestService(repo, httpClient)

	result, err := svc.FetchAndSaveRates("2024-06-15")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "status 429")
}

func TestFetchAndSaveRates_InvalidJSON(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	httpClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte("not json"))),
			}, nil
		},
	}
	svc := newTestService(repo, httpClient)

	result, err := svc.FetchAndSaveRates("2024-06-15")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse API response")
}

func TestFetchAndSaveRates_InvalidDateInResponse(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	resp := currencyBeaconResponse{
		Rates: map[string]float64{"EUR": 0.85},
		Date:  "not-a-date",
		Base:  "USD",
	}
	body, _ := json.Marshal(resp)
	httpClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
			}, nil
		},
	}
	svc := newTestService(repo, httpClient)

	result, err := svc.FetchAndSaveRates("2024-06-15")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to parse date from API response")
}

func TestFetchAndSaveRates_RepoSaveError(t *testing.T) {
	repo := &mockExchangeRateRepo{
		SaveRateFunc: func(rate *models.ExchangeRateHistory) error {
			return errors.New("unique constraint violation")
		},
	}
	httpClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return makeAPIResponse(map[string]float64{"EUR": 0.85}, "2024-06-15", "USD"), nil
		},
	}
	svc := newTestService(repo, httpClient)

	result, err := svc.FetchAndSaveRates("2024-06-15")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to save rates")
}

func TestFetchAndSaveRates_URLConstructedCorrectly(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	var capturedURL string
	httpClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			capturedURL = req.URL.String()
			return makeAPIResponse(map[string]float64{"EUR": 0.92}, "2024-06-15", "USD"), nil
		},
	}
	svc := newTestService(repo, httpClient)

	_, err := svc.FetchAndSaveRates("2024-06-15")
	require.NoError(t, err)
	assert.Equal(t, "https://api.currencybeacon.com/v1/historical?api_key=test-api-key&date=2024-06-15", capturedURL)
}

func TestFetchAndSaveRates_ServiceNameSetCorrectly(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	httpClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return makeAPIResponse(map[string]float64{"EUR": 0.92}, "2024-06-15", "USD"), nil
		},
	}
	svc := newTestService(repo, httpClient)

	_, err := svc.FetchAndSaveRates("2024-06-15")
	require.NoError(t, err)

	require.Len(t, repo.SavedRates, 1)
	assert.Equal(t, "CurrencyBeacon", repo.SavedRates[0].ServiceName)
	assert.Equal(t, "USD", repo.SavedRates[0].BaseCurrencyCode)
}

// ==================== FetchAndSaveRatesRange Tests ====================

func TestFetchAndSaveRatesRange_InvalidStartDate(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	svc := newTestService(repo, nil)

	result, err := svc.FetchAndSaveRatesRange("invalid", "2024-01-31")
	assert.ErrorIs(t, err, ErrInvalidStartDate)
	assert.Nil(t, result)
}

func TestFetchAndSaveRatesRange_InvalidEndDate(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	svc := newTestService(repo, nil)

	result, err := svc.FetchAndSaveRatesRange("2024-01-01", "invalid")
	assert.ErrorIs(t, err, ErrInvalidEndDate)
	assert.Nil(t, result)
}

func TestFetchAndSaveRatesRange_EndBeforeStart(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	svc := newTestService(repo, nil)

	result, err := svc.FetchAndSaveRatesRange("2024-01-31", "2024-01-01")
	assert.ErrorIs(t, err, ErrEndBeforeStart)
	assert.Nil(t, result)
}

func TestFetchAndSaveRatesRange_ExceedsMaxDays(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	svc := newTestService(repo, nil)

	// 366 days
	result, err := svc.FetchAndSaveRatesRange("2023-01-01", "2024-01-01")
	assert.ErrorIs(t, err, ErrDateRangeExceeded)
	assert.Nil(t, result)
}

func TestFetchAndSaveRatesRange_ExactlyMaxDays(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	httpClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return makeAPIResponse(map[string]float64{"EUR": 0.92}, "2024-01-01", "USD"), nil
		},
	}
	svc := newTestService(repo, httpClient)

	// 2024-01-01 to 2024-12-30 = 365 days
	result, err := svc.FetchAndSaveRatesRange("2024-01-01", "2024-12-30")
	require.NoError(t, err)
	assert.Equal(t, 365, result.DatesProcessed)
	assert.Equal(t, "2024-01-01", result.StartDate)
	assert.Equal(t, "2024-12-30", result.EndDate)
}

func TestFetchAndSaveRatesRange_SingleDay(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	httpClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return makeAPIResponse(map[string]float64{"EUR": 0.92}, "2024-01-15", "USD"), nil
		},
	}
	svc := newTestService(repo, httpClient)

	result, err := svc.FetchAndSaveRatesRange("2024-01-15", "2024-01-15")
	require.NoError(t, err)
	assert.Equal(t, 1, result.DatesProcessed)
}

func TestFetchAndSaveRatesRange_MultiDay(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	apiCallCount := 0
	httpClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			apiCallCount++
			return makeAPIResponse(map[string]float64{"EUR": 0.92}, "2024-01-01", "USD"), nil
		},
	}
	svc := newTestService(repo, httpClient)

	result, err := svc.FetchAndSaveRatesRange("2024-01-01", "2024-01-03")
	require.NoError(t, err)
	assert.Equal(t, 3, result.DatesProcessed)
	assert.Equal(t, 3, apiCallCount)
}

func TestFetchAndSaveRatesRange_APIFailureMidRange(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	callCount := 0
	httpClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 2 {
				return nil, errors.New("network error")
			}
			return makeAPIResponse(map[string]float64{"EUR": 0.92}, "2024-01-01", "USD"), nil
		},
	}
	svc := newTestService(repo, httpClient)

	result, err := svc.FetchAndSaveRatesRange("2024-01-01", "2024-01-03")
	assert.Error(t, err)
	assert.Nil(t, result)

	var dateErr *DateError
	require.True(t, errors.As(err, &dateErr))
	assert.Equal(t, "2024-01-02", dateErr.Date)
	assert.Equal(t, "fetch", dateErr.Operation)
}

func TestFetchAndSaveRatesRange_SaveFailureMidRange(t *testing.T) {
	callCount := 0
	repo := &mockExchangeRateRepo{
		SaveRateFunc: func(rate *models.ExchangeRateHistory) error {
			callCount++
			if callCount == 2 {
				return errors.New("db write error")
			}
			rate.ID = callCount
			return nil
		},
	}
	httpClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return makeAPIResponse(map[string]float64{"EUR": 0.92}, "2024-01-01", "USD"), nil
		},
	}
	svc := newTestService(repo, httpClient)

	result, err := svc.FetchAndSaveRatesRange("2024-01-01", "2024-01-03")
	assert.Error(t, err)
	assert.Nil(t, result)

	var dateErr *DateError
	require.True(t, errors.As(err, &dateErr))
	assert.Equal(t, "2024-01-02", dateErr.Date)
	assert.Equal(t, "save", dateErr.Operation)
}

func TestFetchAndSaveRatesRange_APIKeyNotConfigured(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	svc := New(repo, "https://api.currencybeacon.com", "", testLogger())

	result, err := svc.FetchAndSaveRatesRange("2024-01-01", "2024-01-03")
	assert.Error(t, err)
	assert.Nil(t, result)

	// Should be a DateError wrapping ErrAPIKeyNotConfigured
	var dateErr *DateError
	require.True(t, errors.As(err, &dateErr))
	assert.Equal(t, "2024-01-01", dateErr.Date)
	assert.Equal(t, "fetch", dateErr.Operation)
	assert.ErrorIs(t, dateErr.Err, ErrAPIKeyNotConfigured)
}

// ==================== SetHTTPClient Tests ====================

func TestSetHTTPClient(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	svc := New(repo, "https://api.currencybeacon.com", "test-key", testLogger())

	// Service should have default client
	assert.NotNil(t, svc.httpClient)

	// Set custom client
	customClient := &mockHTTPClient{}
	svc.SetHTTPClient(customClient)
	assert.Equal(t, customClient, svc.httpClient)
}

// ==================== DateError Tests ====================

func TestDateError_Error(t *testing.T) {
	err := &DateError{
		Date:      "2024-01-15",
		Operation: "fetch",
		Err:       errors.New("connection timeout"),
	}
	assert.Equal(t, "fetch rates for 2024-01-15: connection timeout", err.Error())
}

func TestDateError_Unwrap(t *testing.T) {
	inner := errors.New("connection timeout")
	err := &DateError{
		Date:      "2024-01-15",
		Operation: "fetch",
		Err:       inner,
	}
	assert.Equal(t, inner, errors.Unwrap(err))
}

func TestDateError_UnwrapSentinel(t *testing.T) {
	err := &DateError{
		Date:      "2024-01-15",
		Operation: "fetch",
		Err:       ErrAPIKeyNotConfigured,
	}
	assert.ErrorIs(t, err, ErrAPIKeyNotConfigured)
}

// ==================== Constants Tests ====================

func TestConstants(t *testing.T) {
	assert.Equal(t, 365, MaxDateRangeDays)
	assert.Equal(t, 1<<20, maxResponseBytes)
	assert.Equal(t, "CurrencyBeacon", serviceName)
	assert.Equal(t, "v1", apiVersion)
}

// ==================== FetchAndSaveRates Body Read Error ====================

func TestFetchAndSaveRates_BodyReadError(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	httpClient := &mockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(&errorReader{}),
			}, nil
		},
	}
	svc := newTestService(repo, httpClient)

	result, err := svc.FetchAndSaveRates("2024-06-15")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to read response body")
}

// errorReader always returns an error on Read.
type errorReader struct{}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, errors.New("read error")
}

// ==================== FetchAndSaveRates Malformed URL ====================

func TestFetchAndSaveRates_MalformedURL(t *testing.T) {
	repo := &mockExchangeRateRepo{}
	// Use a URL that will cause http.NewRequest to fail
	svc := New(repo, "://invalid-url", "test-key", testLogger())

	result, err := svc.FetchAndSaveRates("2024-06-15")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create request")
}
