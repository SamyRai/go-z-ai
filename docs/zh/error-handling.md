# 错误处理

每个 service 方法都返回一个普通的 `error`。传输层失败（DNS、连接被拒绝、
超时）会以 `fmt.Errorf` 包装后返回；任何被 Z.AI API 自身拒绝的请求会以
`*client.APIError` 形式返回，其中带有结构化字段，你可以据此分支处理，而不必
解析消息字符串。

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

`APIError` 字段：

| Field | Meaning |
|---|---|
| `HTTPStatus` | HTTP 状态码 |
| `Code` | Z.AI 的业务错误码（int） |
| `Message` | API 返回的原始消息 |
| `Category` | 下列分类之一 |
| `UserMessage` | 预先写好的、更友好的描述 |
| `IsRetriable` | client 自身的重试逻辑是否将其视为瞬时错误 |
| `RequestID` | 用于支持/调试，当 API 返回时携带 |

辅助谓词：`IsAuthError()`、`IsRateLimitError()`、`IsQuotaError()`、
`IsParameterError()`、`IsServerError()`——等价于直接检查 `.Category`，提供
它们是为了在调用处更易读。

## 开箱即用的重试行为

`Client.doRequest`（被每个 service 使用）已经对 429/5xx/网络错误进行重试，
支持指数退避、抖动以及 `Retry-After`，最多重试到 `Config.MaxRetries`
（默认为 3；设为 `-1` 可禁用）。你通常不需要自己写重试循环——
`APIError.IsRetriable` 会告诉你某个到达你代码的错误是否已经耗尽了那些重试。

## 错误码参考

| Code | Constant | Category | Retriable |
|---|---|---|---|
| 1000 | `ErrCodeAuthFailed` | Auth | 否 |
| 1001 | `ErrCodeAuthNotFound` | Auth | 否 |
| 1003 | `ErrCodeAuthTokenExpired` | Auth | 否 |
| 1005 | `ErrCodeAuthNeed2FA` | Auth | 否 |
| 1113 | `ErrCodeInsufficientBalance` | Quota | 否 |
| 1302 | `ErrCodeRateLimitReached` | RateLimit | 是 |
| 1305 | `ErrCodeServiceOverloaded` | Server | 是 |
| 1308 | `ErrCodeUsageLimitReached` | Quota | 否 |
| 1309 | `ErrCodeCodingPlanExpired` | Quota | 否 |
| 1310 | `ErrCodeWeeklyMonthlyExhausted` | Quota | 否 |
| 1311 | `ErrCodeModelNotIncluded` | Quota | 否 |
| 1313 | `ErrCodeFairUsageViolation` | Quota | 否 |
| 1314 | `ErrCodeEnterpriseExpired` | Quota | 否 |
| 1315 | `ErrCodeEnterpriseKeyOnly` | Quota | 否 |
| 1316–1321 | usage-limit variants | Quota | 否 |
| 1210 | `ErrCodeInvalidParameter` | Parameter | 否 |
| 1211 | `ErrCodeUnknownModel` | Parameter | 否 |
| 1212 | `ErrCodeMethodNotSupported` | Parameter | 否 |
| 1213 | `ErrCodeParameterMissing` | Parameter | 否 |
| 1214 | `ErrCodeParameterInvalid` | Parameter | 否 |
| 1215 | `ErrCodeParametersConflict` | Parameter | 否 |
| 1221 | `ErrCodeAPITakenOffline` | Parameter | 否 |
| 1222 | `ErrCodeAPINotExist` | Parameter | 否 |
| 1261 | `ErrCodePromptTooLong` | Parameter | 否 |
| 1301 | `ErrCodeUnsafeContent` | Content | 否 |
| 1220 | `ErrCodeNoPermission` | Permission | 否 |
| -1 | `ErrCodeInternalError` | Server | 是 |
| 1200 | `ErrCodeAPICallError` | Server | 是 |
| 1230 | `ErrCodeProcessError` | Server | 是 |
| 1234 | `ErrCodeNetworkError` | Server | 是 |

本 client 未识别的错误码默认归为 `ErrorCategoryServer` 且
`IsRetriable: true`——这是一个合理的默认值（把未知错误当作瞬时服务器问题
处理），但上表中列出的常量才是具有专门定制的 `UserMessage` 和重试决策的
那些。

事实来源：[`pkg/client/errors.go`](../../pkg/client/errors.go)。

## HTTP 200 内嵌失败 的怪异行为

少数端点（Agents 的 `Invoke`/`AsyncResult`）即使在业务层面调用失败时也会
返回 HTTP 200——失败信息内嵌在响应体中，而非通过 HTTP 状态码指示。从这些
方法收到非 nil 的 `error` 意味着*传输层*失败；要检测一个成功响应内部的
业务级失败，请检查 `resp.Failed()`（或直接查看 `resp.Error`/`resp.Status`）。
两种响应类型都在其 `Failed()` 方法上对此做了说明——如果你只检查
`err != nil`，很容易遗漏这一点。
