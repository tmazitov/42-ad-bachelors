package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/auth"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/avatar"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/progress"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/config"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/infrastructure/ft"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/infrastructure/sqlite"
	httpapi "github.com/tmazitov/42-ad-bachelors/backend/internal/interfaces/http"
)

func main() {
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
		BaseURL:      cfg.FT.APIBase,
		ClientID:     cfg.FT.ClientID,
		ClientSecret: cfg.FT.ClientSecret,
		RedirectURI:  cfg.FT.RedirectURI,
	})
	students := sqlite.NewStudentRepository(db)
	bachelor := curriculum.ADBachelor()
	progressSvc := progress.NewService(ftClient, students, bachelor)

	app := httpapi.NewServer(cfg, httpapi.Deps{
		DB:         db,
		Curriculum: bachelor,
		Auth:       auth.NewService(ftClient, students, progressSvc, log),
		Progress:   progressSvc,
		Avatar:     avatar.NewService(ftClient),
		Log:        log,
	})

	serveErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.HTTPAddr)
		serveErr <- app.Listen(cfg.HTTPAddr, fiber.ListenConfig{DisableStartupMessage: true})
	}()

	select {
	case err := <-serveErr:
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
