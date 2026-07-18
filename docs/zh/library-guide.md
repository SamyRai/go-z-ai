# 库使用指南

`pkg/client` 是一个独立的 Go 库——CLI 所做的一切，都是通过调用这个 package
完成的。你可以直接依赖它，完全不需要带上 CLI。

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"
```

## 创建客户端

```go
c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
})
```

或者，如果你只想要默认的环境变量配置、不要其他选项：

```go
c, err := client.NewClientFromEnv() // reads ZAI_API_KEY, ZAI_API_BASE_URL
```

`Config` 字段：

| Field | Default | Notes |
|---|---|---|
| `APIKey` | — | 必填 |
| `BaseURL` | `https://api.z.ai/api/paas/v4` | 用于覆盖 coding-plan 端点等 |
| `HTTPClient` | 内部配置的 `*http.Client` | 如需自定义 TLS/代理行为，可传入自己的 transport |
| `Timeout` | 30s | 仅限制拨号/TLS/响应头等待时间——**不**包含整个响应体的读取，因此绝不会截断正在进行的 SSE 流 |
| `MaxRetries` | 3 | 在 429/5xx/网络错误时重试。`-1` 表示完全禁用重试 |
| `RetryDelay` | 200ms | 指数退避的基础延迟 |
| `ChinaAPIKey` | 回退到 `APIKey` | 仅在你持有单独的 bigmodel.cn 专用凭据时才需要——见 [账户与配额](accounts-and-quota.md#regional-gateways-apiza--openbigmodelcn) |
| `Region` | `RegionGlobal` | 选择 monitor/biz/agents/detection 的主机：`RegionGlobal`（api.z.ai）或 `RegionChina`（open.bigmodel.cn）。不会覆盖 `BaseURL`（聊天接口）或 Embeddings/Moderations 的主机。见 [账户与配额](accounts-and-quota.md#regional-gateways-apiza--openbigmodelcn)。 |

每个服务方法的第一个参数都是 `context.Context`，并一路传递到底层的 HTTP
调用——取消它即可中止请求或正在等待的重试退避。

## 服务

`Client` 为每个服务暴露一个方法，全部遵循同样的 `c.<Service>().<Method>(ctx, ...)`
形式：

| Accessor | Covers |
|---|---|
| `c.Chat()` | Completions — `Create`、`CreateAsync`、`CreateStream`、`CreateSimple`、`RunWithTools` |
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
| `c.Embeddings()` | `Create`（路由到 `open.bigmodel.cn`） |
| `c.Moderations()` | `Create`（路由到 `open.bigmodel.cn`） |
| `c.Rerank()` | `Create` |
| `c.Tools()` | `WebSearch`、`WebReader`、`Tokenize` |
| `c.Usage()`、`c.Quota()`、`c.Detection()`、`c.Account()` | GLM Coding Plan 用量/配额/账户监控 |
| `c.GetAsyncResult(ctx, id)`、`c.WaitForResult(ctx, id, interval)` | 异步图像/视频/聊天任务的共享轮询 |

每一项请求校验（必填字段等）都在请求发送之前于客户端完成——对于缺少
`model` 这类问题，你会立刻拿到一个本地 `error`，而不是白白跑一次往返。

## 聊天补全

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

### 流式

```go
err := c.Chat().CreateStream(ctx, req, func(chunk client.StreamChunk) error {
    if len(chunk.Choices) > 0 {
        fmt.Print(chunk.Choices[0].Delta.Content)
    }
    return nil // a non-nil return aborts the stream
})
```

设置 `req.StreamToolCall = true`（GLM-4.6+）可以把工具调用的增量
跨多个事件流式地投放到 `chunk.Choices[0].Delta.ToolCalls` 里，
而不是在该轮结束时一次性整批返回。这在 UI 上展示"模型正在调用工具…"的进度时
很有用。尚未实测验证——见 [路线图](roadmap.md)。

### 异步

```go
task, _ := c.Chat().CreateAsync(ctx, req)
result, err := c.WaitForResult(ctx, task.ID, 3*time.Second)
```

### 视觉（消息中的图像）

```go
req.Messages[len(req.Messages)-1].Images = []string{
    "https://example.com/photo.jpg", // or a data: URI
}
req.Model = "glm-4.6v"
```

### 结构化输出

```go
req.ResponseFormat = client.NewJSONSchemaFormat("my_schema", rawJSONSchema, true /* strict */)
```

### 函数调用

如果需要手动控制，自行检查 `resp.Choices[0].Message.ToolCalls`，
然后在再次调用 `Create` 之前追加 `role: "tool"` 消息。对于常见场景，
`RunWithTools` 会替你驱动这个循环：

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

它会执行每一次工具调用，追加 assistant 和 tool 消息，并循环重复，
直到模型返回非工具类型的 finish reason 或超过 `ToolMaxRounds`（8）——
可用 `RunWithToolsLimit` 设置不同的上限。工具执行器返回的错误会以工具结果
（`"error: ..."`）的形式回传给模型，而不是返回给你的调用方，这样模型可以
自行恢复，整轮对话也不会因此失败。

#### 工具类型

一个 `Tool` 携带三种 payload 之一，由其 `Type` 决定：

| Constructor | `Type` | Payload |
|---|---|---|
| `NewFunctionTool(name, desc, params)` | `ToolTypeFunction`（`"function"`） | `FunctionDef`——模型按名称调用的可执行体 |
| `NewRetrievalTool(knowledgeID, prompt)` | `ToolTypeRetrieval`（`"retrieval"`） | `Retrieval`——用于为回答提供依据的知识库 |
| `NewWebSearchTool(queries...)` | `ToolTypeWebSearch`（`"web_search"`） | `WebSearch`——一个 `search_query` 列表 |

`retrieval` 和 `web_search` 在 docs.z.ai 上有文档，但此处**尚未实测
验证**——只有 `function` 已对照真实 API 确认。`web_search` 的 payload
形态（`{"search_query":[...]}`）遵循官方 Python SDK 示例；见
[路线图](roadmap.md)。

`validateChatRequest` 在客户端强制执行三条文档化的规则，这样你会拿到
清晰的本地错误，而不是服务器返回的晦涩错误：

- **工具名格式**——`tools[].function.name` 必须匹配
  `^[A-Za-z0-9_-]{1,64}$`。
- **函数上限**——每次请求最多 `ToolMaxFunctions`（128）个 function 工具。
- **按类型匹配 payload**——`function`/`retrieval`/`web_search` 工具必须
  携带对应的 payload；未知类型会被拒绝。

#### 值得检查的响应字段

除了 `resp.Choices[0].Message.Content` 之外：

- `resp.Choices[0].FinishReason`——与 `FinishReason*` 常量
  （`FinishReasonStop`、`FinishReasonToolCalls`、`FinishReasonLength`、
  `FinishReasonSensitive`、`FinishReasonModelContextWindowExceeded`、
  `FinishReasonNetworkError`）做比较。后三个是文档化的真实取值，
  表示非内容性终止。
- `resp.WebSearch`——响应在 `web_search` 工具触发时携带的顶层
  `web_search` 数组（每条的 `Link`/`Title`/`Content` 就是回答所依据的
  来源）。尚未实测验证。

#### 工具 schema 兼容性

GLM 的聊天端点对工具 `parameters` 使用严格的 JSON-Schema 解析器：
包含 `anyOf`、`oneOf`、`allOf`，或 `$ref`/`$defs` 引用的 schema 会让
它返回 **HTTP 500**，而不是一个可用的错误。这些构造恰恰就是带类型的语言
所生成的——一个可为 null 的字段会变成
`anyOf: [{…}, {"type":"null"}]`，一个被复用的 struct 会变成 `$ref`。

默认情况下，客户端会在每次聊天请求之前把工具 schema 改写成 GLM 接受的
扁平子集（可为 null 的联合类型坍缩为底层类型、`allOf` 合并、`$ref`
内联展开），并尽量保留类型/描述信息。对于已经在受支持子集内的 schema
这是 no-op，并且永远不会改动你的 `req.Tools`。

- 想自行规范化 schema（例如你在别处构造请求）：
  `client.SanitizeToolSchemas(tools)`。
- 想原样发送 schema（用于调试，或将来某个支持完整 draft 的端点）：
  设置 `Config.DisableToolSchemaCompat = true`。

### Anthropic 兼容的 Messages API

Z.AI 还在 `/api/anthropic` 暴露了一个 Anthropic 协议接口——也就是
GLM Coding Plan 把 Claude Code 指向的同一个端点。`c.Anthropic()`
是其 `POST /v1/messages` 的类型化客户端，与 OpenAI 风格接口的
`c.Chat()` 并列。它用你的 z.ai key 以 Bearer token 形式鉴权（不是
Anthropic 的 `x-api-key`），并自动发送 `anthropic-version` 头。

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

流式传输会交付 Anthropic 原生的 SSE 事件（`message_start`、
`content_block_delta`、……），包含事件名和 JSON payload，你按事件类型
分别 unmarshal：

```go
err := c.Anthropic().CreateStream(ctx, req, func(ev client.AnthropicStreamEvent) error {
    if ev.Type == "content_block_delta" {
        // ev.Data is {"delta":{"type":"text_delta","text":"…"}, …}
    }
    return nil
})
```

通过 `AnthropicTool.InputSchema` 声明的工具会得到与聊天工具相同的 GLM
schema 规范化（见上文）。`Config.DisableToolSchemaCompat` 可关闭它。

深度思考（GLM 模型是推理模型）按请求开启，并用 `resp.Thinking()` 读回：

```go
req.Thinking = &client.AnthropicThinking{Type: "enabled", BudgetTokens: 2048}
resp, _ := c.Anthropic().Create(ctx, req)
fmt.Println(resp.Thinking()) // thinking blocks, or reasoning_content if the
                             // endpoint surfaces reasoning that way instead
fmt.Println(resp.Text())     // the answer, without the reasoning mixed in
```

成功路径上的响应形态是按 Anthropic 文档化的 Messages API 建模的，此处
尚未实测验证——见 [路线图](roadmap.md)。

## 错误处理

完整的 `APIError` 参考、错误码，以及默认的重试行为，请见
[错误处理](error-handling.md)。

## 多账户凭据管理

多账户凭据存储以及 GLM Coding Plan 的凭据/配置写入器位于
`internal/accounts` 和 `internal/coding`。它们对本模块是内部的——不属于
可导入的公开 API——因此可以不受 semver 约束地演进。`pkg/client` 是唯一
受支持的公开 package；`accounts` 和 `coding` CLI 命令是驱动这些功能的
稳定入口。（这些 package 之前位于 `pkg/` 下且可导入；迁移说明见
[CHANGELOG](../../CHANGELOG.md)。）

## 用本客户端测试你自己的代码

每个服务方法都是无接口的具体类型上的普通函数，因此常规的 Go 做法是把
`Config.BaseURL` 指向你控制的 `httptest.Server`。如果你想回放**真实**录制
的 Z.AI 流量而不是手写 stub，可以参考本仓库自身的测试是如何用
[go-vcr](https://github.com/dnaeon/go-vcr) 做的——`pkg/client/*_test.go`
和 `pkg/client/testdata/cassettes/`——并阅读
[贡献指南 § 实测验证约定](../../CONTRIBUTING.md) 了解背后原因。

## 架构说明

关于服务在内部是如何组织的（`doRequest` 门面、重试/退避设计、为何有些服务
针对不同的 base URL 鉴权），请见 [架构](architecture.md)。
