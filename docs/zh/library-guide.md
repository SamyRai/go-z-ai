# 库使用指南

`pkg/client` 是一个独立的 Go 库（仅依赖标准库）——CLI 所做的一切，都是通过调用
这个 package 完成的。你可以直接依赖它，完全不需要带上 CLI。

## 目录

- [创建客户端](#创建客户端)
- [服务](#服务)
- [默认模型](#默认模型)
- [聊天补全](#聊天补全)
- [Anthropic 兼容的 Messages API](#anthropic-兼容的-messages-api)
- [Responses API（Codex 协议）](#responses-apicodex-协议)
- [账户检测与状态](#账户检测与状态)
- [错误处理](#错误处理)
- [可观测性钩子](#可观测性钩子)
- [多账户凭据管理](#多账户凭据管理)
- [用本客户端测试你自己的代码](#用本客户端测试你自己的代码)
- [架构说明](#架构说明)

```bash
go get github.com/SamyRai/go-z-ai
```

本模块需要 Go 1.26（见 `go.mod`）。

```go
import "github.com/SamyRai/go-z-ai/pkg/client"
```

## 创建客户端

```go
c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
})
if err != nil {
    log.Fatal(err)
}
```

或者，从环境变量配置一切——与 CLI 所遵循的变量相同：

```go
c, err := client.NewClientFromEnv()
if err != nil {
    log.Fatal(err)
}
```

| Variable | 对应 | 说明 |
|---|---|---|
| `ZAI_API_KEY` | `Config.APIKey` | 必填 |
| `ZAI_API_BASE_URL` | `Config.BaseURL` | 可选，覆盖 chat/PaaS 根地址 |
| `ZAI_REGION` | `Config.Region` | 用 `client.ParseRegion` 解析：`global`（默认）或 `china` |
| `ZAI_CHINA_API_KEY` | `Config.ChinaAPIKey` | 可选，单独的 bigmodel.cn 凭据 |
| `ZAI_MONITOR_TIMEZONE` | `Config.MonitorTimezone` | IANA 名称、`UTC`、形如 `UTC+8` 的偏移量，或 `local`；无效值会报错 |

`Config` 字段：

| Field | Default | Notes |
|---|---|---|
| `APIKey` | — | 必填 |
| `BaseURL` | `Region.PaaSBaseURL()`——global 区域为 `https://api.z.ai/api/paas/v4` | 显式设置的值永远优先。Coding Plan 的 key 使用 `Region.CodingBaseURL()`（或 `DetectedAccount.BaseURL`，见 [账户检测与状态](#账户检测与状态)）。 |
| `HTTPClient` | 内部配置的 `*http.Client` | 如需自定义 TLS/代理行为，可传入自己的 transport |
| `Timeout` | 30s（`DefaultTimeout`） | 仅限制拨号/TLS/响应头等待时间——**不**包含整个响应体的读取，因此绝不会截断正在进行的 SSE 流 |
| `MaxRetries` | 3 | 在 429/5xx/网络错误时重试。`-1` 表示完全禁用重试 |
| `RetryDelay` | 200ms | 指数退避的基础延迟 |
| `ChinaAPIKey` | 回退到 `APIKey` | 仅在你持有单独的 bigmodel.cn 专用凭据时才需要——见 [账户与配额](accounts-and-quota.md#区域网关apizai--openbigmodelcn) |
| `Region` | `RegionGlobal` | 选择网关：`RegionGlobal`（api.z.ai）或 `RegionChina`（open.bigmodel.cn）。见 [区域](#区域)。 |
| `MonitorTimezone` | `client.MonitorServerTZ`（UTC+8） | monitor（配额/用量）API 中不带时区的时间戳所采用的时区。仅当实际抓包显示服务器时区不同时才覆盖它。 |
| `DisableToolSchemaCompat` | `false` | 原样发送工具 schema——见 [工具 schema 兼容性](#工具-schema-兼容性) |
| `UserAgent` | `"go-z-ai/<version>"` | 覆盖每个请求所发送的 `User-Agent` 头。默认值会向 Z.AI 的 API 标明 go-z-ai 的身份——这对 coding 端点的 [使用政策](coding-tools.md#合规与使用政策-) 很重要。仅当你需要一个不同的标识（下游应用、代理、MCP 服务器）时才覆盖；覆盖的字符串会原样发送。 |
| `Hooks` | `nil` | 在每次请求/响应/错误/流数据块时触发的可观测性钩子（[`Hook`](library-guide.md#可观测性钩子)）。默认为空——无钩子路径零成本。具体实现位于 `pkg/observe`（OpenTelemetry）。 |

每个服务方法的第一个参数都是 `context.Context`，并一路传递到底层的 HTTP
调用——取消它即可中止请求或正在等待的重试退避。

### 区域

`Region` 拥有每一个网关 URL；库中其他任何地方都不会硬编码主机。`Config.Region`
为区域相关的服务选择主机——monitor（配额/用量）、biz（账户）、agents、Anthropic
兼容的 Messages API、Responses API 和账户类型检测——并且当 `BaseURL` 为空时，它
也决定默认的 chat 根。`c.Region()` 返回已配置的值；空的或无法识别的 `Region`
都表示 global。

| 方法 | `Region.Host()` 下的路径 |
|---|---|
| `PaaSBaseURL()` | `/api/paas/v4`——pay-as-you-go，OpenAI 兼容 |
| `CodingBaseURL()` | `/api/coding/paas/v4`——GLM Coding Plan |
| `AnthropicBaseURL()` | `/api/anthropic` |
| `ResponsesBaseURL()` | `/api/v1`——OpenAI Responses 协议（Codex 所使用的） |
| `MonitorBaseURL()` | `/api/monitor`——配额/用量 |
| `BizBaseURL()` | `/api/biz`——账户 |
| `AgentsBaseURL()` | `/api` |
| `MCPServerURL(name)` | `/api/mcp/<name>/mcp`——Z.AI 托管的 MCP 服务器 |

`Host()` 是 `https://api.z.ai` 或 `https://open.bigmodel.cn`（`client.GlobalHost`
/ `client.ChinaHost`）；`ConsoleURL()` 是 Web 控制台（`https://z.ai` /
`https://bigmodel.cn`）；`BaseURLFor(accountType)` 会为某个 `AccountType` 返回
coding 或 PaaS 的 chat 根。`client.ParseRegion` 接受 `china`（别名 `cn`、
`bigmodel`）和 `global`（别名 `west`，以及空值），不区分大小写；未知值会被解析为
global，而不是失败。

```go
c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    Region: client.RegionChina, // chat, quota, account, agents, Anthropic -> open.bigmodel.cn
})
if err != nil {
    log.Fatal(err)
}
```

Embeddings 和 Moderations 是例外：无论 `Region` 如何，它们始终使用
`ChinaAPIKey` 调用 `client.BigModelBaseURL`（open.bigmodel.cn）。monitor/biz/agents
路由在中国镜像上的情况尚未实测验证——见 [路线图](roadmap.md)。

## 服务

`Client` 为每个服务暴露一个方法，全部遵循同样的 `c.<Service>().<Method>(ctx, ...)`
形式：

| Accessor | Covers |
|---|---|
| `c.Chat()` | Completions — `Create`、`CreateAsync`、`Stream`、`RunWithTools`、`RunWithToolsLimit` |
| `c.Models()` | `List`、`Get`、`GetTextModels`、`GetVisionModels`、`GetFreeModels`、`RefreshCache` |
| `c.Images()` | `Generate`、`GenerateAsync` |
| `c.Videos()` | `Generate`（始终异步） |
| `c.Audio()` | `Transcribe`、`Speech` |
| `c.Voice()` | `Clone`、`Delete`、`List`——GLM-TTS 声音克隆 |
| `c.Layout()` | `Parse`、`HandwritingOCR` |
| `c.FileParser()` | `Create`、`Sync`、`Result`——用于 RAG 的文档转文本 |
| `c.Files()` | `Upload`、`List`、`Delete`、`Content` |
| `c.Batch()` | `Create`、`Retrieve`、`List`、`Cancel` |
| `c.Agents()` | `Invoke`、`AsyncResult` |
| `c.Anthropic()` | `Create`、`Stream`——Anthropic 协议 `/v1/messages` 接口 |
| `c.Responses()` | `Create`、`Stream`——OpenAI Responses 协议 `/api/v1/responses`，即 Codex 所使用的接口 |
| `c.Embeddings()` | `Create`（路由到 `open.bigmodel.cn`） |
| `c.Moderations()` | `Create`（路由到 `open.bigmodel.cn`） |
| `c.Rerank()` | `Create` |
| `c.Tools()` | `WebSearch`、`WebReader`、`Tokenize` |
| `c.Quota()` | GLM Coding Plan 配额与用量——`GetQuotaLimit`、`GetModelUsage`、`GetToolUsage` |
| `c.Detection()` | `DetectAccountType`、`CheckAccountStatus` |
| `c.Account()` | `Balance`、`Subscriptions` |
| `c.GetAsyncResult(ctx, id)`、`c.WaitForResult(ctx, id, interval)` | 异步图像/视频/聊天任务的共享轮询 |

每一项请求校验（必填字段等）都在请求发送之前于客户端完成——对于缺少
`model` 这类问题，你会立刻拿到一个本地 `error`，而不是白白跑一次往返。

客户端没有用量追踪器：请用 `Pricing.Cost` 计算一次请求的成本（见
[值得检查的响应字段](#值得检查的响应字段)），并从 `c.Quota()` 读取套餐的消耗情况。

## 默认模型

请使用这些常量，而不要硬编码模型 ID；它们与 CLI、TUI 和编码工具配置写入器所用
的默认值相同。它们背后的目录（`pkg/client/models_catalog.go`）还为每个模型携带
上下文大小、输出上限、价格、能力和可接受的推理强度；`client.CatalogEntry(id)`
返回其中一行，而 `c.Models().List` 会把目录合并进线上的 `/models` 列表。

| 常量 | 模型 | 用途 |
|---|---|---|
| `DefaultModel` | `glm-5.3` | 用于推理、编码和 agent 工作的旗舰文本模型（100 万上下文） |
| `DefaultFastModel` | `glm-5.3-flash` | 快速、低成本档位；原生多模态 |
| `DefaultVisionModel` | `glm-5.3-flash` | 与 `DefaultFastModel` 相同：图像、视频和文件输入 |
| `DefaultOCRModel` | `glm-ocr` | `Layout().Parse` |
| `DefaultASRModel` | `glm-asr-2512` | `Audio().Transcribe` |
| `DefaultTTSModel` | `glm-tts` | `Audio().Speech` |

图像模型是 `client.ModelGLMImage`（默认；唯一支持 `GenerateAsync` 的）和
`client.ModelCogView4`。视频模型位于 `client.VideoModels` 中，以
`client.ModelCogVideoX3` 为首。

## 聊天补全

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

`Temperature`、`TopP` 和 `MaxTokens` 为零时默认使用服务器端的取值；
`Temperature` 和 `TopP` 必须在 [0, 1] 范围内。`DoSample` 是一个 `*bool`，因此
显式的 `false` 也会被发送。`RequestID` 和 `UserID`（6–128 个字符，用于滥用监控）
是可选的。

### 流式

`Stream` 返回一个你可以 range 遍历的迭代器（`iter.Seq2[StreamChunk, error]`）；
它会替你设置 `stream=true`：

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

流在循环内部同步运行——没有生产者 goroutine。跳出循环会停止读取并关闭连接，
而取消 `ctx` 会中止一次被阻塞的读取。连接阶段的瞬时失败（429/5xx/网络）会像
`Create` 一样最多重试到 `Config.MaxRetries`；一旦流已经开始，失败会作为终止的
`err` 抛出，并且绝不重试，因为响应的一部分已经被消费了。校验失败的请求会在发送
任何内容之前，从迭代器中产出一个错误。

推理内容通过 `chunk.Choices[0].Delta.ReasoningContent` 到达，最后一个数据块携带
`chunk.Usage`。

#### 带内流错误

服务器可能在 HTTP 200 已经发出之后才报告失败：OpenAI 风格的 `{"error": {...}}`
数据块、Anthropic 的 `event: error`，或 Responses 的 `error` /
`response.failed` 事件。客户端会把它们各自转换为一个 `*APIError`（其
`HTTPStatus` 为 200），并以它作为终止的 `err` 结束该流，因此同一套 `errors.As`
处理对两种失败都适用：

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

带内错误即使其错误码被标记为可重试也不会被重试：流已经开始了。见
[错误处理](error-handling.md)。

#### 流式工具调用

设置 `req.ToolStream = true`（GLM-4.6 及更高版本）可以把工具调用的参数增量地
跨多个事件投放到 `chunk.Choices[0].Delta.ToolCalls` 里，而不是在该轮结束时一次性
整批返回。同一次调用的各个片段共享同一个 `ToolCall.Index`；把它们的
`Function.Arguments` 拼接起来即可。这在 UI 上展示"模型正在调用工具…"的进度时
很有用。尚未实测验证——见 [路线图](roadmap.md)。

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

### 推理强度

`ChatRequest.ReasoningEffort` 设定思考型模型推理的力度。请使用 `Effort*` 常量
（`EffortMax`、`EffortXhigh`、`EffortHigh`、`EffortMedium`、`EffortLow`、
`EffortMinimal`、`EffortNone`）；`client.AllEfforts` 列出了全部。API 默认值为
`max`。

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

该值会在客户端按模型在目录中的条目进行校验（`ModelCatalogEntry.ReasoningEfforts`，
同时也通过 `ModelDetails.ReasoningEfforts` 暴露），因此不受支持的等级会是一个
本地错误，而不是一次服务器往返：

- `glm-5.2` 接受所有等级。
- GLM-5.3 系列（`glm-5.3`、`glm-5.3-flash`、`glm-5.3-flashx`）只接受 `low`、
  `high` 和 `max`，并且始终会思考——这里不支持关闭思考。
- 目录中收录、但没有强度列表的模型（例如 `glm-5.1`）会拒绝这个参数。
- 不在目录中的模型只会按 `AllEfforts` 检查。

服务器会把 `none`/`minimal` 规范化为"跳过思考"，`low`/`medium` 规范化为
`high`，`xhigh` 规范化为 `max`。这些等级遵循 docs.z.ai；此处尚未对它们做实测验证。

若要跨轮次保留推理（保留式思考，preserved thinking），请把
`ThinkingConfig.ClearThinking` 设为 false，并把每个 assistant 轮次的
`ReasoningContent` 原样回传。`RunWithTools` 会替你做这件事。

```go
keep := false
req.Thinking = &client.ThinkingConfig{Type: client.ThinkingEnabled, ClearThinking: &keep}
req.Messages = append(req.Messages, client.Message{
    Role:             "assistant",
    Content:          msg.Content,
    ReasoningContent: msg.ReasoningContent,
})
```

### 异步

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

`WaitForResult` 在任务离开 `PROCESSING` 时返回，无论成功与否，所以请检查
`TaskStatus`（`TaskStatusSuccess` / `TaskStatusFail`）。图像和视频任务使用同样的
轮询。图像任务的结果位于 `AsyncResultResponse.ImageResult`（每一项是一个
`GeneratedImage`，其 `URL` 在 30 天后过期）；视频任务的结果位于 `VideoResult`：

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

`VideoGenerationRequest.OffPeak` 会把视频任务排入非高峰时段处理，价格更低。

### 多模态消息（图像、视频、文件）

设置 `Message.Images`、`Message.Videos` 或 `Message.Files` 即可附加媒体。每个条目
是一个 `https://` URL 或一个 `data:` URI。客户端会替你把该消息的 `content` 切换
为 content-parts 的 wire 形式：先是文本部分，然后每个条目对应一个 `image_url`、
`video_url` 或 `file_url` 部分（`Part*` 常量给出了这些部分的类型名）。

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

请选择其目录能力涵盖你所发送媒体的模型（`CapVision`、`CapVideo`、`CapFile`）：

```go
entry, ok := client.CatalogEntry(client.DefaultVisionModel)
if ok && slices.Contains(entry.Capabilities, client.CapVideo) {
    // video input is supported
}
```

### 结构化输出

Z.AI 没有 `json_schema` 响应格式。请请求 JSON 对象输出，并把 schema 放进系统
提示词；`JSONSchemaPrompt` 会构造这条指令：

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

`JSONObjectFormat()` 请求一个 JSON 对象；对 schema 的符合程度来自提示词，所以请
自行校验解码后的结果。

### 函数调用

如果需要手动控制，自行检查 `resp.Choices[0].Message.ToolCalls`，然后在再次调用
`Create` 之前追加 `role: "tool"` 消息。对于常见场景，`RunWithTools` 会替你驱动
这个循环：

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

它会执行每一次工具调用，追加 assistant 和 tool 消息（包括该 assistant 轮次的
`ReasoningContent`），并循环重复，直到模型返回非工具类型的 finish reason 或超过
`ToolMaxRounds`（8）——可用 `RunWithToolsLimit` 设置不同的上限。工具执行器返回的
错误会以工具结果（`"error: ..."`）的形式回传给模型，而不是返回给你的调用方，这样
模型可以自行恢复，整轮对话也不会因此失败。

#### 工具类型

一个 `Tool` 携带四种 payload 之一，由其 `Type` 决定：

| Constructor | `Type` | Payload |
|---|---|---|
| `NewFunctionTool(name, desc, params)` | `ToolTypeFunction`（`"function"`） | `FunctionDef`——模型按名称调用的可执行体 |
| `NewWebSearchTool(engine)` | `ToolTypeWebSearch`（`"web_search"`） | `WebSearchDef`——内置网页搜索，在服务端运行 |
| `NewRetrievalTool(knowledgeID, promptTemplate)` | `ToolTypeRetrieval`（`"retrieval"`） | `Retrieval`——用于为回答提供依据的知识库 |
| `NewMCPTool(label, url, allowedTools...)` | `ToolTypeMCP`（`"mcp"`） | `MCPServer`——远程的或 Z.AI 托管的 MCP 服务器，在服务端调用 |

只有 `function` 已对照真实 API 确认。`web_search`、`retrieval` 和 `mcp` 遵循
docs.z.ai 和官方 SDK，此处**尚未实测验证**；见 [路线图](roadmap.md)。

**web_search。** `NewWebSearchTool` 构造一个已启用、并会返回其来源的工具。可通过
`WebSearch` payload 进行调优（`Count` 1–50、`SearchDomainFilter`、
`SearchRecencyFilter`、`ContentSize`、`SearchPrompt`，以及用来强制指定查询的
`SearchQuery`）。`client.SearchEnginePrime` 是 global 平台的引擎；
`client.SearchEngines` 列出的其他引擎属于中国平台。这与独立的
`c.Tools().WebSearch` 端点不同。

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

**retrieval。** 知识库产品由中国平台提供。`promptTemplate` 可以使用
`{{knowledge}}` 和 `{{question}}` 占位符。

```go
kb := client.NewRetrievalTool("your-knowledge-id", "Use {{knowledge}} to answer: {{question}}")
req.Tools = []client.Tool{kb}
```

**mcp。** `ServerLabel` 为服务器命名；URL 留空则按标签使用 Z.AI 托管的 MCP
服务器之一，或者给出一个 URL 来使用你自己的服务器。可选的 `allowedTools` 用于限制
模型可以调用它的哪些工具。传输方式默认是 streamable HTTP
（`MCPTransportStreamableHTTP`）；`MCPTransportSSE` 是另一种选择。模型的 MCP
活动会在 `ToolCall.MCP`（`MCPCall`：工具列表或一次带有输出或错误的调用）中返回。

```go
remote := client.NewMCPTool("docs", "https://mcp.example.com/mcp", "search", "fetch")
remote.MCP.TransportType = client.MCPTransportSSE
remote.MCP.Headers = map[string]string{"Authorization": "Bearer " + os.Getenv("DOCS_MCP_TOKEN")}

hosted := client.NewMCPTool("zread", "") // Z.AI-hosted server, chosen by label
req.Tools = []client.Tool{remote, hosted}
```

客户端会在发送前校验这些规则，这样你会拿到清晰的本地错误，而不是服务器返回的
晦涩错误：

- **工具名格式**——`tools[].function.name` 必须匹配
  `^[A-Za-z0-9_-]{1,64}$`。
- **函数上限**——每次请求最多 `ToolMaxFunctions`（128）个 function 工具。
- **按类型匹配 payload**——工具必须携带与其类型对应的 payload：一个
  `function` payload、一个 `web_search` payload、带有 `knowledge_id` 的
  `retrieval` payload，或带有 `server_label` 的 `mcp` payload。未知类型会被拒绝。

#### 值得检查的响应字段

除了 `resp.Choices[0].Message.Content` 之外：

- `resp.Choices[0].FinishReason`——与 `FinishReason*` 常量
  （`FinishReasonStop`、`FinishReasonToolCalls`、`FinishReasonLength`、
  `FinishReasonSensitive`、`FinishReasonModelContextWindowExceeded`、
  `FinishReasonNetworkError`）做比较。后三个是文档化的取值，表示非内容性终止。
- `resp.Choices[0].Message.ReasoningContent`——模型的推理内容（针对思考型模型）。
- `resp.WebSearch`——响应在 `web_search` 工具触发时携带的顶层 `web_search` 数组
  （每条的 `Link`/`Title`/`Content` 就是回答所依据的来源）。尚未实测验证。
- `resp.Usage`——token 计数，包括 `CompletionTokensDetails.ReasoningTokens` 和
  `PromptTokensDetails.CachedTokens`。`resp.RequestID` 携带平台的请求 ID。

`Pricing.Cost` 会按某个模型在目录中的费率（每 100 万 token 的美元价格）把一个
`Usage` 换算成美元金额，并按缓存费率对缓存的 prompt token 计费。它是基于目录
快照的估算，而不是账单：

```go
if entry, ok := client.CatalogEntry(req.Model); ok && entry.Pricing != nil {
    fmt.Printf("about $%.6f\n", entry.Pricing.Cost(resp.Usage))
}
```

#### 工具 schema 兼容性

GLM 的聊天端点对工具 `parameters` 使用严格的 JSON-Schema 解析器：包含 `anyOf`、
`oneOf`、`allOf`，或 `$ref`/`$defs` 引用的 schema 会让它返回 **HTTP 500**，而不是
一个可用的错误。这些构造恰恰就是带类型的语言所生成的——一个可为 null 的字段会变成
`anyOf: [{…}, {"type":"null"}]`，一个被复用的 struct 会变成 `$ref`。

默认情况下，客户端会在每次 chat、Anthropic 和 Responses 请求之前，把工具 schema
改写成 GLM 接受的扁平子集（可为 null 的联合类型坍缩为底层类型、`allOf` 合并、
`$ref` 内联展开），并尽量保留类型/描述信息。对于已经在受支持子集内的 schema 这是
no-op，并且永远不会改动你的 `req.Tools`。

- 想自行规范化 schema（例如你在别处构造请求）：
  `client.SanitizeToolSchemas(tools)`。
- 想原样发送 schema（用于调试，或将来某个支持完整 draft 的端点）：
  设置 `Config.DisableToolSchemaCompat = true`。

## Anthropic 兼容的 Messages API

Z.AI 还在 `/api/anthropic` 暴露了一个 Anthropic 协议接口——也就是 GLM Coding Plan
把 Claude Code 指向的同一个端点。`c.Anthropic()` 是其 `POST /v1/messages` 的类型化
客户端，与 OpenAI 风格接口的 `c.Chat()` 并列。它调用所在区域的
`Region.AnthropicBaseURL()`（独立于 `Config.BaseURL`），用你的 z.ai key 以 Bearer
token 形式鉴权（不是 Anthropic 的 `x-api-key`），并自动发送 `anthropic-version`
头。

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

`Stream` 返回一个遍历 Anthropic 原生 SSE 事件（`message_start`、
`content_block_delta`、……）的迭代器，语义与 `Chat().Stream` 相同。每个
`AnthropicStreamEvent` 在 `Type` 中携带事件名，在 `Data` 中携带 JSON payload。
`Delta()` 会把一个 `content_block_delta` 事件解码为 `AnthropicDelta`（`Type` 为
`text_delta`、`thinking_delta`、`input_json_delta` 或 `signature_delta`；`Text`、
`Thinking` 和 `PartialJSON` 携带片段内容），对其他任何事件则报告 `ok == false`，
这些事件你需要自行从 `Data` 中 unmarshal：

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

通过 `AnthropicTool.InputSchema` 声明的工具会得到与聊天工具相同的 GLM schema 规范化
（见上文）。`Config.DisableToolSchemaCompat` 可关闭它。

扩展思考（GLM 模型是推理模型）按请求开启，并用 `resp.Thinking()` 读回：

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

成功路径上的响应形态是按 Anthropic 文档化的 Messages API 建模的，此处尚未实测
验证——见 [路线图](roadmap.md)。

## Responses API（Codex 协议）

Z.AI 在 `/api/v1` 提供 OpenAI Responses 协议——也就是 Codex 所配置的端点，文档面向
GLM Coding Plan（[文档](https://docs.z.ai/devpack/tool/codex)）。`c.Responses()` 会对
`Region.ResponsesBaseURL()`（`https://api.z.ai/api/v1`，`RegionChina` 则为
`https://open.bigmodel.cn/api/v1`）调用 `POST /responses`，独立于
`Config.BaseURL`。输入是一个 item 列表而不是 messages，`Instructions` 起到系统消息
的作用：

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

- `Reasoning.Effort` 会像 `ChatRequest.ReasoningEffort` 一样，按目录进行校验。
- Function 工具使用 Responses API 的扁平形态（`ResponsesTool{Name, Description,
  Parameters}`；`Type` 默认为 `function`），并得到与聊天工具相同的 schema 改写。
  `resp.FunctionCalls()` 返回这些调用；请用追加到下一次请求 `Input` 中的
  `client.ResponsesFunctionOutput(call.CallID, output)` 来回应每一个调用。
- `Status` 为 `failed` 的响应会以 `*APIError` 的形式返回。
- `resp.GetUsage()` 把 token 计数映射到 `client.Usage`，这样钩子看到它们的方式与
  聊天用量相同。

`Stream` 产出 `ResponsesStreamEvent`。文本通过 `ResponsesEventOutputTextDelta` 事件的
`Delta` 到达，推理内容通过 `ResponsesEventReasoningTextDelta` 或
`ResponsesEventReasoningSummaryDelta` 到达，最终响应通过
`ResponsesEventCompleted` 到达；`error` 和 `response.failed` 事件会以一个
`*APIError` 结束该流：

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

尚未实测验证：Z.AI 为该端点提供了文档，但没有给出它的 schema，因此这些类型遵循
OpenAI 的 Responses API；见 [路线图](roadmap.md)。

## 账户检测与状态

一把 Z.AI key 要么是 pay-as-you-go，要么是 GLM Coding Plan 订阅，两者使用不同的
chat 根。`Detection().DetectAccountType` 在不花费任何费用的情况下对 key 分类：它先
在客户端所在区域上调用 coding-plan 专属的 quota 端点，再到另一个区域上调用（中国
签发的 coding-plan key 只会在 open.bigmodel.cn 上应答）。得到结构良好的 quota 响应
即确认为 coding plan；如果两个网关都没能识别出订阅，但至少有一个网关作了应答，则
该 key 被视为 pay-as-you-go。如果两个网关都无法连通，则返回传输错误，而不是瞎猜。
结果会按客户端缓存。

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

只有 coding plan 的 `Confirmed` 才为 true，因为 quota 端点能正向识别它。没有任何
端点能识别 pay-as-you-go 的 key，所以那个结果是一种排除法的推断，永远不会被确认。

`CheckAccountStatus` 回答"这把 key 能用吗？现在能花钱吗？"：

```go
st, err := c.Detection().CheckAccountStatus(ctx)
if err != nil {
    return err
}
fmt.Println(st.APIAccessible, st.HasBalance, st.Message)
```

对 coding-plan 的 key 来说，它是一次免费的 quota 检查（当某个 model-usage 窗口已用
尽时 `HasBalance` 为 false）。Z.AI 没有针对 pay-as-you-go key 的文档化余额查询调用，
所以对这类 key，它会在 `client.BalanceProbeModel` 上发送一次计费的、单 token 的补全
请求，并对结果分类（可用、无余额、被限流、key 无效）。请按需调用，绝不要放在定时器
里。

配额窗口来自 `c.Quota()`；`QuotaData.Exhausted`、`QuotaLimit.IsModelLimit` 和
`QuotaLimit.UsedFraction` 会同时解读基于 credit 的窗口和旧版 token 窗口（见
[账户与配额](accounts-and-quota.md)）：

```go
q, err := c.Quota().GetQuotaLimit(ctx)
if err != nil {
    return err
}
for _, l := range q.Data.Limits {
    fmt.Printf("%s: %.0f%% used\n", l.WindowDescription(), l.UsedFraction()*100)
}
```

### 余额与订阅

尚未实测验证。`Account().Balance` 和 `Account().Subscriptions` 调用 biz API
（`Region.BizBaseURL()`），Z.AI 并未为其提供文档；路由和字段名来自 Z.AI 自家的
ZCode 客户端以及社区的用量工具（见 [路线图](roadmap.md)）。在 HTTP 200 内报告的
业务失败会以 `*APIError` 的形式返回。余额金额以该区域的计费币种表示（api.z.ai 为
USD，open.bigmodel.cn 为 CNY）；API 不发送币种字段。

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

## 错误处理

完整的 `APIError` 参考、错误码、带内流错误，以及默认的重试行为，请见
[错误处理](error-handling.md)。

## 可观测性钩子

`Config.Hooks` 用来挂载可观测性钩子（追踪、指标、日志），它们会在每次请求、响应、
错误和流数据块上触发——无需你去包装 `http.RoundTripper`。该接口仅依赖标准库，因此
`pkg/client` 能保持零依赖；具体实现位于 `pkg/observe`（OpenTelemetry），你也可以
自己编写。

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

`Hook` 会在以下时机触发：

- `OnRequest(ctx, meta) context.Context`——每次尝试在 HTTP 发送之前；返回的
  context 会取代输入的 context（在这里挂载追踪 span）。
- `OnResponse(ctx, meta)`——一次尝试成功时（已解析出 2xx 响应，或流正常结束），
  附带状态码、耗时和 token 用量（当响应携带时）。
- `OnError(ctx, meta, err)`——一次尝试失败时：传输错误、非 2xx 响应（包括即将被
  重试的那种）、无法解码的响应体、流中途失败，或你提前停止 range 遍历的流
  （`context.Canceled`）。在最后一次尝试上，`err` 就是调用方所看到的错误。
- `OnStreamChunk(ctx, meta, chunk)`——对来自 `Chat().Stream`、`Anthropic().Stream`
  或 `Responses().Stream` 的每个数据块触发；`chunk` 是一个 `client.StreamChunk`、
  一个 `client.AnthropicStreamEvent` 或一个 `client.ResponsesStreamEvent`。

每个 `OnRequest` 之后，对同一次尝试都恰好跟着一个 `OnResponse` 或 `OnError`，因此
在 `OnRequest` 中开启的 span 总是能在那里被结束。

`RequestMeta` 携带 `Service`、`Method`、`Endpoint`、`Model` 和 `Attempt` 字段。
大多数服务会自动把 `Service`（以及请求携带的 `Model`）写入请求——chat、anthropic、
embeddings、audio、files、quota 等；其余服务则将它们留空，除非你在调用前通过
`client.WithService(ctx, "...")` / `client.WithModel(ctx, "...")` 自己写入。这些
context 值也会覆盖服务自己设置的内容。

nil/空的 `Hooks` 切片会跳过所有调用——无钩子路径零分配，没有可测量的开销。

## 多账户凭据管理

多账户凭据存储以及 GLM Coding Plan 的凭据/配置写入器位于 `internal/accounts` 和
`internal/coding`。它们对本模块是内部的——不属于可导入的公开 API——因此可以不受
semver 约束地演进。公开的 package 是 `pkg/client` 和可选的 `pkg/observe`；
`accounts` 和 `coding` CLI 命令是驱动这些功能的稳定入口。（这些 package 之前位于
`pkg/` 下且可导入；迁移说明见 [CHANGELOG](../../CHANGELOG.md)。）

## 用本客户端测试你自己的代码

每个服务方法都是无接口的具体类型上的普通函数，因此常规的 Go 做法是把
`Config.BaseURL` 指向你控制的 `httptest.Server`。如果你想回放**真实**录制的 Z.AI
流量而不是手写 stub，可以参考本仓库自身的测试是如何用
[go-vcr](https://github.com/dnaeon/go-vcr) 做的——`pkg/client/*_test.go` 和
`pkg/client/testdata/cassettes/`——并阅读
[贡献指南 § 实测验证约定](../../CONTRIBUTING.md) 了解背后原因。

## 架构说明

关于服务在内部是如何组织的（请求门面、重试与超时设计、为何有些服务会访问不同的
主机），请见 [架构](architecture.md)。
