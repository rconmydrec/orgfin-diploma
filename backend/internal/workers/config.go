package workers

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/hibiken/asynq"
)

// Queue names used for task routing.
const (
	QueueCritical = "critical"
	QueueDefault  = "default"
	QueueExport   = "export"
)

// QueueConfig returns the queue priority configuration for the asynq server.
// Weights determine the relative processing priority of each queue.
func QueueConfig() map[string]int {
	return map[string]int{
		QueueCritical: 6,
		QueueDefault:  3,
		QueueExport:   1,
	}
}

// ParseRedisURL parses a Redis URL (e.g., "redis://user:pass@host:port/db")
// into an asynq.RedisClientOpt suitable for creating asynq clients and servers.
func ParseRedisURL(redisURL string) (asynq.RedisClientOpt, error) {
	var zero asynq.RedisClientOpt

	parsed, err := url.Parse(redisURL)
	if err != nil {
		return zero, fmt.Errorf("invalid redis URL: %w", err)
	}

	host := parsed.Hostname()
	if host == "" {
		host = "localhost"
	}

	port := parsed.Port()
	if port == "" {
		port = "6379"
	}

	addr := host + ":" + port

	password := ""
	if parsed.User != nil {
		password, _ = parsed.User.Password()
	}

	db := 0
	if parsed.Path != "" && parsed.Path != "/" {
		dbStr := parsed.Path[1:] // strip leading "/"
		if dbStr != "" {
			db, err = strconv.Atoi(dbStr)
			if err != nil {
				return zero, fmt.Errorf("invalid redis database number: %w", err)
			}
		}
	}

	return asynq.RedisClientOpt{
		Addr:     addr,
		Password: password,
		DB:       db,
	}, nil
}
