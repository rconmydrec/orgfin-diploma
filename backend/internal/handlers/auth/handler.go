package auth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-budget/backend/internal/handlers/common"
	"github.com/go-budget/backend/internal/serviceerrors"
	"github.com/labstack/echo/v4"
	"google.golang.org/api/idtoken"
)

// GoogleTokenValidator interface for validating Google OAuth tokens
type GoogleTokenValidator interface {
	Validate(ctx context.Context, token string, audience string) (*idtoken.Payload, error)
}

// DefaultGoogleValidator uses the real Google idtoken library
type DefaultGoogleValidator struct{}

func (v *DefaultGoogleValidator) Validate(ctx context.Context, token string, audience string) (*idtoken.Payload, error) {
	return idtoken.Validate(ctx, token, audience)
}

type Handler struct {
	authService     AuthService
	googleClientID  string
	googleValidator GoogleTokenValidator
	logger          *slog.Logger
}

func New(authService AuthService, googleClientID string, logger *slog.Logger) *Handler {
	return &Handler{
		authService:     authService,
		googleClientID:  googleClientID,
		googleValidator: &DefaultGoogleValidator{},
		logger:          logger,
	}
}

// NewWithValidator creates handler with custom validator (for testing)
func NewWithValidator(authService AuthService, googleClientID string, validator GoogleTokenValidator, logger *slog.Logger) *Handler {
	return &Handler{
		authService:     authService,
		googleClientID:  googleClientID,
		googleValidator: validator,
		logger:          logger,
	}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/register/", h.Register)
	g.POST("/login/", h.Login)
	g.GET("/profile/", h.GetProfile)
	g.GET("/activate/:token", h.Activate)
	g.POST("/oauth/", h.OAuth)
	g.POST("/change-password/", h.ChangePassword)
}

func (h *Handler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Validation failed"})
	}

	result, err := h.authService.Register(req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		if serviceerrors.IsKind(err, serviceerrors.Conflict) {
			return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "User with this email already exists"})
		}
		h.logger.Error("registration failed", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Registration failed"})
	}

	return c.JSON(http.StatusOK, UserResponse{
		ID:        result.User.ID,
		Email:     result.User.Email,
		FirstName: result.User.FirstName,
		LastName:  result.User.LastName,
	})
}

func (h *Handler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Validation failed"})
	}

	token, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.InvalidInput:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "Invalid credentials or user not found"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("login failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Authentication failed"})
		}
	}

	return c.JSON(http.StatusOK, TokenResponse{
		AccessToken: token,
		TokenType:   "bearer",
	})
}

func (h *Handler) GetProfile(c echo.Context) error {
	userID := c.Get("user_id").(int)

	user, settings, err := h.authService.GetProfile(userID)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.NotFound:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not found"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("get profile failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to get profile"})
		}
	}

	response := ProfileResponse{
		ID:           user.ID,
		Email:        user.Email,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		BaseCurrency: user.BaseCurrency.Code,
	}

	if settings != nil {
		response.Settings = &SettingsDTO{
			Language:          settings.Settings.Language,
			ProjectionEndDate: settings.Settings.ProjectionEndDate,
			ProjectionPeriod:  settings.Settings.ProjectionPeriod,
		}
	}

	return c.JSON(http.StatusOK, response)
}

func (h *Handler) Activate(c echo.Context) error {
	token := c.Param("token")

	err := h.authService.ActivateUser(token)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.NotFound:
			return c.JSON(http.StatusNotFound, common.ErrorResponse{Detail: "Token not found"})
		case serviceerrors.Unauthorized:
			return c.JSON(http.StatusBadRequest, common.ErrorResponse{Detail: "Token expired"})
		default:
			h.logger.Error("activation failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Activation failed"})
		}
	}

	return c.JSON(http.StatusOK, true)
}

func (h *Handler) OAuth(c echo.Context) error {
	var req OAuthRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Validation failed"})
	}

	// Verify Google token
	payload, err := h.googleValidator.Validate(context.Background(), req.Credential, h.googleClientID)
	if err != nil {
		h.logger.Error("google token validation failed", "error", err)
		return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "Invalid credentials or user not found"})
	}

	email, ok := payload.Claims["email"].(string)
	if !ok || email == "" {
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "No email provided"})
	}

	emailVerified, _ := payload.Claims["email_verified"].(bool)
	if !emailVerified {
		return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "Email not verified"})
	}

	firstName, _ := payload.Claims["given_name"].(string)
	lastName, _ := payload.Claims["family_name"].(string)

	token, err := h.authService.LoginOrRegisterOAuth(email, firstName, lastName)
	if err != nil {
		if serviceerrors.IsKind(err, serviceerrors.Unauthorized) {
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		}
		h.logger.Error("oauth login/register failed", "error", err)
		return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Authentication failed"})
	}

	return c.JSON(http.StatusOK, TokenResponse{
		AccessToken: token,
		TokenType:   "bearer",
	})
}

func (h *Handler) ChangePassword(c echo.Context) error {
	userID := c.Get("user_id").(int)

	var req ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("bind error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Invalid request data"})
	}
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validation error", "error", err)
		return c.JSON(http.StatusUnprocessableEntity, common.ErrorResponse{Detail: "Validation failed"})
	}

	err := h.authService.ChangePassword(userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch serviceerrors.GetKind(err) {
		case serviceerrors.NotFound:
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not found"})
		case serviceerrors.Unauthorized:
			// Both ErrUserNotActivated and ErrIncorrectPassword are Unauthorized kind.
			// Use the ServiceError message to preserve specific client responses.
			if serviceerrors.Message(err) == "current password is incorrect" {
				return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "Current password is incorrect"})
			}
			return c.JSON(http.StatusUnauthorized, common.ErrorResponse{Detail: "User not activated"})
		default:
			h.logger.Error("change password failed", "error", err)
			return c.JSON(http.StatusInternalServerError, common.ErrorResponse{Detail: "Failed to change password"})
		}
	}

	return c.JSON(http.StatusOK, ChangePasswordResponse{
		Success: true,
		Message: "Password changed successfully",
	})
}
