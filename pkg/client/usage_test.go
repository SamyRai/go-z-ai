package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestBalance returns the underlying error unchanged so callers can inspect the
// structured *APIError (business code + HTTP status). It must NOT flatten a
// balance error into a plain string — that dropped the code GetAccountStatus
// needs to classify it (the bug this replaced).
func TestTestBalanceReturnsStructuredAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1113","message":"Insufficient balance or no active resource package"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	err := c.Usage().TestBalance(context.Background())
	if err == nil {
		t.Fatal("expected an error")
	}
	apiErr, ok := errors.AsType[*APIError](err)
	if !ok {
		t.Fatalf("expected a wrapped *APIError, got %T: %v", err, err)
	}
	if apiErr.Code != ErrCodeInsufficientBalance {
		t.Errorf("expected code %d, got %d", ErrCodeInsufficientBalance, apiErr.Code)
	}
	if apiErr.HTTPStatus != http.StatusTooManyRequests {
		t.Errorf("expected HTTP 429, got %d", apiErr.HTTPStatus)
	}
}

// An insufficient-balance error means the key authenticated and reached
// billing, so the API is accessible — there's just no balance. Classified from
// the structured *APIError code (1113), not string-matching. (This previously
// fell through to APIAccessible=false because TestBalance flattened the error.)
func TestGetAccountStatusInsufficientBalance(t *testing.T) {
	for _, tc := range []struct {
		name string
		code string
	}{
		{"1113 insufficient balance", "1113"},
		{"1316 hourly limit no balance", "1316"},
		{"1317 weekly limit no balance", "1317"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"`+tc.code+`","message":"Insufficient balance or no active resource package"}}`)
			}))
			defer srv.Close()

			c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
			status, err := c.Usage().GetAccountStatus(context.Background())
			if err != nil {
				t.Fatalf("GetAccountStatus itself should not error: %v", err)
			}
			if !status.APIAccessible {
				t.Error("expected APIAccessible=true (key works, just no balance)")
			}
			if status.HasBalance {
				t.Error("expected HasBalance=false for an insufficient-balance error")
			}
			const want = "API accessible but insufficient balance - please recharge at https://z.ai"
			if status.Message != want {
				t.Errorf("expected message %q, got %q", want, status.Message)
			}
		})
	}
}

// A plain 429 rate-limit (no balance code) is accessible-but-throttled, not a
// balance problem — distinguished by HTTP status once the balance codes miss.
func TestGetAccountStatusRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1302","message":"API rate limit reached"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	status, err := c.Usage().GetAccountStatus(context.Background())
	if err != nil {
		t.Fatalf("GetAccountStatus itself should not error: %v", err)
	}
	if !status.APIAccessible {
		t.Error("expected APIAccessible=true for a rate-limit response")
	}
	if status.HasBalance {
		t.Error("expected HasBalance=false while rate limited")
	}
	const want = "API accessible but rate limited - try again later"
	if status.Message != want {
		t.Errorf("expected message %q, got %q", want, status.Message)
	}
}

// GetAccountStatus must classify an auth failure (401) distinctly from a
// balance issue — the "401"/"unauthorized" substrings are also buried
// mid-string in the wrapped error.
func TestGetAccountStatusDetectsAuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusUnauthorized, `{"error":{"code":"1000","message":"Authentication Failed, Please check your Authorization Header"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	status, err := c.Usage().GetAccountStatus(context.Background())
	if err != nil {
		t.Fatalf("GetAccountStatus itself should not error: %v", err)
	}
	if status.APIAccessible {
		t.Error("expected APIAccessible=false for a 401 response")
	}
	const want = "API key authentication failed - check your API key"
	if status.Message != want {
		t.Errorf("expected message %q, got %q", want, status.Message)
	}
}
