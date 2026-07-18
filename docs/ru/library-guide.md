# Руководство по библиотеке

`pkg/client` — это самостоятельная библиотека на Go: всё, что делает CLI,
достигается за счёт вызовов к этому пакету. Вы можете зависеть от него
напрямую, вообще без CLI.

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"
```

## Создание клиента

```go
c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
})
```

Или, если нужен только вариант с переменными окружения по умолчанию и без
другой конфигурации:

```go
c, err := client.NewClientFromEnv() // reads ZAI_API_KEY, ZAI_API_BASE_URL
```

Поля `Config`:

| Поле | По умолчанию | Примечания |
|---|---|---|
| `APIKey` | — | Обязательно |
| `BaseURL` | `https://api.z.ai/api/paas/v4` | Переопределите для эндпоинта coding-plan и т. п. |
| `HTTPClient` | внутренне настроенный `*http.Client` | Подключите собственный транспорт, если нужно кастомное поведение TLS/прокси |
| `Timeout` | 30с | Ограничивает ожидание dial/TLS/заголовков ответа — **не** чтение всего тела ответа, поэтому никогда не обрезает активный SSE-поток |
| `MaxRetries` | 3 | Повторные попытки при ошибках 429/5xx/сети. `-1` полностью отключает повторные попытки |
| `RetryDelay` | 200мс | Базовая задержка экспоненциального backoff |
| `ChinaAPIKey` | берётся из `APIKey`, если не задан | Нужен только при наличии отдельного креда только для bigmodel.cn — см. [Аккаунты и квоты](accounts-and-quota.md#regional-gateways-apiza--openbigmodelcn) |
| `Region` | `RegionGlobal` | Выбирает хост для monitor/biz/agents/detection: `RegionGlobal` (api.z.ai) или `RegionChina` (open.bigmodel.cn). Не переопределяет `BaseURL` (чатовая поверхность) и хост Embeddings/Moderations. См. [Аккаунты и квоты](accounts-and-quota.md#regional-gateways-apiza--openbigmodelcn). |

Каждый метод сервиса принимает `context.Context` первым аргументом и
прокидывает его вплоть до HTTP-вызова — отмените его, чтобы прервать запрос
или ожидающий backoff перед повторной попыткой.

## Сервисы

`Client` предоставляет по одному методу на сервис, все в едином формате
`c.<Service>().<Method>(ctx, ...)`:

| Аксессор | Что покрывает |
|---|---|
| `c.Chat()` | Completions — `Create`, `CreateAsync`, `CreateStream`, `CreateSimple`, `RunWithTools` |
| `c.Models()` | `List`, `Get`, `GetTextModels`, `GetVisionModels`, `GetFreeModels`, `RefreshCache` |
| `c.Images()` | `Generate`, `GenerateAsync` |
| `c.Videos()` | `Generate` (всегда асинхронно) |
| `c.Audio()` | `Transcribe`, `Speech` |
| `c.Voice()` | `Clone`, `Delete`, `List` — клонирование голоса GLM-TTS |
| `c.Layout()` | `Parse`, `HandwritingOCR` |
| `c.FileParser()` | `Create`, `Sync`, `Result` — документ в текст для RAG |
| `c.Files()` | `Upload`, `List`, `Delete`, `Content` |
| `c.Batch()` | `Create`, `Retrieve`, `List`, `Cancel` |
| `c.Agents()` | `Invoke`, `AsyncResult` |
| `c.Embeddings()` | `Create` (маршрутизируется на `open.bigmodel.cn`) |
| `c.Moderations()` | `Create` (маршрутизируется на `open.bigmodel.cn`) |
| `c.Rerank()` | `Create` |
| `c.Tools()` | `WebSearch`, `WebReader`, `Tokenize` |
| `c.Usage()`, `c.Quota()`, `c.Detection()`, `c.Account()` | Мониторинг использования/квоты/аккаунта GLM Coding Plan |
| `c.GetAsyncResult(ctx, id)`, `c.WaitForResult(ctx, id, interval)` | Общий опрос для асинхронных задач image/video/chat |

Все проверки валидации запроса (обязательные поля и т. п.) выполняются на
стороне клиента перед отправкой запроса — вы сразу получаете локальный
`error`, а не делаете лишний рейс за чем-то вроде отсутствующего `model`.

## Завершение чата

```go
resp, err := c.Chat().Create(ctx, client.ChatRequest{
    Model: "glm-5.2",
    Messages: []client.Message{
        {Role: "system", Content: "You are a helpful assistant."},
        {Role: "user", Content: "Explain goroutines in one paragraph"},
    },
    Temperature: 0.7,
})
fmt.Println(resp.Choices[0].Message.Content)
```

### Потоковая передача

```go
err := c.Chat().CreateStream(ctx, req, func(chunk client.StreamChunk) error {
    if len(chunk.Choices) > 0 {
        fmt.Print(chunk.Choices[0].Delta.Content)
    }
    return nil // a non-nil return aborts the stream
})
```

Установите `req.StreamToolCall = true` (GLM-4.6+), чтобы передавать инкременты
tool-call дельт в `chunk.Choices[0].Delta.ToolCalls` порциями в нескольких
событиях, а не получать их одним пакетом в конце хода. Полезно, чтобы
показывать в UI прогресс «модель вызывает инструмент…». НЕ ВЕРИФИЦИРОВАНО В
LIVE — см. [Дорожная карта](roadmap.md).

### Асинхронный режим

```go
task, _ := c.Chat().CreateAsync(ctx, req)
result, err := c.WaitForResult(ctx, task.ID, 3*time.Second)
```

### Визуальный (изображения в сообщении)

```go
req.Messages[len(req.Messages)-1].Images = []string{
    "https://example.com/photo.jpg", // or a data: URI
}
req.Model = "glm-4.6v"
```

### Структурированный вывод

```go
req.ResponseFormat = client.NewJSONSchemaFormat("my_schema", rawJSONSchema, true /* strict */)
```

### Вызов функций

Для ручного управления — сами инспектируйте `resp.Choices[0].Message.ToolCalls`
и добавляете сообщения с `role: "tool"` перед повторным вызовом `Create`. Для
типового сценария `RunWithTools` ведёт этот цикл за вас:

```go
resp, err := c.Chat().RunWithTools(ctx, req, func(name, arguments string) (string, error) {
    switch name {
    case "get_weather":
        return `{"temp_c": 18}`, nil
    default:
        return "", fmt.Errorf("unknown tool %q", name)
    }
})
```

Он выполняет каждый вызов инструмента, добавляет сообщения assistant + tool и
повторяет, пока модель не вернёт причину завершения без вызова инструмента
или пока не будет превышено `ToolMaxRounds` (8) — используйте
`RunWithToolsLimit`, чтобы задать другой лимит. Ошибка исполнителя
инструмента сообщается модели как результат этого инструмента
(`"error: ..."`), а не возвращается вызывающему коду, поэтому модель может
восстановиться вместо того, чтобы провалить весь обмен.

#### Типы инструментов

`Tool` несёт одну из трёх полезных нагрузок, выбираемую полем `Type`:

| Конструктор | `Type` | Полезная нагрузка |
|---|---|---|
| `NewFunctionTool(name, desc, params)` | `ToolTypeFunction` (`"function"`) | `FunctionDef` — вызываемая сущность, которую модель вызывает по имени |
| `NewRetrievalTool(knowledgeID, prompt)` | `ToolTypeRetrieval` (`"retrieval"`) | `Retrieval` — база знаний для обоснования ответа |
| `NewWebSearchTool(queries...)` | `ToolTypeWebSearch` (`"web_search"`) | `WebSearch` — список `search_query` |

`retrieval` и `web_search` описаны на docs.z.ai, но **НЕ ВЕРИФИЦИРОВАНЫ В
LIVE** здесь — только `function` подтверждён против live-API. Форма полезной
нагрузки `web_search` (`{"search_query":[...]}`) следует официальному примеру
из Python SDK; см. [Дорожная карта](roadmap.md).

`validateChatRequest` принудительно применяет три документированных правила на
стороне клиента, так что вы получаете понятную локальную ошибку вместо
непрозрачной ошибки сервера:

- **Паттерн имени инструмента** — `tools[].function.name` должен
  соответствовать `^[A-Za-z0-9_-]{1,64}$`.
- **Лимит функций** — не более `ToolMaxFunctions` (128) инструментов-функций
  на запрос.
- **Полезная нагрузка по типу** — инструмент `function`/`retrieval`/`web_search`
  должен нести соответствующую полезную нагрузку; неизвестные типы
  отклоняются.

#### Поля ответа, которые стоит проверять

Помимо `resp.Choices[0].Message.Content`:

- `resp.Choices[0].FinishReason` — сравнивайте с константами
  `FinishReason*` (`FinishReasonStop`, `FinishReasonToolCalls`,
  `FinishReasonLength`, `FinishReasonSensitive`,
  `FinishReasonModelContextWindowExceeded`, `FinishReasonNetworkError`).
  Последние три — задокументированные live-значения, сигнализирующие о
  завершении не по контенту.
- `resp.WebSearch` — массив верхнего уровня `web_search`, который ответ несёт,
  когда сработал инструмент `web_search` (`Link`/`Title`/`Content` каждой
  записи — это источники, на которых основан ответ). НЕ ВЕРИФИЦИРОВАНО В LIVE.

#### Совместимость schema инструментов

Чатовый эндпоинт GLM использует строгий парсер JSON-Schema для `parameters`
инструментов: schema, содержащая `anyOf`, `oneOf`, `allOf` или ссылку
`$ref`/`$defs`, заставляет его вернуть **HTTP 500** вместо пригодной ошибки.
Именно эти конструкции и порождают языки со статической типизацией — поле,
допускающее null, превращается в
`anyOf: [{…}, {"type":"null"}]`, повторно используемая структура — в `$ref`.

По умолчанию клиент переписывает schema инструментов в плоское подмножество,
которое принимает GLM, перед каждым чатовым запросом (объединения с null
схлопываются в лежащий в основе тип, `allOf` сливается, `$ref`
инлайнится), сохраняя максимум информации о типах/описаниях. Это no-op для
schema уже в поддерживаемом подмножестве и никогда не мутирует ваши
`req.Tools`.

- Чтобы нормализовать schema самостоятельно (например, вы строите запросы в
  другом месте): `client.SanitizeToolSchemas(tools)`.
- Чтобы пропустить schema без изменений (для отладки или будущего эндпоинта,
  поддерживающего полную версию черновика): установите
  `Config.DisableToolSchemaCompat = true`.

### Anthropic-совместимый Messages API

Z.AI также предоставляет Anthropic-протокольную поверхность на
`/api/anthropic` — тот же эндпоинт, на который GLM Coding Plan направляет
Claude Code. `c.Anthropic()` — типизированный клиент для его
`POST /v1/messages`, параллельный `c.Chat()` для OpenAI-стиля поверхности.
Аутентификация выполняется вашим ключом z.ai как Bearer-токеном (не
Anthropic-овским `x-api-key`), а заголовок `anthropic-version` добавляется
автоматически.

```go
resp, err := c.Anthropic().Create(ctx, client.AnthropicMessageRequest{
    Model:     "glm-4.6",
    MaxTokens: 1024, // required by the Messages API
    System:    "You are concise.",
    Messages: []client.AnthropicMessage{
        client.AnthropicTextMessage("user", "Explain goroutines in one line"),
    },
})
fmt.Println(resp.Text()) // concatenated text blocks
```

Потоковая передача отдаёт сырые SSE-события Anthropic (`message_start`,
`content_block_delta`, …) с именем события и JSON-полезной нагрузкой,
которую вы десериализуете по типу события:

```go
err := c.Anthropic().CreateStream(ctx, req, func(ev client.AnthropicStreamEvent) error {
    if ev.Type == "content_block_delta" {
        // ev.Data is {"delta":{"type":"text_delta","text":"…"}, …}
    }
    return nil
})
```

Инструменты, объявленные через `AnthropicTool.InputSchema`, проходят ту же
нормализацию GLM schema, что и чатовые инструменты (см. выше).
`Config.DisableToolSchemaCompat` её отключает.

Расширенное мышление (модели GLM — рассуждающие) включается на уровне запроса
и считывается через `resp.Thinking()`:

```go
req.Thinking = &client.AnthropicThinking{Type: "enabled", BudgetTokens: 2048}
resp, _ := c.Anthropic().Create(ctx, req)
fmt.Println(resp.Thinking()) // thinking blocks, or reasoning_content if the
                             // endpoint surfaces reasoning that way instead
fmt.Println(resp.Text())     // the answer, without the reasoning mixed in
```

Форма ответа успешного пути смоделирована по документированному Messages API
Anthropic и здесь пока не прошла живую верификацию — см. [Дорожная
карта](roadmap.md).

## Обработка ошибок

Полный справочник по `APIError`, кодам ошибок и поведению повторных попыток по
умолчанию — см. [Обработка ошибок](error-handling.md).

## Управление мультиаккаунтными кредами

Хранилище мультиаккаунтных кредов и писатели кредов/конфигурации GLM Coding
Plan живут в `internal/accounts` и `internal/coding`. Они внутренние для этого
модуля — не часть импортируемого публичного API — и потому могут эволюционировать
без semver-ограничений. `pkg/client` — единственный поддерживаемый публичный
пакет; команды CLI `accounts` и `coding` — стабильный способ управлять этой
функциональностью. (Раньше эти пакеты лежали в `pkg/` и были импортируемыми;
о переносе см. [CHANGELOG](../../CHANGELOG.md).)

## Тестирование вашего кода против этого клиента

Каждый метод сервиса — это обычная функция на конкретном типе без интерфейсов,
поэтому стандартный подход в Go — направить `Config.BaseURL` на
`httptest.Server`, которым вы управляете. Если хотите воспроизвести *реальный*
записанный трафик Z.AI вместо рукописного стаба, посмотрите, как это делают
тесты самого репозитория с
[go-vcr](https://github.com/dnaeon/go-vcr) — `pkg/client/*_test.go` и
`pkg/client/testdata/cassettes/` — и прочтите
[Contributing § соглашение о живой верификации](../../CONTRIBUTING.md),
чтобы понять, почему.

## Архитектурные заметки

О том, как сервисы устроены внутри (фасад `doRequest`, дизайн повторных
попыток/backoff, почему часть сервисов аутентифицируется против другого
base-URL), см. [Архитектура](architecture.md).
