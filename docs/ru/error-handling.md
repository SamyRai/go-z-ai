# Обработка ошибок

Каждый метод сервиса возвращает обычный `error`. Сбои на транспортном уровне
(DNS, отказ соединения, таймаут) возвращаются обёрнутыми в `fmt.Errorf`; всё,
что отклонил сам API Z.AI, возвращается как `*client.APIError` (обычно
обёрнутый, поэтому используйте `errors.As` или `errors.AsType`) со
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

`Error()` выдаёт `[code] <UserMessage> (<Message>) (HTTP <status>)`, отбрасывая
пустые или повторяющиеся части.

Вспомогательные предикаты: `IsAuthError()`, `IsRateLimitError()`,
`IsQuotaError()`, `IsParameterError()`, `IsServerError()` — эквивалентны прямой
проверке `.Category`, предоставлены для читаемости в местах вызова.
`IsBalanceError()` уже: ключ прошёл аутентификацию, но тратить ему нечего (1113
либо исчерпанное окно coding plan без баланса для дополнительного
использования, 1316/1317).

## Поведение повторов из коробки

Каждый сервис проходит через единый путь запроса, который повторяет ошибки
429, 5xx и сети с экспоненциальной задержкой и джиттером, до
`Config.MaxRetries` (`0` означает значение по умолчанию, 3; `-1` отключает
повторы). Заголовок `Retry-After`, в секундах или в виде HTTP-даты, задаёт
ожидание вместо этого; любое ожидание ограничено 30 секундами, а отмена
контекста его прерывает. Multipart-загрузки (аудио, файлы, образцы голоса)
отправляются один раз и никогда не повторяются. Обычно вам не нужен
собственный цикл повторов — `APIError.IsRetriable` подскажет, была ли ошибка,
дошедшая до вашего кода, временной, повторы которой исчерпались.

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
| 1316 / 1317 | `ErrCodeHourlyLimitNoBalance` / `ErrCodeWeeklyLimitNoBalance` — 5-часовое / 7-дневное окно исчерпано, нет баланса для дополнительного использования | Quota | Нет |
| 1318 / 1319 | `ErrCodeHourlyLimitNoSpend` / `ErrCodeWeeklyLimitNoSpend` — дополнительное использование недоступно при месячном лимите расходов | Quota | Нет |
| 1320 / 1321 | `ErrCodeHourlyLimitSpendCap` / `ErrCodeWeeklyLimitSpendCap` — достигнут месячный предел расходов | Quota | Нет |
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

Код, которого нет в этой таблице (включая тело вообще без кода),
классифицируется только по HTTP-статусу: 401 — это Auth, 403 — Permission, 429 —
RateLimit (повторяемый), 5xx — Server (повторяемый), а всё остальное —
Parameter. Так что повторяются только троттлинг и сбои на стороне сервера;
неизвестный 4xx никогда не повторяется.

Источник истины: [`pkg/client/errors.go`](../../pkg/client/errors.go).

## Сбои внутри HTTP 200

Несколько поверхностей сообщают о сбое в теле успешного HTTP-ответа. Клиент
превращает большинство из них в `*APIError` с `HTTPStatus` 200 и `IsRetriable`
равным false, так что их все покрывает одна проверка `errors.As`:

- **Monitor и biz API** (квота, использование, баланс, подписки) оборачивают
  каждый ответ в `{code, msg, success, data}`. Неудачный конверт становится
  `*APIError`, у которого `Code` — это код конверта; если этот код похож на
  HTTP-статус (401, 500, …), он также задаёт категорию.
- **Потоки.** После начала потока сервер может прислать фрагмент
  `{"error": {...}}` в стиле OpenAI (chat), `event: error` (Anthropic) или
  событие `error` / `response.failed` (Responses). Каждый из них завершает
  поток, передавая `*APIError` как терминальную ошибку итератора. Он не
  повторяется, потому что часть ответа уже была потреблена.
- **Responses `Create`** возвращает `*APIError`, когда статус ответа —
  `failed`.

Агенты — исключение: `Agents().Invoke` и `AsyncResult` возвращают ответ, даже
если вызов завершился неудачей на бизнес-уровне, потому что конверт сбоя несёт
полезные поля. Ненулевой `error` из них означает, что не сработал
*транспортный уровень*; для бизнес-сбоя проверяйте `resp.Failed()`.
`resp.Error` — это `*AgentError`, который реализует `error`, так что его можно
обернуть:

```go
resp, err := c.Agents().Invoke(ctx, req)
if err != nil {
    return err
}
if resp.Failed() {
    return fmt.Errorf("agent failed: %w", resp.Error)
}
```
