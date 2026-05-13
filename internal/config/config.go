package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	ServerPort  string
	DatabaseURL string
	RedisURL    string
	Environment string
	// APIKeys es el conjunto de claves válidas cargadas desde API_KEYS (csv).
	// Ejemplo: API_KEYS=key-abc123,key-def456
	// En producción usar un gestor de secretos en lugar de variables de entorno.
	APIKeys map[string]struct{}
}

func Load() (*Config, error) {
	port := getEnv("APP_PORT", "8080")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	apiKeys := parseAPIKeys(os.Getenv("API_KEYS"))

	return &Config{
		ServerPort:  port,
		DatabaseURL: dbURL,
		RedisURL:    getEnv("REDIS_URL", ""),
		Environment: getEnv("APP_ENV", "development"),
		APIKeys:     apiKeys,
	}, nil
}

func parseAPIKeys(raw string) map[string]struct{} {
	keys := make(map[string]struct{})
	for _, k := range strings.Split(raw, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			keys[k] = struct{}{}
		}
	}
	return keys
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
