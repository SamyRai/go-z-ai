package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A RegionChina DetectionService must probe the open.bigmodel.cn coding/paas
// hosts, not api.z.ai — otherwise a China-issued key fails auth on the global
// host and gets mis-classified. This pins the parity fix.
func TestDetectionProbesRegionHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a 429-shaped body so testEndpoint treats the host as "working"
		// (the detection logic treats 429/1113/rate-limit as accessible).
		writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1302","message":"rate limit"}}`)
	}))
	defer srv.Close()

	for _, tc := range []struct {
		name     string
		region   Region
		wantHost string
	}{
		{"global probes api.z.ai", RegionGlobal, "api.z.ai"},
		{"empty probes api.z.ai", "", "api.z.ai"},
		{"china probes open.bigmodel.cn", RegionChina, "open.bigmodel.cn"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestClient(t, srv.URL, Config{Region: tc.region})
			var seenHost string
			c.httpClient.Transport = &rewrapTransport{base: srv.URL, seen: &seenHost}

			det, err := c.Detection().DetectAccountType(context.Background())
			if err != nil {
				t.Fatalf("DetectAccountType: %v", err)
			}
			if !strings.Contains(seenHost, tc.wantHost) {
				t.Errorf("expected probe host to contain %q, got %q", tc.wantHost, seenHost)
			}
			if det.Type != AccountTypeCodingPlan {
				t.Errorf("expected coding-plan classification on a 429, got %q", det.Type)
			}
			if !strings.Contains(det.BaseURL, tc.wantHost) {
				t.Errorf("detected BaseURL %q should be on %q", det.BaseURL, tc.wantHost)
			}
		})
	}
}

// probeURLs returns the documented coding/paas hosts per region.
func TestProbeURLsByRegion(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	for _, tc := range []struct {
		region           Region
		wantCoding, want string
	}{
		{RegionGlobal, CodingBaseURL, ProdBaseURL},
		{RegionChina, ChinaCodingBaseURL, ChinaProdBaseURL},
		{"", CodingBaseURL, ProdBaseURL},
	} {
		c.config.Region = tc.region
		gotCoding, gotPaas := c.Detection().probeURLs()
		if gotCoding != tc.wantCoding || gotPaas != tc.want {
			t.Errorf("region %q: got (%q,%q), want (%q,%q)", tc.region, gotCoding, gotPaas, tc.wantCoding, tc.want)
		}
	}
}
