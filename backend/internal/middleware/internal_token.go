package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/labstack/echo/v4"
)

// RequireInternalToken returns an Echo middleware that authenticates requests
// using a static bearer token. This is intended for internal service-to-service
// calls (e.g., backup job -> API) where JWT auth is not appropriate.
//
// If the configured token is empty, all requests are rejected (endpoint disabled).
// Token comparison uses constant-time comparison to prevent timing attacks.
func RequireInternalToken(token string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if token == "" {
				return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "Unauthorized"})
			}

			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "Unauthorized"})
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "Unauthorized"})
			}

			provided := strings.TrimPrefix(authHeader, "Bearer ")
			if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
				return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "Unauthorized"})
			}

			return next(c)
		}
	}
}
