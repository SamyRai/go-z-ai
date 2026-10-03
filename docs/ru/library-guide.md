# Руководство по библиотеке

`pkg/client` — это самостоятельная библиотека на Go (только стандартная
библиотека): всё, что делает CLI, достигается за счёт вызовов к этому пакету.
Вы можете зависеть от него напрямую, вообще без CLI.

## Содержание

- [Создание клиента](#создание-клиента)
- [Сервисы](#сервисы)
- [Модели по умолчанию](#модели-по-умолчанию)
- [Завершение чата](#завершение-чата)
- [Anthropic-совместимый Messages API](#anthropic-совместимый-messages-api)
- [Responses API (протокол Codex)](#responses-api-протокол-codex)
- [Определение аккаунта и статус](#определение-аккаунта-и-статус)
- [Обработка ошибок](#обработка-ошибок)
- [Хуки наблюдаемости](#хуки-наблюдаемости)
- [Управление мультиаккаунтными кредами](#управление-мультиаккаунтными-кредами)
- [Тестирование вашего кода против этого клиента](#тестирование-вашего-кода-против-этого-клиента)
- [Архитектурные заметки](#архитектурные-заметки)

```bash
go get github.com/SamyRai/go-z-ai
```

Модулю требуется Go 1.26 (см. `go.mod`).

```go
import "github.com/SamyRai/go-z-ai/pkg/client"
```

## Создание клиента

```go
c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
})
if err != nil {
    log.Fatal(err)
}
```

Или, чтобы настроить всё из окружения — теми же переменными, которые учитывает
CLI:

```go
c, err := client.NewClientFromEnv()
if err != nil {
    log.Fatal(err)
}
```

| Переменная | Соответствует | Примечания |
|---|---|---|
| `ZAI_API_KEY` | `Config.APIKey` | Обязательно |
| `ZAI_API_BASE_URL` | `Config.BaseURL` | Необязательное переопределение корня chat/PaaS |
| `ZAI_REGION` | `Config.Region` | Разбирается через `client.ParseRegion`: `global` (по умолчанию) или `china` |
| `ZAI_CHINA_API_KEY` | `Config.ChinaAPIKey` | Необязательные отдельные учётные данные bigmodel.cn |
| `ZAI_MONITOR_TIMEZONE` | `Config.MonitorTimezone` | Название IANA, `UTC`, смещение вроде `UTC+8` или `local`; неверное значение — ошибка |

Поля `Config`:

| Поле | По умолчанию | Примечания |
|---|---|---|
| `APIKey` | — | Обязательно |
| `BaseURL` | `Region.PaaSBaseURL()` — `https://api.z.ai/api/paas/v4` для глобального региона | Явное значение всегда побеждает. Ключи Coding Plan используют `Region.CodingBaseURL()` (или `DetectedAccount.BaseURL`, см. [Определение аккаунта и статус](#определение-аккаунта-и-статус)). |
| `HTTPClient` | внутренне настроенный `*http.Client` | Подключите собственный транспорт, если нужно кастомное поведение TLS/прокси |
| `Timeout` | 30с (`DefaultTimeout`) | Ограничивает ожидание dial/TLS/заголовков ответа — **не** чтение всего тела ответа, поэтому никогда не обрезает активный SSE-поток |
| `MaxRetries` | 3 | Повторные попытки при ошибках 429/5xx/сети. `-1` полностью отключает повторные попытки |
| `RetryDelay` | 200мс | Базовая задержка экспоненциального backoff |
| `ChinaAPIKey` | берётся из `APIKey`, если не задан | Нужен только при наличии отдельных учётных данных только для bigmodel.cn — см. [Аккаунты и квоты](accounts-and-quota.md#региональные-шлюзы-apizai--openbigmodelcn) |
| `Region` | `RegionGlobal` | Выбирает шлюз: `RegionGlobal` (api.z.ai) или `RegionChina` (open.bigmodel.cn). См. [Регионы](#регионы). |
| `MonitorTimezone` | `client.MonitorServerTZ` (UTC+8) | Часовой пояс, в котором интерпретируются метки времени без пояса из monitor-API (квота/использование). Переопределяйте, только если живой захват покажет другой пояс сервера. |
| `DisableToolSchemaCompat` | `false` | Отправлять schema инструментов без изменений — см. [Совместимость schema инструментов](#совместимость-schema-инструментов) |
| `UserAgent` | `"go-z-ai/<version>"` | Переопределяет заголовок `User-Agent`, отправляемый с каждым запросом. Значение по умолчанию идентифицирует go-z-ai для API Z.AI — это важно в рамках [политики использования](coding-tools.md#соответствие-политике-и-правила-использования-) coding-эндпоинта. Переопределяйте, только когда нужен отдельный идентификатор (downstream-приложение, прокси, MCP-сервер); строка-переопределение отправляется как есть. |
| `Hooks` | `nil` | Хуки наблюдаемости ([`Hook`](library-guide.md#хуки-наблюдаемости)), срабатывающие на каждый запрос/ответ/ошибку/фрагмент потока. По умолчанию пусто — путь без хуков ничего не стоит. Конкретные реализации живут в `pkg/observe` (OpenTelemetry). |

Каждый метод сервиса принимает `context.Context` первым аргументом и
прокидывает его вплоть до HTTP-вызова — отмените его, чтобы прервать запрос
или ожидающий backoff перед повторной попыткой.

### Регионы

`Region` владеет каждым URL шлюза; больше нигде в библиотеке хост не
зашит в код. `Config.Region` выбирает хост для регионально-зависимых
сервисов — monitor (квота/использование), biz (аккаунт), agents,
Anthropic-совместимый Messages API, Responses API и определение типа
аккаунта — а при пустом `BaseURL` ещё и корень чата по умолчанию.
`c.Region()` возвращает настроенное значение; пустой или нераспознанный
`Region` означает global.

| Метод | Путь под `Region.Host()` |
|---|---|
| `PaaSBaseURL()` | `/api/paas/v4` — pay-as-you-go, OpenAI-совместимый |
| `CodingBaseURL()` | `/api/coding/paas/v4` — GLM Coding Plan |
| `AnthropicBaseURL()` | `/api/anthropic` |
| `ResponsesBaseURL()` | `/api/v1` — протокол OpenAI Responses (его использует Codex) |
| `MonitorBaseURL()` | `/api/monitor` — квота/использование |
| `BizBaseURL()` | `/api/biz` — аккаунт |
| `AgentsBaseURL()` | `/api` |
| `MCPServerURL(name)` | `/api/mcp/<name>/mcp` — размещённые MCP-серверы Z.AI |

`Host()` — это `https://api.z.ai` или `https://open.bigmodel.cn`
(`client.GlobalHost` / `client.ChinaHost`); `ConsoleURL()` — веб-консоль
(`https://z.ai` / `https://bigmodel.cn`); `BaseURLFor(accountType)` возвращает
coding- или PaaS-корень чата для `AccountType`. `client.ParseRegion` принимает
`china` (псевдонимы `cn`, `bigmodel`) и `global` (псевдонимы `west`, пустая
строка) без учёта регистра; неизвестное значение разрешается в global, а не
завершается ошибкой.

```go
c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    Region: client.RegionChina, // chat, quota, account, agents, Anthropic -> open.bigmodel.cn
})
if err != nil {
    log.Fatal(err)
}
```

Embeddings и Moderations — исключение: они всегда обращаются к
`client.BigModelBaseURL` (open.bigmodel.cn) с `ChinaAPIKey`, каков бы ни был
`Region`. Китайское зеркало маршрутов monitor/biz/agents НЕ ВЕРИФИЦИРОВАНО В
LIVE — см. [Дорожная карта](roadmap.md).

## Сервисы

`Client` предоставляет по одному методу на сервис, все в едином формате
`c.<Service>().<Method>(ctx, ...)`:

| Аксессор | Что покрывает |
|---|---|
| `c.Chat()` | Completions — `Create`, `CreateAsync`, `Stream`, `RunWithTools`, `RunWithToolsLimit` |
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
| `c.Anthropic()` | `Create`, `Stream` — Anthropic-протокол поверхности `/v1/messages` |
| `c.Responses()` | `Create`, `Stream` — протокол OpenAI Responses `/api/v1/responses`, поверхность Codex |
| `c.Embeddings()` | `Create` (маршрутизируется на `open.bigmodel.cn`) |
| `c.Moderations()` | `Create` (маршрутизируется на `open.bigmodel.cn`) |
| `c.Rerank()` | `Create` |
| `c.Tools()` | `WebSearch`, `WebReader`, `Tokenize` |
| `c.Quota()` | Квота и использование GLM Coding Plan — `GetQuotaLimit`, `GetModelUsage`, `GetToolUsage` |
| `c.Detection()` | `DetectAccountType`, `CheckAccountStatus` |
| `c.Account()` | `Balance`, `Subscriptions` |
| `c.GetAsyncResult(ctx, id)`, `c.WaitForResult(ctx, id, interval)` | Общий опрос для асинхронных задач image/video/chat |

Все проверки валидации запроса (обязательные поля и т. п.) выполняются на
стороне клиента перед отправкой запроса — вы сразу получаете локальный
`error`, а не делаете лишний рейс за чем-то вроде отсутствующего `model`.

Клиентского счётчика использования нет: вычисляйте стоимость запроса через
`Pricing.Cost` (см. [Поля ответа, которые стоит
проверять](#поля-ответа-которые-стоит-проверять)), а расход по тарифу читайте
из `c.Quota()`.

## Модели по умолчанию

Используйте эти константы вместо зашитых в код ID моделей; это те же значения
по умолчанию, что использует CLI, TUI и генераторы конфигов инструментов для
кода. Каталог за ними (`pkg/client/models_catalog.go`) также несёт для каждой
модели размер контекста, лимит вывода, цены, возможности и допустимые уровни
reasoning effort; `client.CatalogEntry(id)` возвращает одну строку, а
`c.Models().List` сливает каталог с живым списком `/models`.

| Константа | Модель | Применение |
|---|---|---|
| `DefaultModel` | `glm-5.3` | Флагманская текстовая модель для рассуждений, кода и агентных задач (контекст 1M) |
| `DefaultFastModel` | `glm-5.3-flash` | Быстрый недорогой уровень; нативно мультимодальна |
| `DefaultVisionModel` | `glm-5.3-flash` | То же, что `DefaultFastModel`: вход — изображения, видео и файлы |
| `DefaultOCRModel` | `glm-ocr` | `Layout().Parse` |
| `DefaultASRModel` | `glm-asr-2512` | `Audio().Transcribe` |
| `DefaultTTSModel` | `glm-tts` | `Audio().Speech` |

Модели изображений — `client.ModelGLMImage` (по умолчанию; единственная,
поддерживающая `GenerateAsync`) и `client.ModelCogView4`. Видеомодели
перечислены в `client.VideoModels`, во главе с `client.ModelCogVideoX3`.

## Завершение чата

```go
resp, err := c.Chat().Create(ctx, client.ChatRequest{
    Model: client.DefaultModel,
    Messages: []client.Message{
        {Role: "system", Content: "You are a helpful assistant."},
        {Role: "user", Content: "Explain goroutines in one paragraph"},
    },
    Temperature: 0.7,
})
if err != nil {
    return err
}
fmt.Println(resp.Choices[0].Message.Content)
```

`Temperature`, `TopP` и `MaxTokens` при нуле берутся от сервера;
`Temperature` и `TopP` должны лежать в [0, 1]. `DoSample` — это `*bool`,
чтобы явный `false` отправлялся. `RequestID` и `UserID` (6–128 символов, для
мониторинга злоупотреблений) необязательны.

### Потоковая передача

`Stream` возвращает итератор (`iter.Seq2[StreamChunk, error]`), по которому вы
итерируетесь в цикле; `stream=true` он выставляет за вас:

```go
for chunk, err := range c.Chat().Stream(ctx, req) {
    if err != nil {
        return err // terminal — the loop ends after this iteration
    }
    if len(chunk.Choices) > 0 {
        fmt.Print(chunk.Choices[0].Delta.Content)
    }
}
```

Поток выполняется синхронно внутри цикла — горутины-производителя нет. Выход
из цикла прекращает чтение и закрывает соединение, а отмена `ctx` прерывает
заблокированное чтение. Временные сбои фазы соединения (429/5xx/сеть)
повторяются до `Config.MaxRetries` в точности как в `Create`; как только поток
начался, сбой выходит как терминальный `err` и никогда не повторяется, потому
что часть ответа уже была потреблена. Запрос, не прошедший валидацию, даёт одну
ошибку из итератора до отправки чего-либо.

Рассуждения приходят в `chunk.Choices[0].Delta.ReasoningContent`, а последний
фрагмент несёт `chunk.Usage`.

#### Ошибки внутри потока

Сервер может сообщить о сбое после того, как HTTP 200 уже отправлен: фрагмент
`{"error": {...}}` в стиле OpenAI, Anthropic `event: error` или событие
Responses `error` / `response.failed`. Клиент превращает каждое в `*APIError`
(с `HTTPStatus` 200) и завершает поток с ним как с терминальным `err`, так что
одна и та же обработка через `errors.As` работает для обоих видов сбоев:

```go
for chunk, err := range c.Chat().Stream(ctx, req) {
    if err != nil {
        var apiErr *client.APIError
        if errors.As(err, &apiErr) {
            log.Printf("stream failed: [%d] %s", apiErr.Code, apiErr.UserMessage)
        }
        return err
    }
    _ = chunk
}
```

Ошибка внутри потока не повторяется, даже если её код помечен как
повторяемый: поток уже начался. См. [Обработка ошибок](error-handling.md).

#### Потоковая передача вызовов инструментов

Установите `req.ToolStream = true` (GLM-4.6 и новее), чтобы получать
аргументы вызова инструмента инкрементально в
`chunk.Choices[0].Delta.ToolCalls` порциями в нескольких событиях, а не одним
пакетом в конце хода. Фрагменты одного и того же вызова разделяют
`ToolCall.Index`; конкатенируйте их `Function.Arguments`. Полезно, чтобы
показывать в UI прогресс «модель вызывает инструмент…». НЕ ВЕРИФИЦИРОВАНО В
LIVE — см. [Дорожная карта](roadmap.md).

```go
req.ToolStream = true
calls := map[int]*client.FunctionCall{}
for chunk, err := range c.Chat().Stream(ctx, req) {
    if err != nil {
        return err
    }
    for _, choice := range chunk.Choices {
        for _, tc := range choice.Delta.ToolCalls {
            if tc.Function == nil {
                continue
            }
            fc := calls[tc.Index]
            if fc == nil {
                fc = &client.FunctionCall{}
                calls[tc.Index] = fc
            }
            if tc.Function.Name != "" {
                fc.Name = tc.Function.Name
            }
            fc.Arguments += tc.Function.Arguments
        }
    }
}
```

### Reasoning effort

`ChatRequest.ReasoningEffort` задаёт, насколько усердно рассуждает модель с
режимом мышления. Используйте константы `Effort*` (`EffortMax`, `EffortXhigh`,
`EffortHigh`, `EffortMedium`, `EffortLow`, `EffortMinimal`, `EffortNone`);
`client.AllEfforts` перечисляет их все. Значение API по умолчанию — `max`.

```go
resp, err := c.Chat().Create(ctx, client.ChatRequest{
    Model:           client.DefaultModel,
    Messages:        []client.Message{{Role: "user", Content: "Plan a zero-downtime schema migration"}},
    ReasoningEffort: client.EffortHigh,
})
if err != nil {
    return err
}
msg := resp.Choices[0].Message
fmt.Println(msg.ReasoningContent) // the reasoning
fmt.Println(msg.Content)          // the answer
if d := resp.Usage.CompletionTokensDetails; d != nil {
    fmt.Println("reasoning tokens:", d.ReasoningTokens)
}
```

Значение проверяется на стороне клиента по записи модели в каталоге
(`ModelCatalogEntry.ReasoningEfforts`, также доступно как
`ModelDetails.ReasoningEfforts`), так что неподдерживаемый уровень — это
локальная ошибка, а не рейс к серверу:

- `glm-5.2` принимает каждый уровень.
- Семейство GLM-5.3 (`glm-5.3`, `glm-5.3-flash`, `glm-5.3-flashx`) принимает
  только `low`, `high` и `max` и всегда думает — отключение мышления там не
  поддерживается.
- Модель из каталога без списка effort (например, `glm-5.1`) отклоняет этот
  параметр.
- Модель, которой нет в каталоге, проверяется только по `AllEfforts`.

Сервер нормализует `none`/`minimal` в «пропустить мышление», `low`/`medium` в
`high`, а `xhigh` в `max`. Уровни следуют docs.z.ai; здесь они не проходили
живую верификацию.

Чтобы сохранять рассуждения между ходами (preserved thinking), установите
`ThinkingConfig.ClearThinking` в false и возвращайте `ReasoningContent` каждого
хода assistant без изменений. `RunWithTools` делает это за вас.

```go
keep := false
req.Thinking = &client.ThinkingConfig{Type: client.ThinkingEnabled, ClearThinking: &keep}
req.Messages = append(req.Messages, client.Message{
    Role:             "assistant",
    Content:          msg.Content,
    ReasoningContent: msg.ReasoningContent,
})
```

### Асинхронный режим

```go
task, err := c.Chat().CreateAsync(ctx, req)
if err != nil {
    return err
}
result, err := c.WaitForResult(ctx, task.ID, 3*time.Second)
if err != nil {
    return err
}
if result.TaskStatus == client.TaskStatusSuccess {
    fmt.Println(result.Choices[0].Message.Content)
}
```

`WaitForResult` возвращается, когда задача покидает `PROCESSING` — успешно или
нет, так что проверяйте `TaskStatus` (`TaskStatusSuccess` / `TaskStatusFail`).
Задачи изображений и видео используют тот же опрос. Результаты задачи
изображения лежат в `AsyncResultResponse.ImageResult` (каждый — `GeneratedImage`
с `URL`, который истекает через 30 дней); результаты видеозадачи — в
`VideoResult`:

```go
task, err := c.Images().GenerateAsync(ctx, client.ImageGenerationRequest{
    Model:  client.ModelGLMImage,
    Prompt: "A lighthouse at dusk, oil painting",
})
if err != nil {
    return err
}
result, err := c.WaitForResult(ctx, task.ID, 3*time.Second)
if err != nil {
    return err
}
for _, img := range result.ImageResult {
    fmt.Println(img.URL)
}
```

`VideoGenerationRequest.OffPeak` ставит видеозадачу в очередь внепиковой
обработки по более низкой цене.

### Мультимодальные сообщения (изображения, видео, файлы)

Задайте `Message.Images`, `Message.Videos` или `Message.Files`, чтобы
прикрепить медиа. Каждый элемент — URL с `https://` или URI `data:`. Клиент
переключает `content` сообщения на wire-форму content-parts за вас: сначала
текстовая часть, затем по одной части `image_url`, `video_url` или `file_url`
на каждый элемент (константы `Part*` называют типы частей).

```go
resp, err := c.Chat().Create(ctx, client.ChatRequest{
    Model: client.DefaultVisionModel,
    Messages: []client.Message{{
        Role:    "user",
        Content: "What does the chart show, and how does the report explain it?",
        Images:  []string{"https://example.com/chart.png"},
        Files:   []string{"https://example.com/report.pdf"},
        // Videos: []string{"https://example.com/clip.mp4"},
    }},
})
if err != nil {
    return err
}
fmt.Println(resp.Choices[0].Message.Content)
```

Выбирайте модель, чьи возможности в каталоге покрывают отправляемое медиа
(`CapVision`, `CapVideo`, `CapFile`):

```go
entry, ok := client.CatalogEntry(client.DefaultVisionModel)
if ok && slices.Contains(entry.Capabilities, client.CapVideo) {
    // video input is supported
}
```

### Структурированный вывод

У Z.AI нет формата ответа `json_schema`. Запросите вывод в виде JSON-объекта и
поместите schema в системный промпт; `JSONSchemaPrompt` строит эту инструкцию:

```go
schema := json.RawMessage(`{
  "type": "object",
  "properties": {"city": {"type": "string"}, "temp_c": {"type": "number"}},
  "required": ["city", "temp_c"]
}`)
instruction, err := client.JSONSchemaPrompt(schema)
if err != nil {
    return err // the schema is not valid JSON
}
resp, err := c.Chat().Create(ctx, client.ChatRequest{
    Model: client.DefaultModel,
    Messages: []client.Message{
        {Role: "system", Content: instruction},
        {Role: "user", Content: "Current weather in Lisbon?"},
    },
    ResponseFormat: client.JSONObjectFormat(),
})
if err != nil {
    return err
}
var weather struct {
    City  string  `json:"city"`
    TempC float64 `json:"temp_c"`
}
if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &weather); err != nil {
    return err // the model did not return the JSON you asked for
}
```

`JSONObjectFormat()` запрашивает JSON-объект; соответствие schema обеспечивает
промпт, так что проверяйте декодированный результат самостоятельно.

### Вызов функций

Для ручного управления — сами инспектируйте `resp.Choices[0].Message.ToolCalls`
и добавляйте сообщения с `role: "tool"` перед повторным вызовом `Create`. Для
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
if err != nil {
    return err
}
fmt.Println(resp.Choices[0].Message.Content)
```

Он выполняет каждый вызов инструмента, добавляет сообщения assistant + tool
(включая `ReasoningContent` хода assistant) и повторяет, пока модель не вернёт
причину завершения без вызова инструмента или пока не будет превышено
`ToolMaxRounds` (8) — используйте `RunWithToolsLimit`, чтобы задать другой
лимит. Ошибка исполнителя инструмента сообщается модели как результат этого
инструмента (`"error: ..."`), а не возвращается вызывающему коду, поэтому
модель может восстановиться вместо того, чтобы провалить весь обмен.

#### Типы инструментов

`Tool` несёт одну из четырёх полезных нагрузок, выбираемую полем `Type`:

| Конструктор | `Type` | Полезная нагрузка |
|---|---|---|
| `NewFunctionTool(name, desc, params)` | `ToolTypeFunction` (`"function"`) | `FunctionDef` — вызываемая сущность, которую модель вызывает по имени |
| `NewWebSearchTool(engine)` | `ToolTypeWebSearch` (`"web_search"`) | `WebSearchDef` — встроенный веб-поиск, выполняемый на стороне сервера |
| `NewRetrievalTool(knowledgeID, promptTemplate)` | `ToolTypeRetrieval` (`"retrieval"`) | `Retrieval` — база знаний для обоснования ответа |
| `NewMCPTool(label, url, allowedTools...)` | `ToolTypeMCP` (`"mcp"`) | `MCPServer` — удалённый или размещённый Z.AI MCP-сервер, вызываемый на стороне сервера |

Против живого API подтверждён только `function`. `web_search`, `retrieval` и
`mcp` следуют docs.z.ai и официальным SDK и здесь **НЕ ВЕРИФИЦИРОВАНЫ В
LIVE**; см. [Дорожная карта](roadmap.md).

**web_search.** `NewWebSearchTool` строит включённый инструмент, который
возвращает свои источники. Настраивайте его через полезную нагрузку
`WebSearch` (`Count` 1–50, `SearchDomainFilter`, `SearchRecencyFilter`,
`ContentSize`, `SearchPrompt`, `SearchQuery` для принудительного запроса).
`client.SearchEnginePrime` — движок для глобальной платформы;
`client.SearchEngines` перечисляет остальные, которые относятся к китайской
платформе. Это отличается от отдельного эндпоинта `c.Tools().WebSearch`.

```go
search := client.NewWebSearchTool(client.SearchEnginePrime)
search.WebSearch.Count = 5
search.WebSearch.SearchRecencyFilter = client.SearchRecencyOneWeek

resp, err := c.Chat().Create(ctx, client.ChatRequest{
    Model:    client.DefaultModel,
    Messages: []client.Message{{Role: "user", Content: "What changed in Go 1.26?"}},
    Tools:    []client.Tool{search},
})
if err != nil {
    return err
}
for _, src := range resp.WebSearch { // the sources the answer grounded in
    fmt.Println(src.Title, src.Link)
}
```

**retrieval.** Продукт баз знаний обслуживается китайской платформой.
`promptTemplate` может использовать плейсхолдеры `{{knowledge}}` и
`{{question}}`.

```go
kb := client.NewRetrievalTool("your-knowledge-id", "Use {{knowledge}} to answer: {{question}}")
req.Tools = []client.Tool{kb}
```

**mcp.** `ServerLabel` называет сервер; оставьте URL пустым, чтобы использовать
один из размещённых MCP-серверов Z.AI по метке, либо задайте URL для своего.
Необязательный `allowedTools` ограничивает, какие из его инструментов модель
может вызывать. Транспорт по умолчанию — streamable HTTP
(`MCPTransportStreamableHTTP`); альтернатива — `MCPTransportSSE`. Активность
модели в MCP возвращается в `ToolCall.MCP` (`MCPCall`: список инструментов или
вызов с его выводом либо ошибкой).

```go
remote := client.NewMCPTool("docs", "https://mcp.example.com/mcp", "search", "fetch")
remote.MCP.TransportType = client.MCPTransportSSE
remote.MCP.Headers = map[string]string{"Authorization": "Bearer " + os.Getenv("DOCS_MCP_TOKEN")}

hosted := client.NewMCPTool("zread", "") // Z.AI-hosted server, chosen by label
req.Tools = []client.Tool{remote, hosted}
```

Клиент проверяет эти правила перед отправкой, так что вы получаете понятную
локальную ошибку вместо непрозрачной ошибки сервера:

- **Паттерн имени инструмента** — `tools[].function.name` должен
  соответствовать `^[A-Za-z0-9_-]{1,64}$`.
- **Лимит функций** — не более `ToolMaxFunctions` (128) инструментов-функций
  на запрос.
- **Полезная нагрузка по типу** — инструмент должен нести полезную нагрузку
  своего типа: нагрузку `function`, нагрузку `web_search`, нагрузку
  `retrieval` с `knowledge_id` или нагрузку `mcp` с `server_label`.
  Неизвестные типы отклоняются.

#### Поля ответа, которые стоит проверять

Помимо `resp.Choices[0].Message.Content`:

- `resp.Choices[0].FinishReason` — сравнивайте с константами
  `FinishReason*` (`FinishReasonStop`, `FinishReasonToolCalls`,
  `FinishReasonLength`, `FinishReasonSensitive`,
  `FinishReasonModelContextWindowExceeded`, `FinishReasonNetworkError`).
  Последние три — задокументированные значения, сигнализирующие о завершении
  не по контенту.
- `resp.Choices[0].Message.ReasoningContent` — рассуждения модели, для моделей
  с режимом мышления.
- `resp.WebSearch` — массив верхнего уровня `web_search`, который ответ несёт,
  когда сработал инструмент `web_search` (`Link`/`Title`/`Content` каждой
  записи — это источники, на которых основан ответ). НЕ ВЕРИФИЦИРОВАНО В LIVE.
- `resp.Usage` — счётчики token, включая
  `CompletionTokensDetails.ReasoningTokens` и `PromptTokensDetails.CachedTokens`.
  `resp.RequestID` несёт ID запроса платформы.

`Pricing.Cost` превращает `Usage` в сумму в долларах по каталожным ставкам
модели (USD за 1M token), тарифицируя кэшированные prompt-токены по
кэш-ставке. Это оценка по снимку каталога, а не счёт:

```go
if entry, ok := client.CatalogEntry(req.Model); ok && entry.Pricing != nil {
    fmt.Printf("about $%.6f\n", entry.Pricing.Cost(resp.Usage))
}
```

#### Совместимость schema инструментов

Чатовый эндпоинт GLM использует строгий парсер JSON-Schema для `parameters`
инструментов: schema, содержащая `anyOf`, `oneOf`, `allOf` или ссылку
`$ref`/`$defs`, заставляет его вернуть **HTTP 500** вместо пригодной ошибки.
Именно эти конструкции и порождают языки со статической типизацией — поле,
допускающее null, превращается в
`anyOf: [{…}, {"type":"null"}]`, повторно используемая структура — в `$ref`.

По умолчанию клиент переписывает schema инструментов в плоское подмножество,
которое принимает GLM, перед каждым запросом chat, Anthropic и Responses
(объединения с null схлопываются в лежащий в основе тип, `allOf` сливается,
`$ref` инлайнится), сохраняя максимум информации о типах/описаниях. Это no-op
для schema уже в поддерживаемом подмножестве и никогда не мутирует ваши
`req.Tools`.

- Чтобы нормализовать schema самостоятельно (например, вы строите запросы в
  другом месте): `client.SanitizeToolSchemas(tools)`.
- Чтобы пропустить schema без изменений (для отладки или будущего эндпоинта,
  поддерживающего полную версию черновика): установите
  `Config.DisableToolSchemaCompat = true`.

## Anthropic-совместимый Messages API

Z.AI также предоставляет Anthropic-протокольную поверхность на
`/api/anthropic` — тот же эндпоинт, на который GLM Coding Plan направляет
Claude Code. `c.Anthropic()` — типизированный клиент для его
`POST /v1/messages`, параллельный `c.Chat()` для OpenAI-стиля поверхности. Он
вызывает `Region.AnthropicBaseURL()` региона (независимо от `Config.BaseURL`),
аутентифицируется вашим ключом z.ai как Bearer-токеном (не Anthropic-овским
`x-api-key`) и автоматически отправляет заголовок `anthropic-version`.

```go
req := client.AnthropicMessageRequest{
    Model:     client.DefaultModel,
    MaxTokens: 1024, // required by the Messages API
    System:    "You are concise.",
    Messages: []client.AnthropicMessage{
        client.AnthropicTextMessage("user", "Explain goroutines in one line"),
    },
}
resp, err := c.Anthropic().Create(ctx, req)
if err != nil {
    return err
}
fmt.Println(resp.Text()) // concatenated text blocks
```

`Stream` возвращает итератор по сырым SSE-событиям Anthropic
(`message_start`, `content_block_delta`, …) с той же семантикой, что и у
`Chat().Stream`. Каждый `AnthropicStreamEvent` несёт имя события в `Type` и
JSON-полезную нагрузку в `Data`. `Delta()` декодирует событие
`content_block_delta` в `AnthropicDelta` (`Type` — это `text_delta`,
`thinking_delta`, `input_json_delta` или `signature_delta`; `Text`, `Thinking`
и `PartialJSON` несут фрагмент) и сообщает `ok == false` для любого другого
события, которое вы десериализуете из `Data` самостоятельно:

```go
for ev, err := range c.Anthropic().Stream(ctx, req) {
    if err != nil {
        return err // an `event: error` arrives here as an *APIError
    }
    if d, ok := ev.Delta(); ok && d.Type == "text_delta" {
        fmt.Print(d.Text)
    }
}
```

Инструменты, объявленные через `AnthropicTool.InputSchema`, проходят ту же
нормализацию GLM schema, что и чатовые инструменты (см. выше).
`Config.DisableToolSchemaCompat` её отключает.

Расширенное мышление (модели GLM — рассуждающие) включается на уровне запроса
и считывается через `resp.Thinking()`:

```go
req.Thinking = &client.AnthropicThinking{Type: "enabled", BudgetTokens: 2048}
resp, err := c.Anthropic().Create(ctx, req)
if err != nil {
    return err
}
fmt.Println(resp.Thinking()) // thinking blocks, or reasoning_content if the
                             // endpoint surfaces reasoning that way instead
fmt.Println(resp.Text())     // the answer, without the reasoning mixed in
```

Форма ответа успешного пути смоделирована по документированному Messages API
Anthropic и здесь пока не прошла живую верификацию — см. [Дорожная
карта](roadmap.md).

## Responses API (протокол Codex)

Z.AI обслуживает протокол OpenAI Responses на `/api/v1` — эндпоинт, против
которого настроен Codex, документированный для GLM Coding Plan
([docs](https://docs.z.ai/devpack/tool/codex)). `c.Responses()` вызывает
`POST /responses` на `Region.ResponsesBaseURL()` (`https://api.z.ai/api/v1`
либо `https://open.bigmodel.cn/api/v1` для `RegionChina`) независимо от
`Config.BaseURL`. Вход — это список элементов, а не сообщений, а
`Instructions` играет роль системного сообщения:

```go
resp, err := c.Responses().Create(ctx, client.ResponsesRequest{
    Model:        client.DefaultModel,
    Instructions: "You are concise.",
    Input:        []client.ResponsesItem{client.ResponsesMessage("user", "Explain goroutines in one line")},
    Reasoning:    &client.ResponsesReasoning{Effort: client.EffortHigh},
})
if err != nil {
    return err
}
fmt.Println(resp.OutputText())    // the message text
fmt.Println(resp.ReasoningText()) // reasoning summaries, or the full reasoning text
```

- `Reasoning.Effort` проверяется по каталогу, как и
  `ChatRequest.ReasoningEffort`.
- Инструменты-функции используют плоскую форму Responses API
  (`ResponsesTool{Name, Description, Parameters}`; `Type` по умолчанию
  `function`) и проходят то же переписывание schema, что и чатовые
  инструменты. `resp.FunctionCalls()` возвращает вызовы; отвечайте на каждый
  через `client.ResponsesFunctionOutput(call.CallID, output)`, добавляемый в
  `Input` следующего запроса.
- Ответ со `Status` равным `failed` возвращается как `*APIError`.
- `resp.GetUsage()` отображает счётчики token на `client.Usage`, так что хуки
  видят их так же, как использование чата.

`Stream` отдаёт `ResponsesStreamEvent`. Текст приходит в `Delta` в событиях
`ResponsesEventOutputTextDelta`, рассуждения — в
`ResponsesEventReasoningTextDelta` или `ResponsesEventReasoningSummaryDelta`, а
итоговый ответ — в `ResponsesEventCompleted`; события `error` и
`response.failed` завершают поток с `*APIError`:

```go
for ev, err := range c.Responses().Stream(ctx, req) {
    if err != nil {
        return err
    }
    switch ev.Type {
    case client.ResponsesEventOutputTextDelta:
        fmt.Print(ev.Delta)
    case client.ResponsesEventCompleted:
        fmt.Println("\ntokens:", ev.Response.Usage.TotalTokens)
    }
}
```

НЕ ВЕРИФИЦИРОВАНО В LIVE: Z.AI документирует эндпоинт, но не его schema,
поэтому типы следуют Responses API от OpenAI; см. [Дорожная карта](roadmap.md).

## Определение аккаунта и статус

Ключ Z.AI — это либо pay-as-you-go, либо подписка GLM Coding Plan, и у них
разные корни чата. `Detection().DetectAccountType` классифицирует ключ, ничего
не тратя: он вызывает применимый только к coding plan эндпоинт квоты в регионе
клиента, затем в другом регионе (ключ coding plan, выпущенный в Китае, отвечает
только на open.bigmodel.cn). Корректно сформированный ответ с квотой
подтверждает coding plan; если ни один шлюз не опознал подписку, но хотя бы
один ответил, ключ считается pay-as-you-go. Если ни один из шлюзов недоступен,
возвращается транспортная ошибка, а не догадка. Результат кэшируется на
клиента.

```go
acct, err := c.Detection().DetectAccountType(ctx)
if err != nil {
    return err
}
fmt.Println(acct.Type)      // client.AccountTypeCodingPlan or client.AccountTypePayAsYouGo
fmt.Println(acct.Region)    // the gateway the key belongs to
fmt.Println(acct.BaseURL)   // the chat root to use (Region.BaseURLFor)
fmt.Println(acct.Confirmed) // see below
fmt.Println(acct.Level)     // coding-plan tier (lite, pro, max); empty for pay-as-you-go
```

`Confirmed` равно true только для coding plan, который эндпоинт квоты опознаёт
положительно. Ни один эндпоинт не опознаёт ключ pay-as-you-go, поэтому такой
результат — вывод методом исключения и никогда не подтверждается.

`CheckAccountStatus` отвечает на вопрос «работает ли этот ключ и может ли он
тратить прямо сейчас?»:

```go
st, err := c.Detection().CheckAccountStatus(ctx)
if err != nil {
    return err
}
fmt.Println(st.APIAccessible, st.HasBalance, st.Message)
```

Для ключа coding plan это бесплатная проверка квоты (`HasBalance` равно false,
когда окно использования модели исчерпано). У Z.AI нет документированного
вызова баланса для ключей pay-as-you-go, поэтому для них отправляется одно
платное завершение на один token на `client.BalanceProbeModel`, и исход
классифицируется (работает, нет баланса, ограничение скорости, плохой ключ).
Вызывайте по требованию, никогда по таймеру.

Окна квоты приходят из `c.Quota()`; `QuotaData.Exhausted`,
`QuotaLimit.IsModelLimit` и `QuotaLimit.UsedFraction` интерпретируют как окна
на основе кредитов, так и устаревшие токенные окна (см. [Аккаунты и
квоты](accounts-and-quota.md)):

```go
q, err := c.Quota().GetQuotaLimit(ctx)
if err != nil {
    return err
}
for _, l := range q.Data.Limits {
    fmt.Printf("%s: %.0f%% used\n", l.WindowDescription(), l.UsedFraction()*100)
}
```

### Баланс и подписки

НЕ ВЕРИФИЦИРОВАНО В LIVE. `Account().Balance` и `Account().Subscriptions`
вызывают biz API (`Region.BizBaseURL()`), которое Z.AI не документирует;
маршруты и имена полей взяты из собственного клиента ZCode от Z.AI и
общественных инструментов учёта использования (см. [Дорожная карта](roadmap.md)).
Бизнес-сбой, сообщённый внутри HTTP 200, возвращается как `*APIError`. Суммы
баланса указаны в расчётной валюте региона (USD на api.z.ai, CNY на
open.bigmodel.cn); API не присылает поля валюты.

```go
bal, err := c.Account().Balance(ctx)
if err != nil {
    return err
}
fmt.Printf("available %.2f of %.2f\n", bal.AvailableBalance, bal.Balance)

subs, err := c.Account().Subscriptions(ctx) // empty when the account has none
if err != nil {
    return err
}
for _, s := range subs {
    fmt.Println(s.ProductName, s.Status, s.NextRenewTime)
}
```

## Обработка ошибок

Полный справочник по `APIError`, кодам ошибок, ошибкам внутри потока и
поведению повторных попыток по умолчанию — см. [Обработка
ошибок](error-handling.md).

## Хуки наблюдаемости

`Config.Hooks` подключает хуки наблюдаемости (трассировка, метрики,
логирование), срабатывающие на каждый запрос, ответ, ошибку и фрагмент
потока — без необходимости оборачивать `http.RoundTripper`. Интерфейс —
только на stdlib, так что `pkg/client` остаётся без зависимостей; конкретные
реализации живут в `pkg/observe` (OpenTelemetry), либо вы можете написать свою.

```go
import (
    "github.com/SamyRai/go-z-ai/pkg/client"
    "github.com/SamyRai/go-z-ai/pkg/observe"
)

c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    Hooks:  []client.Hook{observe.NewOTelHook("my-service")},
})
```

`Hook` срабатывает:

- `OnRequest(ctx, meta) context.Context` — на каждую попытку, перед HTTP-отправкой;
  возвращённый контекст заменяет входной (здесь прикрепляйте spans
  трассировки).
- `OnResponse(ctx, meta)` — когда попытка успешна (2xx-ответ разобран либо
  поток завершился чисто), с кодом статуса, длительностью и использованием
  token (когда ответ его несёт).
- `OnError(ctx, meta, err)` — когда попытка не удалась: ошибка транспорта,
  не-2xx ответ (включая тот, что вот-вот будет повторён), недекодируемое тело,
  сбой в середине потока или поток, по которому вы перестали итерироваться
  досрочно (`context.Canceled`). На последней попытке `err` — это ошибка,
  которую видит вызывающий.
- `OnStreamChunk(ctx, meta, chunk)` — для каждого фрагмента из
  `Chat().Stream`, `Anthropic().Stream` или `Responses().Stream`; `chunk` — это
  `client.StreamChunk`, `client.AnthropicStreamEvent` или
  `client.ResponsesStreamEvent`.

За каждым `OnRequest` следует ровно один `OnResponse` или `OnError` для той же
попытки, так что span, начатый в `OnRequest`, всегда можно закончить там же.

`RequestMeta` несёт поля `Service`, `Method`, `Endpoint`, `Model` и `Attempt`.
Большинство сервисов проставляют `Service` (и `Model`, когда запрос его несёт)
в запрос автоматически — chat, anthropic, embeddings, audio, files, quota и
другие; остальные оставляют их пустыми, если вы не проставите их сами через
`client.WithService(ctx, "...")` / `client.WithModel(ctx, "...")` перед
вызовом. Значения контекста также переопределяют то, что задаёт сервис.

Nil/пустой срез `Hooks` пропускает любые вызовы — путь без хуков не требует
аллокаций и не имеет измеримых накладных расходов.

## Управление мультиаккаунтными кредами

Хранилище мультиаккаунтных кредов и писатели кредов/конфигурации GLM Coding
Plan живут в `internal/accounts` и `internal/coding`. Они внутренние для этого
модуля — не часть импортируемого публичного API — и потому могут эволюционировать
без semver-ограничений. Публичные пакеты — `pkg/client` и необязательный
`pkg/observe`; команды CLI `accounts` и `coding` — стабильный способ управлять
этой функциональностью. (Раньше эти пакеты лежали в `pkg/` и были
импортируемыми; о переносе см. [CHANGELOG](../../CHANGELOG.md).)

## Тестирование вашего кода против этого клиента

Каждый метод сервиса — это обычная функция на конкретном типе без интерфейсов,
поэтому стандартный подход в Go — направить `Config.BaseURL` на
`httptest.Server`, которым вы управляете. Если хотите воспроизвести *реальный*
записанный трафик Z.AI вместо рукописного стаба, посмотрите, как это делают
тесты самого репозитория с
[go-vcr](https://github.com/dnaeon/go-vcr) — `pkg/client/*_test.go` и
`pkg/client/testdata/cassettes/` — и прочтите
[Участие в проекте § соглашение о живой верификации](../../CONTRIBUTING.md),
чтобы понять, почему.

## Архитектурные заметки

О том, как сервисы устроены внутри (фасад запросов, дизайн повторных
попыток и тайм-аутов, почему часть сервисов обращается к другому хосту), см.
[Архитектура](architecture.md).
