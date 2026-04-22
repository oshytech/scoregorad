package config

import (
	"fmt"
	"os"
)

type Config struct {
	ServerPort  string
	DatabaseURL string
	RedisURL    string
	Environment string
}

func Load() (*Config, error) {
	port := getEnv("APP_PORT", "8080")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return &Config{
		ServerPort:  port,
		DatabaseURL: dbURL,
		RedisURL:    getEnv("REDIS_URL", ""),
		Environment: getEnv("APP_ENV", "development"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
