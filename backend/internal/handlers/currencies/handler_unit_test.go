package currencies

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	currencyservice "github.com/go-budget/backend/internal/services/currency"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// Mock currency service
type MockCurrencyService struct {
	GetAllFunc func() ([]*currencyservice.Currency, error)
}

func (m *MockCurrencyService) GetAll() ([]*currencyservice.Currency, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc()
	}
	return []*currencyservice.Currency{}, nil
}

func setupTestHandler() (*Handler, *echo.Echo, *MockCurrencyService) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	svc := &MockCurrencyService{}
	handler := New(svc, logger)
	e := echo.New()
	return handler, e, svc
}

func TestGetCurrenciesDBError(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.GetAllFunc = func() ([]*currencyservice.Currency, error) {
		return nil, errors.New("db error")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", 1)

	err := handler.GetCurrencies(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to get currencies")
}

func TestGetCurrenciesSuccess(t *testing.T) {
	handler, e, svc := setupTestHandler()
	svc.GetAllFunc = func() ([]*currencyservice.Currency, error) {
		return []*currencyservice.Currency{
			{ID: 1, Code: "USD", Name: "US Dollar"},
			{ID: 2, Code: "EUR", Name: "Euro"},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", 1)

	err := handler.GetCurrencies(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "USD")
	assert.Contains(t, rec.Body.String(), "EUR")
}

func TestRegisterRoutes(t *testing.T) {
	handler, e, _ := setupTestHandler()

	g := e.Group("/currencies")
	handler.RegisterRoutes(g)

	routes := e.Routes()
	routePaths := make(map[string]bool)
	for _, r := range routes {
		routePaths[r.Path] = true
	}

	assert.True(t, routePaths["/currencies/"])
}
