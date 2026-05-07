package currency

import "github.com/go-budget/backend/internal/serviceerrors"

// ErrNoExchangeRates is a sentinel error indicating the exchange_rates table is empty.
// This is a system state error (not user input), so handlers return 500.
var ErrNoExchangeRates = serviceerrors.New(serviceerrors.Other, "exchange rates table is empty")
