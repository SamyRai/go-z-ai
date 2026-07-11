package coding

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ErrInvalidAPIKey is returned when the API rejects the key (HTTP 401).
var ErrInvalidAPIKey = errors.New("API key is invalid or expired")

// ValidateAPIKey confirms a key is valid for a plan by calling the models list
// endpoint, exactly as @z_ai/coding-helper's validateApiKey does. nil means
// valid; ErrInvalidAPIKey means rejected; other errors are network/transport
// failures or unexpected HTTP status.
func ValidateAPIKey(plan, apiKey string) error {
	return ValidateAPIKeyWith(plan, apiKey, http.DefaultClient)
}

// ValidateAPIKeyWith is ValidateAPIKey with a custom HTTP client (for tests).
func ValidateAPIKeyWith(plan, apiKey string, hc *http.Client) error {
	return validateKeyAt(CodingBaseURL(plan), apiKey, hc)
}

// validateKeyAt hits <baseURL>/models with the bearer key and classifies the
// response. Split out so tests can point at an httptest server.
func validateKeyAt(baseURL, apiKey string, hc *http.Client) error {
	if hc.Timeout == 0 {
		hc.Timeout = 30 * time.Second
	}

	req, err := http.NewRequest(http.MethodGet, baseURL+"/models", nil)
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
