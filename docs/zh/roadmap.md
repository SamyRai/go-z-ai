# 路线图与已知限制

这里记录还未收尾的事项、原因，以及让它收尾的确切动作。当某条事项被一个已提交的
cassette 或一次已合并的改动解决后，它就会从本页移除——这是 verify-first（先验证）
约定的工作清单（缘由见
[贡献指南 § the live-verification convention](../../CONTRIBUTING.md) 和
[架构 § 实测验证约定](architecture.md#实测验证约定)）。

模型目录、API 结构和 Coding Plan 的行为最后一次刷新是在 2026-10-02。那次调研期间
无法直接抓取 `docs.z.ai`，因此这些事实来自对官方文档的搜索结果摘要、官方 SDK 和
npm 包，以及（已注明之处）第三方来源——这正是下面有若干条目被列为未验证、而不是
已确认的原因。

> **自行录制 cassette：** 下方每一条点名了某个 `TestVerify*` 测试的事项，都对应
> 一个在你用 `ZAI_RECORD=1` 录制一份成功路径 cassette 之前会一直 SKIP 的测试。
> 测试框架在保存前会把 `Authorization` 改写为 `Bearer REDACTED`——提交前用
> `grep "Bearer " pkg/client/testdata/cassettes/<name>.yaml` 确认一下。
>
> ```sh
> ZAI_RECORD=1 ZAI_API_KEY=<real-key> go test -run TestVerify<Name> ./pkg/client
> ```

## 未实测

分三组：**服务**——其成功路径的响应结构尚未被录制；**chat API 字段**——它们与文档
一致，但尚未被某个 cassette 固定下来；以及**属于快照的数据**——需要定期刷新。

### 仍需成功路径 cassette 的服务

目前使用的开发账户没有这些服务的 PAYG 余额 / 权益，因此只验证了请求结构和错误路径。
一份录制到真实成功响应的 cassette 就能让每条事项收尾。

- **账户（biz）端点**——`AccountService.Balance`
  （`GET /api/biz/account/query-customer-account-report`）和
  `AccountService.Subscriptions`（`GET /api/biz/subscription/list`）。Z.AI 并未为
  biz API 提供文档；路由和字段名来自 Z.AI 自家的 ZCode 客户端以及社区的用量工具，
  两者都通过共享的 `Envelope[T]` 解码。在两个区域上都尚未实测验证；录制
  `TestVerifyAccountBalance` 和 `TestVerifyAccountSubscriptions` 即可定论。
- **`api.z.ai` 上的 Embeddings、Moderations 和 Rerank**（`TestVerifyEmbeddings`、
  `TestVerifyModerations`）——这些在国际平台上到底能不能用，目前尚未确认。
  Embeddings 和 Moderations 被路由到 `open.bigmodel.cn`（`BigModelBaseURL`）；
  Rerank 使用 `Config.BaseURL`。每个测试过的账户，对所试过的模型 ID（embeddings 和
  moderations 在两个 host 上，rerank 在 `api.z.ai` 上）都返回了 `400 Unknown Model`
  （code 1211）；这看起来是权益门槛，而不是路由 bug（见
  [账户与配额](accounts-and-quota.md)），但成功结构尚未见到。Rerank 目前还没有
  `TestVerify*` 测试；`TestRerankCreateLive` 只回放那个 1211。
- **Anthropic Messages**（`TestVerifyAnthropicMessages`）——路由、`anthropic-version`
  header 以及 Bearer 鉴权都已确认（一个假 key 会返回干净的 401，而不是 404 / 超时）。
  cassette 能给出定论的问题是：GLM 是把推理内容作为 Anthropic 的 `thinking` block
  暴露，还是作为 OpenAI 风格的 `reasoning_content` 字段？
  `AnthropicResponse.Thinking()` 两者都会读取。
  （[claude-code-router#1133](https://github.com/musistudio/claude-code-router/issues/1133)）
- **Agents `Invoke` 的成功结构**（`TestVerifyAgentsInvoke`）——目前只有失败信封
  （`ID`/`AgentID`/`Status`/`Error`）经过实测确认，通过
  `testdata/cassettes/agents_invoke.yaml`（一个 200 状态码内嵌失败的响应）。
  `Choices`/`Usage` 的成功结构仅依据文档建模，`async-result` 除失败信封之外的响应
  结构（`agents_async_result.yaml`）以及图像/视频 agents 的内容结构也同样如此。
- **Voice `Clone` / `Delete`**（`TestVerifyVoiceClone`、`TestVerifyVoiceDelete`）
  ——`Voice List` 已经实测确认；clone / delete 需要一段已上传的样音和一个真实的克隆
  voice ID 才能录制。clone 需要 `ZAI_VOICE_SAMPLE_FILE_ID` + `ZAI_VOICE_NAME`；
  delete 需要 `ZAI_VOICE_ID`。
- **配额与用量（monitor）端点**（`TestVerifyQuotaLimit`、
  `TestVerifyQuotaLimitChina`）——目前还没有已提交的 cassette 覆盖它们。基于 credit
  的套餐结构（`CREDIT_LIMIT` 窗口、`QuotaTypeCreditLimit`）是依据 monitor API 的第三方
  样本建模的，而对不带时区的时间字符串所作的 `MonitorServerTZ`（UTC+8）假设，只在
  global host 上核对过。
- **Responses API**（`TestVerifyResponses`）——`ResponsesService` 遵循 OpenAI 的
  Responses 规范；Z.AI 为该端点（面向 Codex）提供了文档，但没有给出其 schema。一次
  录制将能确认 item 和事件的结构、GLM 会发出哪些推理事件
  （`response.reasoning_text.delta` 还是 `response.reasoning_summary_text.delta`），
  以及 pay-as-you-go 的 key 是否被接受，还是只接受 Coding Plan 的 key。
- **Batch 和 Files 端点整体**——还没有专门的 `TestVerify*` 脚手架；需要一个有权益的
  PAYG 账户才能录制。

### 等待 cassette 的 chat API 字段

这些字段被加进了 `pkg/client/chat_types.go` / `chat.go`，用以匹配当前 docs.z.ai 的
chat-completion 规格和官方 SDK。它们都是增量字段并已通过单元测试，但在 cassette 固定
其确切线上结构之前**尚未实测验证**。

- **`ChatRequest.ToolStream`**（`TestVerifyChatStreamToolCall`）——GLM-4.6+
  流式工具调用 delta。cassette 应展示 tool-call delta 分散在多个 SSE chunk 中通过
  `StreamDelta.ToolCalls` 到达。
- **`Tool` 在 `function` / `retrieval` / `web_search` / `mcp` 之间的判别**
  （`NewFunctionTool` / `NewRetrievalTool` / `NewWebSearchTool` / `NewMCPTool`）
  ——规格列出了全部四种类型；目前只有 `function` 被确认。`web_search` 的 payload
  结构遵循官方文档和 SDK 示例，而 `mcp` 工具及其 `MCPCall` 响应条目仅依据文档建模。
- **`ChatResponse.WebSearch`**（`TestVerifyChatWebSearchResponse`）——当某个
  `web_search` 工具触发时返回的顶层 `web_search` 数组。条目结构复用了 `tools.go`
  里的 `WebSearchResult`（独立 web-search 工具的版本已实测验证）；作为顶层数组
  出现的位置是依据文档建模的。
- **`ChatRequest.ReasoningEffort`**——在本地按每个模型在目录中的条目进行校验
  （`ReasoningEfforts`；GLM-5.3 系列接受 `low`/`high`/`max`，GLM-5.2 接受所有等级）。
  每个模型的规则以及服务器对不受支持等级的处理，来自文档和 SDK；没有 cassette 固定
  它们。
- **保留式思考（preserved thinking）**——`ThinkingConfig.ClearThinking`，以及在后续
  轮次中把 `Message.ReasoningContent` 回传（`RunWithTools` 会自动这样做），遵循文档
  化的契约；当推理内容被省略或被改动时服务器的行为尚未被录制。
- **视频和文件输入**——对于原生多模态的模型（`DefaultVisionModel`），
  `Message.Videos` / `Message.Files` 会编码为 `video_url` / `file_url` 的 content
  part；该编码遵循文档，没有 cassette。
- **`FinishReason*` 常量**（`sensitive`、`model_context_window_exceeded`、
  `network_error`）——来自文档；目前还没有 cassette 复现这些终止路径。
- **客户端的工具名正则**（`^[A-Za-z0-9_-]{1,64}$`）以及 **128 个函数的上限**——
  这些是服务端文档化的规则，我们在本地强制执行；尚未确认它们是否就是服务端的精确
  拒绝准则。
- **面向 monitor/biz/agents/Anthropic/detection 的中国区域网关**——`RegionChina`
  把这些路由到 `open.bigmodel.cn`。`/models` 和 `/chat/completions` 已在中国 host 上
  实测验证；其余路径是通过镜像 `api.z.ai` 的布局建模的（`region.go` 中的
  `Region`），需要用一把有权益的中国 key 录制一份 cassette 来确认。

### 需要定期刷新的快照数据

- **模型目录**（`pkg/client/models_catalog.go`）——ID、上下文大小、输出上限、价格、
  能力和推理强度是一份快照，于 2026-10-02 从 docs.z.ai 的定价页和模型页转录而来
  （经由搜索摘要；部分行依赖第三方来源，并且 `glm-5-turbo` / `glm-5v-turbo` 的状态
  不确定）。其中任何内容发生变化时，API 都不会给出信号。该文件头部列出了刷新步骤。
- **高峰时段与非高峰促销**（`internal/usageview/peak.go`）——工作日 14:00-18:00
  UTC+8 的高峰窗口以及计费倍率是在客户端镜像的，因为 quota API 并不暴露它们；而
  `offPeakPromotions` 条目目前对应一项将于 2026-10-07 结束的促销。该日期过后请移除
  或更新它。

### 较早的遗留问题（暂无专门测试）

- **Tool-schema 兼容性改写**——GLM 解析器对哪些 JSON-Schema 构造返回 HTTP 500
  （`anyOf`/`oneOf`/`allOf`/`$ref`）这一集合，来自社区 bug 反馈
  （[claude-code-router#1474](https://github.com/musistudio/claude-code-router/issues/1474)），
  并未在此对真实账户复现。改写本身已通过完整的单元测试，并且对已经是平铺结构的
  schema 完全无效；如果有一份 cassette 精确地固定下哪些构造会 500（以及改写后的
  输出让其中哪些能通过），就能把这条从"文档化的行为"升级为"已实测验证"。见
  `pkg/client/toolschema.go`。

## 后续事项

- **录制待办的验证 cassette**——上面的 `TestVerify*` 测试已经存在，在录制之前会
  SKIP。请用真实的 key 运行它们（余额用 pay-as-you-go 的 key，配额、Responses 和
  订阅用 Coding Plan 的 key，`TestVerifyQuotaLimitChina` 用中国 key），例如
  `ZAI_RECORD=1 ZAI_API_KEY=… go test -run 'TestVerify(AccountBalance|AccountSubscriptions|QuotaLimit|Responses)$' ./pkg/client -v`，
  然后修正真实响应所推翻的任何字段名。录制钩子会对 key 和账户标识做脱敏；提交前请
  检查 YAML。
- **Codex 托管的 MCP 服务器**——与官方助手一致，`go-z-ai coding mcp add codex`
  只提供 Vision 服务器，原因是 openai/codex#14793（Codex 拒绝不带 Content-Type
  的应答，而 Z.AI 托管服务器对 `notifications/initialized` 正是这样应答的）。
  上游 issue 已关闭：一旦确认当前 Codex 版本能接受托管服务器，再为 `codex`
  提供一种远程 MCP 条目结构（`url` 加 `http_headers`）。
- **Agent 会话端点**——未实现。`POST /v1/agents/conversation` 返回某个运行会话的
  agent 的会话历史（文档称只支持 `slides_glm_agent`）。`AgentAsyncResultRequest` 已经
  携带 `ConversationID`；缺失的是历史调用本身。相关：`AgentInvokeRequest.Stream` 存在
  于 wire 类型上，但 agents 没有流式解码器，因此不支持流式的 agent 响应。

## 未实现

- **性能基准**——推迟到真正测量出瓶颈时再做；目前没有任何已知的热点路径值得为其
  建立基准（先 profile，再优化）。

## 已交付（保留供参考）

- **请求/响应可观测性钩子**——在 v0.2.0 中落地。`pkg/client` 中的 `Hook` 接缝
  （`OnRequest`/`OnResponse`/`OnError`/`OnStreamChunk`）会在每个请求生命周期事件上
  触发，每次尝试配对一个终结调用；第一个具体实现是 `pkg/observe` 中的 `OTelHook`
  （OpenTelemetry 的 span 和指标）。见
  [库使用指南 → 可观测性钩子](library-guide.md#可观测性钩子)。

## 故意不实现

- **Assistant API**——已确认被弃用。Z.AI 自己线上的 OpenAPI 规格
  （`docs.bigmodel.cn/openapi/openapi.json`）把每一条 Assistant 路径都标记为
  `"deprecated": true`，并且从 `api.z.ai` 调用它会完全超时，而不是返回错误。
  为一个已下线的 API 编写客户端，在维护成本上不划算——如果 Z.AI 哪天重新启用它，
  上面的规格里就有完整的请求/响应 schema 可以直接转录。
