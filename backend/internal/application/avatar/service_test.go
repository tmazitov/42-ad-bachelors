package avatar

import (
	"context"
	"errors"
	"testing"
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
