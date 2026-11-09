// Package http is the delivery layer: it maps HTTP requests to application
// use cases and domain results to JSON. No business logic lives here.
//
// The frontend is served from the same origin (a reverse proxy in production,
// the Vite dev proxy locally), so no CORS is needed and the session cookie is
// first-party.
package http

import (
	"database/sql"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/auth"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/avatar"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/progress"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/students"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/config"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/interfaces/http/handlers"
)

// Deps are the dependencies the HTTP layer needs.
type Deps struct {
	DB         *sql.DB
	Curriculum curriculum.Curriculum
	Auth       *auth.Service
	Students   *students.Service
	Progress   *progress.Service
	Avatar     *avatar.Service
	Log        *slog.Logger
}

// NewServer builds the Fiber app with middleware and routes.
func NewServer(cfg config.Config, deps Deps) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "42-ad-bachelors",
		// Copy request values (params, query, cookies) instead of pointing
		// into buffers that Fiber reuses for the next request; services keep
		// some of them, e.g. as cache keys.
		Immutable:    true,
		ErrorHandler: errorHandler(deps.Log),
	})

	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(requestLogger(deps.Log))

	app.Get("/healthz", handlers.Health(deps.DB))

	authCfg := handlers.AuthConfig{SecureCookies: cfg.SecureCookies()}
	signedIn := handlers.RequireSession(deps.Auth)

	api := app.Group("/api/v1")
	api.Get("/curriculum", handlers.Curriculum(deps.Curriculum))
	api.Get("/avatar", handlers.Avatar(deps.Avatar))

	api.Get("/auth/login", handlers.Login(deps.Auth, authCfg))
	api.Post("/auth/callback", handlers.Callback(deps.Auth, authCfg, deps.Log))
	api.Post("/auth/logout", handlers.Logout(deps.Auth, authCfg))

	api.Get("/me", signedIn, handlers.Me(deps.Students))
	api.Put("/me/stream", signedIn, handlers.SetMyStream(deps.Students))
	api.Put("/me/modules", signedIn, handlers.MarkMyModule(deps.Students))
	api.Get("/students", signedIn, handlers.SearchStudents(deps.Students))
	api.Get("/students/:login", signedIn, handlers.Student(deps.Students))
	api.Get("/leaderboard", signedIn, handlers.Leaderboard(deps.Progress))

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
