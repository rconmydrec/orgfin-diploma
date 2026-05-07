package export

import "github.com/go-budget/backend/internal/serviceerrors"

// ErrUserNotFound is returned when the user does not exist.
var ErrUserNotFound = serviceerrors.New(serviceerrors.NotFound, "user not found")
