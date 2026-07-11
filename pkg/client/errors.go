package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// API error code constants from Z.ai API
// Reference: https://docs.z.ai/devpack/overview
const (
	// Authentication Errors (HTTP 401)
	ErrCodeAuthFailed       = 1000 // Authentication Failed
	ErrCodeAuthNotFound     = 1001 // Auth parameter not received
	ErrCodeAuthTokenExpired = 1003 // Auth Token expired
	ErrCodeAuthNeed2FA      = 1005 // Need Two-Factor Authentication

	// Rate Limit/Quota Errors (HTTP 429)
	ErrCodeInsufficientBalance    = 1113 // Insufficient balance or no resource package
	ErrCodeRateLimitReached       = 1302 // Rate limit reached for requests
	ErrCodeServiceOverloaded      = 1305 // Service temporarily overloaded
	ErrCodeUsageLimitReached      = 1308 // Usage limit reached for {number} {unit}
	ErrCodeCodingPlanExpired      = 1309 // GLM Coding Plan package expired
	ErrCodeWeeklyMonthlyExhausted = 1310 // Weekly/Monthly Limit Exhausted
	ErrCodeModelNotIncluded       = 1311 // Subscription doesn't include this model
	ErrCodeFairUsageViolation     = 1313 // Fair Usage Policy violation
	ErrCodeEnterpriseExpired      = 1314 // Enterprise package expired
	ErrCodeEnterpriseKeyOnly      = 1315 // API Key limited to enterprise scenarios
	ErrCodeHourlyLimitNoBalance   = 1316 // 5-hour limit reached, no balance for extra
	ErrCodeWeeklyLimitNoBalance   = 1317 // 7-day limit reached, no balance for extra
	ErrCodeHourlyLimitNoSpend     = 1318 // 5-hour limit reached, no monthly spend
	ErrCodeWeeklyLimitNoSpend     = 1319 // 7-day limit reached, no monthly spend
	ErrCodeHourlyLimitSpendCap    = 1320 // 5-hour limit reached, monthly spend cap
	ErrCodeWeeklyLimitSpendCap    = 1321 // 7-day limit reached, monthly spend cap

	// Parameter Errors (HTTP 400)
	ErrCodeInvalidParameter   = 1210 // Invalid API parameter
	ErrCodeUnknownModel       = 1211 // Unknown Model
	ErrCodeMethodNotSupported = 1212 // Model doesn't support method
	ErrCodeParameterMissing   = 1213 // Parameter not received
	ErrCodeParameterInvalid   = 1214 // Parameter invalid
	ErrCodeParametersConflict = 1215 // Parameters conflict
	ErrCodeAPITakenOffline    = 1221 // API taken offline
	ErrCodeAPINotExist        = 1222 // API doesn't exist
	ErrCodePromptTooLong      = 1261 // Prompt too long
	ErrCodeUnsafeContent      = 1301 // Unsafe/sensitive content detected

	// Permission Errors (HTTP 403)
	ErrCodeNoPermission = 1220 // No permission to access

	// Server Errors (HTTP 500)
	ErrCodeInternalError = -1   // Internal Error
	ErrCodeAPICallError  = 1200 // API Call Error
	ErrCodeProcessError  = 1230 // API call process error
	ErrCodeNetworkError  = 1234 // Network error
)

// Error categories for easier error handling
type ErrorCategory string

const (
	ErrorCategoryAuth       ErrorCategory = "authentication"
	ErrorCategoryRateLimit  ErrorCategory = "rate_limit"
	ErrorCategoryParameter  ErrorCategory = "parameter"
	ErrorCategoryPermission ErrorCategory = "permission"
	ErrorCategoryServer     ErrorCategory = "server"
	ErrorCategoryContent    ErrorCategory = "content"
	ErrorCategoryQuota      ErrorCategory = "quota"
)

// APIError represents a structured error from the Z.ai API
type APIError struct {
	HTTPStatus  int           `json:"http_status"`  // HTTP status code
	Code        int           `json:"code"`         // Business error code
	Message     string        `json:"message"`      // Error message
	Category    ErrorCategory `json:"category"`     // Error category
	UserMessage string        `json:"user_message"` // User-friendly message
	IsRetriable bool          `json:"retriable"`    // Can this error be retried
	RequestID   string        `json:"request_id"`   // Request ID for support
	Err         error         `json:"-"`            // Wrapped error for internal use
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	if e.UserMessage != "" {
		return fmt.Sprintf("[%d] %s (HTTP %d)", e.Code, e.UserMessage, e.HTTPStatus)
	}
	if e.Code > 0 {
		return fmt.Sprintf("[%d] %s (HTTP %d)", e.Code, e.Message, e.HTTPStatus)
	}
	return e.Message
}

// IsAuthError reports whether this is an authentication error
func (e *APIError) IsAuthError() bool {
	return e.Category == ErrorCategoryAuth
}

// IsRateLimitError reports whether this is a rate limiting error
func (e *APIError) IsRateLimitError() bool {
	return e.Category == ErrorCategoryRateLimit
}

// IsQuotaError reports whether this is a quota/usage limit error
func (e *APIError) IsQuotaError() bool {
	return e.Category == ErrorCategoryQuota
}

// IsParameterError reports whether this is a parameter/validation error
func (e *APIError) IsParameterError() bool {
	return e.Category == ErrorCategoryParameter
}

// IsServerError reports whether this is a server-side error
func (e *APIError) IsServerError() bool {
	return e.Category == ErrorCategoryServer
}

// ErrorResponse represents the error response structure from Z.ai API
type ErrorResponse struct {
	Error APIErrorDetail `json:"error"`
}

// APIErrorDetail represents the error detail structure
type APIErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// errorConfig provides metadata for each error code
type errorConfig struct {
	category    ErrorCategory
	userMessage string
	retriable   bool
}

// errorMapping maps error codes to their configurations
var errorMapping = map[int]errorConfig{
	// Authentication errors
	ErrCodeAuthFailed:       {ErrorCategoryAuth, "Authentication failed. Please check your API key.", false},
	ErrCodeAuthNotFound:     {ErrorCategoryAuth, "Authentication credentials not found. Please provide a valid API key.", false},
	ErrCodeAuthTokenExpired: {ErrorCategoryAuth, "Authentication token has expired. Please regenerate your API key.", false},
	ErrCodeAuthNeed2FA:      {ErrorCategoryAuth, "Two-factor authentication is required.", false},

	// Rate limit/quota errors
	ErrCodeInsufficientBalance:    {ErrorCategoryQuota, "Insufficient balance or no active resource package. Please recharge.", false},
	ErrCodeRateLimitReached:       {ErrorCategoryRateLimit, "Rate limit reached. Please slow down your requests.", true},
	ErrCodeServiceOverloaded:      {ErrorCategoryServer, "Service is temporarily overloaded. Please try again later.", true},
	ErrCodeUsageLimitReached:      {ErrorCategoryQuota, "Usage limit reached. Please wait for the quota window to reset.", false},
	ErrCodeCodingPlanExpired:      {ErrorCategoryQuota, "Your GLM Coding Plan has expired. Please renew at https://z.ai/subscribe", false},
	ErrCodeWeeklyMonthlyExhausted: {ErrorCategoryQuota, "Weekly or monthly quota exhausted. Please wait for the reset period.", false},
	ErrCodeModelNotIncluded:       {ErrorCategoryQuota, "Your subscription plan doesn't include access to this model.", false},
	ErrCodeFairUsageViolation:     {ErrorCategoryQuota, "Account usage pattern doesn't comply with Fair Usage Policy. Access has been limited.", false},
	ErrCodeEnterpriseExpired:      {ErrorCategoryQuota, "Your enterprise package has expired. Please contact your administrator.", false},
	ErrCodeEnterpriseKeyOnly:      {ErrorCategoryQuota, "This API key is limited to enterprise coding scenarios. Please use the correct API key type.", false},
	ErrCodeHourlyLimitNoBalance:   {ErrorCategoryQuota, "5-hour usage limit reached. Insufficient balance for extra usage.", false},
	ErrCodeWeeklyLimitNoBalance:   {ErrorCategoryQuota, "7-day usage limit reached. Insufficient balance for extra usage.", false},
	ErrCodeHourlyLimitNoSpend:     {ErrorCategoryQuota, "5-hour usage limit reached. Extra usage unavailable due to monthly spend limit.", false},
	ErrCodeWeeklyLimitNoSpend:     {ErrorCategoryQuota, "7-day usage limit reached. Extra usage unavailable due to monthly spend limit.", false},
	ErrCodeHourlyLimitSpendCap:    {ErrorCategoryQuota, "5-hour usage limit reached. Monthly spend cap reached.", false},
	ErrCodeWeeklyLimitSpendCap:    {ErrorCategoryQuota, "7-day usage limit reached. Monthly spend cap reached.", false},

	// Parameter errors
	ErrCodeInvalidParameter:   {ErrorCategoryParameter, "Invalid API parameter. Please check the API documentation.", false},
	ErrCodeUnknownModel:       {ErrorCategoryParameter, "Unknown or unsupported model. Please check available models.", false},
	ErrCodeMethodNotSupported: {ErrorCategoryParameter, "The current model doesn't support this API method.", false},
	ErrCodeParameterMissing:   {ErrorCategoryParameter, "Required parameter not provided. Please check the API documentation.", false},
	ErrCodeParameterInvalid:   {ErrorCategoryParameter, "Invalid parameter value. Please check the API documentation.", false},
	ErrCodeParametersConflict: {ErrorCategoryParameter, "Conflicting parameters provided. Please check the API documentation.", false},
	ErrCodeAPITakenOffline:    {ErrorCategoryParameter, "This API has been taken offline.", false},
	ErrCodeAPINotExist:        {ErrorCategoryParameter, "This API endpoint doesn't exist.", false},
	ErrCodePromptTooLong:      {ErrorCategoryParameter, "Prompt text is too long. Please shorten your input.", false},
	ErrCodeUnsafeContent:      {ErrorCategoryContent, "Content flagged as potentially unsafe or sensitive. Please modify your input.", false},

	// Permission errors
	ErrCodeNoPermission: {ErrorCategoryPermission, "You don't have permission to access this resource.", false},

	// Server errors
	ErrCodeInternalError: {ErrorCategoryServer, "Internal server error. Please try again later.", true},
	ErrCodeAPICallError:  {ErrorCategoryServer, "API call failed. Please try again later.", true},
	ErrCodeProcessError:  {ErrorCategoryServer, "API request processing error. Please try again later.", true},
	ErrCodeNetworkError:  {ErrorCategoryServer, "Network error. Please check your connection and try again.", true},
}

// createAPIError creates a structured APIError from an error code and HTTP status
func createAPIError(code int, httpStatus int, message string) *APIError {
	config, exists := errorMapping[code]
	if !exists {
		// Unknown error code - use server error as default
		config = errorConfig{
			category:    ErrorCategoryServer,
			userMessage: "Unexpected error occurred. Please try again later.",
			retriable:   true,
		}
	}

	return &APIError{
		HTTPStatus:  httpStatus,
		Code:        code,
		Message:     message,
		Category:    config.category,
		UserMessage: config.userMessage,
		IsRetriable: config.retriable,
	}
}

// parseAPIError parses an error response from the Z.ai API
func parseAPIError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return createAPIError(ErrCodeNetworkError, resp.StatusCode, "Failed to read error response")
	}

	var errorResp ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err != nil {
		// If we can't parse the error response, return a generic error
		return createAPIError(ErrCodeInternalError, resp.StatusCode, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)))
	}

	// Parse the error code from string to int
	var code int
	if _, err := fmt.Sscanf(errorResp.Error.Code, "%d", &code); err != nil {
		code = ErrCodeInternalError
	}

	return createAPIError(code, resp.StatusCode, errorResp.Error.Message)
}

// Legacy error types for backward compatibility
var (
	ErrInvalidAPIKey      = fmt.Errorf("invalid API key")
	ErrMissingAPIKey      = fmt.Errorf("API key is required")
	ErrInvalidModel       = fmt.Errorf("invalid model")
	ErrInvalidRequest     = fmt.Errorf("invalid request")
	ErrRateLimitExceeded  = fmt.Errorf("rate limit exceeded")
	ErrQuotaExceeded      = fmt.Errorf("quota exceeded")
	ErrUnauthorized       = fmt.Errorf("unauthorized access")
	ErrNetworkError       = fmt.Errorf("network error")
	ErrInvalidResponse    = fmt.Errorf("invalid response")
	ErrServiceUnavailable = fmt.Errorf("service unavailable")
)

// NewAPIError creates a new API error (legacy compatibility)
func NewAPIError(statusCode int, message string) *APIError {
	return &APIError{
		HTTPStatus:  statusCode,
		Message:     message,
		Category:    ErrorCategoryServer,
		IsRetriable: statusCode >= 500,
	}
}

// WrapAPIError wraps an existing error with API context (legacy compatibility)
func WrapAPIError(err error, message string) *APIError {
	return &APIError{
		Message:     message,
		Err:         err,
		Category:    ErrorCategoryServer,
		IsRetriable: true,
	}
}
