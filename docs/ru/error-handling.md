# Обработка ошибок

Каждый метод сервиса возвращает обычный `error`. Сбои на транспортном уровне
(DNS, отказ соединения, таймаут) возвращаются обёрнутыми в `fmt.Errorf`; всё,
что отклонил сам API Z.AI, возвращается как `*client.APIError` со
структурированными полями, по которым можно ветвить логику, вместо разбора
строк сообщения.

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

Поля `APIError`:

| Поле | Назначение |
|---|---|
| `HTTPStatus` | HTTP-код состояния |
| `Code` | Бизнес-код ошибки Z.AI (int) |
| `Message` | Исходное сообщение от API |
| `Category` | Одна из категорий ниже |
| `UserMessage` | Более понятное, заранее подготовленное описание |
| `IsRetriable` | Считает ли собственная логика повторов клиента эту ошибку временной |
| `RequestID` | Для поддержки/отладки, когда API его вернул |

Вспомогательные предикаты: `IsAuthError()`, `IsRateLimitError()`,
`IsQuotaError()`, `IsParameterError()`, `IsServerError()` — эквивалентны прямой
проверке `.Category`, предоставлены для читаемости в местах вызова.

## Поведение повторов из коробки

`Client.doRequest` (используется каждым сервисом) уже повторяет ошибки
429/5xx/сети с экспоненциальной задержкой, джиттером и поддержкой
`Retry-After`, до `Config.MaxRetries` (по умолчанию 3; установите `-1` для
отключения). Обычно вам не нужен собственный цикл повторов —
`APIError.IsRetriable` подскажет, исчерпала ли ошибка, дошедшая до вашего
кода, эти повторные попытки.

## Справочник кодов ошибок

| Code | Constant | Category | Повторяемый |
|---|---|---|---|
| 1000 | `ErrCodeAuthFailed` | Auth | Нет |
| 1001 | `ErrCodeAuthNotFound` | Auth | Нет |
| 1003 | `ErrCodeAuthTokenExpired` | Auth | Нет |
| 1005 | `ErrCodeAuthNeed2FA` | Auth | Нет |
| 1113 | `ErrCodeInsufficientBalance` | Quota | Нет |
| 1302 | `ErrCodeRateLimitReached` | RateLimit | Да |
| 1305 | `ErrCodeServiceOverloaded` | Server | Да |
| 1308 | `ErrCodeUsageLimitReached` | Quota | Нет |
| 1309 | `ErrCodeCodingPlanExpired` | Quota | Нет |
| 1310 | `ErrCodeWeeklyMonthlyExhausted` | Quota | Нет |
| 1311 | `ErrCodeModelNotIncluded` | Quota | Нет |
| 1313 | `ErrCodeFairUsageViolation` | Quota | Нет |
| 1314 | `ErrCodeEnterpriseExpired` | Quota | Нет |
| 1315 | `ErrCodeEnterpriseKeyOnly` | Quota | Нет |
| 1316–1321 | usage-limit variants | Quota | Нет |
| 1210 | `ErrCodeInvalidParameter` | Parameter | Нет |
| 1211 | `ErrCodeUnknownModel` | Parameter | Нет |
| 1212 | `ErrCodeMethodNotSupported` | Parameter | Нет |
| 1213 | `ErrCodeParameterMissing` | Parameter | Нет |
| 1214 | `ErrCodeParameterInvalid` | Parameter | Нет |
| 1215 | `ErrCodeParametersConflict` | Parameter | Нет |
| 1221 | `ErrCodeAPITakenOffline` | Parameter | Нет |
| 1222 | `ErrCodeAPINotExist` | Parameter | Нет |
| 1261 | `ErrCodePromptTooLong` | Parameter | Нет |
| 1301 | `ErrCodeUnsafeContent` | Content | Нет |
| 1220 | `ErrCodeNoPermission` | Permission | Нет |
| -1 | `ErrCodeInternalError` | Server | Да |
| 1200 | `ErrCodeAPICallError` | Server | Да |
| 1230 | `ErrCodeProcessError` | Server | Да |
| 1234 | `ErrCodeNetworkError` | Server | Да |

Код ошибки, неизвестный этому клиенту, по умолчанию получает
`ErrorCategoryServer` с `IsRetriable: true` — разумное значение по умолчанию
(неизвестные коды трактуются как временные проблемы сервера), но именно для
перечисленных выше констант задано специально подобранное `UserMessage` и
решение о повторе.

Источник истины: [`pkg/client/errors.go`](../../pkg/client/errors.go).

## Особенность: 200 со встроенным сбоем

Несколько эндпоинтов (`Invoke`/`AsyncResult` у агентов) возвращают HTTP 200
даже в случае неудачи на бизнес-уровне — сбой встроен в тело ответа, а не
сигнализируется через HTTP-статус. Ненулевой `error` из этих методов означает,
что не сработал *транспортный уровень*; для обнаружения сбоя на бизнес-уровне
внутри успешного ответа проверяйте `resp.Failed()` (или напрямую `resp.Error`/
`resp.Status`). Оба типа ответов документируют это в методе `Failed()` —
легко пропустить, если проверять только `err != nil`.
