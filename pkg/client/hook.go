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
//   - pkg/observe — OpenTelemetry / Langfuse hooks (depend on otel SDK)
//   - user-provided — slog-based loggers, metrics counters, anything
//
// All methods must be safe for concurrent use. Methods are invoked from the
// doRequest/send/sendMultipart facade (single goroutine per request) and
// from the SSE reader goroutine for OnStreamChunk. A nil/empty Config.Hooks
// slice skips all invocation — the no-hook path is zero-allocation.
//
// Error semantics: a Hook must not panic. A panicking Hook aborts the request
// just like any other panic would; guard accordingly in production hooks.
type Hook interface {
	// OnRequest is invoked once per attempt, before the HTTP request is sent.
	// The returned context replaces the input context for downstream hook
	// methods and the actual HTTP call — use it to attach tracing spans or
	// request-scoped state. Return ctx unchanged if you have nothing to add.
	OnRequest(ctx context.Context, meta RequestMeta) context.Context

	// OnResponse is invoked after a non-streaming response is received and
	// parsed successfully (HTTP 2xx). Not invoked for streaming responses
	// (use OnStreamChunk) or errors (use OnError). meta.Attempt identifies
	// which retry attempt succeeded.
	OnResponse(ctx context.Context, meta ResponseMeta)

	// OnError is invoked when a request ultimately fails — either all
	// retries were exhausted or the error was non-retriable. Not invoked
	// for successful responses. The err is the final error the caller will
	// see (typically a *APIError). meta.Attempt is the last attempt index.
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
	// Service is a short label identifying the calling service
	// ("chat", "anthropic", "embeddings", "models", etc.). Stamped into the
	// context by each service via WithService before doRequest; defaults to
	// "" when a service doesn't bother (the hook still gets Method/Endpoint).
	Service string
	// Method is the HTTP method ("GET", "POST", ...).
	Method string
	// Endpoint is the URL path (without the base URL), e.g. "/chat/completions".
	Endpoint string
	// Model is the model ID for requests that carry one (chat, embeddings,
	// rerank, anthropic). Empty for endpoints that don't take a model.
	// Stamped via WithModel.
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

// WithService stamps a service label into the context for hook extraction.
// Services call this (or WithModel for the model ID) before doRequest; the
// facade reads it back when building RequestMeta. Callers who don't care
// about per-service hook attributes can skip this — RequestMeta.Service will
// be "".
func WithService(ctx context.Context, service string) context.Context {
	return context.WithValue(ctx, ctxKeyService, service)
}

// WithModel stamps a model ID into the context for hook extraction. Same
// pattern as WithService. ChatService, AnthropicService, EmbeddingsService,
// RerankService should stamp the model they're sending so hooks can report
// per-model metrics without re-parsing the request body.
func WithModel(ctx context.Context, model string) context.Context {
	return context.WithValue(ctx, ctxKeyModel, model)
}

// ServiceFromContext returns the service label stamped by WithService, or "".
// Used by the doRequest facade when building RequestMeta for hook invocation.
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

// buildRequestMeta constructs RequestMeta from the facade's known inputs +
// the context-stamped service/model. attempt is the current retry index.
func (c *Client) buildRequestMeta(ctx context.Context, method, endpoint string, attempt int) RequestMeta {
	return RequestMeta{
		Service:  ServiceFromContext(ctx),
		Method:   method,
		Endpoint: endpoint,
		Model:    ModelFromContext(ctx),
		Attempt:  attempt,
	}
}
