package settings

import "github.com/go-budget/backend/internal/serviceerrors"

// Sentinel errors for the settings service.
var (
	ErrSettingsCreateFailed = serviceerrors.New(serviceerrors.Other, "failed to create default settings")
	ErrSettingsNotFound     = serviceerrors.New(serviceerrors.NotFound, "settings not found after creation")
	ErrInvalidCurrency      = serviceerrors.New(serviceerrors.InvalidInput, "invalid currency")
	ErrBaseCurrencyNotSet   = serviceerrors.New(serviceerrors.NotFound, "base currency not set")
	ErrUserNotFound         = serviceerrors.New(serviceerrors.NotFound, "user not found")
)
