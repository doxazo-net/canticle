package musixmatch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sydlexius/canticle/internal/models"
)

// renewSignalBody is the body-level hint=renew response: HTTP 200 at the
// transport layer, with the renewal flag inside message.header.
const renewSignalBody = `{"message":{"header":{"status_code":401,"hint":"renew"}}}`

// missBody is a well-formed macro response reporting no matching track. The
// retry assertions only need the second response to be CONSUMED and classified,
// so a benign miss is a cleaner fixture than a synthetic lyrics payload.
const missBody = `{"message":{"header":{"status_code":200},"body":{"macro_calls":{` +
	`"matcher.track.get":{"message":{"header":{"status_code":404}}}` +
	`}}}}`

func respond(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

type stubRenewer struct {
	token string
	err   error
	calls int
}

func (s *stubRenewer) Renew(context.Context) (string, error) {
	s.calls++
	if s.err != nil {
		return "", s.err
	}
	return s.token, nil
}

// clientWithResponses returns a client whose transport replays the given bodies
// in order, and records the usertoken sent on each request.
func clientWithResponses(t *testing.T, bodies ...string) (*Client, *[]string) {
	t.Helper()
	var sentTokens []string
	i := 0
	c := NewClient("original-token")
	c.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		sentTokens = append(sentTokens, req.URL.Query().Get("usertoken"))
		body := bodies[len(bodies)-1]
		if i < len(bodies) {
			body = bodies[i]
		}
		i++
		return respond(http.StatusOK, body), nil
	})}
	return c, &sentTokens
}

// TestFindLyrics_RenewsOnHintRenew is AC3: the explicit signal mints a
// replacement, installs it, and retries once with the NEW token.
func TestFindLyrics_RenewsOnHintRenew(t *testing.T) {
	c, sent := clientWithResponses(t, renewSignalBody, missBody)
	r := &stubRenewer{token: "renewed-token"}
	c.SetTokenRenewer(r)

	// The retry's outcome is a benign miss; what matters is that the SECOND
	// response was consumed, proving the retry actually happened, and that it
	// carried the renewed token.
	_, err := c.FindLyrics(context.Background(), models.Track{ArtistName: "A", TrackName: "T"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v; want ErrNotFound from the retried request", err)
	}
	if r.calls != 1 {
		t.Errorf("renewer called %d times; want 1", r.calls)
	}
	if len(*sent) != 2 {
		t.Fatalf("requests = %d; want 2 (original + one retry)", len(*sent))
	}
	if (*sent)[0] != "original-token" {
		t.Errorf("first request token = %q; want original-token", (*sent)[0])
	}
	if (*sent)[1] != "renewed-token" {
		t.Errorf("retry token = %q; want renewed-token", (*sent)[1])
	}
}

// TestFindLyrics_BareUnauthorizedDoesNotRenew is the load-bearing safety
// property. Observed behavior attributes bare 401s to throttling, and the mint
// endpoint is itself rate limited, so renewing on them would burn mints during
// exactly the window when they are most likely to be refused.
func TestFindLyrics_BareUnauthorizedDoesNotRenew(t *testing.T) {
	var sentTokens []string
	c := NewClient("original-token")
	c.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		sentTokens = append(sentTokens, req.URL.Query().Get("usertoken"))
		return respond(http.StatusUnauthorized, `{}`), nil
	})}
	r := &stubRenewer{token: "renewed-token"}
	c.SetTokenRenewer(r)

	_, err := c.FindLyrics(context.Background(), models.Track{ArtistName: "A", TrackName: "T"})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v; want ErrUnauthorized", err)
	}
	if r.calls != 0 {
		t.Errorf("renewer called %d times; want 0 on a bare 401", r.calls)
	}
	if len(sentTokens) != 1 {
		t.Errorf("requests = %d; want 1 (no retry)", len(sentTokens))
	}
}

// TestFindLyrics_RenewalFailureReturnsOriginalError verifies a refused mint does
// not loop and surfaces the renewal error to the caller.
func TestFindLyrics_RenewalFailureReturnsOriginalError(t *testing.T) {
	c, sent := clientWithResponses(t, renewSignalBody)
	r := &stubRenewer{err: ErrTokenMintRefused}
	c.SetTokenRenewer(r)

	_, err := c.FindLyrics(context.Background(), models.Track{ArtistName: "A", TrackName: "T"})
	if !errors.Is(err, ErrTokenRenewalRequired) {
		t.Fatalf("err = %v; want ErrTokenRenewalRequired", err)
	}
	if r.calls != 1 {
		t.Errorf("renewer called %d times; want exactly 1 (no retry loop)", r.calls)
	}
	if len(*sent) != 1 {
		t.Errorf("requests = %d; want 1", len(*sent))
	}
}

// TestFindLyrics_RetriesAtMostOnce verifies a second renewal signal on the retry
// does not recurse: the mint endpoint is rate limited, so an unbounded
// renew-retry loop would lock the deployment out of it.
func TestFindLyrics_RetriesAtMostOnce(t *testing.T) {
	c, sent := clientWithResponses(t, renewSignalBody, renewSignalBody)
	r := &stubRenewer{token: "renewed-token"}
	c.SetTokenRenewer(r)

	_, err := c.FindLyrics(context.Background(), models.Track{ArtistName: "A", TrackName: "T"})
	if !errors.Is(err, ErrTokenRenewalRequired) {
		t.Fatalf("err = %v; want ErrTokenRenewalRequired", err)
	}
	if r.calls != 1 {
		t.Errorf("renewer called %d times; want exactly 1", r.calls)
	}
	if len(*sent) != 2 {
		t.Errorf("requests = %d; want exactly 2", len(*sent))
	}
}

// TestFindLyrics_NoRenewerLeavesBehaviorUnchanged covers every caller that
// supplies its own operator token: renewal stays off.
func TestFindLyrics_NoRenewerLeavesBehaviorUnchanged(t *testing.T) {
	c, sent := clientWithResponses(t, renewSignalBody)

	_, err := c.FindLyrics(context.Background(), models.Track{ArtistName: "A", TrackName: "T"})
	if !errors.Is(err, ErrTokenRenewalRequired) {
		t.Fatalf("err = %v; want ErrTokenRenewalRequired", err)
	}
	if len(*sent) != 1 {
		t.Errorf("requests = %d; want 1 (no retry without a renewer)", len(*sent))
	}
}
