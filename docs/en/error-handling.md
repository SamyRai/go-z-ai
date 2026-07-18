# Error Handling

Every service method returns a plain `error`. Transport-level failures (DNS,
connection refused, timeout) come back wrapped in `fmt.Errorf`; anything the
Z.AI API itself rejected comes back as `*client.APIError`, with structured
fields you can branch on instead of parsing message strings.

```go
resp, err := c.Chat().Create(ctx, req)
if err != nil {
    var apiErr *client.APIError
    if errors.As(err, &apiErr) {
        fmt.Printf("[%d] %s\n", apiErr.Code, apiErr.UserMessage)

        switch apiErr.Category {
        case client.ErrorCategoryAuth:
            // bad/expired key — don't retry, tell the user
        case client.ErrorCategoryQuota:
            // out of balance/quota — don't retry, surface to the user
        case client.ErrorCategoryRateLimit:
            if apiErr.IsRetriable {
                // the client already retries this internally up to
                // Config.MaxRetries — you'll only see it here if retries
                // were exhausted or disabled (MaxRetries: -1)
            }
        }
        return
    }
    // transport-level failure — network, DNS, timeout
}
```

`APIError` fields:

| Field | Meaning |
|---|---|
| `HTTPStatus` | HTTP status code |
| `Code` | Z.AI's business error code (int) |
| `Message` | Raw message from the API |
| `Category` | One of the categories below |
| `UserMessage` | A friendlier, pre-written description |
| `IsRetriable` | Whether the client's own retry logic considers this transient |
| `RequestID` | For support/debugging, when the API returned one |

Helper predicates: `IsAuthError()`, `IsRateLimitError()`, `IsQuotaError()`,
`IsParameterError()`, `IsServerError()` — equivalent to checking `.Category`
directly, provided for readability at call sites.

## Retry behavior you get for free

`Client.doRequest` (used by every service) already retries 429/5xx/network
errors with exponential backoff, jitter, and `Retry-After` support, up to
`Config.MaxRetries` (default 3; set to `-1` to disable). You generally don't
need your own retry loop — `APIError.IsRetriable` tells you whether an error
that reached your code already exhausted those retries.

## Error code reference

| Code | Constant | Category | Retriable |
|---|---|---|---|
| 1000 | `ErrCodeAuthFailed` | Auth | No |
| 1001 | `ErrCodeAuthNotFound` | Auth | No |
| 1003 | `ErrCodeAuthTokenExpired` | Auth | No |
| 1005 | `ErrCodeAuthNeed2FA` | Auth | No |
| 1113 | `ErrCodeInsufficientBalance` | Quota | No |
| 1302 | `ErrCodeRateLimitReached` | RateLimit | Yes |
| 1305 | `ErrCodeServiceOverloaded` | Server | Yes |
| 1308 | `ErrCodeUsageLimitReached` | Quota | No |
| 1309 | `ErrCodeCodingPlanExpired` | Quota | No |
| 1310 | `ErrCodeWeeklyMonthlyExhausted` | Quota | No |
| 1311 | `ErrCodeModelNotIncluded` | Quota | No |
| 1313 | `ErrCodeFairUsageViolation` | Quota | No |
| 1314 | `ErrCodeEnterpriseExpired` | Quota | No |
| 1315 | `ErrCodeEnterpriseKeyOnly` | Quota | No |
| 1316–1321 | usage-limit variants | Quota | No |
| 1210 | `ErrCodeInvalidParameter` | Parameter | No |
| 1211 | `ErrCodeUnknownModel` | Parameter | No |
| 1212 | `ErrCodeMethodNotSupported` | Parameter | No |
| 1213 | `ErrCodeParameterMissing` | Parameter | No |
| 1214 | `ErrCodeParameterInvalid` | Parameter | No |
| 1215 | `ErrCodeParametersConflict` | Parameter | No |
| 1221 | `ErrCodeAPITakenOffline` | Parameter | No |
| 1222 | `ErrCodeAPINotExist` | Parameter | No |
| 1261 | `ErrCodePromptTooLong` | Parameter | No |
| 1301 | `ErrCodeUnsafeContent` | Content | No |
| 1220 | `ErrCodeNoPermission` | Permission | No |
| -1 | `ErrCodeInternalError` | Server | Yes |
| 1200 | `ErrCodeAPICallError` | Server | Yes |
| 1230 | `ErrCodeProcessError` | Server | Yes |
| 1234 | `ErrCodeNetworkError` | Server | Yes |

An error code this client hasn't seen before defaults to `ErrorCategoryServer`
with `IsRetriable: true` — a reasonable default (treat unknowns as transient
server issues) but the constants above are the ones with a specifically
tailored `UserMessage` and retry decision.

Source of truth: [`pkg/client/errors.go`](../../pkg/client/errors.go).

## The 200-with-embedded-failure quirk

A few endpoints (Agents' `Invoke`/`AsyncResult`) return HTTP 200 even when the
call fails at the business level — the failure is embedded in the response
body, not signaled via HTTP status. A non-nil `error` from these methods
means the *transport* failed; check `resp.Failed()` (or `resp.Error`/
`resp.Status` directly) for a business-level failure inside a successful
response. Both response types document this on their `Failed()` method — it's
easy to miss if you only check `err != nil`.
