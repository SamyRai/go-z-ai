package coding

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrInvalidAPIKey is returned when the API rejects the key (HTTP 401).
var ErrInvalidAPIKey = errors.New("API key is invalid or expired")

// validateTimeout bounds a validation call regardless of hc's own Timeout
// (which validateKeyAt must never mutate — see its doc comment).
const validateTimeout = 30 * time.Second

// ValidateAPIKey confirms a key is valid for a plan by calling the models list
// endpoint, exactly as @z_ai/coding-helper's validateApiKey does. nil means
// valid; ErrInvalidAPIKey means rejected; other errors are network/transport
// failures or unexpected HTTP status.
func ValidateAPIKey(ctx context.Context, plan, apiKey string) error {
	return ValidateAPIKeyWith(ctx, plan, apiKey, http.DefaultClient)
}

// ValidateAPIKeyWith is ValidateAPIKey with a custom HTTP client (for tests).
func ValidateAPIKeyWith(ctx context.Context, plan, apiKey string, hc *http.Client) error {
	return validateKeyAt(ctx, CodingBaseURL(plan), apiKey, hc)
}

// validateKeyAt hits <baseURL>/models with the bearer key and classifies the
// response. Split out so tests can point at an httptest server.
//
// It never mutates hc (in particular never sets hc.Timeout): ValidateAPIKey
// passes http.DefaultClient, a shared package-level global, and previously
// this function set DefaultClient.Timeout on first use — a data race under
// concurrent callers (this runs inside a bubbletea tea.Cmd goroutine in the
// TUI) and a surprising process-wide side effect on every other caller of
// http.DefaultClient. The 30s bound is applied via context instead, which
// composes with whatever timeout hc already carries rather than overwriting it.
func validateKeyAt(ctx context.Context, baseURL, apiKey string, hc *http.Client) error {
	ctx, cancel := context.WithTimeout(ctx, validateTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return ErrInvalidAPIKey
	case http.StatusOK:
		return nil
	default:
		return fmt.Errorf("unexpected response: HTTP %d", resp.StatusCode)
	}
}
