package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/auth"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/avatar"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/progress"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/students"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/config"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/infrastructure/ft"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/infrastructure/sqlite"
	httpapi "github.com/tmazitov/42-ad-bachelors/backend/internal/interfaces/http"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/interfaces/scheduler"
)

func main() {
	// `server healthcheck` probes /healthz; used by the container health check
	// because the distroless image has no shell or curl.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := sqlite.Open(ctx, cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := sqlite.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	ftClient := ft.NewClient(ft.Config{
		BaseURL:       cfg.FT.APIBase,
		ClientID:      cfg.FT.ClientID,
		ClientSecret:  cfg.FT.ClientSecret,
		RedirectURI:   cfg.FT.RedirectURI,
		RatePerSecond: cfg.FT.RatePerSecond,
		RatePerHour:   cfg.FT.RatePerHour,
	})
	studentRepo := sqlite.NewStudentRepository(db)
	bachelor := curriculum.ADBachelor()
	progressSvc := progress.NewService(ftClient, ftClient, studentRepo, bachelor)
	authSvc := auth.NewService(ftClient, studentRepo, sqlite.NewSessionRepository(db), progressSvc, cfg.SessionTTL, log)

	studentsSvc := students.NewService(ftClient, progressSvc, ftClient, studentRepo, bachelor,
		cfg.StudentMaxAge, cfg.VisitorMaxAge, log)
	defer studentsSvc.Wait() // background refreshes, before the database is closed

	app := httpapi.NewServer(cfg, httpapi.Deps{
		DB:         db,
		Curriculum: bachelor,
		Auth:       authSvc,
		Students:   studentsSvc,
		Progress:   progressSvc,
		Avatar:     avatar.NewService(ftClient),
		Log:        log,
	})

	// Background sync; waited for before the database is closed.
	var background sync.WaitGroup
	defer background.Wait()
	if cfg.SyncInterval > 0 {
		background.Go(func() {
			scheduler.Every(ctx, cfg.SyncInterval, func(ctx context.Context) {
				syncAll(ctx, log, progressSvc, cfg.SyncMaxAge)
				if n, err := authSvc.DeleteExpiredSessions(ctx); err != nil {
					log.Error("deleting expired sessions failed", "err", err)
				} else if n > 0 {
					log.Info("deleted expired sessions", "count", n)
				}
			})
		})
	} else {
		log.Info("background sync disabled")
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.HTTPAddr, "public_url", cfg.PublicURL, "oauth_redirect_uri", cfg.FT.RedirectURI)
		serveErr <- app.Listen(cfg.HTTPAddr, fiber.ListenConfig{DisableStartupMessage: true})
	}()

	select {
	case err := <-serveErr:
		stop() // let the background sync finish before returning
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

// syncAll re-syncs students not synced within maxAge and logs the outcome.
func syncAll(ctx context.Context, log *slog.Logger, svc *progress.Service, maxAge time.Duration) {
	log.Info("sync started", "max_age", maxAge.String())
	report, err := svc.SyncAll(ctx, maxAge)
	for _, f := range report.Failures {
		log.Warn("sync failed for student", "login", f.Login, "err", f.Err)
	}

	attrs := []any{
		"students", report.Students,
		"skipped", report.Skipped,
		"synced", report.Synced,
		"failed", len(report.Failures),
		"duration", report.Duration.Round(time.Millisecond).String(),
	}
	switch {
	case errors.Is(err, context.Canceled):
		log.Info("sync interrupted", attrs...)
	case err != nil:
		log.Error("sync aborted", append(attrs, "err", err)...)
	default:
		log.Info("sync finished", attrs...)
	}
}

// healthcheck returns 0 if the server on HTTP_ADDR answers /healthz with 200.
func healthcheck() int {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	client := http.Client{Timeout: 3 * time.Second}
	res, err := client.Get("http://" + addr + "/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: status", res.StatusCode)
		return 1
	}
	return 0
}
