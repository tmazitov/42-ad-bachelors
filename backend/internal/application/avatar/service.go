// Package avatar serves 42 profile pictures. The 42 CDN only answers
// authenticated requests, which an <img> tag cannot make, so images are
// fetched server-side and cached in memory.
package avatar

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	// ErrInvalidURL means the URL does not point to a 42 user image.
	ErrInvalidURL = errors.New("not a 42 user image URL")
	// ErrNotFound means the CDN has no such image.
	ErrNotFound = errors.New("image not found")
)

const (
	allowedHost = "cdn.intra.42.fr"
	allowedPath = "/users/"
)

// Image is a fetched picture.
type Image struct {
	ContentType string
	Data        []byte
}

// Source fetches an image from the 42 CDN.
type Source interface {
	Image(ctx context.Context, url string) (Image, error)
}

type entry struct {
	image   Image
	expires time.Time
}

// Service fetches and caches 42 user images.
type Service struct {
	source     Source
	ttl        time.Duration
	maxEntries int
	now        func() time.Time

	mu    sync.Mutex
	cache map[string]entry
}

func NewService(source Source) *Service {
	return &Service{
		source:     source,
		ttl:        24 * time.Hour,
		maxEntries: 1000,
		now:        time.Now,
		cache:      map[string]entry{},
	}
}

// Get returns the image at rawURL, which must be a 42 CDN user image.
func (s *Service) Get(ctx context.Context, rawURL string) (Image, error) {
	if !isAllowed(rawURL) {
		return Image{}, ErrInvalidURL
	}
	// The URL is kept as a cache key, so it must not share memory with the
	// caller: HTTP frameworks such as Fiber reuse request buffers, which would
	// later turn this key into another user's URL.
	rawURL = strings.Clone(rawURL)

	if img, ok := s.cached(rawURL); ok {
		return img, nil
	}

	img, err := s.source.Image(ctx, rawURL)
	if err != nil {
		return Image{}, err
	}
	s.store(rawURL, img)
	return img, nil
}

func isAllowed(rawURL string) bool {
	u, err := url.Parse(rawURL)
	return err == nil &&
		u.Scheme == "https" &&
		u.Host == allowedHost &&
		u.User == nil &&
		strings.HasPrefix(u.Path, allowedPath) &&
		!strings.Contains(u.Path, "..")
}

func (s *Service) cached(key string) (Image, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.cache[key]
	if !ok || s.now().After(e.expires) {
		return Image{}, false
	}
	return e.image, true
}

func (s *Service) store(key string, img Image) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.cache) >= s.maxEntries {
		now := s.now()
		for k, e := range s.cache {
			if now.After(e.expires) {
				delete(s.cache, k)
			}
		}
		// Still full: drop an arbitrary entry.
		for k := range s.cache {
			if len(s.cache) < s.maxEntries {
				break
			}
			delete(s.cache, k)
		}
	}
	s.cache[key] = entry{image: img, expires: s.now().Add(s.ttl)}
}
