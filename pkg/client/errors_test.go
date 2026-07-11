package client

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestAPIErrorParsing tests the parsing of API error responses
func TestAPIErrorParsing(t *testing.T) {
	tests := []struct {
		name              string
		responseBody      string
		httpStatus        int
		expectedCode      int
		expectedCat       ErrorCategory
		expectedRetriable bool
	}{
		{
			name:              "Authentication error",
			responseBody:      `{"error":{"code":"1001","message":"Authentication parameter not received"}}`,
			httpStatus:        401,
			expectedCode:      1001,
			expectedCat:       ErrorCategoryAuth,
			expectedRetriable: false,
		},
		{
			name:              "Quota exceeded error",
			responseBody:      `{"error":{"code":"1308","message":"Usage limit reached"}}`,
			httpStatus:        429,
			expectedCode:      1308,
			expectedCat:       ErrorCategoryQuota,
			expectedRetriable: false,
		},
		{
			name:              "Rate limit error",
			responseBody:      `{"error":{"code":"1302","message":"Rate limit reached"}}`,
			httpStatus:        429,
			expectedCode:      1302,
			expectedCat:       ErrorCategoryRateLimit,
			expectedRetriable: true,
		},
		{
			name:              "Parameter error",
			responseBody:      `{"error":{"code":"1214","message":"Parameter invalid"}}`,
			httpStatus:        400,
			expectedCode:      1214,
			expectedCat:       ErrorCategoryParameter,
			expectedRetriable: false,
		},
		{
			name:              "Server error",
			responseBody:      `{"error":{"code":"1200","message":"API Call Error"}}`,
			httpStatus:        500,
			expectedCode:      1200,
			expectedCat:       ErrorCategoryServer,
			expectedRetriable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock HTTP response
			resp := &http.Response{
				StatusCode: tt.httpStatus,
				Header:     make(http.Header),
			}

			// Create a read closer with the response body
			resp.Body = io.NopCloser(strings.NewReader(tt.responseBody))

			// Parse the error
			err := parseAPIError(resp)

			// Type assert to APIError
			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("Expected APIError, got %T", err)
			}

			// Verify error properties
			if apiErr.Code != tt.expectedCode {
				t.Errorf("Expected code %d, got %d", tt.expectedCode, apiErr.Code)
			}

			if apiErr.HTTPStatus != tt.httpStatus {
				t.Errorf("Expected HTTP status %d, got %d", tt.httpStatus, apiErr.HTTPStatus)
			}

			if apiErr.Category != tt.expectedCat {
				t.Errorf("Expected category %s, got %s", tt.expectedCat, apiErr.Category)
			}

			if apiErr.IsRetriable != tt.expectedRetriable {
				t.Errorf("Expected retriable %v, got %v", tt.expectedRetriable, apiErr.IsRetriable)
			}
		})
	}
}

// TestAPIErrorHelpers tests the helper methods
func TestAPIErrorHelpers(t *testing.T) {
	tests := []struct {
		name             string
		errorCode        int
		isAuthError      bool
		isQuotaError     bool
		isRateLimitError bool
		isParameterError bool
		isServerError    bool
	}{
		{
			name:        "Authentication error",
			errorCode:   1001,
			isAuthError: true,
		},
		{
			name:         "Quota error",
			errorCode:    1308,
			isQuotaError: true,
		},
		{
			name:             "Rate limit error",
			errorCode:        1302,
			isRateLimitError: true,
		},
		{
			name:             "Parameter error",
			errorCode:        1214,
			isParameterError: true,
		},
		{
			name:          "Server error",
			errorCode:     1200,
			isServerError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := createAPIError(tt.errorCode, 400, "Test error")

			if apiErr.IsAuthError() != tt.isAuthError {
				t.Errorf("Expected IsAuthError()=%v, got %v", tt.isAuthError, apiErr.IsAuthError())
			}

			if apiErr.IsQuotaError() != tt.isQuotaError {
				t.Errorf("Expected IsQuotaError()=%v, got %v", tt.isQuotaError, apiErr.IsQuotaError())
			}

			if apiErr.IsRateLimitError() != tt.isRateLimitError {
				t.Errorf("Expected IsRateLimitError()=%v, got %v", tt.isRateLimitError, apiErr.IsRateLimitError())
			}

			if apiErr.IsParameterError() != tt.isParameterError {
				t.Errorf("Expected IsParameterError()=%v, got %v", tt.isParameterError, apiErr.IsParameterError())
			}

			if apiErr.IsServerError() != tt.isServerError {
				t.Errorf("Expected IsServerError()=%v, got %v", tt.isServerError, apiErr.IsServerError())
			}
		})
	}
}

// TestUnknownErrorCode tests handling of unknown error codes
func TestUnknownErrorCode(t *testing.T) {
	resp := &http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"9999","message":"Unknown error"}}`)),
		Header:     make(http.Header),
	}

	err := parseAPIError(resp)
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}

	// Unknown errors should default to server error category
	if apiErr.Category != ErrorCategoryServer {
		t.Errorf("Expected category %s for unknown error, got %s", ErrorCategoryServer, apiErr.Category)
	}

	// Unknown errors should be retriable
	if !apiErr.IsRetriable {
		t.Error("Expected unknown errors to be retriable")
	}
}

// TestCreateAPIError tests the createAPIError function
func TestCreateAPIError(t *testing.T) {
	apiErr := createAPIError(1308, 429, "Usage limit reached")

	if apiErr == nil {
		t.Fatal("Expected non-nil APIError")
	}

	if apiErr.Code != 1308 {
		t.Errorf("Expected code 1308, got %d", apiErr.Code)
	}

	if apiErr.HTTPStatus != 429 {
		t.Errorf("Expected HTTP status 429, got %d", apiErr.HTTPStatus)
	}

	if apiErr.Message != "Usage limit reached" {
		t.Errorf("Expected message 'Usage limit reached', got '%s'", apiErr.Message)
	}

	if apiErr.Category != ErrorCategoryQuota {
		t.Errorf("Expected category %s, got %s", ErrorCategoryQuota, apiErr.Category)
	}

	// Check that user message is set
	if apiErr.UserMessage == "" {
		t.Error("Expected user message to be set")
	}
}

// TestAPIErrorString tests the Error() method output
func TestAPIErrorString(t *testing.T) {
	apiErr := createAPIError(1308, 429, "Usage limit reached")

	errStr := apiErr.Error()
	if errStr == "" {
		t.Error("Expected non-empty error string")
	}

	// Error string should contain the code
	if !strings.Contains(errStr, "1308") {
		t.Errorf("Expected error string to contain code 1308, got: %s", errStr)
	}
}

// TestLegacyErrorCompatibility tests backward compatibility with legacy error types
func TestLegacyErrorCompatibility(t *testing.T) {
	// Test NewAPIError
	legacyErr := NewAPIError(500, "Internal error")
	if legacyErr.HTTPStatus != 500 {
		t.Errorf("Expected HTTP status 500, got %d", legacyErr.HTTPStatus)
	}

	// Test WrapAPIError
	baseErr := createAPIError(1200, 500, "Base error")
	wrappedErr := WrapAPIError(baseErr, "Wrapped error message")

	if wrappedErr.Message != "Wrapped error message" {
		t.Errorf("Expected message 'Wrapped error message', got '%s'", wrappedErr.Message)
	}

	if wrappedErr.Err == nil {
		t.Error("Expected wrapped error to contain original error")
	}
}
