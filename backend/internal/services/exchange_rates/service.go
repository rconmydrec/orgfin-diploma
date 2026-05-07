package exchange_rates

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/types"
)

// maxResponseBytes is the maximum allowed size for external API response bodies (1 MB).
const maxResponseBytes = 1 << 20

// serviceName is the identifier written to exchange_rates.service_name, matching the Python app.
const serviceName = "CurrencyBeacon"

// apiVersion is the CurrencyBeacon API version path segment.
const apiVersion = "v1"

// currencyBeaconResponse represents the JSON response from the CurrencyBeacon API.
type currencyBeaconResponse struct {
	Rates map[string]float64 `json:"rates"`
	Date  string             `json:"date"`
	Base  string             `json:"base"`
}

// Service encapsulates all exchange rates business logic: fetching rates from
// the CurrencyBeacon API, persisting them, and querying stored rates.
type Service struct {
	exchangeRateRepo ExchangeRateRepository
	apiURL           string
	apiKey           string
	httpClient       HTTPClient
	logger           *slog.Logger
}

// New creates a new exchange rates Service with the given dependencies.
// A default http.Client with a 30s timeout is created for API calls.
func New(
	exchangeRateRepo ExchangeRateRepository,
	apiURL string,
	apiKey string,
	logger *slog.Logger,
) *Service {
	return &Service{
		exchangeRateRepo: exchangeRateRepo,
		apiURL:           apiURL,
		apiKey:           apiKey,
		httpClient:       &http.Client{Timeout: 30 * time.Second},
		logger:           logger,
	}
}

// SetHTTPClient replaces the default HTTP client (used in tests).
func (s *Service) SetHTTPClient(client HTTPClient) {
	s.httpClient = client
}

// GetRatesForDate returns all exchange rates stored for the given date.
func (s *Service) GetRatesForDate(date string) (*types.ExchangeRateSnapshot, error) {
	return s.exchangeRateRepo.GetAllRatesForDate(date)
}

// FetchAndSaveRates fetches exchange rates from the CurrencyBeacon API for the
// given date and saves them to the database. It returns a FetchResult with
// metadata about the operation.
func (s *Service) FetchAndSaveRates(date string) (*FetchResult, error) {
	apiResp, err := s.fetchFromAPI(date)
	if err != nil {
		return nil, err
	}

	if err := s.saveRates(apiResp); err != nil {
		return nil, err
	}

	return &FetchResult{
		BaseCurrency: apiResp.Base,
		RateCount:    len(apiResp.Rates),
	}, nil
}

// FetchAndSaveRatesRange fetches and saves exchange rates for each day in the
// given date range (inclusive). It validates the date format, range order, and
// maximum range length. Processing stops on the first error.
func (s *Service) FetchAndSaveRatesRange(startDateStr, endDateStr string) (*FetchAndSaveRatesRangeResult, error) {
	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidStartDate, err)
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidEndDate, err)
	}

	if endDate.Before(startDate) {
		return nil, ErrEndBeforeStart
	}

	daysDiff := int(endDate.Sub(startDate).Hours()/24) + 1
	if daysDiff > MaxDateRangeDays {
		return nil, ErrDateRangeExceeded
	}

	datesProcessed := 0
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")

		apiResp, fetchErr := s.fetchFromAPI(dateStr)
		if fetchErr != nil {
			return nil, &DateError{Date: dateStr, Operation: "fetch", Err: fetchErr}
		}

		if saveErr := s.saveRates(apiResp); saveErr != nil {
			return nil, &DateError{Date: dateStr, Operation: "save", Err: saveErr}
		}

		datesProcessed++
	}

	return &FetchAndSaveRatesRangeResult{
		DatesProcessed: datesProcessed,
		StartDate:      startDateStr,
		EndDate:        endDateStr,
	}, nil
}

// fetchFromAPI calls the CurrencyBeacon historical endpoint for the given date
// and returns the parsed response.
func (s *Service) fetchFromAPI(date string) (*currencyBeaconResponse, error) {
	if s.apiKey == "" {
		return nil, ErrAPIKeyNotConfigured
	}

	url := fmt.Sprintf("%s/%s/historical?api_key=%s&date=%s",
		s.apiURL, apiVersion, s.apiKey, date)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rates: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CurrencyBeacon API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp currencyBeaconResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	return &apiResp, nil
}

// saveRates persists a CurrencyBeacon API response to the exchange_rates table.
func (s *Service) saveRates(apiResp *currencyBeaconResponse) error {
	actualDate, err := time.Parse("2006-01-02", apiResp.Date)
	if err != nil {
		return fmt.Errorf("failed to parse date from API response: %w", err)
	}

	rate := &models.ExchangeRateHistory{
		Rates:            apiResp.Rates,
		ActualDate:       actualDate,
		BaseCurrencyCode: apiResp.Base,
		ServiceName:      serviceName,
	}

	if err := s.exchangeRateRepo.SaveRate(rate); err != nil {
		return fmt.Errorf("failed to save rates: %w", err)
	}

	return nil
}
