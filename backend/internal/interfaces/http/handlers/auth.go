package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/auth"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

const (
	sessionCookie = "session"
	stateCookie   = "oauth_state"
	stateTTL      = 10 * time.Minute
	studentKey    = "student"
)

// AuthConfig holds the HTTP settings for sign-in.
type AuthConfig struct {
	SecureCookies bool
}

// Login starts the 42 OAuth flow: GET /auth/login redirects to 42 with a
// random state that the callback checks against a short-lived cookie.
func Login(svc *auth.Service, cfg AuthConfig) fiber.Handler {
	return func(c fiber.Ctx) error {
		state, err := randomToken()
		if err != nil {
			return err
		}
		c.Cookie(&fiber.Cookie{
			Name:     stateCookie,
			Value:    state,
			Path:     "/api/v1/auth",
			MaxAge:   int(stateTTL.Seconds()),
			HTTPOnly: true,
			Secure:   cfg.SecureCookies,
			SameSite: fiber.CookieSameSiteLaxMode,
		})
		return c.Redirect().To(svc.AuthorizeURL(state))
	}
}

// Callback finishes the OAuth flow. 42 redirects the browser to the app's
// /auth/callback page (which shows a loader), and that page posts the query
// here: {"code", "state", "error"}. On success it sets the session cookie and
// returns 204; otherwise 400 with {"error": "denied" | "state" | "failed"}.
func Callback(svc *auth.Service, cfg AuthConfig, log *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			Code  string `json:"code"`
			State string `json:"state"`
			Error string `json:"error"`
		}
		fail := func(reason string) error {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": reason})
		}

		expected := c.Cookies(stateCookie)
		c.Cookie(&fiber.Cookie{Name: stateCookie, Path: "/api/v1/auth", MaxAge: -1, HTTPOnly: true,
			Secure: cfg.SecureCookies, SameSite: fiber.CookieSameSiteLaxMode})

		if err := c.Bind().JSON(&req); err != nil {
			return fail("failed")
		}
		if req.Error != "" {
			return fail("denied") // the user declined on 42
		}
		if req.State == "" || expected == "" || subtle.ConstantTimeCompare([]byte(req.State), []byte(expected)) != 1 {
			return fail("state")
		}
		if req.Code == "" {
			return fail("failed")
		}

		token, expires, err := svc.Login(c.Context(), req.Code)
		if err != nil {
			if !errors.Is(err, auth.ErrInvalidGrant) {
				log.ErrorContext(c.Context(), "login failed", "err", err)
			}
			return fail("failed")
		}

		c.Cookie(&fiber.Cookie{
			Name:     sessionCookie,
			Value:    token,
			Path:     "/",
			Expires:  expires,
			HTTPOnly: true,
			Secure:   cfg.SecureCookies,
			SameSite: fiber.CookieSameSiteLaxMode,
		})
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// Logout ends the session and clears its cookie.
func Logout(svc *auth.Service, cfg AuthConfig) fiber.Handler {
	return func(c fiber.Ctx) error {
		if err := svc.Logout(c.Context(), c.Cookies(sessionCookie)); err != nil {
			return err
		}
		c.Cookie(&fiber.Cookie{Name: sessionCookie, Path: "/", MaxAge: -1, HTTPOnly: true,
			Secure: cfg.SecureCookies, SameSite: fiber.CookieSameSiteLaxMode})
		return c.SendStatus(fiber.StatusNoContent)
	}
}

// RequireSession rejects requests without a valid session with 401 and
// stores the signed-in student's ID for SignedIn.
func RequireSession(svc *auth.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		id, err := svc.Authenticate(c.Context(), c.Cookies(sessionCookie))
		if errors.Is(err, auth.ErrUnauthenticated) {
			return fiber.NewError(fiber.StatusUnauthorized, "not signed in")
		}
		if err != nil {
			return err
		}
		fiber.Locals(c, studentKey, id)
		return c.Next()
	}
}

// SignedIn returns the student ID stored by RequireSession.
func SignedIn(c fiber.Ctx) student.ID {
	return fiber.Locals[student.ID](c, studentKey)
}

func randomToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
