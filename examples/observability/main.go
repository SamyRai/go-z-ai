// Command observability demonstrates wiring an OpenTelemetry hook onto the
// go-z-ai client, so every API call emits a span carrying the GenAI
// semantic-convention attributes (gen_ai.request.model, gen_ai.system=z.ai,
// gen_ai.usage.input_tokens, etc.) plus metrics (duration, request count,
// token usage) when a MeterProvider is registered. This example registers
// only a TracerProvider, so spans are emitted to stdout (swap the exporter
// for OTLP/Jaeger/Tempo in production); metrics land in the no-op meter
// until you register a MeterProvider alongside it.
//
// Usage:
//
//	export ZAI_API_KEY=your_api_key_here
//	go run ./examples/observability "Explain goroutines in one paragraph"
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/SamyRai/go-z-ai/pkg/observe"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	prompt := "Explain goroutines in one paragraph"
	if len(os.Args) > 1 {
		prompt = os.Args[1]
	}

	// Configure a global TracerProvider with a stdout exporter. In production,
	// swap stdouttrace for otlptracehttp/otlptracegrpc pointing at your
	// collector (Tempo, Jaeger, Honeycomb, etc.). Call Shutdown on exit to
	// flush buffered spans.
	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatalf("stdout exporter: %v", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(ctx) // flush
	}()

	// Wire the OTelHook onto the client. Every Chat/Anthropic/Embeddings call
	// now emits a span with gen_ai.* attributes and metrics, with no further
	// per-call code. The hook reads Config.Hooks once at construction.
	c, err := client.NewClient(client.Config{
		APIKey: os.Getenv("ZAI_API_KEY"),
		Hooks:  []client.Hook{observe.NewOTelHook("go-z-ai-example")},
	})
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	ctx := context.Background()
	for chunk, err := range c.Chat().Stream(ctx, client.ChatRequest{
		Model:    "glm-5.2",
		Messages: []client.Message{{Role: "user", Content: prompt}},
		TopP:     0.95,
	}) {
		if err != nil {
			log.Fatalf("stream: %v", err)
		}
		if len(chunk.Choices) > 0 {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}
	fmt.Println()
}
