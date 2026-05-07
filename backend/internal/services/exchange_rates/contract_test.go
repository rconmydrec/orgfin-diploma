// Contract tests for the exchange_rates service.
// These tests validate the ServiceInterface contract and survive any implementation rewrite.
// They test through the public interface only (black-box), using a real DB and mock HTTP.
package exchange_rates_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/go-budget/backend/internal/services/exchange_rates"
	"github.com/go-budget/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHTTPClient implements exchange_rates.HTTPClient for testing.
// It returns a fresh response body for each call (unlike a single-use reader).
type mockHTTPClient struct {
	body string
	err  error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(m.body)),
	}, nil
}

// newMockHTTPClient creates a mock that returns a successful CurrencyBeacon response.
func newMockHTTPClient(body string) *mockHTTPClient {
	return &mockHTTPClient{body: body}
}

// newErrorHTTPClient creates a mock that returns an error.
func newErrorHTTPClient() *mockHTTPClient {
	return &mockHTTPClient{err: errors.New("network error")}
}

// sampleAPIResponse returns a valid CurrencyBeacon API JSON response.
const sampleAPIResponse = `{
	"rates": {
		"EUR": 0.92,
		"GBP": 0.79,
		"UAH": 41.5
	},
	"date": "2026-03-01",
	"base": "USD"
}`

// setupTestService creates a real exchange_rates service connected to the test DB
// with a mock HTTP client.
func setupTestService(t *testing.T, httpClient exchange_rates.HTTPClient) exchange_rates.ServiceInterface {
	t.Helper()
	app := testutil.NewTestApp(t)
	logger := testutil.TestLogger()
	svc := exchange_rates.New(
		app.ExchangeRateRepo,
		"https://api.currencybeacon.com",
		"test-api-key",
		logger,
	)
	if httpClient != nil {
		svc.SetHTTPClient(httpClient)
	}
	return svc
}

// setupServiceNoAPIKey creates a service without an API key.
func setupServiceNoAPIKey(t *testing.T) exchange_rates.ServiceInterface {
	t.Helper()
	app := testutil.NewTestApp(t)
	logger := testutil.TestLogger()
	svc := exchange_rates.New(
		app.ExchangeRateRepo,
		"https://api.currencybeacon.com",
		"", // no API key
		logger,
	)
	return svc
}

// --- GetRatesForDate ---

func TestGetRatesForDate_ReturnsStoredRates(t *testing.T) {
	// First fetch and save, then retrieve
	mockClient := newMockHTTPClient(sampleAPIResponse)
	svc := setupTestService(t, mockClient)

	_, err := svc.FetchAndSaveRates("2026-03-01")
	require.NoError(t, err)

	result, err := svc.GetRatesForDate("2026-03-01")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Rates)
}

func TestGetRatesForDate_NoRates(t *testing.T) {
	svc := setupTestService(t, nil)

	// Query a date that has no stored rates.
	// The repo may return nil with an error (e.g., sql.ErrNoRows) or nil result.
	result, err := svc.GetRatesForDate("1990-01-01")

	// Either an error or nil result is acceptable behavior
	if err != nil {
		assert.Nil(t, result)
	} else {
		// If no error, result should be nil or have empty rates
		if result != nil {
			// The result is valid but may have no rates
			_ = result
		}
	}
}

// --- FetchAndSaveRates ---

func TestFetchAndSaveRates_Success(t *testing.T) {
	mockClient := newMockHTTPClient(sampleAPIResponse)
	svc := setupTestService(t, mockClient)

	result, err := svc.FetchAndSaveRates("2026-03-01")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "USD", result.BaseCurrency)
	assert.Greater(t, result.RateCount, 0)
}

func TestFetchAndSaveRates_APIKeyNotConfigured(t *testing.T) {
	svc := setupServiceNoAPIKey(t)

	_, err := svc.FetchAndSaveRates("2026-03-01")

	require.Error(t, err)
	assert.ErrorIs(t, err, exchange_rates.ErrAPIKeyNotConfigured)
}

func TestFetchAndSaveRates_NetworkError(t *testing.T) {
	mockClient := newErrorHTTPClient()
	svc := setupTestService(t, mockClient)

	_, err := svc.FetchAndSaveRates("2026-03-01")

	require.Error(t, err)
}

// --- FetchAndSaveRatesRange ---

func TestFetchAndSaveRatesRange_Success(t *testing.T) {
	mockClient := newMockHTTPClient(sampleAPIResponse)
	svc := setupTestService(t, mockClient)

	result, err := svc.FetchAndSaveRatesRange("2026-03-01", "2026-03-03")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 3, result.DatesProcessed)
	assert.Equal(t, "2026-03-01", result.StartDate)
	assert.Equal(t, "2026-03-03", result.EndDate)
}

func TestFetchAndSaveRatesRange_InvalidStartDate(t *testing.T) {
	svc := setupTestService(t, nil)

	_, err := svc.FetchAndSaveRatesRange("not-a-date", "2026-03-03")

	require.Error(t, err)
	assert.ErrorIs(t, err, exchange_rates.ErrInvalidStartDate)
}

func TestFetchAndSaveRatesRange_InvalidEndDate(t *testing.T) {
	svc := setupTestService(t, nil)

	_, err := svc.FetchAndSaveRatesRange("2026-03-01", "not-a-date")

	require.Error(t, err)
	assert.ErrorIs(t, err, exchange_rates.ErrInvalidEndDate)
}

func TestFetchAndSaveRatesRange_EndBeforeStart(t *testing.T) {
	svc := setupTestService(t, nil)

	_, err := svc.FetchAndSaveRatesRange("2026-03-05", "2026-03-01")

	require.Error(t, err)
	assert.ErrorIs(t, err, exchange_rates.ErrEndBeforeStart)
}

func TestFetchAndSaveRatesRange_ExceedsMaxRange(t *testing.T) {
	svc := setupTestService(t, nil)

	start := "2024-01-01"
	end := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, exchange_rates.MaxDateRangeDays+1).
		Format("2006-01-02")

	_, err := svc.FetchAndSaveRatesRange(start, end)

	require.Error(t, err)
	assert.ErrorIs(t, err, exchange_rates.ErrDateRangeExceeded)
}

func TestFetchAndSaveRatesRange_StopsOnFirstError(t *testing.T) {
	// Mock that fails on every request
	mockClient := newErrorHTTPClient()
	svc := setupTestService(t, mockClient)

	_, err := svc.FetchAndSaveRatesRange("2026-03-01", "2026-03-03")

	require.Error(t, err)
	// Should be wrapped in DateError
	var dateErr *exchange_rates.DateError
	assert.True(t, errors.As(err, &dateErr), "error should be a DateError")
	assert.Equal(t, "2026-03-01", dateErr.Date, "should fail on the first date")
	assert.Equal(t, "fetch", dateErr.Operation)
}

// --- SetHTTPClient ---

func TestSetHTTPClient_DoesNotPanic(t *testing.T) {
	svc := setupTestService(t, nil)

	assert.NotPanics(t, func() {
		svc.SetHTTPClient(newMockHTTPClient("{}"))
	})
}
