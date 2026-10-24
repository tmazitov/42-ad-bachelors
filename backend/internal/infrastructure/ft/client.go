// Package ft is a client for the 42 intra API (https://api.intra.42.fr/apidoc).
package ft

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/auth"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/progress"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/curriculum"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

// Default 42 API limits per application.
const (
	defaultRatePerSecond = 2
	defaultRatePerHour   = 1200
)

// maxRetries is how many times a 429 response is retried.
const maxRetries = 3

// Config holds the 42 OAuth application credentials and rate limits.
type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	// RatePerSecond and RatePerHour cap API requests; 0 means the 42 default.
	RatePerSecond int
	RatePerHour   int
}

// Client talks to the 42 API. It implements auth.Provider,
// progress.ProjectSource and avatar.Source. All API requests share one rate
// limiter.
type Client struct {
	cfg  Config
	http *http.Client
	// Both limiters must allow a request: a short-term one for the per-second
	// limit and a bucket holding the hourly budget.
	perSecond *rate.Limiter
	perHour   *rate.Limiter
	app       appTokenCache
}

func NewClient(cfg Config) *Client {
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.RatePerSecond <= 0 {
		cfg.RatePerSecond = defaultRatePerSecond
	}
	if cfg.RatePerHour <= 0 {
		cfg.RatePerHour = defaultRatePerHour
	}
	return &Client{
		cfg:       cfg,
		http:      &http.Client{Timeout: 15 * time.Second},
		perSecond: rate.NewLimiter(rate.Limit(cfg.RatePerSecond), cfg.RatePerSecond),
		perHour:   rate.NewLimiter(rate.Limit(float64(cfg.RatePerHour)/3600), cfg.RatePerHour),
	}
}

var (
	_ auth.Provider          = (*Client)(nil)
	_ progress.ProjectSource = (*Client)(nil)
)

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
	if _, err := c.do(req, &body); err != nil {
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
	if _, err := c.do(req, &me); err != nil {
		return auth.Profile{}, fmt.Errorf("42 /v2/me: %w", err)
	}

	image := me.Image.Versions.Medium
	if image == "" {
		image = me.Image.Link
	}
	return auth.Profile{ID: me.ID, Login: me.Login, DisplayName: me.DisplayName, ImageURL: image}, nil
}

const (
	projectsPerPage = 100
	maxProjectPages = 10
)

type projectUserResponse struct {
	FinalMark *int       `json:"final_mark"`
	Validated *bool      `json:"validated?"`
	MarkedAt  *time.Time `json:"marked_at"`
	Project   struct {
		ID int `json:"id"`
	} `json:"project"`
}

// CompletedProjects returns the user's validated finished projects, following
// pagination.
func (c *Client) CompletedProjects(ctx context.Context, accessToken string, id student.ID) ([]student.CompletedProject, error) {
	var out []student.CompletedProject
	for page := 1; page <= maxProjectPages; page++ {
		u := fmt.Sprintf("%s/v2/users/%d/projects_users?filter[status]=finished&per_page=%d&page=%d",
			c.cfg.BaseURL, id, projectsPerPage, page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		var batch []projectUserResponse
		header, err := c.do(req, &batch)
		if err != nil {
			return nil, fmt.Errorf("42 projects_users page %d: %w", page, err)
		}

		for _, pu := range batch {
			if pu.Validated == nil || !*pu.Validated {
				continue
			}
			cp := student.CompletedProject{ProjectID: curriculum.ProjectID(pu.Project.ID)}
			if pu.FinalMark != nil {
				cp.FinalMark = *pu.FinalMark
			}
			if pu.MarkedAt != nil {
				cp.MarkedAt = *pu.MarkedAt
			}
			out = append(out, cp)
		}

		total, _ := strconv.Atoi(header.Get("X-Total"))
		if len(batch) < projectsPerPage || (total > 0 && page*projectsPerPage >= total) {
			return out, nil
		}
	}
	return nil, fmt.Errorf("42 projects_users: more than %d pages", maxProjectPages)
}

// do waits for the rate limiters, sends req and decodes a 2xx JSON response
// into out. 429 responses are retried after Retry-After. An OAuth
// "invalid_grant" error is returned as auth.ErrInvalidGrant.
func (c *Client) do(req *http.Request, out any) (http.Header, error) {
	ctx := req.Context()
	req.Header.Set("Accept", "application/json")

	for attempt := 0; ; attempt++ {
		if err := c.wait(ctx); err != nil {
			return nil, err
		}

		if attempt > 0 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			req.Body = body
		}

		res, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}

		if res.StatusCode == http.StatusTooManyRequests && attempt < maxRetries {
			res.Body.Close()
			if err := sleep(ctx, retryAfter(res.Header.Get("Retry-After"))); err != nil {
				return nil, err
			}
			continue
		}
		return decode(res, out)
	}
}

func (c *Client) wait(ctx context.Context) error {
	if err := c.perHour.Wait(ctx); err != nil {
		return err
	}
	return c.perSecond.Wait(ctx)
}

// retryAfter parses a Retry-After header in seconds, defaulting to 1s.
func retryAfter(v string) time.Duration {
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return time.Second
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func decode(res *http.Response, out any) (http.Header, error) {
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		var oauthErr struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
		}
		_ = json.Unmarshal(body, &oauthErr)
		if oauthErr.Error == "invalid_grant" {
			return nil, auth.ErrInvalidGrant
		}
		return nil, fmt.Errorf("status %d: %s %s", res.StatusCode, oauthErr.Error, oauthErr.Description)
	}
	return res.Header, json.Unmarshal(body, out)
}
