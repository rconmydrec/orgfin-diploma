package middleware

import (
	"net/http"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/labstack/echo/v4"
)

// RequireAdmin returns an Echo middleware that restricts access to admin users.
// It must be placed AFTER RequireActiveUser in the middleware chain because it
// reads "active_user_email" from the context (set by RequireActiveUser middleware).
//
// The middleware checks if the authenticated user's email is present in the
// adminEmails slice. If the user IS an admin, the request proceeds. Otherwise
// a 403 Forbidden response with {"detail":"Admin access required"} is returned.
func RequireAdmin(adminEmails []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			email, ok := c.Get("active_user_email").(string)
			if !ok || email == "" {
				return c.JSON(http.StatusForbidden, common.ErrorResponse{Detail: "Admin access required"})
			}

			for _, adminEmail := range adminEmails {
				if adminEmail == email {
					return next(c)
				}
			}

			return c.JSON(http.StatusForbidden, common.ErrorResponse{Detail: "Admin access required"})
		}
	}
}
