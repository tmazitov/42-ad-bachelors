package avatar

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unsafe"
)

type countingSource struct{ calls int }

func (c *countingSource) Image(context.Context, string) (Image, error) {
	c.calls++
	return Image{ContentType: "image/jpeg", Data: []byte("jpg")}, nil
}

func TestGetRejectsNon42URLs(t *testing.T) {
	svc := NewService(&countingSource{})
	for _, u := range []string{
		"http://cdn.intra.42.fr/users/x/a.jpg",       // not https
		"https://evil.com/users/x/a.jpg",             // other host
		"https://cdn.intra.42.fr.evil.com/users/a",   // suffix trick
		"https://cdn.intra.42.fr/other/a.jpg",        // other path
		"https://user@cdn.intra.42.fr/users/a.jpg",   // userinfo
		"https://cdn.intra.42.fr/users/../admin.jpg", // traversal
		"",
	} {
		if _, err := svc.Get(context.Background(), u); !errors.Is(err, ErrInvalidURL) {
			t.Errorf("Get(%q) err = %v, want ErrInvalidURL", u, err)
		}
	}
}

func TestGetCaches(t *testing.T) {
	src := &countingSource{}
	svc := NewService(src)
	const u = "https://cdn.intra.42.fr/users/abc/medium_jdoe.jpg"

	for range 3 {
		img, err := svc.Get(context.Background(), u)
		if err != nil {
			t.Fatal(err)
		}
		if img.ContentType != "image/jpeg" {
			t.Errorf("content type = %q", img.ContentType)
		}
	}
	if src.calls != 1 {
		t.Errorf("source called %d times, want 1", src.calls)
	}
}

type echoSource struct{}

func (echoSource) Image(_ context.Context, u string) (Image, error) {
	return Image{ContentType: "image/jpeg", Data: []byte(u)}, nil
}

// A caller may pass a string backed by a buffer it reuses later (Fiber does
// this with query values). The cache must not pick up the new contents.
func TestCacheKeyDoesNotAliasCaller(t *testing.T) {
	svc := NewService(echoSource{})
	const a = "https://cdn.intra.42.fr/users/aaaa/medium_alice.jpg"
	const b = "https://cdn.intra.42.fr/users/bbbb/medium_bobby.jpg"
	buf := []byte(a)
	if _, err := svc.Get(context.Background(), unsafe.String(&buf[0], len(buf))); err != nil {
		t.Fatal(err)
	}
	copy(buf, b) // the buffer is reused for the next request

	img, err := svc.Get(context.Background(), b)
	if err != nil {
		t.Fatal(err)
	}
	if string(img.Data) != b {
		t.Errorf("got the image of %s for %s", img.Data, b)
	}
	for k := range svc.items {
		if k != a && k != b {
			t.Errorf("unexpected cache key %q", k)
		}
	}
	if _, ok := svc.items[a]; !ok {
		t.Error("entry for the first URL lost its key")
	}
}

// sizedSource returns images of a fixed size and counts calls per URL.
type sizedSource struct {
	size  int
	calls map[string]int
}

func (s *sizedSource) Image(_ context.Context, u string) (Image, error) {
	if s.calls == nil {
		s.calls = map[string]int{}
	}
	s.calls[u]++
	if strings.Contains(u, "missing") {
		return Image{}, ErrNotFound
	}
	return Image{ContentType: "image/jpeg", Data: make([]byte, s.size)}, nil
}

func TestQueryAndFragmentDoNotBypassTheCache(t *testing.T) {
	src := &sizedSource{size: 10}
	svc := NewService(src)
	const base = "https://cdn.intra.42.fr/users/abc/medium_jdoe.jpg"
	for _, u := range []string{base, base + "?a=1", base + "?a=2", base + "#x", base + "?"} {
		if _, err := svc.Get(context.Background(), u); err != nil {
			t.Fatalf("Get(%q): %v", u, err)
		}
	}
	if len(src.calls) != 1 || src.calls[base] != 1 {
		t.Errorf("source calls = %v, want one call for the bare URL", src.calls)
	}
}

func TestCacheIsBoundedByBytes(t *testing.T) {
	src := &sizedSource{size: 400}
	svc := NewService(src)
	svc.maxBytes = 1000 // room for 2 images
	get := func(name string) {
		t.Helper()
		if _, err := svc.Get(context.Background(), "https://cdn.intra.42.fr/users/x/"+name+".jpg"); err != nil {
			t.Fatal(err)
		}
	}

	get("a")
	get("b")
	get("a") // a is now the most recently used
	get("c") // evicts b
	if svc.bytes != 800 || svc.order.Len() != 2 {
		t.Errorf("bytes = %d, entries = %d; want 800, 2", svc.bytes, svc.order.Len())
	}
	get("a")
	get("b")
	if n := src.calls["https://cdn.intra.42.fr/users/x/a.jpg"]; n != 1 {
		t.Errorf("a fetched %d times, want 1 (it was recently used)", n)
	}
	if n := src.calls["https://cdn.intra.42.fr/users/x/b.jpg"]; n != 2 {
		t.Errorf("b fetched %d times, want 2 (it was evicted)", n)
	}

	// An image larger than the whole cache is served but not stored.
	src.size = 2000
	get("huge")
	if svc.bytes > svc.maxBytes {
		t.Errorf("bytes = %d over the cap", svc.bytes)
	}
}

func TestNotFoundIsCachedBriefly(t *testing.T) {
	src := &sizedSource{size: 10}
	svc := NewService(src)
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	const u = "https://cdn.intra.42.fr/users/x/missing.jpg"

	for range 3 {
		if _, err := svc.Get(context.Background(), u); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	}
	if src.calls[u] != 1 {
		t.Errorf("calls = %d, want 1", src.calls[u])
	}
	now = now.Add(11 * time.Minute)
	svc.Get(context.Background(), u)
	if src.calls[u] != 2 {
		t.Errorf("calls = %d after 11 minutes, want 2", src.calls[u])
	}
}
