package client

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// maxRetryDelay caps any single backoff, including a server's Retry-After.
const maxRetryDelay = 30 * time.Second

// isRetriable reports whether a failed attempt may be retried: an *APIError
// says so itself (see APIError.IsRetriable); any other error is a transport
// failure where the server never answered, which is safe to retry.
func isRetriable(err error) bool {
	if apiErr, ok := errors.AsType[*APIError](err); ok {
		return apiErr.IsRetriable
	}
	return true
}

// backoff sleeps before a retry, honoring a Retry-After header value when
// present, otherwise exponential backoff with jitter. It returns early when
// ctx is cancelled; the caller's next ctx check then aborts the retry loop.
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
// seconds or an HTTP date) wins when present; otherwise base * 2^attempt with
// up to 25% jitter to avoid thundering herds. The result never exceeds
// maxRetryDelay.
func (c *Client) retryDelay(retryAfter string, attempt int) time.Duration {
	if d, ok := parseRetryAfter(retryAfter); ok {
		return min(d, maxRetryDelay)
	}
	d := c.config.RetryDelay << uint(attempt)
	if d <= 0 || d > maxRetryDelay {
		d = maxRetryDelay
	}
	jitter := time.Duration(rand.Int64N(int64(d)/4 + 1))
	return min(d+jitter, maxRetryDelay)
}

// parseRetryAfter decodes a Retry-After header: delay-seconds or an HTTP-date.
func parseRetryAfter(v string) (time.Duration, bool) {
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second, true
	}
	if at, err := http.ParseTime(v); err == nil {
		return max(time.Until(at), 0), true
	}
	return 0, false
}
