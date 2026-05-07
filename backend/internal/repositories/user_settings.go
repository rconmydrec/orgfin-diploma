package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/go-budget/backend/internal/models"
)

type UserSettingsRepositoryImpl struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewUserSettingsRepository(db *sql.DB, logger *slog.Logger) *UserSettingsRepositoryImpl {
	return &UserSettingsRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *UserSettingsRepositoryImpl) CreateDefault(userID int) error {
	defaultSettings := models.SettingsData{
		Language: "en",
	}
	settingsJSON, err := json.Marshal(defaultSettings)
	if err != nil {
		return fmt.Errorf("marshal default settings: %w", err)
	}

	query := `
		INSERT INTO user_settings (user_id, settings)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`

	_, err = r.db.Exec(query, userID, settingsJSON)
	if err != nil {
		r.logger.Error("failed to create default user settings", "error", err, "user_id", userID)
		return err
	}

	return nil
}

func (r *UserSettingsRepositoryImpl) GetByUserID(userID int) (*models.UserSettings, error) {
	query := `
		SELECT id, user_id, settings, created_at, updated_at
		FROM user_settings
		WHERE user_id = $1
	`

	settings := &models.UserSettings{}
	var settingsJSON []byte

	err := r.db.QueryRow(query, userID).Scan(
		&settings.ID, &settings.UserID, &settingsJSON,
		&settings.CreatedAt, &settings.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Parse JSON settings
	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &settings.Settings); err != nil {
			return nil, fmt.Errorf("unmarshal settings: %w", err)
		}
	}

	return settings, nil
}

func (r *UserSettingsRepositoryImpl) Update(settings *models.UserSettings) error {
	settingsJSON, err := json.Marshal(settings.Settings)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	query := `
		UPDATE user_settings
		SET settings = $1, updated_at = NOW()
		WHERE user_id = $2
	`

	_, err = r.db.Exec(query, settingsJSON, settings.UserID)
	return err
}
