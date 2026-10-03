package client

import (
	"context"
	"time"
)

// Hook is the observability seam. Implementations receive metadata about
// every request, response, error, and stream chunk flowing through the
// client, without needing to wrap the http.RoundTripper (which is coarser
// and can't see parse-level errors or per-attempt boundaries).
//
// The interface is intentionally minimal and stdlib-only so pkg/client stays
// dependency-free. Concrete implementations live in separate packages:
//
//   - pkg/observe — OpenTelemetry hooks (depend on the otel SDK)
//   - user-provided — slog-based loggers, metrics counters, anything
//
// All methods must be safe for concurrent use. They are invoked from the
// request transport (Client.open and its callers) on the calling goroutine;
// OnStreamChunk fires on the goroutine ranging over a Stream iterator. A
// nil/empty Config.Hooks slice skips all invocation — the no-hook path is
// zero-allocation.
//
// Pairing contract: every OnRequest is followed by exactly one terminal call
// for the same attempt — OnResponse on success, OnError otherwise (including
// a failed attempt that is about to be retried, and a stream the caller
// stops ranging over early). A span started in OnRequest can therefore
// always be ended in the terminal call.
//
// Error semantics: a Hook must not panic. A panicking Hook aborts the request
// just like any other panic would; guard accordingly in production hooks.
type Hook interface {
	// OnRequest is invoked once per attempt, before the HTTP request is sent.
	// The returned context replaces the input context for downstream hook
	// methods and the actual HTTP call — use it to attach tracing spans or
	// request-scoped state. Return ctx unchanged if you have nothing to add.
	OnRequest(ctx context.Context, meta RequestMeta) context.Context

	// OnResponse is invoked when an attempt succeeds: a response was received
	// and parsed (HTTP 2xx), or a stream ended cleanly. meta.Attempt
	// identifies which retry attempt succeeded.
	OnResponse(ctx context.Context, meta ResponseMeta)

	// OnError is invoked when an attempt fails — a transport error, a
	// non-2xx response (retried or not), a body that fails to decode, a
	// mid-stream failure, or a stream abandoned early (context.Canceled).
	// meta.Attempt identifies the failed attempt; when it is the last one,
	// err is the error the caller sees (typically a *APIError).
	OnError(ctx context.Context, meta RequestMeta, err error)

	// OnStreamChunk is invoked for each chunk parsed from a streaming
	// response (ChatService.Stream / AnthropicService.Stream). The chunk is
	// passed as `any` (typed as StreamChunk or AnthropicStreamEvent at call
	// time) so the Hook interface stays generic over both protocols without
	// a type-parameter explosion. Type-assert in the implementation:
	//
	//   switch c := chunk.(type) {
	//   case client.StreamChunk: ...
	//   case client.AnthropicStreamEvent: ...
	//   }
	//
	// Not invoked for non-streaming requests.
	OnStreamChunk(ctx context.Context, meta RequestMeta, chunk any)
}

// RequestMeta describes an in-flight request for hook invocation.
type RequestMeta struct {
	// Service is a short label identifying the calling service ("chat",
	// "anthropic", "embeddings", ...). Set by the service issuing the
	// request; a WithService context value overrides it. Empty for
	// endpoints without a natural label (the hook still gets Endpoint).
	Service string
	// Method is the HTTP method ("GET", "POST", ...).
	Method string
	// Endpoint is the URL path (without the base URL), e.g. "/chat/completions".
	Endpoint string
	// Model is the model ID for requests that carry one (chat, embeddings,
	// rerank, anthropic, ...). Empty for endpoints that don't take a model.
	// A WithModel context value overrides it.
	Model string
	// Attempt is the 0-indexed retry attempt for this request. 0 on the
	// first try; incremented on each retry.
	Attempt int
}

// ResponseMeta describes a completed response for OnResponse.
type ResponseMeta struct {
	RequestMeta
	// StatusCode is the HTTP status code received.
	StatusCode int
	// Duration is the wall-clock time from request send to response parsed.
	Duration time.Duration
	// Usage is the token usage reported by the API, when the response
	// includes one (chat completions, embeddings). Nil for responses that
	// don't carry usage (models list, file ops, etc.).
	Usage *Usage
}

// ctxKey is unexported so callers can't collide with our context keys.
type ctxKey int

const (
	ctxKeyService ctxKey = iota
	ctxKeyModel
)

// WithService stamps a service label into the context, overriding the label
// the issuing service sets on RequestMeta.Service — e.g. to attribute calls
// to an application feature instead.
func WithService(ctx context.Context, service string) context.Context {
	return context.WithValue(ctx, ctxKeyService, service)
}

// WithModel stamps a model ID into the context, overriding the model the
// issuing service sets on RequestMeta.Model. Same pattern as WithService.
func WithModel(ctx context.Context, model string) context.Context {
	return context.WithValue(ctx, ctxKeyModel, model)
}

// ServiceFromContext returns the service label stamped by WithService, or "".
func ServiceFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyService).(string); ok {
		return v
	}
	return ""
}

// ModelFromContext returns the model ID stamped by WithModel, or "".
func ModelFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyModel).(string); ok {
		return v
	}
	return ""
}

// callHooksRequest invokes OnRequest on every configured hook. Returns the
// (possibly modified) context. Nil-safe: returns ctx unchanged when no hooks
// are configured.
func (c *Client) callHooksRequest(ctx context.Context, meta RequestMeta) context.Context {
	for _, h := range c.hooks {
		ctx = h.OnRequest(ctx, meta)
	}
	return ctx
}

func (c *Client) callHooksResponse(ctx context.Context, meta ResponseMeta) {
	for _, h := range c.hooks {
		h.OnResponse(ctx, meta)
	}
}

func (c *Client) callHooksError(ctx context.Context, meta RequestMeta, err error) {
	for _, h := range c.hooks {
		h.OnError(ctx, meta, err)
	}
}

func (c *Client) callHooksStreamChunk(ctx context.Context, meta RequestMeta, chunk any) {
	for _, h := range c.hooks {
		h.OnStreamChunk(ctx, meta, chunk)
	}
}

// requestMeta builds the RequestMeta for attempt n of r. Context stamps
// (WithService/WithModel) override the labels the issuing service set.
func (c *Client) requestMeta(ctx context.Context, r apiRequest, n int) RequestMeta {
	meta := RequestMeta{
		Service:  r.service,
		Method:   r.method,
		Endpoint: r.path,
		Model:    r.model,
		Attempt:  n,
	}
	if s := ServiceFromContext(ctx); s != "" {
		meta.Service = s
	}
	if m := ModelFromContext(ctx); m != "" {
		meta.Model = m
	}
	return meta
}
