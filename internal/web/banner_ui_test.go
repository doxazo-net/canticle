package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sydlexius/canticle/internal/config"
)

// spicetifyFAQURL was once linked from the banner as a way to obtain a Musixmatch
// token by hand (#385). It is now asserted ABSENT: serve mode provisions a token
// automatically and retries on a later start, so the banner's actionable remedies
// are Settings and the tokenless provider. Walking an operator through extracting
// a credential out of another application's traffic is not guidance canticle
// ships, and this constant is retained purely so the assertion below keeps the
// link from being reintroduced.
const spicetifyFAQURL = "https://spicetify.app/docs/faq#sometimes-popup-lyrics-andor-lyrics-plus-seem-to-not-work"

// renderShell mounts a (public, no-auth) UI and fetches the Reports workspace
// shell page, which renders through the shared Layout where the banner lives.
// /reports needs no data source on the no-key path (it shows the placeholder),
// so it exercises the shell without wiring a database.
func renderShell(t *testing.T, inactive bool) string {
	t.Helper()
	mux := http.NewServeMux()
	NewUI(config.Config{}, "v-test", WithMusixmatchInactive(inactive)).Register(mux)
	req := httptest.NewRequest(http.MethodGet, "/reports", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /reports status = %d; want 200", rec.Code)
	}
	return rec.Body.String()
}

func TestMusixmatchInactiveBannerRendersWhenInactive(t *testing.T) {
	body := renderShell(t, true)
	if !strings.Contains(body, "mx-banner") {
		t.Fatal("banner element (.mx-banner) missing when musixmatchInactive=true")
	}
	if !strings.Contains(body, `href="/settings"`) {
		t.Fatal("banner must link to /settings to add the token")
	}
	// The banner must NOT teach the operator to obtain a token by hand. This is the
	// inverse of the original #385 assertion and exists to stop the link coming back.
	if strings.Contains(body, spicetifyFAQURL) {
		t.Fatalf("banner must not link to a manual token-extraction guide (%s)", spicetifyFAQURL)
	}
	if !strings.Contains(body, "Musixmatch") {
		t.Fatal("banner copy must mention Musixmatch")
	}
	// The banner must offer the tokenless alternative (PetitLyrics), not imply all
	// lyric fetching is off (#385 follow-up: a tokenless provider may be covering).
	if !strings.Contains(body, "PetitLyrics") {
		t.Fatal("banner copy must mention the tokenless PetitLyrics alternative")
	}
}

func TestMusixmatchInactiveBannerAbsentWhenActive(t *testing.T) {
	body := renderShell(t, false)
	if strings.Contains(body, "mx-banner") {
		t.Fatal("banner element (.mx-banner) must not render when musixmatchInactive=false")
	}
	if strings.Contains(body, spicetifyFAQURL) {
		t.Fatal("Spicetify FAQ link must not render when musixmatchInactive=false")
	}
}
