package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const minSecretLen = 32

type Config struct {
	AppPort string

	DatabaseURL string

	JWTSecret string
	JWTTTL    time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		AppPort: getEnv("APP_PORT", "8080"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		JWTSecret: os.Getenv("JWT_SECRET"),
	}

	var problems []string

	if cfg.DatabaseURL == "" {
		problems = append(problems, "DATABASE_URL is required")
	}

	if len(cfg.JWTSecret) < minSecretLen {
		problems = append(problems, fmt.Sprintf("JWT_SECRET must be at least %d characters", minSecretLen))
	}

	exp, err := time.ParseDuration(getEnv("JWT_TTL", "1h"))
	if err != nil {
		problems = append(problems, fmt.Sprintf("JWT_TTL is not a valid duration: %v", err))
	} else {
		cfg.JWTTTL = exp
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("invalid configuration:\n  - %s", strings.Join(problems, "\n  - "))
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
