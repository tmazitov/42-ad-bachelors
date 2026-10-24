package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Config is the runtime configuration, read from environment variables.
type Config struct {
	HTTPAddr        string
	DatabasePath    string
	CORSOrigins     []string
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
}

// Load reads the configuration from the environment.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		DatabasePath:    getenv("DATABASE_PATH", "./data/app.db"),
		CORSOrigins:     splitList(getenv("CORS_ALLOW_ORIGINS", "http://localhost:5173")),
		ShutdownTimeout: 10 * time.Second,
	}

	if err := cfg.LogLevel.UnmarshalText([]byte(getenv("LOG_LEVEL", "info"))); err != nil {
		return Config{}, fmt.Errorf("LOG_LEVEL: %w", err)
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
