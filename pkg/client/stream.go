package client

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"strings"
)

// maxSSELine bounds one SSE line (and so the memory a misbehaving server can
// make the scanner hold).
const maxSSELine = 1 << 20

// errStreamStopped is returned through a stream decoder when the consumer
// stops ranging; it never reaches the caller.
var errStreamStopped = errors.New("stream stopped by consumer")

// streamSSE opens r as a server-sent-events stream and exposes the decoded
// values as an iterator for Go's range-over-func:
//
//	for v, err := range seq {
//	    if err != nil { /* terminal — the loop ends */ }
//	}
//
// The stream runs synchronously inside the iterator — there is no producer
// goroutine — so breaking out of the loop simply stops reading and closes the
// body, and cancelling ctx aborts a blocked read (the request is bound to
// ctx). Connect-level transient failures are retried like any request (see
// Client.open); once the stream has begun, a failure is surfaced as the
// terminal error, never retried, since part of the response was consumed.
//
// Hooks: every decoded value fires OnStreamChunk on the connected attempt;
// the attempt ends with OnResponse on a clean end, or OnError on a mid-stream
// failure or an early break (context.Canceled).
func streamSSE[T any](
	ctx context.Context,
	c *Client,
	r apiRequest,
	decode func(ctx context.Context, body io.Reader, emit func(T) error) error,
) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		a, err := c.open(ctx, r)
		if err != nil {
			yield(zero, err)
			return
		}
		defer a.resp.Body.Close()

		stopped := false
		err = decode(ctx, a.resp.Body, func(v T) error {
			c.callHooksStreamChunk(a.ctx, a.meta, v)
			if !yield(v, nil) {
				stopped = true
				return errStreamStopped
			}
			return nil
		})
		switch {
		case stopped:
			a.fail(context.Canceled)
		case err != nil:
			a.fail(err)
			yield(zero, err)
		default:
			a.succeed(nil)
		}
	}
}

// errorIter returns an iterator that yields exactly one error — for failures
// (such as request validation) detected before a stream can be opened.
func errorIter[T any](err error) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		yield(zero, err)
	}
}

// scanSSE reads a text/event-stream body and calls onEvent once per event
// with its event name ("" when the stream sends none) and data, multiple
// data: lines being joined with "\n". A blank line dispatches the pending
// event; comment lines (":...") and id:/retry: fields are ignored; an event
// with neither a name nor data is skipped. A trailing event without its
// terminating blank line is still delivered.
func scanSSE(ctx context.Context, body io.Reader, onEvent func(event, data string) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSSELine)

	var event string
	var data strings.Builder
	hasData := false
	dispatch := func() error {
		if event == "" && !hasData {
			return nil
		}
		ev, d := event, data.String()
		event, hasData = "", false
		data.Reset()
		return onEvent(ev, d)
	}

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := scanner.Text()
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimSpace(value)
		switch {
		case line == "":
			if err := dispatch(); err != nil {
				return err
			}
		case field == "": // comment / keep-alive
		case field == "event":
			event = value
		case field == "data":
			if hasData {
				data.WriteByte('\n')
			}
			data.WriteString(value)
			hasData = true
		}
	}
	if err := scanner.Err(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("stream read error: %w", err)
	}
	return dispatch()
}
