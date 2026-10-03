package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// hostRoutes answers each request by its host (api.z.ai / open.bigmodel.cn),
// recording the hosts it saw in order, so detection can be exercised against
// both gateways without a network.
type hostRoutes struct {
	mu     sync.Mutex
	seen   []string
	routes map[string]stubReply
}

type stubReply struct {
	status int
	body   string
	err    error
}

func (h *hostRoutes) RoundTrip(r *http.Request) (*http.Response, error) {
	h.mu.Lock()
	h.seen = append(h.seen, r.URL.Host)
	h.mu.Unlock()
	reply, ok := h.routes[r.URL.Host]
	if !ok {
		reply = stubReply{status: http.StatusNotFound, body: `{"error":{"code":"1222","message":"no route"}}`}
	}
	if reply.err != nil {
		return nil, reply.err
	}
	return &http.Response{
		StatusCode: reply.status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(reply.body)),
		Request:    r,
	}, nil
}

func detectWith(t *testing.T, region Region, routes map[string]stubReply) (*DetectedAccount, []string, error) {
	t.Helper()
	tr := &hostRoutes{routes: routes}
	c, err := NewClient(Config{APIKey: "k", Region: region, MaxRetries: -1, HTTPClient: &http.Client{Transport: tr}})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	det, err := c.Detection().DetectAccountType(context.Background())
	return det, tr.seen, err
}

const (
	quotaOK     = `{"success":true,"data":{"level":"pro","limits":[]}}`
	authFailure = `{"error":{"code":"1000","message":"auth failed"}}`
)

// A clean quota response on the home gateway confirms a coding plan there,
// without ever touching the other gateway.
func TestDetectCodingPlanOnHomeRegion(t *testing.T) {
	det, seen, err := detectWith(t, RegionGlobal, map[string]stubReply{
		"api.z.ai": {status: http.StatusOK, body: quotaOK},
	})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if det.Type != AccountTypeCodingPlan || !det.Confirmed || det.Level != "pro" {
		t.Fatalf("got %+v, want confirmed pro coding plan", det)
	}
	if det.Region != RegionGlobal || det.BaseURL != CodingBaseURL {
		t.Errorf("got region %q base %q, want global coding base", det.Region, det.BaseURL)
	}
	if len(seen) != 1 || seen[0] != "api.z.ai" {
		t.Errorf("probed %v, want only api.z.ai", seen)
	}
}

// A China-issued coding-plan key fails on the global gateway and is found on
// open.bigmodel.cn — the parity fix for China keys being misclassified.
func TestDetectCodingPlanFallsThroughToChina(t *testing.T) {
	det, seen, err := detectWith(t, "", map[string]stubReply{
		"api.z.ai":         {status: http.StatusUnauthorized, body: authFailure},
		"open.bigmodel.cn": {status: http.StatusOK, body: quotaOK},
	})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if det.Region != RegionChina || det.BaseURL != ChinaCodingBaseURL || !det.Confirmed {
		t.Fatalf("got %+v, want confirmed China coding plan", det)
	}
	if strings.Join(seen, ",") != "api.z.ai,open.bigmodel.cn" {
		t.Errorf("probe order %v, want global then china", seen)
	}
}

// A RegionChina client probes its own gateway first.
func TestDetectProbesConfiguredRegionFirst(t *testing.T) {
	_, seen, err := detectWith(t, RegionChina, map[string]stubReply{
		"open.bigmodel.cn": {status: http.StatusOK, body: quotaOK},
	})
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(seen) == 0 || seen[0] != "open.bigmodel.cn" {
		t.Errorf("probe order %v, want open.bigmodel.cn first", seen)
	}
}

// When both gateways answer without identifying a subscription, the key is
// pay-as-you-go — unconfirmed, since nothing positively identifies one. In
// particular an out-of-balance answer (1113) is NOT a coding plan: the old
// billed-probe detector misclassified exactly that case.
func TestDetectPayAsYouGoByElimination(t *testing.T) {
	cases := map[string]stubReply{
		"auth error":       {status: http.StatusUnauthorized, body: authFailure},
		"no balance":       {status: http.StatusTooManyRequests, body: `{"error":{"code":"1113","message":"insufficient balance"}}`},
		"success=false":    {status: http.StatusOK, body: `{"success":false,"data":{}}`},
		"empty level":      {status: http.StatusOK, body: `{"success":true,"data":{"level":"","limits":[]}}`},
		"undecodable body": {status: http.StatusOK, body: `not json`},
	}
	for name, reply := range cases {
		t.Run(name, func(t *testing.T) {
			det, _, err := detectWith(t, RegionGlobal, map[string]stubReply{
				"api.z.ai": reply, "open.bigmodel.cn": reply,
			})
			if err != nil {
				t.Fatalf("detect: %v", err)
			}
			if det.Type != AccountTypePayAsYouGo || det.Confirmed {
				t.Errorf("got %+v, want unconfirmed pay_as_you_go", det)
			}
			if det.BaseURL != DefaultBaseURL {
				t.Errorf("base %q, want %q", det.BaseURL, DefaultBaseURL)
			}
		})
	}
}

// If neither gateway can be reached, detection reports the failure instead
// of guessing pay-as-you-go.
func TestDetectUnreachableIsAnError(t *testing.T) {
	down := stubReply{err: io.ErrUnexpectedEOF}
	if _, _, err := detectWith(t, RegionGlobal, map[string]stubReply{"api.z.ai": down, "open.bigmodel.cn": down}); err == nil {
		t.Fatal("expected an error when no gateway is reachable")
	}
}

// The result is cached per client: a second call makes no requests.
func TestDetectCaches(t *testing.T) {
	tr := &hostRoutes{routes: map[string]stubReply{"api.z.ai": {status: http.StatusOK, body: quotaOK}}}
	c, err := NewClient(Config{APIKey: "k", MaxRetries: -1, HTTPClient: &http.Client{Transport: tr}})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	for range 2 {
		if _, err := c.Detection().DetectAccountType(context.Background()); err != nil {
			t.Fatalf("detect: %v", err)
		}
	}
	if len(tr.seen) != 1 {
		t.Errorf("made %d requests, want 1 (cached)", len(tr.seen))
	}
}
