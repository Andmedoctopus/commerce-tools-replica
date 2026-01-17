package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration parsed from environment variables.
type Config struct {
	HTTPAddr        string
	DBConnString    string
	ShutdownTimeout time.Duration
	FileURLHost     string
}

// FromEnv builds Config with defaults, overridden by environment variables.
func FromEnv() Config {
	return Config{
		HTTPAddr:        envOrDefault("HTTP_ADDR", ":8080"),
		DBConnString:    envOrDefault("DB_DSN", "postgres://commerce:commerce@localhost:5432/commerce?sslmode=disable"),
		ShutdownTimeout: envDuration("SHUTDOWN_TIMEOUT_SECONDS", 10*time.Second),
		FileURLHost:     envOrDefault("FILE_URL_HOST", ""),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		seconds, err := strconv.Atoi(v)
		if err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return def
}
