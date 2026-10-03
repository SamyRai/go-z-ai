# Error Handling

Every service method returns a plain `error`. Transport-level failures (DNS,
connection refused, timeout) come back wrapped in `fmt.Errorf`; anything the
Z.AI API itself rejected comes back as `*client.APIError` (usually wrapped, so
use `errors.As` or `errors.AsType`), with structured fields you can branch on
instead of parsing message strings.

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

`Error()` reads `[code] <UserMessage> (<Message>) (HTTP <status>)`, dropping
the parts that are empty or repeated.

Helper predicates: `IsAuthError()`, `IsRateLimitError()`, `IsQuotaError()`,
`IsParameterError()`, `IsServerError()` — equivalent to checking `.Category`
directly, provided for readability at call sites. `IsBalanceError()` is
narrower: the key authenticated but has nothing to spend (1113, or an
exhausted coding-plan window with no balance for extra usage, 1316/1317).

## Retry behavior you get for free

Every service goes through one request path that retries 429, 5xx, and
network errors with exponential backoff and jitter, up to
`Config.MaxRetries` (`0` means the default of 3; `-1` disables retries). A
`Retry-After` header, in seconds or as an HTTP date, sets the wait instead;
any wait is capped at 30 seconds, and cancelling the context aborts it.
Multipart uploads (audio, files, voice samples) are sent once and never
retried. You generally don't need your own retry loop —
`APIError.IsRetriable` tells you whether an error that reached your code was
a transient one whose retries ran out.

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
| 1316 / 1317 | `ErrCodeHourlyLimitNoBalance` / `ErrCodeWeeklyLimitNoBalance` — 5-hour / 7-day window used up, no balance for extra usage | Quota | No |
| 1318 / 1319 | `ErrCodeHourlyLimitNoSpend` / `ErrCodeWeeklyLimitNoSpend` — extra usage unavailable under the monthly spend limit | Quota | No |
| 1320 / 1321 | `ErrCodeHourlyLimitSpendCap` / `ErrCodeWeeklyLimitSpendCap` — monthly spend cap reached | Quota | No |
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

A code this table doesn't know (including a body with no code at all) is
classified by the HTTP status alone: 401 is Auth, 403 Permission, 429
RateLimit (retriable), 5xx Server (retriable), and anything else Parameter.
So only throttling and server-side failures are ever retried; an unknown 4xx
is never retried.

Source of truth: [`pkg/client/errors.go`](../../pkg/client/errors.go).

## Failures inside an HTTP 200

Several surfaces report a failure in the body of a successful HTTP response.
The client turns most of them into an `*APIError` with `HTTPStatus` 200 and
`IsRetriable` false, so one `errors.As` check covers them:

- **Monitor and biz APIs** (quota, usage, balance, subscriptions) wrap every
  answer in `{code, msg, success, data}`. A failed envelope becomes an
  `*APIError` whose `Code` is the envelope's code; when that code looks like
  an HTTP status (401, 500, …) it also sets the category.
- **Streams.** After a stream has started, the server can send an
  OpenAI-style `{"error": {...}}` chunk (chat), an `event: error` (Anthropic),
  or an `error` / `response.failed` event (Responses). Each ends the stream
  with an `*APIError` as the iterator's terminal error. It is not retried,
  because part of the response was already consumed.
- **Responses `Create`** returns an `*APIError` when the response's status is
  `failed`.

Agents are the exception: `Agents().Invoke` and `AsyncResult` return the
response even when the call failed at the business level, because the
failure envelope carries useful fields. A non-nil `error` from them means the
*transport* failed; check `resp.Failed()` for a business failure.
`resp.Error` is an `*AgentError`, which implements `error`, so you can wrap
it:

```go
resp, err := c.Agents().Invoke(ctx, req)
if err != nil {
    return err
}
if resp.Failed() {
    return fmt.Errorf("agent failed: %w", resp.Error)
}
```
