package ft

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/auth"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/avatar"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/upstream"
	"github.com/tmazitov/42-ad-bachelors/backend/internal/domain/student"
)

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewClient(Config{BaseURL: srv.URL, ClientID: "id", ClientSecret: "secret", RedirectURI: "http://app/cb"})
}

func TestExchangeCode(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		want := map[string]string{
			"grant_type": "authorization_code", "code": "abc", "redirect_uri": "http://app/cb",
			"client_id": "id", "client_secret": "secret",
		}
		for k, v := range want {
			if got := r.PostForm.Get(k); got != v {
				t.Errorf("%s = %q, want %q", k, got, v)
			}
		}
		w.Write([]byte(`{"access_token":"a","refresh_token":"r","token_type":"bearer","expires_in":7200,"created_at":1700000000}`))
	})

	token, err := c.ExchangeCode(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "a" || token.RefreshToken != "r" || token.ExpiresIn != 7200 || token.CreatedAt != 1700000000 {
		t.Errorf("token = %+v", token)
	}
}

func TestInvalidGrant(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant","error_description":"expired"}`))
	})

	if _, err := c.ExchangeCode(context.Background(), "old"); !errors.Is(err, auth.ErrInvalidGrant) {
		t.Fatalf("err = %v, want ErrInvalidGrant", err)
	}
}

func TestInvalidClientIsNotInvalidGrant(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid_client"}`))
	})

	_, err := c.ExchangeCode(context.Background(), "abc")
	if err == nil || errors.Is(err, auth.ErrInvalidGrant) {
		t.Fatalf("err = %v, want a non-grant error", err)
	}
}

func TestProfileFallsBackToImageLink(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		w.Write([]byte(`{"id":7,"login":"jdoe","displayname":"John","alumni?":true,"image":{"link":"http://img","versions":{"medium":null}}}`))
	})

	p, err := c.Profile(context.Background(), "tok")
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != 7 || p.Login != "jdoe" || p.DisplayName != "John" || p.ImageURL != "http://img" || !p.Alumni || p.Staff {
		t.Errorf("profile = %+v", p)
	}
}

func TestUserByLogin(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/users/jdoe" {
			w.Write([]byte(`{"id":7,"login":"jdoe","displayname":"John","staff?":true}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{}`))
	})

	u, err := c.UserByLogin(context.Background(), "app", "jdoe")
	if err != nil || u.ID != 7 || !u.Staff {
		t.Errorf("UserByLogin(jdoe) = %+v, %v", u, err)
	}
	if _, err := c.UserByLogin(context.Background(), "app", "ghost"); !errors.Is(err, student.ErrNotFound) {
		t.Errorf("UserByLogin(ghost) err = %v, want ErrNotFound", err)
	}
}

func TestAuthorizeURL(t *testing.T) {
	c := NewClient(Config{BaseURL: "https://api.intra.42.fr", ClientID: "id", RedirectURI: "http://app/api/v1/auth/callback"})
	u, err := url.Parse(c.AuthorizeURL("xyz"))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Path != "/oauth/authorize" || q.Get("client_id") != "id" || q.Get("state") != "xyz" ||
		q.Get("redirect_uri") != "http://app/api/v1/auth/callback" || q.Get("response_type") != "code" {
		t.Errorf("authorize URL = %s", u)
	}
}

func TestCompletedProjectsPaginatesAndFiltersValidated(t *testing.T) {
	var pages []string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/users/7/projects_users" || r.URL.Query().Get("filter[status]") != "finished" {
			t.Errorf("unexpected request %s", r.URL)
		}
		page := r.URL.Query().Get("page")
		pages = append(pages, page)
		w.Header().Set("X-Total", "101")
		if page == "1" {
			// 100 items: 99 unvalidated + 1 validated
			w.Write([]byte("["))
			for i := range 99 {
				fmt.Fprintf(w, `{"validated?":false,"project":{"id":%d}},`, 5000+i)
			}
			w.Write([]byte(`{"validated?":true,"final_mark":125,"marked_at":"2024-03-01T10:00:00.000Z","project":{"id":1314,"slug":"42cursus-libft"}}]`))
			return
		}
		w.Write([]byte(`[{"validated?":true,"final_mark":null,"marked_at":null,"project":{"id":1327}},{"validated?":null,"project":{"id":1}}]`))
	})

	got, err := c.CompletedProjects(context.Background(), "tok", 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 2 {
		t.Errorf("requested pages %v, want [1 2]", pages)
	}
	want := []student.CompletedProject{
		{ProjectID: 1314, Slug: "42cursus-libft", FinalMark: 125, MarkedAt: time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)},
		{ProjectID: 1327},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i].ProjectID != want[i].ProjectID || got[i].Slug != want[i].Slug || got[i].FinalMark != want[i].FinalMark || !got[i].MarkedAt.Equal(want[i].MarkedAt) {
			t.Errorf("#%d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestImageUsesCachedAppToken(t *testing.T) {
	var tokenRequests int
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			tokenRequests++
			r.ParseForm()
			if r.PostForm.Get("grant_type") != "client_credentials" {
				t.Errorf("grant_type = %q", r.PostForm.Get("grant_type"))
			}
			fmt.Fprintf(w, `{"access_token":"app","expires_in":7200,"created_at":%d}`, time.Now().Unix())
		case "/users/x/medium_a.jpg":
			if r.Header.Get("Authorization") != "Bearer app" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write([]byte("jpeg-bytes"))
		default:
			w.WriteHeader(http.StatusForbidden)
		}
	})

	for range 2 {
		img, err := c.Image(context.Background(), c.cfg.BaseURL+"/users/x/medium_a.jpg")
		if err != nil {
			t.Fatal(err)
		}
		if img.ContentType != "image/jpeg" || string(img.Data) != "jpeg-bytes" {
			t.Errorf("image = %+v", img)
		}
	}
	if tokenRequests != 1 {
		t.Errorf("token requested %d times, want 1", tokenRequests)
	}

	if _, err := c.Image(context.Background(), c.cfg.BaseURL+"/users/x/missing.jpg"); !errors.Is(err, avatar.ErrNotFound) {
		t.Errorf("missing image err = %v, want ErrNotFound", err)
	}
}

func TestRetriesOn429(t *testing.T) {
	var calls int
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		r.ParseForm()
		if r.PostForm.Get("code") != "abc" {
			t.Errorf("attempt %d: body not resent, code = %q", calls, r.PostForm.Get("code"))
		}
		if calls == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"access_token":"a"}`))
	})

	token, err := c.ExchangeCode(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || token.AccessToken != "a" {
		t.Errorf("calls = %d, token = %+v", calls, token)
	}
}

func TestHourlyBudgetBlocks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id":1,"login":"x"}`))
	}))
	t.Cleanup(srv.Close)
	c := NewClient(Config{BaseURL: srv.URL, RatePerSecond: 100, RatePerHour: 2})

	ctx := context.Background()
	for i := range 2 {
		if _, err := c.Profile(ctx, "tok"); err != nil {
			t.Fatalf("request %d within budget: %v", i+1, err)
		}
	}
	// The third request would have to wait ~30 minutes for the bucket to
	// refill: it fails at once instead of blocking.
	start := time.Now()
	if _, err := c.Profile(ctx, "tok"); !errors.Is(err, upstream.ErrBusy) {
		t.Fatalf("third request err = %v, want ErrBusy", err)
	}
	if d := time.Since(start); d > 100*time.Millisecond {
		t.Errorf("busy request took %v, want an immediate failure", d)
	}
}

func TestShortWaitsAreAllowedAndRefusedOnesCostNothing(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"id":1,"login":"x"}`))
	}))
	t.Cleanup(srv.Close)
	ctx := context.Background()

	// 1 request per second, wait up to 2s: the second request waits ~1s.
	c := NewClient(Config{BaseURL: srv.URL, RatePerSecond: 1, RatePerHour: 100, MaxWait: 2 * time.Second})
	for i := range 2 {
		if _, err := c.Profile(ctx, "tok"); err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
	}

	// Wait up to 50ms: the second request is refused without using the budget.
	c = NewClient(Config{BaseURL: srv.URL, RatePerSecond: 1, RatePerHour: 100, MaxWait: 50 * time.Millisecond})
	if _, err := c.Profile(ctx, "tok"); err != nil {
		t.Fatal(err)
	}
	before := c.perHour.Tokens()
	if _, err := c.Profile(ctx, "tok"); !errors.Is(err, upstream.ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
	if after := c.perHour.Tokens(); after < before-0.01 {
		t.Errorf("hourly tokens %.2f -> %.2f: a refused request used the budget", before, after)
	}
	if calls != 3 {
		t.Errorf("42 calls = %d, want 3", calls)
	}
}

func TestLongRetryAfterIsBusy(t *testing.T) {
	var calls int
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	start := time.Now()
	if _, err := c.Profile(context.Background(), "tok"); !errors.Is(err, upstream.ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
	if d := time.Since(start); d > time.Second || calls != 1 {
		t.Errorf("took %v with %d calls, want one call and no sleep", d, calls)
	}
}

func TestPersistent429IsBusy(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	c.cfg.RatePerSecond = 100
	if _, err := c.Profile(context.Background(), "tok"); !errors.Is(err, upstream.ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
}
