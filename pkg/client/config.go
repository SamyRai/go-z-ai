package client

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

// Configuration defaults.
const (
	DefaultTimeout    = 30 * time.Second       // connect + TLS + response-header wait
	DefaultMaxRetries = 3                      // retries on transient (429/5xx/network) failures
	DefaultRetryDelay = 200 * time.Millisecond // base exponential-backoff delay
)

// Config holds the client configuration.
type Config struct {
	APIKey string
	// BaseURL is the OpenAI-compatible API root used by chat and most other
	// services. Empty means Region.PaaSBaseURL() (DefaultBaseURL for the
	// global region). Coding Plan keys use Region.CodingBaseURL().
	BaseURL    string
	HTTPClient *http.Client
	// Timeout bounds connection setup (dial + TLS handshake) and the wait
	// for response headers — not the time spent reading the response body.
	// It is deliberately NOT the whole-request http.Client.Timeout, which
	// would cut off a long-running stream (SSE bodies can stay open for
	// minutes) partway through a generation. Defaults to DefaultTimeout.
	Timeout time.Duration
	// MaxRetries is the number of retry attempts on transient failures
	// (429, 5xx, network errors). The zero value means "unset" and resolves
	// to DefaultMaxRetries; set -1 to disable retries entirely.
	MaxRetries int
	// RetryDelay is the base delay for exponential backoff. Defaults to
	// DefaultRetryDelay.
	RetryDelay time.Duration
	// ChinaAPIKey authenticates against BigModelBaseURL (open.bigmodel.cn),
	// used by EmbeddingsService and ModerationsService. Falls back to
	// APIKey when empty — live-verified 2026-07-11 that a z.ai key
	// authenticates identically on both platforms, so the fallback is the
	// common case. Set this only for a distinct bigmodel.cn-only credential.
	ChinaAPIKey string
	// DisableToolSchemaCompat turns off automatic normalization of tool
	// (function) parameter schemas before they are sent. By default the
	// client rewrites the JSON Schema constructs GLM's parser rejects with
	// HTTP 500 — `anyOf`, `oneOf`, `allOf`, and `$ref`/`$defs` — into the flat
	// subset it accepts (see SanitizeToolSchemas).
	DisableToolSchemaCompat bool
	// Region selects the regional gateway (api.z.ai vs open.bigmodel.cn). It
	// picks the host for the region-scoped services — monitor (quota/usage),
	// biz (account), agents, Anthropic-compatible Messages, and account-type
	// detection — and the default BaseURL when BaseURL is empty. An explicit
	// BaseURL always wins. Defaults to RegionGlobal. See region.go.
	Region Region
	// MonitorTimezone overrides the timezone assumed for the monitor (quota/
	// usage) API, which exchanges zoneless time strings the server interprets
	// as its own wall-clock (MonitorServerTZ, UTC+8, by default). Set this
	// only if a live capture shows a different server zone for your account.
	// The CLI mirrors it as ZAI_MONITOR_TIMEZONE.
	MonitorTimezone *time.Location
	// UserAgent overrides the default User-Agent header ("go-z-ai/<version>")
	// sent on every request. The default identifies go-z-ai to Z.AI's API —
	// important under the coding endpoint's usage policy, which treats
	// unidentified clients as policy violations (see docs/en/coding-tools.md).
	UserAgent string
	// Hooks attaches observability hooks (tracing, metrics, logging) that
	// fire on every request/response/error/stream-chunk. Empty by default —
	// the no-hook path is zero-cost. See Hook and pkg/observe.
	Hooks []Hook
}

// withDefaults returns a copy of c with every unset field resolved.
func (c Config) withDefaults() Config {
	if c.Region == "" {
		c.Region = RegionGlobal
	}
	if c.BaseURL == "" {
		c.BaseURL = c.Region.PaaSBaseURL()
	}
	if c.Timeout == 0 {
		c.Timeout = DefaultTimeout
	}
	if c.HTTPClient == nil {
		c.HTTPClient = newHTTPClient(c.Timeout)
	}
	switch {
	case c.MaxRetries == 0:
		c.MaxRetries = DefaultMaxRetries
	case c.MaxRetries < 0:
		c.MaxRetries = 0
	}
	if c.RetryDelay <= 0 {
		c.RetryDelay = DefaultRetryDelay
	}
	if c.MonitorTimezone == nil {
		c.MonitorTimezone = MonitorServerTZ
	}
	if c.UserAgent == "" {
		c.UserAgent = userAgent
	}
	return c
}

func (c Config) validate() error {
	if c.APIKey == "" {
		return errors.New("API key is required")
	}
	return nil
}

// newHTTPClient builds the default transport. There is no http.Client.Timeout:
// that field bounds the entire request including the body read, which would
// kill a streaming SSE read after timeout even while tokens are still
// arriving. Bounding dial/TLS/response-header wait instead protects against a
// hung or unreachable server without capping a live stream.
func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   timeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   timeout,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: timeout,
		},
		CheckRedirect: stripAuthOnCrossHostRedirect,
	}
}

// stripAuthOnCrossHostRedirect drops the Authorization header on any redirect
// to a different host:port, so a compromised or misconfigured upstream (or a
// transparent proxy) can't capture the bearer token by answering with a 3xx
// to a host it controls. net/http's default only strips it on a scheme or
// domain change, not on a port change.
func stripAuthOnCrossHostRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	if req.URL.Host != via[0].URL.Host {
		req.Header.Del("Authorization")
	}
	return nil
}

// NewClientFromEnv creates a client from environment variables: ZAI_API_KEY
// (required), ZAI_API_BASE_URL, ZAI_REGION, ZAI_CHINA_API_KEY, and
// ZAI_MONITOR_TIMEZONE — the same variables the CLI honors.
func NewClientFromEnv() (*Client, error) {
	apiKey := os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("ZAI_API_KEY environment variable not set")
	}
	monitorTZ, err := ParseTimezone(os.Getenv("ZAI_MONITOR_TIMEZONE"))
	if err != nil {
		return nil, fmt.Errorf("ZAI_MONITOR_TIMEZONE: %w", err)
	}
	return NewClient(Config{
		APIKey:          apiKey,
		BaseURL:         os.Getenv("ZAI_API_BASE_URL"),
		ChinaAPIKey:     os.Getenv("ZAI_CHINA_API_KEY"),
		Region:          ParseRegion(os.Getenv("ZAI_REGION")),
		MonitorTimezone: monitorTZ,
	})
}
