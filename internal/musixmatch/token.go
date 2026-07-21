package musixmatch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/valyala/fastjson"
)

// tokenURL is the unauthenticated endpoint that issues a desktop-app user token.
// It is the same host the lyrics endpoint uses, on the token.get path.
const tokenURL = "https://apic-desktop.musixmatch.com/ws/1.1/token.get" //nolint:gosec // reason: G101 - this is the public endpoint URL, not a credential; the name matches the API path

// desktopAppID identifies the client to the API. It matches the app_id the
// lyrics request already sends, so a minted token is issued for the same client
// identity that will later use it.
const desktopAppID = "web-desktop-app-v1.0"

// maxTokenBodyBytes caps the token response read. The body is a single small
// JSON object; anything larger is a wrong endpoint or an error page.
const maxTokenBodyBytes = 64 << 10

// ErrTokenMintRefused reports that the token endpoint declined to issue a token,
// which in practice means rate limiting: #554 measured three rapid successive
// mints from one egress all returning 401.
//
// Callers MUST treat this as "back off and keep whatever token you already
// have", never as a reason to retry immediately. A restart loop that re-mints
// would lock itself out of the bootstrap endpoint entirely.
var ErrTokenMintRefused = errors.New("musixmatch: token mint refused (rate limited)")

// TokenMinter obtains a Musixmatch user token from the unauthenticated
// token.get endpoint, so canticle can bootstrap itself with no operator-supplied
// credential (#554).
//
// A minted token MUST be persisted by the caller. Minting on every start would
// trip the endpoint's rate limit and leave the deployment with no token at all.
type TokenMinter struct {
	httpClient *http.Client
	// baseURL is overridden in tests; production always uses tokenURL.
	baseURL string
}

// NewTokenMinter returns a minter using httpClient, or a client with a sane
// timeout when httpClient is nil.
func NewTokenMinter(httpClient *http.Client) *TokenMinter {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &TokenMinter{httpClient: httpClient, baseURL: tokenURL}
}

// Mint requests a fresh user token.
//
// It returns ErrTokenMintRefused when the endpoint answers with status_code 401
// (rate limited). Every other failure -- transport, malformed body, or a 200
// carrying no token -- returns a plain error, because persisting an empty or
// unparsable value would be worse than having no token at all.
func (m *TokenMinter) Mint(ctx context.Context) (string, error) {
	u, err := url.Parse(m.baseURL)
	if err != nil {
		return "", fmt.Errorf("musixmatch: parse token URL: %w", err)
	}
	q := u.Query()
	q.Set("app_id", desktopAppID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("musixmatch: build token request: %w", err)
	}
	resp, err := m.httpClient.Do(req) //nolint:gosec // reason: G704 - the URL is a package constant (tokenURL), overridden only by tests
	if err != nil {
		return "", fmt.Errorf("musixmatch: token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("musixmatch: token endpoint HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTokenBodyBytes))
	if err != nil {
		return "", fmt.Errorf("musixmatch: read token response: %w", err)
	}

	var p fastjson.Parser
	v, err := p.ParseBytes(body)
	if err != nil {
		return "", fmt.Errorf("musixmatch: parse token response: %w", err)
	}
	if code := v.GetInt("message", "header", "status_code"); code == http.StatusUnauthorized {
		return "", ErrTokenMintRefused
	} else if code != http.StatusOK {
		return "", fmt.Errorf("musixmatch: token endpoint status_code %d", code)
	}
	token := string(v.GetStringBytes("message", "body", "user_token"))
	if token == "" {
		return "", errors.New("musixmatch: token response carried no user_token")
	}
	return token, nil
}
