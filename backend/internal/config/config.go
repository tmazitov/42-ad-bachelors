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
	FT              FTConfig
}

// FTConfig holds the 42 OAuth application settings
// (https://profile.intra.42.fr/oauth/applications).
type FTConfig struct {
	APIBase      string
	ClientID     string
	ClientSecret string
	RedirectURI  string // must match the redirect URI the frontend sends users to
}

// Load reads the configuration from the environment.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		DatabasePath:    getenv("DATABASE_PATH", "./data/app.db"),
		CORSOrigins:     splitList(getenv("CORS_ALLOW_ORIGINS", "http://localhost:5173")),
		ShutdownTimeout: 10 * time.Second,
		FT: FTConfig{
			APIBase:      getenv("FT_API_BASE", "https://api.intra.42.fr"),
			ClientID:     os.Getenv("FT_CLIENT_ID"),
			ClientSecret: os.Getenv("FT_CLIENT_SECRET"),
			RedirectURI:  os.Getenv("FT_REDIRECT_URI"),
		},
	}

	var missing []string
	for _, kv := range [][2]string{
		{"FT_CLIENT_ID", cfg.FT.ClientID},
		{"FT_CLIENT_SECRET", cfg.FT.ClientSecret},
		{"FT_REDIRECT_URI", cfg.FT.RedirectURI},
	} {
		if kv[1] == "" {
			missing = append(missing, kv[0])
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
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
	for part := range strings.SplitSeq(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
