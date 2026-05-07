package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/go-budget/backend/internal/auth"
	"github.com/go-budget/backend/internal/config"
	"github.com/go-budget/backend/internal/models"
	"github.com/go-budget/backend/internal/workers/tasks"
)

const (
	ActivationTokenLength = 16 // produces 32 char hex string
	ActivationTokenExpiry = 24 * time.Hour
)

type Service struct {
	userRepo        UserRepository
	tokenRepo       ActivationTokenRepository
	settingsRepo    UserSettingsRepository
	categoryRepo    CategoryRepository
	currencyRepo    CurrencyRepository
	jwtService      *auth.JWTService
	enqueuer        TaskEnqueuer
	subscriptionSvc SubscriptionTrialCreator // optional, can be nil
	cfg             *config.Config
	logger          *slog.Logger
}

func New(
	userRepo UserRepository,
	tokenRepo ActivationTokenRepository,
	settingsRepo UserSettingsRepository,
	categoryRepo CategoryRepository,
	currencyRepo CurrencyRepository,
	jwtService *auth.JWTService,
	enqueuer TaskEnqueuer,
	subscriptionSvc SubscriptionTrialCreator,
	cfg *config.Config,
	logger *slog.Logger,
) *Service {
	return &Service{
		userRepo:        userRepo,
		tokenRepo:       tokenRepo,
		settingsRepo:    settingsRepo,
		categoryRepo:    categoryRepo,
		currencyRepo:    currencyRepo,
		jwtService:      jwtService,
		enqueuer:        enqueuer,
		subscriptionSvc: subscriptionSvc,
		cfg:             cfg,
		logger:          logger,
	}
}

func (s *Service) Register(email, password, firstName, lastName string) (*RegisterResult, error) {
	// Check if user exists
	existingUser, err := s.userRepo.GetByEmail(email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserExists
	}

	// Get default currency (USD)
	currency, err := s.currencyRepo.GetByCode("USD")
	if err != nil {
		s.logger.Error("failed to get default currency", "error", err)
		return nil, err
	}

	// Create user
	user := &models.User{
		Email:          email,
		FirstName:      &firstName,
		LastName:       &lastName,
		IsActive:       false,
		BaseCurrencyID: currency.ID,
	}
	if err := user.SetPassword(password); err != nil {
		return nil, err
	}

	createdUser, err := s.userRepo.Create(user)
	if err != nil {
		return nil, err
	}

	// Create activation token
	tokenBytes := make([]byte, ActivationTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	tokenString := hex.EncodeToString(tokenBytes)

	activationToken := &models.ActivationToken{
		UserID:    createdUser.ID,
		Token:     tokenString,
		ExpiresAt: time.Now().Add(ActivationTokenExpiry),
	}
	if err := s.tokenRepo.Create(activationToken); err != nil {
		return nil, err
	}

	// Create default user settings
	if err := s.settingsRepo.CreateDefault(createdUser.ID); err != nil {
		s.logger.Error("failed to create default settings", "error", err, "user_id", createdUser.ID)
	}

	// Copy default categories to user
	if err := s.categoryRepo.CopyDefaultCategories(createdUser.ID); err != nil {
		s.logger.Error("failed to copy default categories", "error", err, "user_id", createdUser.ID)
	}

	// Create trial subscription (non-blocking: failure does not prevent registration)
	if s.subscriptionSvc != nil {
		if err := s.subscriptionSvc.CreateTrialSubscription(createdUser.ID); err != nil {
			s.logger.Error("failed to create trial subscription", "userID", createdUser.ID, "error", err)
		}
	}

	// Queue activation email task
	if s.enqueuer != nil {
		activationTask, err := tasks.NewActivationEmailTask(tasks.ActivationEmailPayload{
			UserID:      createdUser.ID,
			Email:       email,
			Token:       tokenString,
			AppURL:      s.cfg.AppURL,
			FrontendURL: s.cfg.FrontendURL,
		})
		if err != nil {
			s.logger.Error("failed to create activation email task", "error", err, "user_id", createdUser.ID)
		} else if _, err := s.enqueuer.Enqueue(activationTask); err != nil {
			s.logger.Error("failed to enqueue activation email task", "error", err, "user_id", createdUser.ID)
		}
	}

	return &RegisterResult{
		User:            toUser(createdUser),
		ActivationToken: tokenString,
	}, nil
}

func (s *Service) Login(email, password string) (string, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}

	if !user.CheckPassword(password) {
		return "", ErrInvalidCredentials
	}

	if !user.IsActive {
		return "", ErrUserNotActivated
	}

	token, err := s.jwtService.GenerateToken(user.ID, user.Email)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) GetProfile(userID int) (*User, *UserSettings, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrUserNotFound
		}
		return nil, nil, err
	}

	if !user.IsActive {
		return nil, nil, ErrUserNotActivated
	}

	settings, err := s.settingsRepo.GetByUserID(userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, nil, err
	}

	return toUser(user), toUserSettings(settings), nil
}

func (s *Service) ActivateUser(token string) error {
	activationToken, err := s.tokenRepo.GetByToken(token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTokenNotFound
		}
		return err
	}

	if time.Now().After(activationToken.ExpiresAt) {
		return ErrTokenExpired
	}

	// Check that user exists and is not deleted
	user, err := s.userRepo.GetByID(activationToken.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	// Already active — idempotent, just clean up the token
	if user.IsActive {
		if err := s.tokenRepo.Delete(activationToken.ID); err != nil {
			s.logger.Error("failed to delete activation token", "error", err, "token_id", activationToken.ID)
		}
		return nil
	}

	if err := s.userRepo.Activate(activationToken.UserID); err != nil {
		return err
	}

	if err := s.tokenRepo.Delete(activationToken.ID); err != nil {
		s.logger.Error("failed to delete activation token", "error", err, "token_id", activationToken.ID)
	}

	return nil
}

func (s *Service) ChangePassword(userID int, currentPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}

	if !user.IsActive {
		return ErrUserNotActivated
	}

	if !user.CheckPassword(currentPassword) {
		return ErrIncorrectPassword
	}

	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	return s.userRepo.UpdatePassword(userID, user.PasswordHash)
}

func (s *Service) LoginOrRegisterOAuth(email, firstName, lastName string) (string, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	if user != nil {
		// Existing user - login
		if !user.IsActive {
			return "", ErrUserNotActivated
		}
		return s.jwtService.GenerateToken(user.ID, user.Email)
	}

	// New user - register with OAuth (auto-activate)
	currency, err := s.currencyRepo.GetByCode("USD")
	if err != nil {
		return "", err
	}

	// Generate random password for OAuth users
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	randomPassword := hex.EncodeToString(randomBytes)

	newUser := &models.User{
		Email:          email,
		FirstName:      &firstName,
		LastName:       &lastName,
		IsActive:       true, // OAuth users are auto-activated
		BaseCurrencyID: currency.ID,
	}
	if err := newUser.SetPassword(randomPassword); err != nil {
		return "", err
	}

	createdUser, err := s.userRepo.Create(newUser)
	if err != nil {
		return "", err
	}

	// Create default settings and categories
	if err := s.settingsRepo.CreateDefault(createdUser.ID); err != nil {
		s.logger.Error("failed to create default settings for OAuth user", "error", err)
	}
	if err := s.categoryRepo.CopyDefaultCategories(createdUser.ID); err != nil {
		s.logger.Error("failed to copy default categories for OAuth user", "error", err)
	}

	// Create trial subscription (non-blocking: failure does not prevent registration)
	if s.subscriptionSvc != nil {
		if err := s.subscriptionSvc.CreateTrialSubscription(createdUser.ID); err != nil {
			s.logger.Error("failed to create trial subscription for OAuth user", "userID", createdUser.ID, "error", err)
		}
	}

	return s.jwtService.GenerateToken(createdUser.ID, createdUser.Email)
}

// toUser converts a models.User to the service domain type.
func toUser(m *models.User) *User {
	if m == nil {
		return nil
	}
	u := &User{
		ID:             m.ID,
		Email:          m.Email,
		IsActive:       m.IsActive,
		FirstName:      m.FirstName,
		LastName:       m.LastName,
		BaseCurrencyID: m.BaseCurrencyID,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
	if m.BaseCurrency != nil {
		u.BaseCurrency = &Currency{
			ID:   m.BaseCurrency.ID,
			Code: m.BaseCurrency.Code,
			Name: m.BaseCurrency.Name,
		}
	}
	return u
}

// toUserSettings converts a models.UserSettings to the service domain type.
func toUserSettings(m *models.UserSettings) *UserSettings {
	if m == nil {
		return nil
	}
	return &UserSettings{
		ID:     m.ID,
		UserID: m.UserID,
		Settings: SettingsData{
			Language:          m.Settings.Language,
			ProjectionEndDate: m.Settings.ProjectionEndDate,
			ProjectionPeriod:  m.Settings.ProjectionPeriod,
		},
	}
}
