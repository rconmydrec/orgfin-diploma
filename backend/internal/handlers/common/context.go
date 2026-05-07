package common

import "github.com/labstack/echo/v4"

// ActiveUserID extracts the active user's ID from the echo context.
func ActiveUserID(c echo.Context) int {
	return c.Get("user_id").(int)
}

// ActiveUserEmail extracts the active user's email from the echo context.
func ActiveUserEmail(c echo.Context) string {
	return c.Get("active_user_email").(string)
}

// ActiveUserBaseCurrencyID extracts the active user's base currency ID from the echo context.
func ActiveUserBaseCurrencyID(c echo.Context) int {
	return c.Get("active_user_base_currency_id").(int)
}

// ActiveUserDisplayName extracts the active user's display name from the echo context.
func ActiveUserDisplayName(c echo.Context) string {
	return c.Get("active_user_display_name").(string)
}
