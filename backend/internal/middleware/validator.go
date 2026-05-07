package middleware

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// CustomValidator wraps go-playground/validator and surfaces raw
// `validator.ValidationErrors` so handlers can introspect them with
// `errors.As`. Handlers are responsible for shaping the HTTP response —
// this middleware never wraps the error in `echo.NewHTTPError` and never
// echoes validator strings to the client.
type CustomValidator struct {
	validator *validator.Validate
}

// NewValidator constructs a CustomValidator. It registers a tag name
// function so `FieldError.Field()` returns the JSON tag name (e.g. `label`)
// instead of the Go struct field name (e.g. `Label`). This allows handlers
// to map (jsonField, tag) pairs to error codes server-side without ever
// echoing the JSON field back to the client.
func NewValidator() *CustomValidator {
	v := validator.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		// Use the first segment of the json tag (e.g. `accountId,omitempty` -> `accountId`).
		// If the tag is missing or "-", fall back to the Go field name.
		tag := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if tag == "" || tag == "-" {
			return fld.Name
		}
		return tag
	})
	return &CustomValidator{validator: v}
}

// Validate runs struct validation. On failure it returns the raw
// `validator.ValidationErrors` (or other underlying error) so callers can
// inspect failed fields and tags. The error is NEVER wrapped in an HTTP
// error here — handlers are responsible for converting it to a response
// without leaking internal field names or tag literals.
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}
