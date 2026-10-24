package handlers

import (
	"database/sql"

	"github.com/gofiber/fiber/v3"
)

// Health reports whether the service and its database are reachable.
func Health(db *sql.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		if err := db.PingContext(c.Context()); err != nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "database unavailable")
		}
		return c.JSON(fiber.Map{"status": "ok"})
	}
}
