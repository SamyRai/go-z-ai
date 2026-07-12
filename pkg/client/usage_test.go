package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestBalance wraps the underlying error ("failed to create chat
// completion: [1113] ... (HTTP 429)"), so the "1113"/"Insufficient balance"
// substrings never appear at the start of err.Error(). Regression test for
// the bug where the old contains() helper only checked prefixes: it would
// never match here, so TestBalance would return the raw wrapped error
// instead of the friendly "insufficient balance" message.
func TestTestBalanceDetectsInsufficientBalanceMidString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1113","message":"Insufficient balance or no active resource package"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	err := c.Usage().TestBalance(context.Background())
	if err == nil {
		t.Fatal("expected an error")
	}
	const want = "insufficient balance: please recharge your account at https://z.ai"
	if err.Error() != want {
		t.Fatalf("expected friendly balance message %q, got %q", want, err.Error())
	}
}

// GetAccountStatus's own "429 && (1113 || Insufficient balance)" branch is
// unreachable for this scenario regardless of the contains() fix: TestBalance
// (which GetAccountStatus always calls first) already intercepts exactly
// this error and returns its own clean "insufficient balance: ..." message,
// so the raw "429"/"1113" markers never reach GetAccountStatus's
// classification. This locks in the real (if debatably imperfect —
// APIAccessible ends up false, though the API did respond) current
// behavior; see docs/roadmap.md for the follow-up.
func TestGetAccountStatusInsufficientBalanceViaTestBalanceShortcut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"1113","message":"Insufficient balance or no active resource package"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	status, err := c.Usage().GetAccountStatus(context.Background())
	if err != nil {
		t.Fatalf("GetAccountStatus itself should not error: %v", err)
	}
	const want = "insufficient balance: please recharge your account at https://z.ai"
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
