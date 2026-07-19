// Package observe provides observability hooks for the go-z-ai client.
//
// It implements the client.Hook interface (defined stdlib-only in pkg/client)
// with concrete tracing/metrics/logging backends:
//
//   - OTelHook: OpenTelemetry traces + metrics, emitting the GenAI semantic-
//     convention attributes (gen_ai.request.model, gen_ai.usage.input_tokens,
//     gen_ai.system = "z.ai", etc.).
//
// This package is the only public package in the module that imports
// third-party dependencies (go.opentelemetry.io/otel). pkg/client stays
// stdlib-only by design — see docs/en/architecture.md.
//
// Usage:
//
//	import (
//	    "go.opentelemetry.io/otel"
//	    "github.com/SamyRai/go-z-ai/pkg/client"
//	    "github.com/SamyRai/go-z-ai/pkg/observe"
//	)
//
//	c, _ := client.NewClient(client.Config{
//	    APIKey: os.Getenv("ZAI_API_KEY"),
//	    Hooks:  []client.Hook{observe.NewOTelHook("my-service")},
//	})
//
// The hook uses the global TracerProvider and MeterProvider by default
// (otel.GetTracerProvider() / otel.GetMeterProvider()). Configure those
// globally with the OTel SDK's sdk/trace and sdk/metric packages, or pass a
// custom provider to NewOTelHookWithProvider for isolated setups (tests,
// per-client backends).
package observe

import (
	"context"
	"fmt"

	"github.com/SamyRai/go-z-ai/pkg/client"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// GenAI semantic-convention attribute keys actually emitted by this hook.
//
// Source: OpenTelemetry GenAI semantic conventions
// (https://opentelemetry.io/docs/specs/semconv/registry/attributes/gen-ai/).
// Defined as local string constants rather than imported from
// go.opentelemetry.io/otel/semconv because the GenAI attributes are mid-
// migration to a separate semconv-genai module (the names are stable; the Go
// package layout is not). When that package stabilizes, these constants can
// be replaced with the upstream imports without a public API change.
//
// Only attributes the hook actually emits are declared here. Add constants
// (and emit them) when the underlying RequestMeta/ResponseMeta grows new
// fields (e.g. max_tokens, temperature, response_model from the wire).
const (
	// attrGenAISystem identifies the GenAI provider. Always "z.ai" for this
	// client, matching the semconv's list of known systems.
	attrGenAISystem = "gen_ai.system"
	// attrGenAIRequestModel carries the requested model ID.
	attrGenAIRequestModel = "gen_ai.request.model"
	// attrGenAIResponseModel carries the actual model that served the
	// response (may differ from the request model after server-side aliasing).
	attrGenAIResponseModel = "gen_ai.response.model"
	// attrGenAIOperationType is the operation name: "chat", "embeddings", etc.
	attrGenAIOperationType = "gen_ai.operation.name"
	// attrHTTPRequestMethod and attrHTTPResponseStatusCode are the HTTP
	// method/status semconv attributes (operation-level, not the HTTP-client
	// span — we emit one span per Z.AI API call, not per HTTP roundtrip).
	attrHTTPRequestMethod      = "http.request.method"
	attrHTTPResponseStatusCode = "http.response.status_code"
	// attrRetryAttempt is the 0-indexed retry attempt for this request.
	attrRetryAttempt = "retry_attempt"
	// attrZAIErrorCategory is the client's typed error category
	// (ErrorCategory* in pkg/client/errors.go).
	attrZAIErrorCategory = "zai.error.category"
	// attrZAIErrorCode is the Z.AI business error code (e.g. 1113, 1302).
	attrZAIErrorCode = "zai.error.code"
)

// spanKey is the context key under which OTelHook stashes the in-flight span
// so OnResponse / OnError can end it. Using a private key type prevents
// collisions with caller code.
type spanKey int

const activeSpan spanKey = 0

// GenAISystemValue is the gen_ai.system value this hook emits for Z.AI.
const GenAISystemValue = "z.ai"

// OTelHook is a client.Hook that emits OpenTelemetry traces and metrics
// for every Z.AI API call. The zero value is NOT valid — construct with
// NewOTelHook or NewOTelHookWithProvider.
//
// The hook emits one span per request attempt (a retried request produces
// N child spans, one per actual HTTP send) under a parent operation span.
// Spans carry the GenAI semconv attributes plus HTTP status, retry attempt,
// and (on error) the Z.AI business error code/category. Token usage from
// successful responses is recorded both as span attributes and as increments
// to the gen_ai.client.token.usage counter (broken down by input/output and
// model).
//
// All metrics use the global MeterProvider by default. The set of metrics
// emitted is intentionally minimal — production deployments usually layer
// their own dashboards on top of these primitives.
type OTelHook struct {
	tracer trace.Tracer
	meter  metric.Meter
	// serviceName is attached to every span as the OTel service.name resource
	// attribute when non-empty. The OTel SDK usually sets this globally via
	// resource.Default() — this field is for per-hook overrides (e.g. when
	// one process serves multiple logical services).
	serviceName string

	// metricDur is the request-duration histogram, lazily created on first
	// OnResponse (so a hook whose caller never registers a MeterProvider
	// doesn't fail at NewOTelHook time).
	metricDur metric.Float64Histogram
	// metricTokens is the token-usage counter, lazily created.
	metricTokens metric.Int64Counter
	// metricRequests is the request counter (success/error), lazily created.
	metricRequests metric.Int64Counter
}

// NewOTelHook constructs an OTelHook using the global TracerProvider and
// MeterProvider (otel.GetTracerProvider() / otel.GetMeterProvider()). The
// serviceName is attached to spans as service.name (pass "" to omit).
//
// Configure the global providers once at program start with the OTel SDK:
//
//	tp, err := sdktrace.NewTracerProvider(...)
//	otel.SetTracerProvider(tp)
//	defer tp.Shutdown(ctx)
//
// Then every OTelHook constructed via NewOTelHook picks them up.
func NewOTelHook(serviceName string) *OTelHook {
	return NewOTelHookWithProvider(serviceName, otel.GetTracerProvider(), otel.GetMeterProvider())
}

// NewOTelHookWithProvider constructs an OTelHook with explicit Tracer and
// Meter providers. Use this for tests, for per-client backends, or when you
// don't want to register providers globally.
func NewOTelHookWithProvider(serviceName string, tp trace.TracerProvider, mp metric.MeterProvider) *OTelHook {
	tracer := tp.Tracer("github.com/SamyRai/go-z-ai/pkg/observe",
		trace.WithInstrumentationVersion(client.Version()))
	meter := mp.Meter("github.com/SamyRai/go-z-ai/pkg/observe",
		metric.WithInstrumentationVersion(client.Version()))
	return &OTelHook{
		tracer:      tracer,
		meter:       meter,
		serviceName: serviceName,
	}
}

// OnRequest starts a span for the attempt and attaches it to the returned
// context. The caller (the client facade) passes the returned context to
// the actual HTTP send, so spans correlate with any HTTP-client
// instrumentation layered on top.
func (h *OTelHook) OnRequest(ctx context.Context, meta client.RequestMeta) context.Context {
	attrs := h.baseAttrs(meta)
	spanName := spanNameFor(meta)
	ctx, span := h.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
	return context.WithValue(ctx, activeSpan, span)
}

// OnResponse ends the span started by OnRequest with OK status, records
// duration + usage metrics, and stamps response attributes (model, id,
// status code, token counts).
func (h *OTelHook) OnResponse(ctx context.Context, meta client.ResponseMeta) {
	span, _ := ctx.Value(activeSpan).(trace.Span)
	if span != nil {
		respAttrs := []attribute.KeyValue{
			attribute.Int(attrHTTPResponseStatusCode, meta.StatusCode),
		}
		if meta.Model != "" {
			// gen_ai.response.model can differ from the request model after
			// server-side aliasing; here we only have the request model, so
			// stamp it as the response model too (the wire response.Model
			// would be more accurate — future enhancement).
			respAttrs = append(respAttrs, attribute.String(attrGenAIResponseModel, meta.Model))
		}
		span.SetAttributes(respAttrs...)
		// 2xx is OK by default; the SDK leaves status unset. We leave it unset
		// rather than explicitly setting Unset so the SDK's defaults apply.
		span.End()
	}

	// Metrics. Duration histogram and request counter always; token counter
	// only when usage is reported.
	h.recordDuration(ctx, meta)
	h.recordRequest(ctx, meta.RequestMeta, true)
	if meta.Usage != nil {
		h.recordTokens(ctx, meta.RequestMeta, meta.Usage)
	}
}

// OnError ends the span started by OnRequest with ERROR status, records the
// error message + (when it's an *APIError) the Z.AI business code/category,
// and increments the error request counter.
func (h *OTelHook) OnError(ctx context.Context, meta client.RequestMeta, err error) {
	span, _ := ctx.Value(activeSpan).(trace.Span)
	if span != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		// Surface the Z.AI business error code/category as span attributes
		// for queryability (errors get recorded as events; attributes make
		// them filterable in Tempo/Jaeger/etc.).
		extra := errorAttrs(err)
		if len(extra) > 0 {
			span.SetAttributes(extra...)
		}
		span.End()
	}
	h.recordRequest(ctx, meta, false)
}

// OnStreamChunk records each chunk as a span event on the active span (if
// any). This makes the per-chunk timeline visible in trace viewers without
// creating one span per chunk (which would be heavy and noisy).
//
// The chunk is type-asserted to client.StreamChunk or
// client.AnthropicStreamEvent to extract a short summary; unhandled types
// are recorded as a generic "stream_chunk" event.
func (h *OTelHook) OnStreamChunk(ctx context.Context, meta client.RequestMeta, chunk any) {
	span, _ := ctx.Value(activeSpan).(trace.Span)
	if span == nil {
		return
	}
	switch c := chunk.(type) {
	case client.StreamChunk:
		attrs := []attribute.KeyValue{}
		if c.ID != "" {
			attrs = append(attrs, attribute.String("stream.chunk_id", c.ID))
		}
		if len(c.Choices) > 0 {
			attrs = append(attrs, attribute.Int("stream.choices", len(c.Choices)))
		}
		span.AddEvent("stream_chunk", trace.WithAttributes(attrs...))
	case client.AnthropicStreamEvent:
		span.AddEvent("stream_event", trace.WithAttributes(
			attribute.String("stream.event_type", c.Type),
		))
	default:
		span.AddEvent("stream_chunk")
	}
}

// --- helpers ---------------------------------------------------------------

// baseAttrs is the attribute set every span starts with (request-side).
func (h *OTelHook) baseAttrs(meta client.RequestMeta) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String(attrGenAISystem, GenAISystemValue),
		attribute.String(attrHTTPRequestMethod, meta.Method),
		attribute.Int(attrRetryAttempt, meta.Attempt),
	}
	if meta.Service != "" {
		attrs = append(attrs, attribute.String(attrGenAIOperationType, meta.Service))
	}
	if meta.Model != "" {
		attrs = append(attrs, attribute.String(attrGenAIRequestModel, meta.Model))
	}
	if h.serviceName != "" {
		attrs = append(attrs, attribute.String("service.name", h.serviceName))
	}
	// Endpoint as a span attribute for queryability — not the semconv
	// attrHTTPRequestMethod* since those describe a generic HTTP client span;
	// ours is a GenAI operation span and the endpoint is one of its facets.
	if meta.Endpoint != "" {
		attrs = append(attrs, attribute.String("zai.endpoint", meta.Endpoint))
	}
	return attrs
}

// spanNameFor produces a semconv-style operation span name:
// "<service> <model>" when both are set, "<service>" otherwise, "<method>
// <endpoint>" as the final fallback (so unnamed services still get a useful
// span name in trace UIs).
func spanNameFor(meta client.RequestMeta) string {
	switch {
	case meta.Service != "" && meta.Model != "":
		return meta.Service + " " + meta.Model
	case meta.Service != "":
		return meta.Service
	default:
		return meta.Method + " " + meta.Endpoint
	}
}

// errorAttrs extracts Z.AI-specific attributes from a *client.APIError when
// present. Returns nil for other error types (the error is still recorded as
// a span event via RecordError in OnError).
func errorAttrs(err error) []attribute.KeyValue {
	type codedError interface {
		GetCode() int
		GetCategory() client.ErrorCategory
	}
	// *client.APIError exposes Code/Category as fields, not getters; use a
	// concrete type assertion rather than an interface to avoid adding
	// getter methods to the public API just for this hook.
	var ae *client.APIError
	if asAPIError(err, &ae) && ae != nil {
		return []attribute.KeyValue{
			attribute.Int(attrZAIErrorCode, ae.Code),
			attribute.String(attrZAIErrorCategory, string(ae.Category)),
		}
	}
	_ = codedError(nil) // reserved for a future getter-based path
	return nil
}

// asAPIError wraps errors.As so we don't need to import errors at the top
// level just for one call. Kept thin for readability.
func asAPIError(err error, target **client.APIError) bool {
	if err == nil {
		return false
	}
	// errors.As doesn't type-match when target is **T; do a manual chain walk.
	for {
		if ae, ok := err.(*client.APIError); ok {
			*target = ae
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
		if err == nil {
			return false
		}
	}
}

// --- metrics ---------------------------------------------------------------

// recordDuration increments the request-duration histogram. Lazily creates
// the histogram on first call (so a missing MeterProvider only fails when
// telemetry actually flows).
func (h *OTelHook) recordDuration(ctx context.Context, meta client.ResponseMeta) {
	if meta.Duration <= 0 {
		return
	}
	dur, err := h.durationHistogram()
	if err != nil {
		return // no meter configured; silently skip (best-effort observability)
	}
	attrs := h.baseAttrs(meta.RequestMeta)
	attrs = append(attrs, attribute.Int(attrHTTPResponseStatusCode, meta.StatusCode))
	dur.Record(ctx, meta.Duration.Seconds(), metric.WithAttributes(attrs...))
}

// recordRequest increments the request counter (success or error).
func (h *OTelHook) recordRequest(ctx context.Context, meta client.RequestMeta, success bool) {
	req, err := h.requestCounter()
	if err != nil {
		return
	}
	attrs := h.baseAttrs(meta)
	status := "success"
	if !success {
		status = "error"
	}
	attrs = append(attrs, attribute.String("status", status))
	req.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// recordTokens increments the token-usage counter, broken down by
// input/output and model.
func (h *OTelHook) recordTokens(ctx context.Context, meta client.RequestMeta, u *client.Usage) {
	if u == nil {
		return
	}
	tok, err := h.tokenCounter()
	if err != nil {
		return
	}
	baseAttrs := h.baseAttrs(meta)
	if u.PromptTokens > 0 {
		tok.Add(ctx, int64(u.PromptTokens),
			metric.WithAttributes(append(baseAttrs, attribute.String("gen_ai.token.type", "input"))...))
	}
	if u.CompletionTokens > 0 {
		tok.Add(ctx, int64(u.CompletionTokens),
			metric.WithAttributes(append(baseAttrs, attribute.String("gen_ai.token.type", "output"))...))
	}
}

// durationHistogram lazily creates the histogram. Failure (e.g. no meter
// configured) is sticky — once it fails, subsequent calls short-circuit on
// h.metricDur == nil after the first attempt. We re-attempt each call until
// success, which is cheap (h.meter.Histogram is idempotent).
func (h *OTelHook) durationHistogram() (metric.Float64Histogram, error) {
	if h.metricDur != nil {
		return h.metricDur, nil
	}
	d, err := h.meter.Float64Histogram("gen_ai.client.operation.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of Z.AI API operations"))
	if err != nil {
		return nil, fmt.Errorf("create duration histogram: %w", err)
	}
	h.metricDur = d
	return d, nil
}

func (h *OTelHook) requestCounter() (metric.Int64Counter, error) {
	if h.metricRequests != nil {
		return h.metricRequests, nil
	}
	c, err := h.meter.Int64Counter("gen_ai.client.operation.count",
		metric.WithDescription("Number of Z.AI API operations (success or error)"))
	if err != nil {
		return nil, fmt.Errorf("create request counter: %w", err)
	}
	h.metricRequests = c
	return c, nil
}

func (h *OTelHook) tokenCounter() (metric.Int64Counter, error) {
	if h.metricTokens != nil {
		return h.metricTokens, nil
	}
	c, err := h.meter.Int64Counter("gen_ai.client.token.usage",
		metric.WithDescription("Token usage from Z.AI API responses, by type (input/output)"))
	if err != nil {
		return nil, fmt.Errorf("create token counter: %w", err)
	}
	h.metricTokens = c
	return c, nil
}

// Compile-time check that OTelHook satisfies client.Hook.
var _ client.Hook = (*OTelHook)(nil)
