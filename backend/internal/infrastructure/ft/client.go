// Package ft is a client for the 42 intra API (https://api.intra.42.fr/apidoc).
package ft

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/auth"
)

// Config holds the 42 OAuth application credentials.
type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// Client talks to the 42 API. It implements auth.Provider.
type Client struct {
	cfg  Config
	http *http.Client
}

func NewClient(cfg Config) *Client {
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &Client{cfg: cfg, http: &http.Client{Timeout: 15 * time.Second}}
}

var _ auth.Provider = (*Client)(nil)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	CreatedAt    int64  `json:"created_at"`
}

func (c *Client) ExchangeCode(ctx context.Context, code string) (auth.Token, error) {
	return c.requestToken(ctx, url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {c.cfg.RedirectURI},
	})
}

func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (auth.Token, error) {
	return c.requestToken(ctx, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
}

func (c *Client) requestToken(ctx context.Context, form url.Values) (auth.Token, error) {
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return auth.Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var body tokenResponse
	if err := c.do(req, &body); err != nil {
		return auth.Token{}, fmt.Errorf("42 token request: %w", err)
	}
	return auth.Token(body), nil
}

type meResponse struct {
	ID          int    `json:"id"`
	Login       string `json:"login"`
	DisplayName string `json:"displayname"`
	Image       struct {
		Link     string `json:"link"`
		Versions struct {
			Medium string `json:"medium"`
		} `json:"versions"`
	} `json:"image"`
}

func (c *Client) Profile(ctx context.Context, accessToken string) (auth.Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/v2/me", nil)
	if err != nil {
		return auth.Profile{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	var me meResponse
	if err := c.do(req, &me); err != nil {
		return auth.Profile{}, fmt.Errorf("42 /v2/me: %w", err)
	}

	image := me.Image.Versions.Medium
	if image == "" {
		image = me.Image.Link
	}
	return auth.Profile{ID: me.ID, Login: me.Login, DisplayName: me.DisplayName, ImageURL: image}, nil
}

// do sends req and decodes a 2xx JSON response into out. An OAuth
// "invalid_grant" error is returned as auth.ErrInvalidGrant.
func (c *Client) do(req *http.Request, out any) error {
	req.Header.Set("Accept", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		var oauthErr struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		_ = json.Unmarshal(body, &oauthErr)
		if oauthErr.Error == "invalid_grant" {
			return auth.ErrInvalidGrant
		}
		return fmt.Errorf("status %d: %s %s", res.StatusCode, oauthErr.Error, oauthErr.Description)
	}
	return json.Unmarshal(body, out)
}
