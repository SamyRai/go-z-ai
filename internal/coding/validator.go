package coding

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// ErrInvalidAPIKey is returned when the API rejects the key (HTTP 401).
var ErrInvalidAPIKey = errors.New("API key is invalid or expired")

// validateTimeout bounds one validation call.
const validateTimeout = 30 * time.Second

// ValidateAPIKey confirms a key is valid for a plan by listing models on the
// plan's coding endpoint, as @z_ai/coding-helper's validateApiKey does. nil
// means valid; ErrInvalidAPIKey means rejected; any other error is a network
// failure or an unexpected response.
func ValidateAPIKey(ctx context.Context, plan, apiKey string) error {
	return validateKeyAt(ctx, Region(plan).CodingBaseURL(), apiKey)
}

// validateKeyAt is ValidateAPIKey against an explicit base URL (tests point
// it at an httptest server).
func validateKeyAt(ctx context.Context, baseURL, apiKey string) error {
	ctx, cancel := context.WithTimeout(ctx, validateTimeout)
	defer cancel()

	c, err := client.NewClient(client.Config{APIKey: apiKey, BaseURL: baseURL, MaxRetries: -1})
	if err != nil {
		return err
	}
	_, err = c.Models().List(ctx)
	if apiErr, ok := errors.AsType[*client.APIError](err); ok && apiErr.HTTPStatus == http.StatusUnauthorized {
		return ErrInvalidAPIKey
	}
	return err
}
