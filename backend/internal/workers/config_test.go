package workers

import (
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRedisURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected asynq.RedisClientOpt
		wantErr  bool
	}{
		{
			name: "valid URL with host and port",
			url:  "redis://localhost:6379",
			expected: asynq.RedisClientOpt{
				Addr:     "localhost:6379",
				Password: "",
				DB:       0,
			},
		},
		{
			name: "valid URL without port defaults to 6379",
			url:  "redis://myhost",
			expected: asynq.RedisClientOpt{
				Addr:     "myhost:6379",
				Password: "",
				DB:       0,
			},
		},
		{
			name: "valid URL with database number",
			url:  "redis://localhost:6380/5",
			expected: asynq.RedisClientOpt{
				Addr:     "localhost:6380",
				Password: "",
				DB:       5,
			},
		},
		{
			name: "valid URL with password",
			url:  "redis://user:secret@redis.example.com:6379/2",
			expected: asynq.RedisClientOpt{
				Addr:     "redis.example.com:6379",
				Password: "secret",
				DB:       2,
			},
		},
		{
			name: "URL with only password (no username)",
			url:  "redis://:mypass@localhost:6379",
			expected: asynq.RedisClientOpt{
				Addr:     "localhost:6379",
				Password: "mypass",
				DB:       0,
			},
		},
		{
			name: "URL with trailing slash but no database",
			url:  "redis://localhost:6379/",
			expected: asynq.RedisClientOpt{
				Addr:     "localhost:6379",
				Password: "",
				DB:       0,
			},
		},
		{
			name:    "invalid database number",
			url:     "redis://localhost:6379/notanumber",
			wantErr: true,
		},
		{
			name: "empty URL defaults to localhost:6379",
			url:  "",
			expected: asynq.RedisClientOpt{
				Addr:     "localhost:6379",
				Password: "",
				DB:       0,
			},
		},
		{
			name: "URL with no host defaults to localhost",
			url:  "redis://:6380",
			expected: asynq.RedisClientOpt{
				Addr:     "localhost:6380",
				Password: "",
				DB:       0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseRedisURL(tt.url)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			assert.Equal(t, tt.expected.Addr, result.Addr)
			assert.Equal(t, tt.expected.Password, result.Password)
			assert.Equal(t, tt.expected.DB, result.DB)
		})
	}
}

func TestQueueConfig(t *testing.T) {
	qc := QueueConfig()

	t.Run("returns expected queue names", func(t *testing.T) {
		assert.Contains(t, qc, QueueCritical)
		assert.Contains(t, qc, QueueDefault)
		assert.Contains(t, qc, QueueExport)
	})

	t.Run("critical has highest priority", func(t *testing.T) {
		assert.Greater(t, qc[QueueCritical], qc[QueueDefault])
		assert.Greater(t, qc[QueueDefault], qc[QueueExport])
	})

	t.Run("returns correct priority values", func(t *testing.T) {
		assert.Equal(t, 6, qc[QueueCritical])
		assert.Equal(t, 3, qc[QueueDefault])
		assert.Equal(t, 1, qc[QueueExport])
	})

	t.Run("contains exactly three queues", func(t *testing.T) {
		assert.Len(t, qc, 3)
	})
}
