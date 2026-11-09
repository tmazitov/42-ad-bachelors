// Package avatar serves 42 profile pictures. The 42 CDN only answers
// authenticated requests, which an <img> tag cannot make, so images are
// fetched server-side and cached in memory.
package avatar

import (
	"container/list"
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

	defaultTTL         = 24 * time.Hour
	defaultNotFoundTTL = 10 * time.Minute
	// defaultMaxBytes caps the image data held in memory.
	defaultMaxBytes = 50 << 20
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

// entry is a cached image, or a cached "not found" (notFound set, no data).
type entry struct {
	key      string
	image    Image
	notFound bool
	expires  time.Time
}

// Service fetches and caches 42 user images in a least-recently-used cache
// bounded by the total size of the images.
type Service struct {
	source      Source
	ttl         time.Duration
	notFoundTTL time.Duration
	maxBytes    int
	now         func() time.Time

	mu    sync.Mutex
	order *list.List               // front = most recently used; values are *entry
	items map[string]*list.Element // by normalized URL
	bytes int
}

func NewService(source Source) *Service {
	return &Service{
		source:      source,
		ttl:         defaultTTL,
		notFoundTTL: defaultNotFoundTTL,
		maxBytes:    defaultMaxBytes,
		now:         time.Now,
		order:       list.New(),
		items:       map[string]*list.Element{},
	}
}

// Get returns the image at rawURL, which must be a 42 CDN user image. The
// query and fragment are ignored, so they cannot be used to bypass the cache.
func (s *Service) Get(ctx context.Context, rawURL string) (Image, error) {
	// normalize builds a new string, so the cache key never shares memory
	// with the caller (HTTP frameworks such as Fiber reuse request buffers).
	key, ok := normalize(rawURL)
	if !ok {
		return Image{}, ErrInvalidURL
	}

	if e, ok := s.cached(key); ok {
		if e.notFound {
			return Image{}, ErrNotFound
		}
		return e.image, nil
	}

	img, err := s.source.Image(ctx, key)
	switch {
	case errors.Is(err, ErrNotFound):
		s.store(&entry{key: key, notFound: true, expires: s.now().Add(s.notFoundTTL)})
		return Image{}, err
	case err != nil:
		return Image{}, err
	}
	s.store(&entry{key: key, image: img, expires: s.now().Add(s.ttl)})
	return img, nil
}

// normalize returns the URL without query and fragment, if it is a 42 CDN
// user image.
func normalize(rawURL string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil ||
		u.Scheme != "https" ||
		u.Host != allowedHost ||
		u.User != nil ||
		!strings.HasPrefix(u.Path, allowedPath) ||
		strings.Contains(u.Path, "..") {
		return "", false
	}
	u.RawQuery, u.ForceQuery, u.Fragment, u.RawFragment = "", false, "", ""
	return u.String(), true
}

func (s *Service) cached(key string) (*entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	el, ok := s.items[key]
	if !ok {
		return nil, false
	}
	e := el.Value.(*entry)
	if !s.now().Before(e.expires) {
		s.remove(el)
		return nil, false
	}
	s.order.MoveToFront(el)
	return e, true
}

func (s *Service) store(e *entry) {
	size := len(e.image.Data)
	if size > s.maxBytes {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if el, ok := s.items[e.key]; ok {
		s.remove(el)
	}
	for s.bytes+size > s.maxBytes && s.order.Len() > 0 {
		s.remove(s.order.Back())
	}
	s.items[e.key] = s.order.PushFront(e)
	s.bytes += size
}

func (s *Service) remove(el *list.Element) {
	e := s.order.Remove(el).(*entry)
	delete(s.items, e.key)
	s.bytes -= len(e.image.Data)
}
