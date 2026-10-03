# 错误处理

每个 service 方法都返回一个普通的 `error`。传输层失败（DNS、连接被拒绝、
超时）会以 `fmt.Errorf` 包装后返回；任何被 Z.AI API 自身拒绝的请求会以
`*client.APIError` 形式返回（通常是被包装过的，因此请使用 `errors.As` 或
`errors.AsType`），其中带有结构化字段，你可以据此分支处理，而不必解析消息
字符串。

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

`Error()` 的输出形如 `[code] <UserMessage> (<Message>) (HTTP <status>)`，会省略
为空或重复的部分。

辅助谓词：`IsAuthError()`、`IsRateLimitError()`、`IsQuotaError()`、
`IsParameterError()`、`IsServerError()`——等价于直接检查 `.Category`，提供
它们是为了在调用处更易读。`IsBalanceError()` 的范围更窄：密钥已通过鉴权，但
已经没有可花的额度（1113，或 coding-plan 窗口已用尽且没有余额用于额外用量，
即 1316/1317）。

## 开箱即用的重试行为

每个 service 都走同一条请求路径，它会对 429、5xx 和网络错误进行重试，采用
指数退避加抖动，最多重试到 `Config.MaxRetries`（`0` 表示使用默认值 3；
`-1` 表示禁用重试）。`Retry-After` 头（秒数或 HTTP 日期形式）会改为用它来设定
等待时长；任何等待都以 30 秒为上限，取消 context 会中止等待。Multipart 上传
（音频、文件、语音样本）只发送一次，绝不重试。你通常不需要自己写重试循环——
`APIError.IsRetriable` 会告诉你某个到达你代码的错误是否是重试已经耗尽的瞬时
错误。

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
| 1316 / 1317 | `ErrCodeHourlyLimitNoBalance` / `ErrCodeWeeklyLimitNoBalance` —— 5 小时 / 7 天窗口已用尽，没有余额用于额外用量 | Quota | 否 |
| 1318 / 1319 | `ErrCodeHourlyLimitNoSpend` / `ErrCodeWeeklyLimitNoSpend` —— 受月度花费上限限制，无法使用额外用量 | Quota | 否 |
| 1320 / 1321 | `ErrCodeHourlyLimitSpendCap` / `ErrCodeWeeklyLimitSpendCap` —— 已达月度花费上限 | Quota | 否 |
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

本表未识别的错误码（包括响应体中根本没有错误码的情况）仅按 HTTP 状态码分类：
401 为 Auth，403 为 Permission，429 为 RateLimit（可重试），5xx 为 Server（可
重试），其余一律为 Parameter。因此只有限流和服务端故障才会被重试；未知的 4xx
绝不会被重试。

事实来源：[`pkg/client/errors.go`](../../pkg/client/errors.go)。

## HTTP 200 内的失败

有若干接口会在成功的 HTTP 响应体中报告失败。client 会把其中大多数转换为一个
`HTTPStatus` 为 200、`IsRetriable` 为 false 的 `*APIError`，因此一次
`errors.As` 检查就能全部覆盖：

- **Monitor 和 biz API**（配额、用量、余额、订阅）把每个应答都包在
  `{code, msg, success, data}` 里。失败的信封会变成一个 `*APIError`，其 `Code`
  即信封中的 code；当该 code 看起来像 HTTP 状态码（401、500……）时，还会据此
  设置 category。
- **流。** 流开始之后，服务器可能发送 OpenAI 风格的 `{"error": {...}}` 数据块
  （chat）、`event: error`（Anthropic），或 `error` / `response.failed` 事件
  （Responses）。每一种都会以一个 `*APIError` 作为迭代器的终止错误来结束该流。
  它不会被重试，因为响应的一部分已经被消费了。
- **Responses 的 `Create`** 在响应的 status 为 `failed` 时会返回一个
  `*APIError`。

Agents 是例外：`Agents().Invoke` 和 `AsyncResult` 即使调用在业务层面失败也会
返回该响应，因为失败信封中携带有用的字段。从它们收到非 nil 的 `error` 意味着
*传输层*失败；要检测业务级失败，请检查 `resp.Failed()`。`resp.Error` 是一个
`*AgentError`，它实现了 `error`，因此你可以对它进行包装：

```go
resp, err := c.Agents().Invoke(ctx, req)
if err != nil {
    return err
}
if resp.Failed() {
    return fmt.Errorf("agent failed: %w", resp.Error)
}
```
