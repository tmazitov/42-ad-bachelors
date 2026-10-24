package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the runtime configuration, read from environment variables.
type Config struct {
	HTTPAddr     string
	DatabasePath string
	// PublicURL is where users open the app (frontend and API share it), e.g.
	// https://bachelors.example.com. Cookies are marked Secure when it is https.
	PublicURL       string
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
	// SessionTTL is how long a sign-in lasts.
	SessionTTL time.Duration
	// StudentMaxAge: a student's data older than this is refreshed from 42
	// before it is returned by the API.
	StudentMaxAge time.Duration
	// SyncInterval is how often the background sync runs; 0 disables it.
	SyncInterval time.Duration
	// SyncMaxAge: the background sync only re-syncs students whose last sync
	// (at login, by a profile view or by a previous run) is at least this old.
	SyncMaxAge time.Duration
	FT         FTConfig
}

// SecureCookies reports whether cookies need the Secure flag.
func (c Config) SecureCookies() bool {
	return strings.HasPrefix(c.PublicURL, "https://")
}

// FTConfig holds the 42 OAuth application settings
// (https://profile.intra.42.fr/oauth/applications).
type FTConfig struct {
	APIBase      string
	ClientID     string
	ClientSecret string
	// RedirectURI is the app's OAuth callback page, which posts the code to
	// the API; it must be registered on the 42 application. Defaults to
	// PublicURL + /auth/callback.
	RedirectURI string
	// API request limits shared by logins, profile refreshes and the sync.
	RatePerSecond int
	RatePerHour   int
}

// Load reads the configuration from the environment.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		DatabasePath:    getenv("DATABASE_PATH", "./data/app.db"),
		PublicURL:       strings.TrimRight(getenv("PUBLIC_URL", "http://localhost:5173"), "/"),
		ShutdownTimeout: 10 * time.Second,
		FT: FTConfig{
			APIBase:      getenv("FT_API_BASE", "https://api.intra.42.fr"),
			ClientID:     os.Getenv("FT_CLIENT_ID"),
			ClientSecret: os.Getenv("FT_CLIENT_SECRET"),
		},
	}
	cfg.FT.RedirectURI = getenv("FT_REDIRECT_URI", cfg.PublicURL+"/auth/callback")

	var missing []string
	for _, kv := range [][2]string{
		{"FT_CLIENT_ID", cfg.FT.ClientID},
		{"FT_CLIENT_SECRET", cfg.FT.ClientSecret},
	} {
		if kv[1] == "" {
			missing = append(missing, kv[0])
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}
	if u, err := url.Parse(cfg.PublicURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return Config{}, fmt.Errorf("PUBLIC_URL: must be an http(s) URL, got %q", cfg.PublicURL)
	}

	var err error
	for _, d := range []struct {
		key      string
		fallback string
		dst      *time.Duration
		min      time.Duration
	}{
		{"SESSION_TTL", "720h", &cfg.SessionTTL, time.Minute},
		{"STUDENT_MAX_AGE", "2m", &cfg.StudentMaxAge, 0},
		{"SYNC_INTERVAL", "1h", &cfg.SyncInterval, 0},
		{"SYNC_MAX_AGE", "24h", &cfg.SyncMaxAge, 0},
	} {
		if *d.dst, err = time.ParseDuration(getenv(d.key, d.fallback)); err != nil || *d.dst < d.min {
			return Config{}, fmt.Errorf("%s: invalid duration %q", d.key, os.Getenv(d.key))
		}
	}
	if cfg.FT.RatePerSecond, err = getenvInt("FT_RATE_PER_SECOND", 2); err != nil {
		return Config{}, err
	}
	if cfg.FT.RatePerHour, err = getenvInt("FT_RATE_PER_HOUR", 1200); err != nil {
		return Config{}, err
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

func getenvInt(key string, fallback int) (int, error) {
	v := getenv(key, "")
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s: must be a positive integer, got %q", key, v)
	}
	return n, nil
}
