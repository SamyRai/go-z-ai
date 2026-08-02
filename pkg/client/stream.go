package client

import (
	"context"
	"fmt"
	"io"
	"iter"
	"net/http"
)

// streamItem is one element pushed from the SSE-reader goroutine to the
// iterator consumer. Exactly one of chunk or err is meaningful per item:
//   - a regular chunk: chunk set, err nil
//   - a fatal error during connect or stream read: chunk zero, err set
//   - stream end (SSE [DONE] or clean EOF): the goroutine closes the channel
//
// The error item is terminal — the goroutine closes the channel right after
// sending it — so consumers see at most one error item, last.
type streamItem[T any] struct {
	chunk T
	err   error
}

// pullStream is the shared iterator plumbing for both Chat and Anthropic
// streaming. It runs the connect+stream lifecycle on a goroutine and exposes
// the chunks as an iter.Seq2 the caller ranges over.
//
// connect must perform the retry/backoff loop (mirroring CreateStream's
// outer attempt loop), open the response body on success, and return a
// non-nil resp + nil err once the SSE stream has begun (past the retry
// boundary). A non-nil err from connect is the terminal error and resp is
// ignored.
//
// deliver reads resp.Body to completion, invoking yield(chunk) for each
// parsed SSE event. A non-nil return from deliver is the terminal error.
// pullStream owns resp.Body.Close() regardless of outcome.
//
// Why a goroutine + channel (not a callback-driven pull function): the
// natural Go 1.23+ idiom for adapting a push-style source to iter.Seq2 is a
// producer goroutine feeding a buffered channel, with the range-loop's
// yield function as the backpressure signal. Early consumer exit (a `break`
// out of the range) is signalled to the producer by an internal
// context.WithCancel that pullStream owns and cancels on every exit path —
// iter.Seq2's range-over-func provides no stop callback, so without this the
// producer would deadlock on its next channel send when the consumer walks
// away. The parent ctx is also honored: cancelling it propagates to the
// internal context and tears down both sides.
func pullStream[T any](
	ctx context.Context,
	connect func() (*http.Response, error),
	deliver func(resp *http.Response, yield func(T) error) error,
) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		// pullStream owns an internal cancellable context derived from the
		// caller's ctx, plus a handle to the response body. The consumer
		// cancels the context AND closes the body on EVERY exit path (early
		// break, error, clean end). This is essential because iter.Seq2's
		// range-over-func gives the producer no other signal when a caller
		// `break`s out of the range: the runtime simply stops calling yield.
		// Without this teardown, the producer would block forever — either on
		// its channel send (yieldToChan) or, more commonly, on a blocked body
		// read inside deliver/readSSE. Cancel alone isn't enough: a bufio
		// Scanner mid-Read on the response body is not reliably interrupted by
		// request-context cancellation, so we close the body directly, which
		// causes the Read to return immediately and deliver to return.
		streamCtx, cancel := context.WithCancel(ctx)
		defer cancel() // belt-and-suspenders: cancel on every return path.

		// bodyCh carries the connected response so the consumer can close
		// resp.Body itself on early exit (interrupting a blocked producer
		// read). Buffered (1) so the producer never blocks publishing it. The
		// producer still owns Close on its normal exit path; double-close is
		// safe (http.Response.Body.Close is idempotent).
		bodyCh := make(chan io.ReadCloser, 1)
		closeBody := func() {
			select {
			case b, ok := <-bodyCh:
				if ok && b != nil {
					b.Close()
				}
			default:
			}
		}

		// Buffered channel: capacity 1 gives the producer one chunk of slack
		// so a slow consumer's range loop doesn't block the SSE read on every
		// token (which would inflate tail latency). Larger buffers trade
		// memory for smoother throughput; 1 is the safe default that never
		// drops ordering and never blocks the producer on the first chunk.
		items := make(chan streamItem[T], 1)

		// Producer goroutine: runs connect + deliver, pushes items, closes.
		producerDone := make(chan struct{})
		go func() {
			defer close(producerDone)
			defer close(items)

			resp, err := connect()
			if err != nil {
				items <- streamItem[T]{err: err}
				return
			}
			// Publish the body so the consumer can close it on early exit.
			// The producer's own defer handles Close on its normal exit.
			bodyCh <- resp.Body
			defer resp.Body.Close()

			// yield adapters: the deliver function takes a func(T) error
			// (the existing callback shape, so we reuse readSSE/readAnthropicSSE
			// verbatim). We translate callback-error-abort semantics into
			// channel-push + return.
			yieldToChan := func(chunk T) error {
				select {
				case items <- streamItem[T]{chunk: chunk}:
					return nil
				// The consumer stopped ranging (returned false from its yield,
				// or the stream was cancelled). The consumer cancels streamCtx
				// and closes the body on exit, so this branch fires promptly
				// and the producer unblocks — deliver returns, the goroutine
				// exits.
				case <-streamCtx.Done():
					return streamCtx.Err()
				}
			}
			if derr := deliver(resp, yieldToChan); derr != nil {
				// Push the terminal error if there's still a consumer; if the
				// consumer walked away, streamCtx is already cancelled and the
				// ctx.Done branch fires, letting us exit without blocking.
				select {
				case items <- streamItem[T]{err: derr}:
				case <-streamCtx.Done():
				}
			}
		}()

		// Consumer loop: range over items until channel closes or yield
		// signals stop. We watch streamCtx.Done() so a cancelled parent ctx
		// unblocks the range even if the producer is slow to close.
		for {
			select {
			case <-streamCtx.Done():
				// Parent ctx cancelled. Tear down: cancel + close the body so
				// a producer blocked in a body read unblocks, then wait for it.
				cancel()
				closeBody()
				<-producerDone
				yield(zero[T](), streamCtx.Err())
				return
			case item, ok := <-items:
				if !ok {
					return // clean stream end
				}
				if !yield(item.chunk, item.err) {
					// Consumer returned false from yield (early break/return).
					// Cancel + close the body to unblock a producer that may be
					// blocked either on its next channel send OR on a body read
					// inside deliver, then wait for it to exit cleanly.
					cancel()
					closeBody()
					<-producerDone
					return
				}
			}
		}
	}
}

// zero returns the zero value of T. Used to send a placeholder chunk when
// reporting a ctx-cancellation error (the chunk is ignored by the consumer
// when err is non-nil).
func zero[T any]() T {
	var z T
	return z
}

// --- Chat streaming --------------------------------------------------------

// Stream sends a streaming chat completion, returning an iterator the caller
// ranges over with Go 1.23+'s range-over-func:
//
//	for chunk, err := range c.Chat().Stream(ctx, req) {
//	    if err != nil { /* fatal — stream ends */ break }
//	    if len(chunk.Choices) > 0 {
//	        fmt.Print(chunk.Choices[0].Delta.Content)
//	    }
//	}
//
// Connect-level transient failures (429, 5xx, network errors) are retried up
// to Config.MaxRetries exactly like Create; once a stream has begun, mid-
// stream failures are surfaced as the terminal err item from the iterator
// (the next range step ends the loop).
//
// Context cancellation propagates both ways: cancelling ctx stops the range
// loop and tears down the in-flight SSE read.
//
// This is the recommended streaming API. The older callback-based
// CreateStream is deprecated and delegates here.
func (s *ChatService) Stream(ctx context.Context, req ChatRequest) iter.Seq2[StreamChunk, error] {
	if err := validateChatRequest(&req); err != nil {
		return singletonErrIter[StreamChunk](fmt.Errorf("invalid chat request: %w", err))
	}
	req.Stream = true
	req.Tools = s.compatTools(req.Tools)

	// Stamp service + model so hooks (OnRequest, OnStreamChunk) can attribute
	// the stream. The connect path reads these via buildRequestMeta.
	streamCtx := WithModel(WithService(ctx, "chat"), req.Model)

	inner := pullStream[StreamChunk](streamCtx,
		func() (*http.Response, error) {
			return s.connectChatStream(streamCtx, req)
		},
		func(resp *http.Response, yield func(StreamChunk) error) error {
			return s.readSSE(streamCtx, resp, yield)
		},
	)

	// Wrap the iterator so each chunk handed to the caller also fires
	// OnStreamChunk. pullStream still owns the connect/stream lifecycle;
	// we only observe chunk delivery.
	return func(yield func(StreamChunk, error) bool) {
		meta := s.client.buildRequestMeta(streamCtx, "POST", "/chat/completions", 0)
		for chunk, err := range inner {
			if err != nil {
				s.client.callHooksError(streamCtx, meta, err)
				if !yield(chunk, err) {
					return
				}
				return
			}
			s.client.callHooksStreamChunk(streamCtx, meta, chunk)
			if !yield(chunk, nil) {
				return
			}
		}
	}
}

// connectChatStream runs the outer retry/backoff loop that CreateStream used
// to inline. Returns a non-nil *http.Response once a 200 stream has begun;
// any other outcome is a terminal error. Pulled out so both Stream and
// CreateStream share the identical connect logic.
func (s *ChatService) connectChatStream(ctx context.Context, req ChatRequest) (*http.Response, error) {
	maxRetries := s.client.config.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// OnRequest fires per attempt; the hook may return a derived ctx
		// (e.g. child span) for the actual send.
		reqMeta := s.client.buildRequestMeta(ctx, "POST", "/chat/completions", attempt)
		hookCtx := s.client.callHooksRequest(ctx, reqMeta)
		resp, err := s.client.send(hookCtx, s.client.config.BaseURL, s.client.config.APIKey, "POST", "/chat/completions", req)
		if err != nil {
			lastErr = fmt.Errorf("failed to execute request: %w", err)
			if attempt < maxRetries {
				s.client.backoff(hookCtx, "", attempt)
				continue
			}
			return nil, lastErr
		}
		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}
		retryAfter := resp.Header.Get("Retry-After")
		apiErr := parseAPIError(resp)
		resp.Body.Close()
		retriable := false
		if ae, ok := apiErr.(*APIError); ok {
			retriable = ae.IsRetriable
		}
		lastErr = apiErr
		if attempt < maxRetries && retriable {
			s.client.backoff(hookCtx, retryAfter, attempt)
			continue
		}
		return nil, apiErr
	}
	return nil, lastErr
}

// --- Anthropic streaming ---------------------------------------------------

// Stream sends a streaming POST /v1/messages request, returning an iterator
// over Anthropic's raw SSE events. See ChatService.Stream for the iterator
// semantics; the per-event shape (AnthropicStreamEvent with Type + raw JSON
// Data) is identical to the callback-based CreateStream.
//
// This is the recommended streaming API. The older callback-based
// AnthropicService.CreateStream is deprecated and delegates here.
func (s *AnthropicService) Stream(ctx context.Context, req AnthropicMessageRequest) iter.Seq2[AnthropicStreamEvent, error] {
	if err := validateAnthropicRequest(&req); err != nil {
		return singletonErrIter[AnthropicStreamEvent](fmt.Errorf("invalid anthropic request: %w", err))
	}
	req.Stream = true
	req.Tools = s.compatTools(req.Tools)

	streamCtx := WithModel(WithService(ctx, "anthropic"), req.Model)

	inner := pullStream[AnthropicStreamEvent](streamCtx,
		func() (*http.Response, error) {
			return s.connectAnthropicStream(streamCtx, req)
		},
		func(resp *http.Response, yield func(AnthropicStreamEvent) error) error {
			return readAnthropicSSE(streamCtx, resp, yield)
		},
	)

	return func(yield func(AnthropicStreamEvent, error) bool) {
		meta := s.client.buildRequestMeta(streamCtx, "POST", anthropicMessagesEndpoint, 0)
		for ev, err := range inner {
			if err != nil {
				s.client.callHooksError(streamCtx, meta, err)
				if !yield(ev, err) {
					return
				}
				return
			}
			s.client.callHooksStreamChunk(streamCtx, meta, ev)
			if !yield(ev, nil) {
				return
			}
		}
	}
}

// connectAnthropicStream mirrors connectChatStream for the Anthropic surface.
func (s *AnthropicService) connectAnthropicStream(ctx context.Context, req AnthropicMessageRequest) (*http.Response, error) {
	maxRetries := s.client.config.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		reqMeta := s.client.buildRequestMeta(ctx, "POST", anthropicMessagesEndpoint, attempt)
		hookCtx := s.client.callHooksRequest(ctx, reqMeta)
		resp, err := s.client.sendHeaders(hookCtx, AnthropicBaseURL, s.client.config.APIKey, "POST", anthropicMessagesEndpoint, req, anthropicHeaders())
		if err != nil {
			lastErr = fmt.Errorf("failed to execute request: %w", err)
			if attempt < maxRetries {
				s.client.backoff(hookCtx, "", attempt)
				continue
			}
			return nil, lastErr
		}
		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}
		retryAfter := resp.Header.Get("Retry-After")
		apiErr := parseAPIError(resp)
		resp.Body.Close()
		retriable := false
		if ae, ok := apiErr.(*APIError); ok {
			retriable = ae.IsRetriable
		}
		lastErr = apiErr
		if attempt < maxRetries && retriable {
			s.client.backoff(hookCtx, retryAfter, attempt)
			continue
		}
		return nil, apiErr
	}
	return nil, lastErr
}

// singletonErrIter returns an iterator that yields exactly one error and
// stops. Used when request validation fails before we can even open a
// connection — we still want Stream to return an iterator (not a separate
// error return), so the failure surfaces as the iterator's terminal err.
func singletonErrIter[T any](err error) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		yield(zero[T](), err)
	}
}
