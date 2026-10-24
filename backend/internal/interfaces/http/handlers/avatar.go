package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/avatar"
)

// Avatar proxies a 42 CDN user image: GET /avatar?url=<image url>.
// The URL must come from a 42 API "image" field.
func Avatar(svc *avatar.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		img, err := svc.Get(c.Context(), c.Query("url"))
		switch {
		case errors.Is(err, avatar.ErrInvalidURL):
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		case errors.Is(err, avatar.ErrNotFound):
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		case err != nil:
			return err
		}

		// Image URLs contain a content hash, so they can be cached for long.
		c.Set(fiber.HeaderCacheControl, "public, max-age=604800")
		c.Set(fiber.HeaderContentType, img.ContentType)
		return c.Send(img.Data)
	}
}
