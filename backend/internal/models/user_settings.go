package models

import (
	"encoding/json"
	"time"
)

type SettingsData struct {
	Language          string  `json:"language,omitempty"`
	ProjectionEndDate *string `json:"projectionEndDate,omitempty"`
	ProjectionPeriod  *string `json:"projectionPeriod,omitempty"`
}

type UserSettings struct {
	ID        int          `json:"id" db:"id"`
	UserID    int          `json:"userId" db:"user_id"`
	Settings  SettingsData `json:"settings"`
	CreatedAt time.Time    `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time    `json:"updatedAt" db:"updated_at"`
}

// SettingsJSON is used for scanning JSON from database
type SettingsJSON []byte

func (s *SettingsJSON) Scan(value interface{}) error {
	if value == nil {
		*s = []byte("{}")
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*s = v
	case string:
		*s = []byte(v)
	}
	return nil
}

func ParseSettingsJSON(data []byte) SettingsData {
	var settings SettingsData
	if len(data) > 0 {
		json.Unmarshal(data, &settings)
	}
	return settings
}
