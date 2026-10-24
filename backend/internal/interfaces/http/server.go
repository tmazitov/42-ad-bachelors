// Package http is the delivery layer: it maps HTTP requests to application
// use cases and domain results to JSON. No business logic lives here.
package http

import (
	"database/sql"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/auth"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/progress"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/config"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/interfaces/http/handlers"
)

// Deps are the dependencies the HTTP layer needs.
type Deps struct {
	DB         *sql.DB
	Curriculum curriculum.Curriculum
	Auth       *auth.Service
	Progress   *progress.Service
	Log        *slog.Logger
}

// NewServer builds the Fiber app with middleware and routes.
func NewServer(cfg config.Config, deps Deps) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "42-ad-bachelors",
		ErrorHandler: errorHandler(deps.Log),
	})

	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(requestLogger(deps.Log))
	app.Use(cors.New(cors.Config{
		AllowOrigins:  cfg.CORSOrigins,
		AllowHeaders:  []string{"Content-Type", "Authorization"},
		ExposeHeaders: []string{"X-Total", "X-Page", "X-Per-Page"},
	}))

	app.Get("/healthz", handlers.Health(deps.DB))

	api := app.Group("/api/v1")
	api.Get("/curriculum", handlers.Curriculum(deps.Curriculum))
	api.Post("/oauth/token", handlers.OAuthToken(deps.Auth))
	api.Post("/oauth/refresh", handlers.OAuthRefresh(deps.Auth))
	api.Get("/leaderboard", handlers.Leaderboard(deps.Progress))

	return app
}

// errorHandler renders every error as {"error": "..."}. Errors that are not
// *fiber.Error are logged and hidden behind a generic 500.
func errorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		var fe *fiber.Error
		if errors.As(err, &fe) {
			return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
		}
		log.Error("unhandled error",
			"err", err,
			"method", c.Method(),
			"path", c.Path(),
			"request_id", requestid.FromContext(c),
		)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
}
