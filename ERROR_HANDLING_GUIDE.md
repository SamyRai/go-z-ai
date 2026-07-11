# Z.ai API Error Handling System

## Overview

The Zai client now includes a comprehensive error handling system that maps Z.ai API error codes to structured, user-friendly error messages with proper categorization and retry logic.

## Error Categories

All errors are categorized into logical groups for easier handling:

- **`authentication`**: Authentication and authorization failures
- **`rate_limit`**: Rate limiting and temporary throttling  
- **`quota`**: Usage limits, quota exhaustion, subscription issues
- **`parameter`**: Invalid parameters, model errors, validation failures
- **`permission`**: Access permission issues
- **`server`**: Server-side errors and network issues
- **`content`**: Content safety and policy violations

## API Error Codes

### Authentication Errors (HTTP 401)

| Code | Constant | Description | Retriable |
|------|----------|-------------|-----------|
| 1000 | `ErrCodeAuthFailed` | Authentication Failed | No |
| 1001 | `ErrCodeAuthNotFound` | Auth parameter not received | No |
| 1003 | `ErrCodeAuthTokenExpired` | Auth Token expired | No |
| 1005 | `ErrCodeAuthNeed2FA` | Need Two-Factor Authentication | No |

### Rate Limit & Quota Errors (HTTP 429)

| Code | Constant | Description | Retriable |
|------|----------|-------------|-----------|
| 1113 | `ErrCodeInsufficientBalance` | Insufficient balance | No |
| 1302 | `ErrCodeRateLimitReached` | Rate limit reached | Yes |
| 1305 | `ErrCodeServiceOverloaded` | Service overloaded | Yes |
| 1308 | `ErrCodeUsageLimitReached` | Usage limit reached | No |
| 1309 | `ErrCodeCodingPlanExpired` | GLM Coding Plan expired | No |
| 1310 | `ErrCodeWeeklyMonthlyExhausted` | Weekly/Monthly limit exhausted | No |
| 1311 | `ErrCodeModelNotIncluded` | Subscription doesn't include model | No |
| 1313 | `ErrCodeFairUsageViolation` | Fair Usage Policy violation | No |
| 1314 | `ErrCodeEnterpriseExpired` | Enterprise package expired | No |
| 1315 | `ErrCodeEnterpriseKeyOnly` | API Key limited to enterprise | No |
| 1316-1321 | Various | Usage limit variants | No |

### Parameter Errors (HTTP 400)

| Code | Constant | Description | Retriable |
|------|----------|-------------|-----------|
| 1210 | `ErrCodeInvalidParameter` | Invalid API parameter | No |
| 1211 | `ErrCodeUnknownModel` | Unknown Model | No |
| 1212 | `ErrCodeMethodNotSupported` | Model doesn't support method | No |
| 1213 | `ErrCodeParameterMissing` | Required parameter not provided | No |
| 1214 | `ErrCodeParameterInvalid` | Invalid parameter value | No |
| 1215 | `ErrCodeParametersConflict` | Conflicting parameters | No |
| 1221 | `ErrCodeAPITakenOffline` | API taken offline | No |
| 1222 | `ErrCodeAPINotExist` | API doesn't exist | No |
| 1261 | `ErrCodePromptTooLong` | Prompt too long | No |
| 1301 | `ErrCodeUnsafeContent` | Unsafe content detected | No |

### Permission Errors (HTTP 403)

| Code | Constant | Description | Retriable |
|------|----------|-------------|-----------|
| 1220 | `ErrCodeNoPermission` | No permission to access | No |

### Server Errors (HTTP 500)

| Code | Constant | Description | Retriable |
|------|----------|-------------|-----------|
| -1 | `ErrCodeInternalError` | Internal Error | Yes |
| 1200 | `ErrCodeAPICallError` | API Call Error | Yes |
| 1230 | `ErrCodeProcessError` | API call process error | Yes |
| 1234 | `ErrCodeNetworkError` | Network error | Yes |

## Usage Examples

### Basic Error Handling

```go
client, _ := client.NewClient(config)
resp, err := client.Chat().Create(request)

if err != nil {
    if apiErr, ok := err.(*client.APIError); ok {
        fmt.Printf("Error %d: %s\n", apiErr.Code, apiErr.UserMessage)
        
        // Handle different error categories
        switch apiErr.Category {
        case client.ErrorCategoryAuth:
            // Handle authentication errors
            fmt.Println("Please check your API key")
        case client.ErrorCategoryQuota:
            // Handle quota errors
            fmt.Println("Please check your subscription")
        case client.ErrorCategoryRateLimit:
            // Handle rate limiting with retry
            if apiErr.IsRetriable {
                time.Sleep(time.Second * 5)
                // Retry the request
            }
        }
    }
}
```

### Error Type Checking

```go
if apiErr.IsAuthError() {
    // Handle authentication errors
    log.Println("Authentication failed")
} else if apiErr.IsQuotaError() {
    // Handle quota errors
    log.Println("Quota exceeded")
} else if apiErr.IsRateLimitError() {
    // Handle rate limiting
    log.Println("Rate limited")
} else if apiErr.IsServerError() {
    // Handle server errors (often retriable)
    if apiErr.IsRetriable {
        // Implement retry logic
    }
}
```

### Retry Logic

```go
func makeRequestWithRetry(client *client.Client, request interface{}) (*client.ChatResponse, error) {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        resp, err := client.Chat().Create(request)
        if err == nil {
            return resp, nil
        }
        
        apiErr, ok := err.(*client.APIError)
        if !ok || !apiErr.IsRetriable {
            return nil, err // Don't retry non-retriable errors
        }
        
        // Wait before retry
        time.Sleep(time.Second * time.Duration(i+1) * 2)
    }
    
    return nil, fmt.Errorf("max retries exceeded")
}
```

## Error Response Structure

The Z.ai API returns errors in this format:

```json
{
  "error": {
    "code": "1214",
    "message": "Parameter `${field}` is invalid. Please check the documentation."
  }
}
```

The client automatically parses this and creates a structured `APIError` with:

- `HTTPStatus`: The HTTP status code
- `Code`: The business error code as integer
- `Message`: The original API error message
- `Category`: The error category for filtering
- `UserMessage`: A user-friendly description
- `IsRetriable`: Whether the error can be retried
- `RequestID`: Request ID for support (if available)

## Integration with Existing Code

### Updating Error Handling

**Before:**
```go
resp, err := client.Chat().Create(request)
if err != nil {
    log.Fatalf("Request failed: %v", err)
}
```

**After:**
```go
resp, err := client.Chat().Create(request)
if err != nil {
    if apiErr, ok := err.(*client.APIError); ok {
        switch apiErr.Category {
        case client.ErrorCategoryAuth:
            log.Fatal("Authentication error - check API key")
        case client.ErrorCategoryQuota:
            log.Fatal("Quota error - check subscription")
        case client.ErrorCategoryRateLimit:
            log.Println("Rate limited - will retry")
            // Implement retry logic
        default:
            log.Fatalf("Request failed: %v", apiErr)
        }
    } else {
        log.Fatalf("Request failed: %v", err)
    }
}
```

## Helper Methods

The `APIError` type provides several helper methods:

```go
// Type checking methods
apiErr.IsAuthError()        // Authentication error?
apiErr.IsRateLimitError()   // Rate limit error?
apiErr.IsQuotaError()       // Quota/usage error?
apiErr.IsParameterError()   // Parameter validation error?
apiErr.IsServerError()     // Server-side error?

// Error details
apiErr.Error()              // Full error string
apiErr.Code                 // Error code (int)
apiErr.HTTPStatus           // HTTP status code
apiErr.Category             // Error category
apiErr.UserMessage          // User-friendly message
apiErr.IsRetriable          // Can retry?
```

## Adding New Error Codes

When Z.ai adds new error codes, add them to the constants:

```go
// In pkg/client/errors.go
const (
    // ... existing constants ...
    
    // New error codes
    ErrCodeNewErrorType = 2000 // New error type description
)
```

Then add the error mapping:

```go
var errorMapping = map[int]errorConfig{
    // ... existing mappings ...
    
    // New error mapping
    ErrCodeNewErrorType: {
        category:    ErrorCategoryParameter,
        userMessage: "User-friendly description of the new error",
        retriable:   false,
    },
}
```

## Testing Error Handling

### Unit Tests

```go
func TestErrorParsing(t *testing.T) {
    // Mock HTTP response with error
    mockResp := &http.Response{
        StatusCode: 401,
        Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"1001","message":"Auth failed"}}`)),
    }
    
    err := parseAPIError(mockResp)
    apiErr, ok := err.(*APIError)
    
    assert.True(t, ok)
    assert.Equal(t, 1001, apiErr.Code)
    assert.Equal(t, 401, apiErr.HTTPStatus)
    assert.Equal(t, ErrorCategoryAuth, apiErr.Category)
    assert.True(t, apiErr.IsAuthError())
    assert.False(t, apiErr.IsRetriable)
}
```

### Integration Testing

```go
func TestAPIErrors(t *testing.T) {
    // Test with invalid API key
    client, _ := NewClient(Config{APIKey: "invalid_key"})
    
    _, err := client.Models().List()
    assert.Error(t, err)
    
    if apiErr, ok := err.(*APIError); ok {
        assert.True(t, apiErr.IsAuthError())
    }
}
```

## Error Handling Best Practices

1. **Always check error categories**: Use `Is*Error()` methods rather than checking error codes directly
2. **Implement retry logic**: Use `IsRetriable` for automatic retries on server errors
3. **Provide user feedback**: Use `UserMessage` for display to end users
4. **Log technical details**: Log `Code` and `Message` for debugging
5. **Handle gracefully**: Don't crash on quota/rate limit errors - inform users

## Troubleshooting

### Common Issues

**"Authentication failed" errors:**
- Check API key validity
- Verify API key has correct permissions
- Ensure API key hasn't expired

**"Quota exceeded" errors:**
- Check quota status with `zai-client accounts quota`
- Consider upgrading subscription
- Wait for quota window reset

**"Rate limit" errors:**
- Implement exponential backoff
- Reduce request frequency
- Use retry logic for retriable errors

**"Parameter invalid" errors:**
- Check API documentation
- Verify parameter values
- Ensure model compatibility

## Related Documentation

- [Z.ai API Documentation](https://docs.z.ai/devpack/overview)
- [Quota Limits Guide](QUOTA_LIMITS_GUIDE.md)
- [Developer Guide](QUOTA_SYSTEM_DEVELOPER_GUIDE.md)
