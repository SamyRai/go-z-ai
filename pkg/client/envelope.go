package client

import (
	"context"
	"fmt"
	"net/http"
)

// Envelope is the {code, msg, success, data} wrapper the monitor
// (quota/usage) and biz (account) APIs answer with. Those APIs report
// business failures inside an HTTP 200 body, so every service returning an
// Envelope checks it and turns a failure into an *APIError: an Envelope a
// service hands back always succeeded, and callers only read Data.
type Envelope[T any] struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Success *bool  `json:"success,omitempty"`
	Data    T      `json:"data"`
}

// OK reports whether the envelope carries a success: the success flag when
// present, otherwise a 0 or 200 code (both are used).
func (e *Envelope[T]) OK() bool {
	if e.Success != nil {
		return *e.Success
	}
	return e.Code == 0 || e.Code == http.StatusOK
}

// check returns an *APIError describing a failed envelope, or nil.
func (e *Envelope[T]) check() error {
	if e.OK() {
		return nil
	}
	// The envelope code often mirrors an HTTP status (401, 500); classify by
	// it when it does, while recording that the transport answered 200.
	classifyAs := http.StatusOK
	if e.Code >= 400 && e.Code < 600 {
		classifyAs = e.Code
	}
	apiErr := createAPIError(e.Code, classifyAs, e.Msg)
	apiErr.HTTPStatus = http.StatusOK
	apiErr.IsRetriable = false
	return apiErr
}

// fetchEnvelope performs r and decodes an Envelope[T], returning a business
// failure as an error. what names the resource for error messages.
func fetchEnvelope[T any](ctx context.Context, c *Client, r apiRequest, what string) (*Envelope[T], error) {
	var env Envelope[T]
	if err := c.do(ctx, r, &env); err != nil {
		return nil, fmt.Errorf("failed to get %s: %w", what, err)
	}
	if err := env.check(); err != nil {
		return nil, fmt.Errorf("failed to get %s: %w", what, err)
	}
	return &env, nil
}
