package scenarios_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-budget/backend/internal/testutil"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

// makeRequestWithHeaders creates an HTTP request with the provided headers.
func makeRequestWithHeaders(t *testing.T, method, path, body string, headers map[string]string) *http.Request {
	t.Helper()

	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

// executeRequest sends a request through the test app's Echo router.
func executeRequest(t *testing.T, app *testutil.TestApp, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec, req)
	return rec
}

// doRequest is a shared helper that executes an HTTP request through the Echo router.
func doRequest(t *testing.T, app *testutil.TestApp, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if token != "" {
		req.Header.Set("auth-token", token)
	}
	rec := httptest.NewRecorder()
	app.Echo.ServeHTTP(rec, req)
	return rec
}

// createTx is a helper that creates a transaction via the HTTP API.
func createTx(t *testing.T, app *testutil.TestApp, token string, accountID, categoryID int, amount float64, isIncome bool) {
	t.Helper()

	body := fmt.Sprintf(`{
		"accountId": %d,
		"categoryId": %d,
		"amount": %.2f,
		"isIncome": %t,
		"isTransfer": false
	}`, accountID, categoryID, amount, isIncome)

	rec := doRequest(t, app, http.MethodPost, "/transactions/", body, token)
	require.Equal(t, http.StatusOK, rec.Code, "create tx: %s", rec.Body.String())
}
