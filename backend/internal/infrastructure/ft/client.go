// Package ft is a client for the 42 intra API (https://api.intra.42.fr/apidoc).
package ft

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/upstream"
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

// defaultMaxWait is how long a request may wait for the rate limiters or a
// 429's Retry-After before failing with upstream.ErrBusy.
const defaultMaxWait = 5 * time.Second

// Config holds the 42 OAuth application credentials and rate limits.
type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	// RatePerSecond and RatePerHour cap API requests; 0 means the 42 default.
	RatePerSecond int
	RatePerHour   int
	// MaxWait caps the wait for a rate limiter or Retry-After; longer waits
	// fail with upstream.ErrBusy. 0 means 5s.
	MaxWait time.Duration
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
	if cfg.MaxWait <= 0 {
		cfg.MaxWait = defaultMaxWait
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

// AuthorizeURL is the 42 page where the user approves the sign-in; 42 then
// redirects to RedirectURI with a code and the given state.
func (c *Client) AuthorizeURL(state string) string {
	q := url.Values{
		"client_id":     {c.cfg.ClientID},
		"redirect_uri":  {c.cfg.RedirectURI},
		"response_type": {"code"},
		"scope":         {"public projects"},
		"state":         {state},
	}
	return c.cfg.BaseURL + "/oauth/authorize?" + q.Encode()
}

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

type userResponse struct {
	ID          int    `json:"id"`
	Login       string `json:"login"`
	DisplayName string `json:"displayname"`
	Staff       bool   `json:"staff?"`
	Alumni      bool   `json:"alumni?"`
	Image       struct {
		Link     string `json:"link"`
		Versions struct {
			Medium string `json:"medium"`
		} `json:"versions"`
	} `json:"image"`
}

// Profile returns the user who owns accessToken.
func (c *Client) Profile(ctx context.Context, accessToken string) (student.Student, error) {
	return c.user(ctx, accessToken, "/v2/me")
}

// UserByLogin returns a 42 user by login, or student.ErrNotFound.
func (c *Client) UserByLogin(ctx context.Context, accessToken, login string) (student.Student, error) {
	return c.user(ctx, accessToken, "/v2/users/"+url.PathEscape(login))
}

func (c *Client) user(ctx context.Context, accessToken, path string) (student.Student, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+path, nil)
	if err != nil {
		return student.Student{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	var u userResponse
	if _, err := c.do(req, &u); err != nil {
		var se *statusError
		if errors.As(err, &se) && se.Code == http.StatusNotFound {
			return student.Student{}, student.ErrNotFound
		}
		return student.Student{}, fmt.Errorf("42 %s: %w", path, err)
	}

	image := u.Image.Versions.Medium
	if image == "" {
		image = u.Image.Link
	}
	s, err := student.New(student.ID(u.ID), u.Login, u.DisplayName, image)
	if err != nil {
		return student.Student{}, fmt.Errorf("42 %s: %w", path, err)
	}
	s.Staff, s.Alumni = u.Staff, u.Alumni
	return s, nil
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
		ID   int    `json:"id"`
		Slug string `json:"slug"`
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
			cp := student.CompletedProject{ProjectID: curriculum.ProjectID(pu.Project.ID), Slug: pu.Project.Slug}
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
// into out. 429 responses are retried after Retry-After. Waits longer than
// MaxWait, and a 429 that outlasts the retries, return upstream.ErrBusy. An
// OAuth "invalid_grant" error is returned as auth.ErrInvalidGrant.
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

		if res.StatusCode == http.StatusTooManyRequests {
			res.Body.Close()
			delay := retryAfter(res.Header.Get("Retry-After"))
			if attempt >= maxRetries || delay > c.cfg.MaxWait {
				return nil, upstream.ErrBusy
			}
			if err := sleep(ctx, delay); err != nil {
				return nil, err
			}
			continue
		}
		return decode(res, out)
	}
}

// wait takes a token from both limiters, sleeping for at most MaxWait. If the
// tokens would come later, nothing is taken and upstream.ErrBusy is returned.
func (c *Client) wait(ctx context.Context) error {
	now := time.Now()
	hour := c.perHour.ReserveN(now, 1)
	second := c.perSecond.ReserveN(now, 1)
	cancel := func() {
		hour.CancelAt(now)
		second.CancelAt(now)
	}

	if !hour.OK() || !second.OK() {
		cancel()
		return upstream.ErrBusy
	}
	delay := max(hour.DelayFrom(now), second.DelayFrom(now))
	if delay > c.cfg.MaxWait {
		cancel()
		return upstream.ErrBusy
	}
	if err := sleep(ctx, delay); err != nil {
		cancel()
		return err
	}
	return nil
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
		return nil, &statusError{Code: res.StatusCode, Message: strings.TrimSpace(oauthErr.Error + " " + oauthErr.Description)}
	}
	return res.Header, json.Unmarshal(body, out)
}

// statusError is a non-2xx response from the 42 API.
type statusError struct {
	Code    int
	Message string
}

func (e *statusError) Error() string {
	return fmt.Sprintf("status %d: %s", e.Code, e.Message)
}
