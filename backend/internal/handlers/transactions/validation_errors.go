package transactions

import (
	"errors"
	"fmt"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-playground/validator/v10"
)

// validationFailureCode is the i18n key returned for a validation failure
// the handler does not have a more specific mapping for. The frontend looks
// it up in its locale catalog and falls back to `Detail` if missing.
const validationFailureCode = "errors.transaction.validationFailed"

// labelTooLongCode / notesTooLongCode are the stable, machine-readable error
// codes the frontend i18n catalog must mirror. They are dot-namespaced under
// `errors.transaction.*` so other domains can introduce their own codes
// without colliding.
const (
	labelTooLongCode = "errors.transaction.labelTooLong"
	notesTooLongCode = "errors.transaction.notesTooLong"
)

// Length limits for transaction label/notes — kept here so the
// validator tag, the error code params, and the documentation strings all
// derive from the same constants.
const (
	transactionLabelMaxLen = 255
	transactionNotesMaxLen = 4000
)

// mapValidationError converts a validator error returned by `c.Validate`
// into a client-safe `common.ErrorResponse` and a list of failed JSON field
// names suitable for server-side logging.
//
// Hard rule: the returned `ErrorResponse` MUST NOT contain Go struct field
// names, validator tag literals, or raw `validator.FieldError` strings. The
// only way new error codes are added is by extending the explicit (jsonField,
// tag) -> (code, detail, params) mapping below.
//
// `failedFields` is returned for log enrichment only — it carries the JSON
// field names (e.g. `label`, `notes`) and is never echoed to the client.
func mapValidationError(err error) (resp common.ErrorResponse, failedFields []string) {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return common.ErrorResponse{
			Detail:    "Validation failed",
			ErrorCode: validationFailureCode,
		}, nil
	}

	failedFields = make([]string, 0, len(ve))
	for _, fe := range ve {
		failedFields = append(failedFields, fe.Field())
	}

	// Emit the first known mapping. UX is a single banner so we surface one
	// code per response. Unmapped failures fall through to the generic code.
	for _, fe := range ve {
		switch {
		case fe.Field() == "label" && fe.Tag() == "max":
			return common.ErrorResponse{
				Detail:    fmt.Sprintf("Label must be at most %d characters", transactionLabelMaxLen),
				ErrorCode: labelTooLongCode,
				Params:    map[string]any{"max": transactionLabelMaxLen},
			}, failedFields
		case fe.Field() == "notes" && fe.Tag() == "max":
			return common.ErrorResponse{
				Detail:    fmt.Sprintf("Notes must be at most %d characters", transactionNotesMaxLen),
				ErrorCode: notesTooLongCode,
				Params:    map[string]any{"max": transactionNotesMaxLen},
			}, failedFields
		}
	}

	return common.ErrorResponse{
		Detail:    "Validation failed",
		ErrorCode: validationFailureCode,
	}, failedFields
}

// notesLen returns the rune length of a `*string`, or 0 if nil. Used for
// log enrichment so the operator can see how big a payload was without
// PII (raw user-typed content is never logged).
func notesLen(s *string) int {
	if s == nil {
		return 0
	}
	return len([]rune(*s))
}
