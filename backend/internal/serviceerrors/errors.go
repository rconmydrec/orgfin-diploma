package serviceerrors

import (
	"errors"
	"fmt"
)

// Kind represents the category of a service error.
// Handlers use Kind to map service errors to HTTP status codes
// without importing individual service packages.
type Kind int

const (
	Other         Kind = iota // Uncategorized error
	NotFound                  // Resource not found
	AccessDenied              // User does not own the resource
	Conflict                  // Duplicate or state conflict (e.g., already exists, same plan)
	InvalidInput              // Validation failure (bad format, invalid reference)
	Unauthorized              // User not activated or not authenticated
	ProviderError             // External service failure (Stripe, email, etc.)
	LimitExceeded             // Plan or resource limit exceeded
	NoChange                  // Operation would have no effect
)

// ServiceError is a structured error returned by services.
// It carries a Kind for handler error-to-HTTP-status mapping
// and a human-readable message for logging (not for client response).
type ServiceError struct {
	Kind    Kind
	Message string
	Err     error // Optional wrapped error
}

func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *ServiceError) Unwrap() error { return e.Err }

// New creates a ServiceError with the given kind and message.
func New(kind Kind, message string) *ServiceError {
	return &ServiceError{Kind: kind, Message: message}
}

// Wrap creates a ServiceError that wraps an existing error.
func Wrap(kind Kind, message string, err error) *ServiceError {
	return &ServiceError{Kind: kind, Message: message, Err: err}
}

// GetKind extracts the Kind from an error. Returns Other if the error
// is not a *ServiceError.
func GetKind(err error) Kind {
	var se *ServiceError
	if errors.As(err, &se) {
		return se.Kind
	}
	return Other
}

// IsKind checks whether an error has the given Kind.
func IsKind(err error, kind Kind) bool {
	return GetKind(err) == kind
}

// IsServiceError checks whether an error is or wraps a *ServiceError.
func IsServiceError(err error) bool {
	var se *ServiceError
	return errors.As(err, &se)
}

// Message extracts the Message field from a *ServiceError.
// Returns an empty string if the error is not a *ServiceError.
func Message(err error) string {
	var se *ServiceError
	if errors.As(err, &se) {
		return se.Message
	}
	return ""
}
