package ft

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tmazitov/42-ad-bachelors/backend/internal/application/auth"
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

	if _, err := c.RefreshToken(context.Background(), "old"); !errors.Is(err, auth.ErrInvalidGrant) {
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
		w.Write([]byte(`{"id":7,"login":"jdoe","displayname":"John","image":{"link":"http://img","versions":{"medium":null}}}`))
	})

	p, err := c.Profile(context.Background(), "tok")
	if err != nil {
		t.Fatal(err)
	}
	if p != (auth.Profile{ID: 7, Login: "jdoe", DisplayName: "John", ImageURL: "http://img"}) {
		t.Errorf("profile = %+v", p)
	}
}
