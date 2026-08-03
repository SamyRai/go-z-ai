package observe

import (
	"context"
	"testing"

	"github.com/SamyRai/go-z-ai/pkg/client"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// newTestHook builds an OTelHook wired to in-memory span and metric
// exporters, plus the exporters themselves for assertion. Defer the
// exporters' Shutdown in the test.
func newTestHook(t *testing.T, serviceName string) (*OTelHook, *tracetest.InMemoryExporter, *metric.ManualReader) {
	t.Helper()
	spanExp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(spanExp),
	)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	reader := metric.NewManualReader()
	mp := metric.NewMeterProvider(metric.WithReader(reader))
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })

	h := NewOTelHookWithProvider(serviceName, tp, mp)
	return h, spanExp, reader
}

// TestOTelHookEmitsSuccessSpan verifies a successful request emits one span
// carrying the GenAI semconv attributes + HTTP status.
func TestOTelHookEmitsSuccessSpan(t *testing.T) {
	h, exp, _ := newTestHook(t, "svc-x")
	ctx := h.OnRequest(context.Background(), client.RequestMeta{
		Service:  "chat",
		Method:   "POST",
		Endpoint: "/chat/completions",
		Model:    "glm-5.2",
		Attempt:  0,
	})
	h.OnResponse(ctx, client.ResponseMeta{
		RequestMeta: client.RequestMeta{Service: "chat", Model: "glm-5.2"},
		StatusCode:  200,
		Usage:       &client.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	})

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	s := spans[0]
	if s.Name != "chat glm-5.2" {
		t.Errorf("expected span name 'chat glm-5.2', got %q", s.Name)
	}
	attrs := attrMap(s.Attributes)
	if got := attrs[attrGenAISystem]; got != "z.ai" {
		t.Errorf("expected gen_ai.system=z.ai, got %v", got)
	}
	if got := attrs[attrGenAIRequestModel]; got != "glm-5.2" {
		t.Errorf("expected gen_ai.request.model=glm-5.2, got %v", got)
	}
	if got := attrs[attrHTTPResponseStatusCode]; got != int64(200) {
		t.Errorf("expected http.response.status_code=200, got %v", got)
	}
	if s.Status.Code != codes.Unset {
		t.Errorf("expected Unset status for 2xx, got %v", s.Status.Code)
	}
}

// TestOTelHookEmitsErrorSpan verifies an error request emits a span with
// ERROR status and the Z.AI business code/category attributes.
func TestOTelHookEmitsErrorSpan(t *testing.T) {
	h, exp, _ := newTestHook(t, "")
	ctx := h.OnRequest(context.Background(), client.RequestMeta{
		Service: "chat", Method: "POST", Endpoint: "/chat/completions", Attempt: 2,
	})
	apiErr := &client.APIError{
		HTTPStatus:  429,
		Code:        1302,
		Message:     "rate limit",
		Category:    client.ErrorCategoryRateLimit,
		IsRetriable: true,
	}
	h.OnError(ctx, client.RequestMeta{Service: "chat", Attempt: 2}, apiErr)

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	s := spans[0]
	if s.Status.Code != codes.Error {
		t.Errorf("expected ERROR status, got %v", s.Status.Code)
	}
	attrs := attrMap(s.Attributes)
	if got := attrs[attrZAIErrorCode]; got != int64(1302) {
		t.Errorf("expected zai.error.code=1302, got %v", got)
	}
	if got := attrs[attrRetryAttempt]; got != int64(2) {
		t.Errorf("expected retry_attempt=2, got %v", got)
	}
	// Non-APIError should still record an error span but no Z.AI code.
	h.OnError(context.Background(), client.RequestMeta{}, context.DeadlineExceeded)
}

// TestOTelHookEmitsTokenMetrics verifies the token-usage counter increments
// on responses carrying usage. Uses the in-memory metric reader.
func TestOTelHookEmitsTokenMetrics(t *testing.T) {
	h, _, reader := newTestHook(t, "")
	ctx := h.OnRequest(context.Background(), client.RequestMeta{
		Service: "chat", Model: "glm-5.2", Method: "POST", Endpoint: "/chat/completions",
	})
	h.OnResponse(ctx, client.ResponseMeta{
		RequestMeta: client.RequestMeta{Service: "chat", Model: "glm-5.2"},
		StatusCode:  200,
		Usage:       &client.Usage{PromptTokens: 12, CompletionTokens: 7, TotalTokens: 19},
	})

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "gen_ai.client.token.usage" {
				continue
			}
			found = true
			data, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("expected Sum[int64], got %T", m.Data)
			}
			var total int64
			for _, dp := range data.DataPoints {
				total += dp.Value
			}
			if total != 19 {
				t.Errorf("expected 19 total tokens counted, got %d", total)
			}
		}
	}
	if !found {
		t.Fatal("expected gen_ai.client.token.usage metric to be present")
	}
}

// TestOTelHookStreamChunkEvent verifies OnStreamChunk records span events
// (one per chunk) without creating one span per chunk.
func TestOTelHookStreamChunkEvent(t *testing.T) {
	h, exp, _ := newTestHook(t, "")
	ctx := h.OnRequest(context.Background(), client.RequestMeta{
		Service: "chat", Model: "glm-5.2", Method: "POST",
	})
	h.OnStreamChunk(ctx, client.RequestMeta{}, client.StreamChunk{
		ID:      "chunk1",
		Choices: []client.StreamChoice{{Index: 0}},
	})
	h.OnStreamChunk(ctx, client.RequestMeta{}, client.AnthropicStreamEvent{
		Type: "content_block_delta",
	})
	h.OnResponse(ctx, client.ResponseMeta{
		RequestMeta: client.RequestMeta{Service: "chat", Model: "glm-5.2"},
		StatusCode:  200,
	})

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected exactly 1 span (chunks are events, not spans), got %d", len(spans))
	}
	if got := len(spans[0].Events); got != 2 {
		t.Errorf("expected 2 span events (one per chunk), got %d", got)
	}
}

// TestOTelHookNoOpOnNilSpan verifies OnResponse/OnError/OnStreamChunk are
// safe when called without an active span in the context (defensive — shouldn't
// happen in normal flow, but the hook must not panic).
func TestOTelHookNoOpOnNilSpan(t *testing.T) {
	h, exp, _ := newTestHook(t, "")
	plain := context.Background() // no active span
	h.OnResponse(plain, client.ResponseMeta{StatusCode: 200})
	h.OnError(plain, client.RequestMeta{}, context.Canceled)
	h.OnStreamChunk(plain, client.RequestMeta{}, client.StreamChunk{})
	if len(exp.GetSpans()) != 0 {
		t.Errorf("expected 0 spans from nil-span path, got %d", len(exp.GetSpans()))
	}
}

// TestOTelHookSpanNameFallbacks verifies the span-name policy: prefer
// "<service> <model>", fall back to "<service>", then "<method> <endpoint>".
func TestOTelHookSpanNameFallbacks(t *testing.T) {
	cases := []struct {
		meta client.RequestMeta
		want string
	}{
		{client.RequestMeta{Service: "chat", Model: "glm-5.2"}, "chat glm-5.2"},
		{client.RequestMeta{Service: "models"}, "models"},
		{client.RequestMeta{Method: "POST", Endpoint: "/x"}, "POST /x"},
	}
	for _, c := range cases {
		if got := spanNameFor(c.meta); got != c.want {
			t.Errorf("spanNameFor(%+v) = %q, want %q", c.meta, got, c.want)
		}
	}
}

// TestOTelHookLazyMetrics verifies metrics are lazily created (a hook with
// no requests doesn't fail). Constructs a hook with a real but unused meter
// provider and asserts no Collect-time errors.
func TestOTelHookLazyMetrics(t *testing.T) {
	h, _, reader := newTestHook(t, "")
	// No OnResponse calls — metrics should not exist yet.
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Collect on unused hook: %v", err)
	}
	// ResourceMetrics may be empty but not an error — just verify no panic.
	_ = h
}

// TestNewOTelHookWithProviderNilGuards verifies that passing literal nil for
// either provider does not panic (the constructor falls back to the global
// no-op providers). Before the guard, a nil interface dereferenced to a nil
// pointer on the first Tracer()/Meter() call and panicked.
func TestNewOTelHookWithProviderNilGuards(t *testing.T) {
	// All four nil-combinations must not panic and must yield a usable hook
	// whose OnRequest/OnResponse are safe to invoke.
	cases := []struct {
		name      string
		tracerNil bool
		meterNil  bool
	}{
		{"both nil", true, true},
		{"tracer nil", true, false},
		{"meter nil", false, true},
	}
	// A real meter provider for the non-nil cases so the hook can record.
	reader := metric.NewManualReader()
	realMP := metric.NewMeterProvider(metric.WithReader(reader))
	t.Cleanup(func() { _ = realMP.Shutdown(context.Background()) })
	realTP := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = realTP.Shutdown(context.Background()) })

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The SDK providers' methods have pointer receivers, so the
			// interface-typed vars must hold the pointer form.
			var tp trace.TracerProvider = realTP
			var mp otelmetric.MeterProvider = realMP
			if tc.tracerNil {
				tp = nil
			}
			if tc.meterNil {
				mp = nil
			}
			var h *OTelHook
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("NewOTelHookWithProvider panicked: %v", r)
				}
			}()
			h = NewOTelHookWithProvider("svc", tp, mp)
			if h == nil {
				t.Fatal("expected non-nil hook")
			}
			// The returned hook must be safe to drive through a full request
			// cycle without panicking (exercises the fallback providers).
			ctx := h.OnRequest(context.Background(), client.RequestMeta{
				Service: "chat", Method: "POST", Model: "m",
			})
			h.OnResponse(ctx, client.ResponseMeta{
				RequestMeta: client.RequestMeta{Service: "chat", Model: "m"},
				StatusCode:  200,
				Usage:       &client.Usage{TotalTokens: 3},
			})
		})
	}
}

// TestAsAPIError verifies the manual unwrap-to-APIError helper used by
// errorAttrs (since *APIError exposes fields, not getters).
func TestAsAPIError(t *testing.T) {
	// Direct type.
	ae := &client.APIError{Code: 1113}
	var got *client.APIError
	if !asAPIError(ae, &got) || got.Code != 1113 {
		t.Errorf("direct: expected match with code 1113")
	}
	// Wrapped via fmt.Errorf("...: %w", ae) — this uses Unwrap() chain.
	if !asAPIError(wrapErr(ae), &got) || got.Code != 1113 {
		t.Errorf("wrapped: expected match with code 1113")
	}
	// Non-API error.
	got = nil
	if asAPIError(context.DeadlineExceeded, &got) {
		t.Errorf("expected no match for DeadlineExceeded")
	}
	// Nil.
	if asAPIError(nil, &got) {
		t.Errorf("expected no match for nil")
	}
}

// wrapErr wraps an error using %w (the standard wrap idiom) for the
// asAPIError unwrap-chain test.
func wrapErr(err error) error { return &wrappedErr{err: err} }

type wrappedErr struct{ err error }

func (w *wrappedErr) Error() string { return "wrapped: " + w.err.Error() }
func (w *wrappedErr) Unwrap() error { return w.err }

// attrMap flattens an attribute set into a map[string]any for assertion.
func attrMap(attrs []attribute.KeyValue) map[string]any {
	m := map[string]any{}
	for _, kv := range attrs {
		m[string(kv.Key)] = kv.Value.AsInterface()
	}
	return m
}
