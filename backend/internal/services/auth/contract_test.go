// Contract tests for the auth service.
// These tests validate the ServiceInterface contract and survive any implementation rewrite.
// They test through the public interface only (black-box), using a real DB.
package auth_test

import (
	"testing"

	jwtauth "github.com/go-budget/backend/internal/auth"
	"github.com/go-budget/backend/internal/services/auth"
	"github.com/go-budget/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestService creates a real auth service connected to the test DB.
func setupTestService(t *testing.T) (auth.ServiceInterface, *testutil.TestApp) {
	t.Helper()
	app := testutil.NewTestApp(t)
	logger := testutil.TestLogger()
	jwtService := jwtauth.NewJWTService(testutil.TestConfig.JWTSecret, testutil.TestConfig.JWTExpirationMinutes)
	mockEnqueuer := testutil.NewMockTaskEnqueuer()

	svc := auth.New(
		app.UserRepo,
		app.TokenRepo,
		app.SettingsRepo,
		app.CategoryRepo,
		app.CurrencyRepo,
		jwtService,
		mockEnqueuer,
		nil, // subscriptionSvc: optional, nil-safe
		testutil.TestConfig,
		logger,
	)
	return svc, app
}

// cleanupEmail ensures a user with the given email is removed before and after test.
func cleanupEmail(t *testing.T, app *testutil.TestApp, email string) {
	t.Helper()
	testutil.CleanupUserByEmail(t, app.DB, email)
	t.Cleanup(func() { testutil.CleanupUserByEmail(t, app.DB, email) })
}

// --- Register ---

func TestRegister_Success(t *testing.T) {
	svc, app := setupTestService(t)
	email := "auth-contract-register@example.com"
	cleanupEmail(t, app, email)

	result, err := svc.Register(email, "SecurePass123!", "John", "Doe")

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.User)
	assert.Equal(t, email, result.User.Email)
	assert.NotZero(t, result.User.ID)
	assert.NotEmpty(t, result.ActivationToken)
	// User should not be active before activation
	assert.False(t, result.User.IsActive)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc, app := setupTestService(t)
	email := "auth-contract-dup@example.com"
	cleanupEmail(t, app, email)

	_, err := svc.Register(email, "SecurePass123!", "", "")
	require.NoError(t, err)

	_, err = svc.Register(email, "AnotherPass456!", "", "")

	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrUserExists)
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	svc, app := setupTestService(t)
	email := "auth-contract-login@example.com"
	cleanupEmail(t, app, email)

	result, err := svc.Register(email, "SecurePass123!", "", "")
	require.NoError(t, err)

	// Activate user
	_, err = app.DB.Exec("UPDATE users SET is_active = true WHERE id = $1", result.User.ID)
	require.NoError(t, err)

	token, err := svc.Login(email, "SecurePass123!")

	require.NoError(t, err)
	assert.NotEmpty(t, token, "login should return a JWT token")
}

func TestLogin_InvalidCredentials(t *testing.T) {
	svc, app := setupTestService(t)
	email := "auth-contract-badlogin@example.com"
	cleanupEmail(t, app, email)

	_, err := svc.Register(email, "SecurePass123!", "", "")
	require.NoError(t, err)

	_, err = svc.Login(email, "WrongPassword!")

	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestLogin_NonExistentUser(t *testing.T) {
	svc, _ := setupTestService(t)

	_, err := svc.Login("nonexistent@example.com", "Password123!")

	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestLogin_UserNotActivated(t *testing.T) {
	svc, app := setupTestService(t)
	email := "auth-contract-notactive@example.com"
	cleanupEmail(t, app, email)

	_, err := svc.Register(email, "SecurePass123!", "", "")
	require.NoError(t, err)
	// Do NOT activate

	_, err = svc.Login(email, "SecurePass123!")

	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrUserNotActivated)
}

// --- GetProfile ---

func TestGetProfile_Success(t *testing.T) {
	svc, app := setupTestService(t)
	email := "auth-contract-profile@example.com"
	cleanupEmail(t, app, email)

	result, err := svc.Register(email, "SecurePass123!", "Alice", "")
	require.NoError(t, err)
	_, err = app.DB.Exec("UPDATE users SET is_active = true WHERE id = $1", result.User.ID)
	require.NoError(t, err)

	user, settings, err := svc.GetProfile(result.User.ID)

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, email, user.Email)
	// Settings may or may not be nil depending on auto-creation
	_ = settings
}

func TestGetProfile_UserNotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	_, _, err := svc.GetProfile(999999)

	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrUserNotFound)
}

// --- ActivateUser ---

func TestActivateUser_Success(t *testing.T) {
	svc, app := setupTestService(t)
	email := "auth-contract-activate@example.com"
	cleanupEmail(t, app, email)

	result, err := svc.Register(email, "SecurePass123!", "", "")
	require.NoError(t, err)

	err = svc.ActivateUser(result.ActivationToken)

	require.NoError(t, err)

	// Verify user is now active
	var isActive bool
	err = app.DB.QueryRow("SELECT is_active FROM users WHERE id = $1", result.User.ID).Scan(&isActive)
	require.NoError(t, err)
	assert.True(t, isActive)
}

func TestActivateUser_InvalidToken(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.ActivateUser("nonexistent-token-12345678")

	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrTokenNotFound)
}

func TestActivateUser_Idempotent(t *testing.T) {
	svc, app := setupTestService(t)
	email := "auth-contract-actidempotent@example.com"
	cleanupEmail(t, app, email)

	result, err := svc.Register(email, "SecurePass123!", "", "")
	require.NoError(t, err)

	// First activation
	err = svc.ActivateUser(result.ActivationToken)
	require.NoError(t, err)

	// Second activation with the same token should not error (idempotent)
	// or return ErrTokenNotFound since the token was already consumed.
	// The invariant says re-activating an already-active user is idempotent.
	// After first activation, token is deleted, so second call returns token not found.
	err = svc.ActivateUser(result.ActivationToken)
	if err != nil {
		assert.ErrorIs(t, err, auth.ErrTokenNotFound)
	}
}

// --- ChangePassword ---

func TestChangePassword_Success(t *testing.T) {
	svc, app := setupTestService(t)
	email := "auth-contract-chgpwd@example.com"
	cleanupEmail(t, app, email)

	result, err := svc.Register(email, "OldPass123!", "", "")
	require.NoError(t, err)
	_, err = app.DB.Exec("UPDATE users SET is_active = true WHERE id = $1", result.User.ID)
	require.NoError(t, err)

	err = svc.ChangePassword(result.User.ID, "OldPass123!", "NewPass456!")

	require.NoError(t, err)

	// Verify login with new password works
	token, err := svc.Login(email, "NewPass456!")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestChangePassword_IncorrectCurrent(t *testing.T) {
	svc, app := setupTestService(t)
	email := "auth-contract-chgbad@example.com"
	cleanupEmail(t, app, email)

	result, err := svc.Register(email, "OldPass123!", "", "")
	require.NoError(t, err)
	_, err = app.DB.Exec("UPDATE users SET is_active = true WHERE id = $1", result.User.ID)
	require.NoError(t, err)

	err = svc.ChangePassword(result.User.ID, "WrongPass!", "NewPass456!")

	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrIncorrectPassword)
}

func TestChangePassword_UserNotFound(t *testing.T) {
	svc, _ := setupTestService(t)

	err := svc.ChangePassword(999999, "Old!", "New!")

	require.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrUserNotFound)
}

// --- LoginOrRegisterOAuth ---

func TestLoginOrRegisterOAuth_NewUser(t *testing.T) {
	svc, app := setupTestService(t)
	email := "auth-contract-oauth-new@example.com"
	cleanupEmail(t, app, email)

	token, err := svc.LoginOrRegisterOAuth(email, "OAuthFirst", "OAuthLast")

	require.NoError(t, err)
	assert.NotEmpty(t, token, "OAuth should return a JWT token")

	// Verify user is auto-activated (OAuth invariant)
	var isActive bool
	err = app.DB.QueryRow("SELECT is_active FROM users WHERE email = $1", email).Scan(&isActive)
	require.NoError(t, err)
	assert.True(t, isActive, "OAuth users should be auto-activated")
}

func TestLoginOrRegisterOAuth_ExistingUser(t *testing.T) {
	svc, app := setupTestService(t)
	email := "auth-contract-oauth-exist@example.com"
	cleanupEmail(t, app, email)

	// Register via OAuth first
	_, err := svc.LoginOrRegisterOAuth(email, "First", "Last")
	require.NoError(t, err)

	// Login again via OAuth
	token, err := svc.LoginOrRegisterOAuth(email, "First", "Last")

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}
