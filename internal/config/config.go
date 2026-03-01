package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"notification-processor/pkg/db"
)

type Config struct {
	// PostgreSQL
	DB db.Config

	// Kafka
	KafkaBrokers []string
	KafkaTopic   string
	KafkaGroupID string

	// Worker pool
	WorkerCount int

	// Retry
	RetryMaxAttempts int
	RetryBaseDelay   time.Duration

	// Logging
	LogLevel string
}

func Load() (*Config, error) {
	cfg := &Config{
		DB: db.Config{
			DSN:      os.Getenv("DATABASE_URL"),
			MaxConns: 10,
			MinConns: 2,
		},

		KafkaTopic:   getEnvOrDefault("KAFKA_TOPIC", "user-events"),
		KafkaGroupID: getEnvOrDefault("KAFKA_GROUP_ID", "notification-processor"),

		WorkerCount:      getEnvInt("WORKER_COUNT", 5),
		RetryMaxAttempts: getEnvInt("RETRY_MAX_ATTEMPTS", 3),
		RetryBaseDelay:   time.Duration(getEnvInt("RETRY_BASE_DELAY_SEC", 1)) * time.Second,

		LogLevel: getEnvOrDefault("LOG_LEVEL", "info"),
	}

	brokersRaw := getEnvOrDefault("KAFKA_BROKERS", "localhost:9092")
	cfg.KafkaBrokers = splitTrim(brokersRaw, ",")

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.DB.DSN == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if len(c.KafkaBrokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS is required")
	}
	if c.WorkerCount < 1 {
		return fmt.Errorf("WORKER_COUNT must be >= 1")
	}
	if c.RetryMaxAttempts < 1 {
		return fmt.Errorf("RETRY_MAX_ATTEMPTS must be >= 1")
	}
	return nil
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func splitTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			result = append(result, t)
		}
	}
	return result
}
