package accounts

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// stubTransport returns a fixed response (or error) for every request,
// regardless of URL — the only way to intercept GetQuotaLimit, which targets a
// hardcoded monitor base URL that Config.BaseURL can't redirect.
type stubTransport struct {
	status int
	body   string
	err    error
}

func (s stubTransport) RoundTrip(*http.Request) (*http.Response, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &http.Response{
		StatusCode: s.status,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
	}, nil
}

func clientWith(t *testing.T, tr http.RoundTripper) *client.Client {
	t.Helper()
	c, err := client.NewClient(client.Config{
		APIKey:     "test-key",
		HTTPClient: &http.Client{Transport: tr},
		MaxRetries: -1, // don't retry/backoff a stubbed error
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// A well-formed successful quota response (the coding-plan-only monitor
// endpoint answering) classifies the key as coding_plan, confirmed.
func TestProbeTypeCodingPlanConfirmed(t *testing.T) {
	c := clientWith(t, stubTransport{
		status: http.StatusOK,
		body:   `{"success":true,"data":{"level":"pro","limits":[]}}`,
	})
	at, confirmed := probeType(context.Background(), c)
	if at != client.AccountTypeCodingPlan || !confirmed {
		t.Errorf("expected coding_plan/confirmed, got %q/%v", at, confirmed)
	}
}

// Anything that isn't a clean success — non-200, unsuccessful body, or a
// transport error — falls back to pay_as_you_go, unconfirmed (inference by
// elimination).
func TestProbeTypeFallsBackToPayAsYouGo(t *testing.T) {
	cases := map[string]stubTransport{
		"non-200":          {status: http.StatusForbidden, body: `{"error":{"code":"1002","message":"nope"}}`},
		"success=false":    {status: http.StatusOK, body: `{"success":false,"data":{}}`},
		"empty level":      {status: http.StatusOK, body: `{"success":true,"data":{"level":"","limits":[]}}`},
		"transport error":  {err: io.ErrUnexpectedEOF},
		"undecodable body": {status: http.StatusOK, body: `not json`},
	}
	for name, tr := range cases {
		t.Run(name, func(t *testing.T) {
			c := clientWith(t, tr)
			at, confirmed := probeType(context.Background(), c)
			if at != client.AccountTypePayAsYouGo || confirmed {
				t.Errorf("expected pay_as_you_go/unconfirmed, got %q/%v", at, confirmed)
			}
		})
	}
}
