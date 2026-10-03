package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// apiRequest describes one API call. It is the only way services talk to the
// network: every call goes through Client.open, which owns authentication,
// retry/backoff, error parsing, and the observability-hook lifecycle — no
// service builds its own http.Request.
//
// Routing fields left empty fall back to the client's configuration, so most
// call sites only set method, path, and body.
type apiRequest struct {
	method  string
	baseURL string            // "" → Config.BaseURL
	path    string            // endpoint path (plus query) under baseURL
	apiKey  string            // "" → Config.APIKey
	header  map[string]string // extra headers, applied after the defaults
	body    any               // JSON-encoded when non-nil
	form    *multipartBody    // multipart/form-data body; excludes body. Never retried.
	service string            // hook attribution; a WithService ctx value wins
	model   string            // hook attribution; a WithModel ctx value wins
}

// multipartBody is an encoded multipart/form-data request body.
type multipartBody struct {
	contentType string
	data        []byte
}

// formFile is the file part of a multipart upload.
type formFile struct {
	field, name string
	data        []byte
}

// newMultipartBody encodes an optional file part followed by text fields.
// fields is a flat name, value, name, value… list; pairs with an empty value
// are skipped (the API treats an absent field as "unset"), and a name may
// repeat for list-valued fields.
func newMultipartBody(file *formFile, fields ...string) (*multipartBody, error) {
	if len(fields)%2 != 0 {
		return nil, fmt.Errorf("multipart fields must be name/value pairs")
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if file != nil {
		fw, err := w.CreateFormFile(file.field, file.name)
		if err != nil {
			return nil, fmt.Errorf("failed to build multipart file field: %w", err)
		}
		if _, err := fw.Write(file.data); err != nil {
			return nil, fmt.Errorf("failed to write multipart file data: %w", err)
		}
	}
	for i := 0; i < len(fields); i += 2 {
		if fields[i+1] == "" {
			continue
		}
		if err := w.WriteField(fields[i], fields[i+1]); err != nil {
			return nil, fmt.Errorf("failed to write multipart field %q: %w", fields[i], err)
		}
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize multipart body: %w", err)
	}
	return &multipartBody{contentType: w.FormDataContentType(), data: buf.Bytes()}, nil
}

// attempt is an open 2xx response plus the hook state of the attempt that
// produced it. The caller owns resp.Body and must finish the attempt with
// exactly one of succeed or fail, so every OnRequest is paired with exactly
// one terminal hook (OnResponse or OnError).
type attempt struct {
	client *Client
	resp   *http.Response
	ctx    context.Context // hook-derived context (e.g. carries a span)
	meta   RequestMeta
	start  time.Time
}

func (a *attempt) succeed(usage *Usage) {
	a.client.callHooksResponse(a.ctx, ResponseMeta{
		RequestMeta: a.meta,
		StatusCode:  a.resp.StatusCode,
		Duration:    time.Since(a.start),
		Usage:       usage,
	})
}

func (a *attempt) fail(err error) {
	a.client.callHooksError(a.ctx, a.meta, err)
}

// open sends r and returns the first 2xx response. Transient failures (429,
// 5xx, network errors) are retried with backoff up to Config.MaxRetries;
// multipart uploads are sent once, since re-uploading is the caller's call.
// Each attempt fires OnRequest, and each attempt that does not produce the
// returned response fires OnError, so hooks see one terminal event per
// attempt and no span outlives its attempt.
func (c *Client) open(ctx context.Context, r apiRequest) (*attempt, error) {
	maxRetries := c.config.MaxRetries
	if r.form != nil {
		maxRetries = 0
	}
	for n := 0; ; n++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		meta := c.requestMeta(ctx, r, n)
		hookCtx := c.callHooksRequest(ctx, meta)
		start := time.Now()

		resp, err := c.send(hookCtx, r)
		retryAfter := ""
		switch {
		case err != nil:
			err = fmt.Errorf("failed to execute request: %w", err)
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			return &attempt{client: c, resp: resp, ctx: hookCtx, meta: meta, start: start}, nil
		default:
			retryAfter = resp.Header.Get("Retry-After")
			err = parseAPIError(resp)
			resp.Body.Close()
		}

		c.callHooksError(hookCtx, meta, err)
		if n >= maxRetries || !isRetriable(err) {
			return nil, err
		}
		c.backoff(hookCtx, retryAfter, n)
	}
}

// do performs r and JSON-decodes a successful response body into result
// (nil discards it).
func (c *Client) do(ctx context.Context, r apiRequest, result any) error {
	a, err := c.open(ctx, r)
	if err != nil {
		return err
	}
	defer a.resp.Body.Close()
	if err := decodeJSON(a.resp.Body, result); err != nil {
		a.fail(err)
		return err
	}
	a.succeed(extractUsage(result))
	return nil
}

// doRaw performs r and returns the raw body of a successful response — for
// binary payloads such as synthesized audio or downloaded file content.
func (c *Client) doRaw(ctx context.Context, r apiRequest) ([]byte, error) {
	a, err := c.open(ctx, r)
	if err != nil {
		return nil, err
	}
	defer a.resp.Body.Close()
	data, err := io.ReadAll(a.resp.Body)
	if err != nil {
		err = fmt.Errorf("failed to read response body: %w", err)
		a.fail(err)
		return nil, err
	}
	a.succeed(nil)
	return data, nil
}

// doRequest is do against Config.BaseURL with the default credential — the
// common case for most endpoints.
func (c *Client) doRequest(ctx context.Context, method, path string, body, result any) error {
	return c.do(ctx, apiRequest{method: method, path: path, body: body}, result)
}

// send builds and issues a single HTTP request for r. The caller owns the
// response body.
func (c *Client) send(ctx context.Context, r apiRequest) (*http.Response, error) {
	baseURL, apiKey := r.baseURL, r.apiKey
	if baseURL == "" {
		baseURL = c.config.BaseURL
	}
	if apiKey == "" {
		apiKey = c.config.APIKey
	}

	var body io.Reader
	contentType := ""
	switch {
	case r.form != nil:
		body, contentType = bytes.NewReader(r.form.data), r.form.contentType
	case r.body != nil:
		data, err := json.Marshal(r.body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body, contentType = bytes.NewReader(data), "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, r.method, baseURL+r.path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept-Language", "en-US,en")
	req.Header.Set("User-Agent", c.config.UserAgent)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Accept", "application/json")
	}
	for k, v := range r.header {
		req.Header.Set(k, v)
	}
	return c.httpClient.Do(req)
}

// decodeJSON reads body and decodes it into result. A nil result or an
// empty body (e.g. 204 No Content) decodes to nothing.
func decodeJSON(body io.Reader, result any) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if result == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, result); err != nil {
		return &decodeError{err: err}
	}
	return nil
}

// decodeError reports a 2xx response whose body does not match the expected
// shape — distinct from an API error, since the server did answer.
type decodeError struct{ err error }

func (e *decodeError) Error() string { return "failed to unmarshal response: " + e.err.Error() }
func (e *decodeError) Unwrap() error { return e.err }

// usageBearer is implemented by response types that carry token usage
// (ChatResponse, EmbeddingsResponse, AsyncResultResponse, …), letting
// extractUsage surface it to OnResponse hooks without reflection.
type usageBearer interface {
	GetUsage() *Usage
}

// extractUsage returns the token usage carried by a decoded response, or nil
// when the response type has none — hooks can tell "not reported" from
// "zero tokens" by the nil check.
func extractUsage(result any) *Usage {
	if u, ok := result.(usageBearer); ok {
		return u.GetUsage()
	}
	return nil
}
