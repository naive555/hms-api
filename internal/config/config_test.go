package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validSecret = "a-secret-that-is-at-least-32-characters"

// setEnv sets every variable Load reads, so the host environment can't leak
// into the test. An empty value means "unset" for Load.
func setEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	for _, key := range []string{"APP_PORT", "DATABASE_URL", "JWT_SECRET", "JWT_TTL", "HIS_TIMEOUT"} {
		t.Setenv(key, vars[key])
	}
}

func TestLoad_Defaults(t *testing.T) {
	setEnv(t, map[string]string{"DATABASE_URL": "postgres://db", "JWT_SECRET": validSecret})

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, &Config{
		AppPort:     "8080",
		DatabaseURL: "postgres://db",
		JWTSecret:   validSecret,
		JWTTTL:      time.Hour,
		HISTimeout:  3 * time.Second,
	}, cfg)
}

func TestLoad_CustomValues(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_PORT":     "9090",
		"DATABASE_URL": "postgres://db",
		"JWT_SECRET":   validSecret,
		"JWT_TTL":      "15m",
		"HIS_TIMEOUT":  "500ms",
	})

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "9090", cfg.AppPort)
	assert.Equal(t, 15*time.Minute, cfg.JWTTTL)
	assert.Equal(t, 500*time.Millisecond, cfg.HISTimeout)
}

func TestLoad_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		vars    map[string]string
		wantMsg string
	}{
		{"missing DATABASE_URL", map[string]string{"JWT_SECRET": validSecret}, "DATABASE_URL is required"},
		{"missing JWT_SECRET", map[string]string{"DATABASE_URL": "postgres://db"}, "JWT_SECRET must be at least 32 characters"},
		{"short JWT_SECRET", map[string]string{"DATABASE_URL": "postgres://db", "JWT_SECRET": "too-short"}, "JWT_SECRET must be at least 32 characters"},
		{"bad JWT_TTL", map[string]string{"DATABASE_URL": "postgres://db", "JWT_SECRET": validSecret, "JWT_TTL": "3600"}, "JWT_TTL is not a valid duration"},
		{"bad HIS_TIMEOUT", map[string]string{"DATABASE_URL": "postgres://db", "JWT_SECRET": validSecret, "HIS_TIMEOUT": "soon"}, "HIS_TIMEOUT is not a valid duration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.vars)

			cfg, err := Load()
			assert.Nil(t, cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantMsg)
		})
	}
}

func TestLoad_ReportsEveryProblem(t *testing.T) {
	setEnv(t, map[string]string{"JWT_TTL": "x", "HIS_TIMEOUT": "y"})

	_, err := Load()
	require.Error(t, err)
	for _, msg := range []string{"DATABASE_URL", "JWT_SECRET", "JWT_TTL", "HIS_TIMEOUT"} {
		assert.Contains(t, err.Error(), msg)
	}
}
