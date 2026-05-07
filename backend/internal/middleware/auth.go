package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-budget/backend/internal/auth"
	"github.com/labstack/echo/v4"
)

// TokenGenerator abstracts JWT token operations needed by AuthMiddleware.
// *auth.JWTService implements this interface.
type TokenGenerator interface {
	GenerateToken(userID int, email string) (string, error)
	ValidateToken(tokenString string) (*auth.Claims, error)
}

type AuthMiddleware struct {
	tokenGenerator TokenGenerator
	logger         *slog.Logger
}

func NewAuthMiddleware(tokenGenerator TokenGenerator, logger *slog.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		tokenGenerator: tokenGenerator,
		logger:         logger,
	}
}

// RequireAuth validates JWT token from auth-token header (not Authorization Bearer)
func (m *AuthMiddleware) RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Python backend uses "auth-token" header, not "Authorization: Bearer"
		token := c.Request().Header.Get("auth-token")
		if token == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"detail": "Missing authorization header",
			})
		}

		claims, err := m.tokenGenerator.ValidateToken(token)
		if err != nil {
			m.logger.Debug("token validation failed", "error", err)
			if errors.Is(err, auth.ErrExpiredToken) {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"detail": "Token has expired",
				})
			}
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"detail": "Invalid token",
			})
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		// Register a Before hook that fires just before headers are sent to the wire.
		// This ensures the new_access_token header is included in the response,
		// unlike setting it after the handler (which would be too late since
		// c.JSON() commits headers+body before returning).
		c.Response().Before(func() {
			status := c.Response().Status
			if status >= 200 && status < 300 {
				newToken, tokenErr := m.tokenGenerator.GenerateToken(claims.UserID, claims.Email)
				if tokenErr != nil {
					m.logger.Error("failed to generate refresh token", "error", tokenErr)
					return
				}
				c.Response().Header().Set("new_access_token", newToken)
			}
		})

		return next(c)
	}
}
