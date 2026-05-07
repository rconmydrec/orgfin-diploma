package middleware

import (
	"log/slog"
	"net/http"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-budget/backend/internal/repositories"
	"github.com/labstack/echo/v4"
)

// RequireActiveUser returns an Echo middleware that validates the authenticated
// user is active and not soft-deleted. It must be placed AFTER RequireAuth in
// the middleware chain because it reads "user_id" from the context (set by JWT
// auth middleware).
//
// On success the middleware stores primitive values in the context:
//   - "active_user_email" (string)
//   - "active_user_base_currency_id" (int)
//   - "active_user_display_name" (string)
//
// The "user_id" (int) key is already set by the auth middleware and is NOT modified.
//
// On failure (user not found, inactive, or deleted) it writes a 401 JSON
// response with {"detail":"User not activated"} and short-circuits (does NOT
// call next).
func RequireActiveUser(userRepo repositories.UserRepository, logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := c.Get("user_id").(int)

			user, err := userRepo.GetByID(userID)
			if err != nil {
				logger.Error("failed to get user", "error", err)
				return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
			}
			if !user.IsActive || user.IsDeleted {
				return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
			}

			c.Set("active_user_email", user.Email)
			c.Set("active_user_base_currency_id", user.BaseCurrencyID)
			c.Set("active_user_display_name", computeDisplayName(user.FirstName, user.LastName))
			return next(c)
		}
	}
}

// computeDisplayName builds a display name from first and last name pointers.
// Duplicates models.User.DisplayName() logic to avoid importing models.
func computeDisplayName(firstName, lastName *string) string {
	var name string
	if firstName != nil {
		name = *firstName
	}
	if lastName != nil {
		if name != "" {
			name += " "
		}
		name += *lastName
	}
	return name
}
