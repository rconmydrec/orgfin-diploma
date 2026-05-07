package auth

// Request DTOs

type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=3,max=50"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=3,max=50"`
}

type OAuthRequest struct {
	Credential string `json:"credential" validate:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,min=3,max=50"`
	NewPassword     string `json:"new_password" validate:"required,min=3,max=50"`
}

// Response DTOs

// UserResponse for POST /auth/register/ - uses camelCase
type UserResponse struct {
	ID        int     `json:"id"`
	Email     string  `json:"email"`
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
}

// TokenResponse for POST /auth/login/ and POST /auth/oauth/
type TokenResponse struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
}

// ProfileResponse for GET /auth/profile/ - mixed case (matches original)
type ProfileResponse struct {
	ID           int          `json:"id"`
	Email        string       `json:"email"`
	FirstName    *string      `json:"first_name,omitempty"`
	LastName     *string      `json:"last_name,omitempty"`
	Settings     *SettingsDTO `json:"settings,omitempty"`
	BaseCurrency string       `json:"baseCurrency"`
}

type SettingsDTO struct {
	Language          string  `json:"language"`
	ProjectionEndDate *string `json:"projectionEndDate,omitempty"`
	ProjectionPeriod  *string `json:"projectionPeriod,omitempty"`
}

type ChangePasswordResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
