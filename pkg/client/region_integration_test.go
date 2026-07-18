package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// stubBody is an httptest handler that emits a minimal valid body for each
// service's decoder. The actual host-recording happens in rewrapTransport, not
// here, so this handler is host-agnostic.
func stubBody(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.Contains(r.URL.Path, "quota") || strings.Contains(r.URL.Path, "usage"):
		fmt.Fprint(w, `{"code":0,"msg":"ok","data":{}}`)
	case strings.Contains(r.URL.Path, "account"):
		fmt.Fprint(w, `{"code":0,"msg":"ok","data":{}}`)
	case strings.Contains(r.URL.Path, "agents"):
		fmt.Fprint(w, `{"id":"x","agent_id":"a","status":"SUCCESS"}`)
	default:
		fmt.Fprint(w, `{}`)
	}
}

// A RegionChina-configured client must route Quota/Account/Agents calls to the
// open.bigmodel.cn host, not api.z.ai. This is the load-bearing parity test —
// it pins the wiring changed in Stream 1. rewrapTransport redirects every
// request to the test server (so no real network call happens) while recording
// the originally-targeted Host, so the test can assert which regional gateway
// the client's service selected.
func TestRegionChinaRoutesToBigModelHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(stubBody))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{Region: RegionChina})

	t.Run("quota", func(t *testing.T) {
		var host string
		c.httpClient.Transport = &rewrapTransport{base: srv.URL, seen: &host}
		_, err := c.Quota().GetQuotaLimit(context.Background())
		if err != nil {
			t.Fatalf("GetQuotaLimit: %v", err)
		}
		assertChinaHost(t, host)
	})

	t.Run("model-usage", func(t *testing.T) {
		var host string
		c.httpClient.Transport = &rewrapTransport{base: srv.URL, seen: &host}
		now := time.Now()
		_, err := c.Quota().GetModelUsage(context.Background(), now.Add(-24*time.Hour), now)
		if err != nil {
			t.Fatalf("GetModelUsage: %v", err)
		}
		assertChinaHost(t, host)
	})

	t.Run("tool-usage", func(t *testing.T) {
		var host string
		c.httpClient.Transport = &rewrapTransport{base: srv.URL, seen: &host}
		now := time.Now()
		_, err := c.Quota().GetToolUsage(context.Background(), now.Add(-24*time.Hour), now)
		if err != nil {
			t.Fatalf("GetToolUsage: %v", err)
		}
		assertChinaHost(t, host)
	})

	t.Run("account-info", func(t *testing.T) {
		var host string
		c.httpClient.Transport = &rewrapTransport{base: srv.URL, seen: &host}
		_, err := c.Account().GetAccountInfo(context.Background())
		if err != nil {
			t.Fatalf("GetAccountInfo: %v", err)
		}
		assertChinaHost(t, host)
	})

	t.Run("agents-invoke", func(t *testing.T) {
		var host string
		c.httpClient.Transport = &rewrapTransport{base: srv.URL, seen: &host}
		_, _ = c.Agents().Invoke(context.Background(), AgentInvokeRequest{
			AgentID:  "general_translation",
			Messages: []AgentMessage{NewAgentTextMessage("user", "hi")},
		})
		// Even on a business-level error, the transport recorded the host.
		assertChinaHost(t, host)
	})
}

// A RegionGlobal-configured (or unset) client must keep routing to api.z.ai —
// the historical behavior is unchanged for existing callers.
func TestRegionGlobalRoutesToAPIZAI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(stubBody))
	defer srv.Close()

	for _, region := range []Region{RegionGlobal, ""} {
		t.Run(string(region), func(t *testing.T) {
			c := newTestClient(t, srv.URL, Config{Region: region})
			var host string
			c.httpClient.Transport = &rewrapTransport{base: srv.URL, seen: &host}

			_, err := c.Quota().GetQuotaLimit(context.Background())
			if err != nil {
				t.Fatalf("GetQuotaLimit: %v", err)
			}
			if !strings.Contains(host, "api.z.ai") {
				t.Errorf("expected api.z.ai host for region %q, got %q", region, host)
			}
		})
	}
}

func assertChinaHost(t *testing.T, host string) {
	t.Helper()
	if !strings.Contains(host, "open.bigmodel.cn") {
		t.Errorf("expected open.bigmodel.cn host, got %q", host)
	}
}

// rewrapTransport redirects every request to a test server (so no real network
// call is made) while preserving the originally-targeted Host in *seen. This
// lets a test assert which regional gateway the client's service selected.
type rewrapTransport struct {
	base string
	seen *string
}

func (t *rewrapTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Record the original host before rewriting, off the un-cloned request.
	if t.seen != nil {
		*t.seen = req.URL.Hostname()
	}
	// Clone before mutating: a RoundTripper must not modify the caller's
	// request (the stdlib Transport reads req.URL.Host after RoundTrip for
	// connection-keying/retries and would race a direct mutation). This
	// matches the roundTripFunc pattern in embeddings_test.go.
	req = req.Clone(req.Context())
	testURL, err := url.Parse(t.base)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = testURL.Scheme
	req.URL.Host = testURL.Host
	req.Host = testURL.Host
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		// Close the body on error to avoid leaking the connection/goroutine —
		// RoundTrip may return (resp, err) with a non-nil body on rare paths.
		if resp != nil {
			resp.Body.Close()
		}
		return nil, err
	}
	return resp, nil
}
