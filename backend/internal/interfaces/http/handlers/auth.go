package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/auth"
)

// tokenResponse mirrors the 42 OAuth token JSON, which the frontend stores as is.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	CreatedAt    int64  `json:"created_at"`
}

// OAuthToken exchanges an authorization code for tokens and registers the
// student. Body: {"code": "..."}. A "redirect_uri" field is accepted for
// compatibility but ignored: the configured FT_REDIRECT_URI is used.
func OAuthToken(svc *auth.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			Code string `json:"code"`
		}
		if err := c.Bind().JSON(&req); err != nil || req.Code == "" {
			return fiber.NewError(fiber.StatusBadRequest, "code is required")
		}

		token, err := svc.Login(c.Context(), req.Code)
		if errors.Is(err, auth.ErrInvalidGrant) {
			return fiber.NewError(fiber.StatusBadRequest, "invalid or expired authorization code")
		}
		if err != nil {
			return err
		}
		return c.JSON(tokenResponse(token))
	}
}

// OAuthRefresh exchanges a refresh token for a new token pair.
// Body: {"refresh_token": "..."}.
func OAuthRefresh(svc *auth.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.Bind().JSON(&req); err != nil || req.RefreshToken == "" {
			return fiber.NewError(fiber.StatusBadRequest, "refresh_token is required")
		}

		token, err := svc.Refresh(c.Context(), req.RefreshToken)
		if errors.Is(err, auth.ErrInvalidGrant) {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired refresh token")
		}
		if err != nil {
			return err
		}
		return c.JSON(tokenResponse(token))
	}
}
