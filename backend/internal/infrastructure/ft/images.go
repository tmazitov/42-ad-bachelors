package ft

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/avatar"
)

var _ avatar.Source = (*Client)(nil)

const maxImageSize = 5 << 20

// appTokenCache holds the client_credentials token shared by server-side calls.
type appTokenCache struct {
	mu      sync.Mutex
	token   string
	expires time.Time
}

// AppToken returns a cached application (client_credentials) token, requesting
// a new one when it is about to expire.
func (c *Client) AppToken(ctx context.Context) (string, error) {
	c.app.mu.Lock()
	defer c.app.mu.Unlock()

	if c.app.token != "" && time.Until(c.app.expires) > time.Minute {
		return c.app.token, nil
	}

	token, err := c.requestToken(ctx, url.Values{"grant_type": {"client_credentials"}})
	if err != nil {
		return "", fmt.Errorf("app token: %w", err)
	}
	c.app.token = token.AccessToken
	c.app.expires = time.Unix(token.CreatedAt, 0).Add(time.Duration(token.ExpiresIn) * time.Second)
	return c.app.token, nil
}

func (c *Client) invalidateAppToken() {
	c.app.mu.Lock()
	c.app.token = ""
	c.app.mu.Unlock()
}

// Image downloads a picture from the 42 CDN with the app token. CDN requests
// do not count against the API rate limit, so the limiter is not used.
func (c *Client) Image(ctx context.Context, imageURL string) (avatar.Image, error) {
	token, err := c.AppToken(ctx)
	if err != nil {
		return avatar.Image{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return avatar.Image{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := c.http.Do(req)
	if err != nil {
		return avatar.Image{}, err
	}
	defer res.Body.Close()

	switch {
	case res.StatusCode == http.StatusUnauthorized:
		c.invalidateAppToken()
		return avatar.Image{}, fmt.Errorf("42 CDN: token rejected")
	case res.StatusCode == http.StatusForbidden, res.StatusCode == http.StatusNotFound:
		return avatar.Image{}, avatar.ErrNotFound
	case res.StatusCode != http.StatusOK:
		return avatar.Image{}, fmt.Errorf("42 CDN: status %d", res.StatusCode)
	}

	contentType := res.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return avatar.Image{}, fmt.Errorf("42 CDN: unexpected content type %q", contentType)
	}

	data, err := io.ReadAll(io.LimitReader(res.Body, maxImageSize+1))
	if err != nil {
		return avatar.Image{}, err
	}
	if len(data) > maxImageSize {
		return avatar.Image{}, fmt.Errorf("42 CDN: image larger than %d bytes", maxImageSize)
	}
	return avatar.Image{ContentType: contentType, Data: data}, nil
}
