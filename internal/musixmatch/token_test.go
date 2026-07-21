package musixmatch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMintToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("app_id"); got != desktopAppID {
			t.Errorf("app_id = %q; want %q", got, desktopAppID)
		}
		_, _ = w.Write([]byte(`{"message":{"header":{"status_code":200},"body":{"user_token":"tok-abcdefghijklmnopqrstuvwxyz"}}}`))
	}))
	defer srv.Close()

	m := NewTokenMinter(srv.Client())
	m.baseURL = srv.URL

	tok, err := m.Mint(context.Background())
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if tok != "tok-abcdefghijklmnopqrstuvwxyz" {
		t.Errorf("token = %q; want the minted value", tok)
	}
}

// TestMintToken_RateLimited pins the constraint #554 measured: three rapid mints
// from one egress all return 401. That must surface as a distinct sentinel so the
// caller can back off rather than spin, which would lock itself out entirely.
func TestMintToken_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"message":{"header":{"status_code":401}}}`))
	}))
	defer srv.Close()

	m := NewTokenMinter(srv.Client())
	m.baseURL = srv.URL

	_, err := m.Mint(context.Background())
	if !errors.Is(err, ErrTokenMintRefused) {
		t.Fatalf("err = %v; want ErrTokenMintRefused", err)
	}
}

// TestMintToken_EmptyTokenIsAnError guards against persisting a useless value:
// a 200 with no user_token must fail rather than store an empty secret.
func TestMintToken_EmptyTokenIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"message":{"header":{"status_code":200},"body":{"user_token":""}}}`))
	}))
	defer srv.Close()

	m := NewTokenMinter(srv.Client())
	m.baseURL = srv.URL

	if _, err := m.Mint(context.Background()); err == nil {
		t.Fatal("Mint: got nil error for an empty user_token; want an error")
	}
}

func TestMintToken_MalformedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	m := NewTokenMinter(srv.Client())
	m.baseURL = srv.URL

	if _, err := m.Mint(context.Background()); err == nil {
		t.Fatal("Mint: got nil error for a malformed body; want an error")
	}
}

func TestMintToken_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	m := NewTokenMinter(srv.Client())
	m.baseURL = srv.URL

	if _, err := m.Mint(context.Background()); err == nil {
		t.Fatal("Mint: got nil error for HTTP 500; want an error")
	}
}
