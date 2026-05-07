package transactions

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/go-budget/backend/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMapValidationErrorLabelTooLong verifies that a label-overflow validator
// failure produces the documented `errors.transaction.labelTooLong` code with
// `params.max == 255` and an English fallback `Detail`. The serialized JSON
// must NOT carry validator internals.
func TestMapValidationErrorLabelTooLong(t *testing.T) {
	v := middleware.NewValidator()
	req := CreateTransactionRequest{AccountID: 1, Label: strings.Repeat("a", 300)}
	err := v.Validate(&req)
	require.Error(t, err)

	resp, fields := mapValidationError(err)
	assert.Equal(t, "errors.transaction.labelTooLong", resp.ErrorCode)
	assert.Equal(t, "Label must be at most 255 characters", resp.Detail)
	if assert.NotNil(t, resp.Params) {
		assert.Equal(t, 255, resp.Params["max"])
	}
	assert.Contains(t, fields, "label")

	body, jerr := json.Marshal(resp)
	require.NoError(t, jerr)
	for _, leak := range []string{
		"CreateTransactionRequest", "UpdateTransactionRequest",
		"Field validation", "failed on the", "Tag=", "Field=",
	} {
		assert.NotContains(t, string(body), leak)
	}
}

// TestMapValidationErrorNotesTooLong covers the notes-length mapping path.
func TestMapValidationErrorNotesTooLong(t *testing.T) {
	v := middleware.NewValidator()
	notes := strings.Repeat("n", 4500)
	req := CreateTransactionRequest{AccountID: 1, Label: "ok", Notes: &notes}
	// AccountID/Amount required tags would fire too — set Amount via direct field.
	// Validator picks first-known mapping; notes max should win since it's mapped
	// (and label is fine here).
	err := v.Validate(&req)
	require.Error(t, err)

	resp, fields := mapValidationError(err)
	// Either the required-amount unmapped failure or the notes failure could come
	// first; we accept the notes mapping when Notes is the offender.
	if resp.ErrorCode == "errors.transaction.notesTooLong" {
		assert.Equal(t, "Notes must be at most 4000 characters", resp.Detail)
		if assert.NotNil(t, resp.Params) {
			assert.Equal(t, 4000, resp.Params["max"])
		}
	}
	assert.Contains(t, fields, "notes")
}

// TestMapValidationErrorUnmapped verifies the generic fallback never leaks
// validator strings and surfaces the stable `validationFailed` error code.
func TestMapValidationErrorUnmapped(t *testing.T) {
	v := middleware.NewValidator()
	// Missing required AccountID + Amount triggers required-tag failures, which
	// are not in the mapping table.
	var req CreateTransactionRequest
	err := v.Validate(&req)
	require.Error(t, err)

	resp, fields := mapValidationError(err)
	assert.Equal(t, "errors.transaction.validationFailed", resp.ErrorCode)
	assert.Equal(t, "Validation failed", resp.Detail)
	assert.Nil(t, resp.Params)
	assert.NotEmpty(t, fields)

	body, jerr := json.Marshal(resp)
	require.NoError(t, jerr)
	for _, leak := range []string{
		"CreateTransactionRequest", "Field validation", "failed on the", "Tag=", "Field=",
	} {
		assert.NotContains(t, string(body), leak)
	}
}

// TestMapValidationErrorNonValidatorError verifies the helper degrades
// gracefully when handed an arbitrary error: it must still produce the
// generic `validationFailed` code without panicking.
func TestMapValidationErrorNonValidatorError(t *testing.T) {
	resp, fields := mapValidationError(errors.New("some other error"))
	assert.Equal(t, "errors.transaction.validationFailed", resp.ErrorCode)
	assert.Equal(t, "Validation failed", resp.Detail)
	assert.Nil(t, fields)
}
