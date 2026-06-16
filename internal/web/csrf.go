package web

import (
	"net/http"
	"net/url"
	"strings"
)

// isSameOriginRequest reports whether r is safe to treat as a same-origin,
// non-cross-site state change. It is a lightweight CSRF guard for the
// cookie-bearing POST endpoints (/setup, /login, /logout): SameSite=Lax does not
// protect /setup (there is no pre-existing session cookie on the first run), so a
// header-based same-origin check is layered in front of every state change.
//
// The predicate, in order:
//   - Sec-Fetch-Site (Fetch Metadata, sent by every modern browser): allow only
//     "same-origin" and "none" (a user-initiated navigation); reject "cross-site"
//     and "same-site".
//   - else Origin: allow only when its host:port equals the request Host.
//   - else (neither header, e.g. curl or another non-browser client): allow,
//     because there is no browser-driven CSRF vector to defend against.
func isSameOriginRequest(r *http.Request) bool {
	if site := strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")); site != "" {
		return site == "same-origin" || site == "none"
	}
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" {
			return false
		}
		return u.Host == r.Host
	}
	return true
}

// enforceSameOrigin rejects a cross-site state-changing request with 403 and
// reports whether the caller may proceed. It is the single entry point wired at
// the top of the state-changing POST handlers.
func enforceSameOrigin(w http.ResponseWriter, r *http.Request) bool {
	if isSameOriginRequest(r) {
		return true
	}
	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	return false
}
