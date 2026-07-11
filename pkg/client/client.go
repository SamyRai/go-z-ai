package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	DefaultBaseURL   = "https://api.z.ai/api/paas/v4"
	ProdBaseURL      = "https://api.z.ai/api/paas/v4"
	CodingBaseURL    = "https://api.z.ai/api/coding/paas/v4"
	AnthropicBaseURL = "https://api.z.ai/api/anthropic"
	MonitorBaseURL   = "https://api.z.ai/api/monitor"
	BizBaseURL       = "https://api.z.ai/api/biz"
	ToolsBaseURL     = "https://api.z.ai/api/tools"

	// Monitor usage endpoints
	QuotaLimitEndpoint = "/usage/quota/limit"
	ModelUsageEndpoint = "/usage/model-usage"
	ToolUsageEndpoint  = "/usage/tool-usage"

	// Retry defaults
	DefaultMaxRetries = 3                      // retries on transient (429/5xx/network) failures
	DefaultRetryDelay = 200 * time.Millisecond // base exponential-backoff delay
	maxRetryDelay     = 30 * time.Second       // cap on any single backoff
)

// Config holds the client configuration
type Config struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	// Timeout bounds connection setup (dial + TLS handshake) and the wait
	// for response headers — not the time spent reading the response body.
	// It is deliberately NOT the whole-request http.Client.Timeout, which
	// would cut off a long-running CreateStream read (SSE bodies can stay
	// open for minutes) partway through a generation. Defaults to 30s.
	Timeout time.Duration
	// MaxRetries is the number of retry attempts on transient failures
	// (429, 5xx, network errors). Defaults to DefaultMaxRetries (3).
	// Set to -1 to disable retries entirely.
	MaxRetries int
	// RetryDelay is the base delay for exponential backoff. Defaults to 200ms.
	RetryDelay time.Duration
}

// Client represents the Z.AI API client
type Client struct {
	config     Config
	httpClient *http.Client
	chat       *ChatService
	models     *ModelsService
	usage      *UsageService
	detection  *DetectionService
	quota      *QuotaService
	account    *AccountService
	tools      *ToolsService
	images     *ImagesService
	videos     *VideosService
	audio      *AudioService
	layout     *LayoutService
	files      *FilesService
	batch      *BatchService
	agents     *AgentsService
}

// NewClient creates a new Z.AI API client with the given configuration
func NewClient(config Config) (*Client, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults
	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.HTTPClient == nil {
		// No http.Client.Timeout here: that field bounds the entire
		// request/response cycle, including reading the body — which would
		// kill a streaming SSE read after config.Timeout even while tokens
		// are still arriving. Bounding dial/TLS/response-header wait
		// instead protects against a hung/unreachable server without
		// capping a live stream.
		config.HTTPClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   config.Timeout,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   config.Timeout,
				ExpectContinueTimeout: 1 * time.Second,
				ResponseHeaderTimeout: config.Timeout,
			},
		}
	}

	// Retry defaults: MaxRetries==0 means "unset" (use default); -1 disables.
	if config.MaxRetries == 0 {
		config.MaxRetries = DefaultMaxRetries
	}
	if config.MaxRetries < 0 {
		config.MaxRetries = 0
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = DefaultRetryDelay
	}

	client := &Client{
		config:     config,
		httpClient: config.HTTPClient,
	}

	// Initialize services
	client.chat = &ChatService{client: client}
	client.models = &ModelsService{client: client}
	client.usage = &UsageService{client: client}
	client.detection = &DetectionService{client: client}
	client.quota = &QuotaService{client: client}
	client.tools = &ToolsService{client: client}
	client.account = &AccountService{client: client}
	client.images = &ImagesService{client: client}
	client.videos = &VideosService{client: client}
	client.audio = &AudioService{client: client}
	client.layout = &LayoutService{client: client}
	client.files = &FilesService{client: client}
	client.batch = &BatchService{client: client}
	client.agents = &AgentsService{client: client}

	return client, nil
}

// NewClientFromEnv creates a new client from environment variables
func NewClientFromEnv() (*Client, error) {
	apiKey := os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ZAI_API_KEY environment variable not set")
	}

	baseURL := os.Getenv("ZAI_API_BASE_URL")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return NewClient(Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
	})
}

// Chat returns the chat service
func (c *Client) Chat() *ChatService {
	return c.chat
}

// Models returns the models service
func (c *Client) Models() *ModelsService {
	return c.models
}

// Usage returns the usage service
func (c *Client) Usage() *UsageService {
	return c.usage
}

// Detection returns the detection service
func (c *Client) Detection() *DetectionService {
	return c.detection
}

// Quota returns the quota service
func (c *Client) Quota() *QuotaService {
	return c.quota

}

// Account returns the account service
func (c *Client) Account() *AccountService {
	return c.account
}

func validateConfig(config Config) error {
	if config.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

// doRequest performs an HTTP request against the client's configured
// BaseURL, with authentication, structured error handling, and automatic
// retry on transient failures (429, 5xx, network errors) up to
// Config.MaxRetries, with exponential backoff and Retry-After.
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	return c.doRequestBase(ctx, c.config.BaseURL, method, endpoint, body, result)
}

// doRequestBase is doRequest against an explicit base URL, for the handful
// of endpoints that don't live under Config.BaseURL (monitor/biz/tools).
// Every service goes through this (or doRequest) — no service should build
// its own http.Client, which would bypass retry, timeout, and error parsing.
func (c *Client) doRequestBase(ctx context.Context, baseURL, method, endpoint string, body interface{}, result interface{}) error {
	maxRetries := c.config.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		resp, err := c.send(ctx, baseURL, method, endpoint, body)
		if err != nil {
			// Transport-level failure: no server response was produced, so the
			// request is safe to retry (the server never answered).
			lastErr = fmt.Errorf("failed to execute request: %w", err)
			if attempt < maxRetries {
				c.backoff(ctx, "", attempt)
				continue
			}
			return lastErr
		}

		if resp.StatusCode == http.StatusOK {
			err = c.decodeBody(resp, result)
			resp.Body.Close()
			return err
		}

		// Non-200: classify via the structured API error mapping.
		retryAfter := resp.Header.Get("Retry-After")
		apiErr := parseAPIError(resp)
		resp.Body.Close()

		retriable := false
		if ae, ok := apiErr.(*APIError); ok {
			retriable = ae.IsRetriable
		}

		lastErr = apiErr
		if attempt < maxRetries && retriable {
			c.backoff(ctx, retryAfter, attempt)
			continue
		}
		return apiErr
	}
	return lastErr
}

// send builds and issues a single HTTP request against baseURL+endpoint.
// The caller owns resp.Body.
func (c *Client) send(ctx context.Context, baseURL, method, endpoint string, body interface{}) (*http.Response, error) {
	url := baseURL + endpoint

	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en-US,en")
	if body != nil {
		req.Header.Set("Accept", "application/json")
	}

	return c.httpClient.Do(req)
}

// sendMultipart issues a single multipart/form-data POST (used by
// AudioService.Transcribe). Unlike doRequest/send it does not retry —
// re-uploading a file on transient failure is left to the caller.
func (c *Client) sendMultipart(ctx context.Context, endpoint, contentType string, body []byte) (*http.Response, error) {
	url := c.config.BaseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

// decodeBody reads and JSON-decodes the response body into result (when non-nil).
func (c *Client) decodeBody(resp *http.Response, result interface{}) error {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}
	return nil
}

// backoff sleeps before a retry, honoring a Retry-After header value (integer
// seconds) when present, otherwise exponential backoff with jitter. It respects
// ctx cancellation so callers can abort a pending retry.
func (c *Client) backoff(ctx context.Context, retryAfter string, attempt int) {
	d := c.retryDelay(retryAfter, attempt)
	if d <= 0 {
		return
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	}
}

// retryDelay computes the delay before the next attempt. Retry-After (integer
// seconds) wins when present; otherwise base * 2^attempt, capped at
// maxRetryDelay, with up to 25% jitter to avoid thundering herds.
func (c *Client) retryDelay(retryAfter string, attempt int) time.Duration {
	base := c.config.RetryDelay
	if base <= 0 {
		base = DefaultRetryDelay
	}
	if retryAfter != "" {
		if secs, err := strconv.Atoi(retryAfter); err == nil && secs >= 0 {
			d := time.Duration(secs) * time.Second
			if d > maxRetryDelay {
				return maxRetryDelay
			}
			return d
		}
	}
	d := base << uint(attempt)
	if d <= 0 || d > maxRetryDelay {
		d = maxRetryDelay
	}
	jitter := time.Duration(rand.Int63n(int64(d)/4 + 1))
	return d + jitter
}

// Tools returns the tools service
func (c *Client) Tools() *ToolsService {
	return c.tools
}

// Images returns the image generation service
func (c *Client) Images() *ImagesService {
	return c.images
}

// Videos returns the video generation service
func (c *Client) Videos() *VideosService {
	return c.videos
}

// Audio returns the audio transcription service
func (c *Client) Audio() *AudioService {
	return c.audio
}

// Layout returns the layout parsing (OCR) service
func (c *Client) Layout() *LayoutService {
	return c.layout
}

// Files returns the file upload/management service
func (c *Client) Files() *FilesService {
	return c.files
}

// Batch returns the batch job service
func (c *Client) Batch() *BatchService {
	return c.batch
}

// Agents returns the agents (specialized-agent invocation) service
func (c *Client) Agents() *AgentsService {
	return c.agents
}
