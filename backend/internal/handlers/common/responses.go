package common

// ErrorResponse is a generic error response returned by all handlers.
//
// `ErrorCode` and `Params` are optional fields used by endpoints that opt into the
// machine-readable error code convention (i18n key + structured params) for the
// frontend. Existing endpoints that construct `ErrorResponse{Detail: "..."}`
// continue to emit identical JSON because both fields are tagged `omitempty`.
//
// The response NEVER carries Go struct field names, validator tags, or raw
// `validator.FieldError` strings. The mapping from validator failure to error
// code lives server-side in the handler.
type ErrorResponse struct {
	Detail    string         `json:"detail"`
	ErrorCode string         `json:"errorCode,omitempty"`
	Params    map[string]any `json:"params,omitempty"`
}

// MessageResponse is a generic success message response.
type MessageResponse struct {
	Message string `json:"message"`
}

// DeleteResponse indicates whether a resource was successfully deleted.
type DeleteResponse struct {
	Deleted bool `json:"deleted"`
}

// CurrencyDTO is a lightweight currency representation used in handler responses.
type CurrencyDTO struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}
