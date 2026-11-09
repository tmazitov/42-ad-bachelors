package http

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/upstream"
)

func TestErrorHandler(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: errorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))})
	app.Get("/busy", func(fiber.Ctx) error { return fmt.Errorf("refresh: %w", upstream.ErrBusy) })
	app.Get("/boom", func(fiber.Ctx) error { return errors.New("secret detail") })
	app.Get("/teapot", func(fiber.Ctx) error { return fiber.NewError(fiber.StatusTeapot, "short and stout") })

	for path, want := range map[string]struct {
		status int
		body   string
	}{
		"/busy":   {503, "42 API is busy"},
		"/boom":   {500, `"internal server error"`},
		"/teapot": {418, "short and stout"},
	} {
		res, err := app.Test(httptest.NewRequest("GET", path, nil))
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		if res.StatusCode != want.status || !strings.Contains(string(body), want.body) {
			t.Errorf("%s = %d %s, want %d containing %q", path, res.StatusCode, body, want.status, want.body)
		}
		if path == "/busy" && res.Header.Get("Retry-After") != "60" {
			t.Errorf("busy Retry-After = %q", res.Header.Get("Retry-After"))
		}
	}
}
