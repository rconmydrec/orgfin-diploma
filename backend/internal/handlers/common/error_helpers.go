package common

import (
	"net/http"

	"github.com/go-budget/backend/internal/serviceerrors"
)

// ErrorHTTPStatus returns the default HTTP status code for a service error Kind.
// Handlers may override these defaults for specific cases.
func ErrorHTTPStatus(kind serviceerrors.Kind) int {
	switch kind {
	case serviceerrors.NotFound:
		return http.StatusNotFound
	case serviceerrors.AccessDenied:
		return http.StatusForbidden
	case serviceerrors.Conflict:
		return http.StatusConflict
	case serviceerrors.InvalidInput:
		return http.StatusUnprocessableEntity
	case serviceerrors.Unauthorized:
		return http.StatusUnauthorized
	case serviceerrors.ProviderError:
		return http.StatusBadGateway
	case serviceerrors.LimitExceeded:
		return http.StatusForbidden
	case serviceerrors.NoChange:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
