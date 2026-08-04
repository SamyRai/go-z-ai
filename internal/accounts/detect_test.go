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

// regionTransport returns different responses based on the request's monitor
// host, so a test can simulate a key that is valid on only one gateway.
type regionTransport struct {
	// respond maps a request URL host to (status, body). A host absent from
	// the map gets a transport error (so the probe falls through to the next
	// region rather than silently succeeding).
	respond map[string]stubTransport
}

func (r regionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if s, ok := r.respond[req.URL.Host]; ok {
		return s.RoundTrip(req)
	}
	return nil, io.ErrUnexpectedEOF
}

// A key issued on the China gateway (open.bigmodel.cn) returns a coding-plan
// success ONLY there; the global host rejects it. Before the both-regions fix,
// ProbeType probed only the global host, so this key was misclassified as
// pay_as_you_go (unconfirmed), routing every subsequent call to the wrong
// endpoint and disabling quota/usage monitoring.
func TestProbeTypeChinaCodingPlanConfirmed(t *testing.T) {
	codingPlanOK := stubTransport{
		status: http.StatusOK,
		body:   `{"success":true,"data":{"level":"pro","limits":[]}}`,
	}
	globalReject := stubTransport{
		status: http.StatusForbidden,
		body:   `{"error":{"code":"1002","message":"invalid key on global host"}}`,
	}
	// ProbeType builds real clients (one per region); wire the transport
	// through both by patching the default... instead, exercise probeType
	// directly against two clients sharing one region-aware transport.
	tr := regionTransport{respond: map[string]stubTransport{
		"api.z.ai":         globalReject, // global host rejects the China key
		"open.bigmodel.cn": codingPlanOK, // China host confirms coding_plan
	}}
	globalC := clientWith(t, tr)
	chinaC := clientWithRegion(t, tr, client.RegionChina)

	// Global alone: pay_as_you_go (the old buggy behavior).
	if at, ok := probeType(context.Background(), globalC); ok || at != client.AccountTypePayAsYouGo {
		t.Fatalf("global probe should fall through for a China key; got %q/confirmed=%v", at, ok)
	}
	// China: coding_plan confirmed.
	if at, ok := probeType(context.Background(), chinaC); !ok || at != client.AccountTypeCodingPlan {
		t.Fatalf("china probe should confirm coding_plan; got %q/confirmed=%v", at, ok)
	}
}

// clientWithRegion is clientWith but with an explicit Region (so the client's
// monitor calls land on the China host for the region-aware test transport).
func clientWithRegion(t *testing.T, tr http.RoundTripper, region client.Region) *client.Client {
	t.Helper()
	c, err := client.NewClient(client.Config{
		APIKey:     "test-key",
		Region:     region,
		HTTPClient: &http.Client{Transport: tr},
		MaxRetries: -1,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}
