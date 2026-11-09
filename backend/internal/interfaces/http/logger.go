package http

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/upstream"
)

// requestLogger logs one structured line per request.
func requestLogger(log *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		status := c.Response().StatusCode()
		if err != nil {
			// The error handler runs after middleware; derive the status it will use.
			status = fiber.StatusInternalServerError
			if fe, ok := err.(*fiber.Error); ok {
				status = fe.Code
			} else if errors.Is(err, upstream.ErrBusy) {
				status = fiber.StatusServiceUnavailable
			}
		}

		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		}
		log.Log(c.Context(), level, "request",
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"duration_ms", float64(time.Since(start).Microseconds())/1000,
			"request_id", requestid.FromContext(c),
		)
		return err
	}
}
