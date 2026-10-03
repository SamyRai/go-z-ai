# 架构

## 目录

- [包布局](#包布局)
- [CLI 约定](#cli-约定)
- [请求门面](#请求门面)
- [流式](#流式)
- [重试与超时设计](#重试与超时设计)
- [为什么有些服务会访问不同的主机](#为什么有些服务会访问不同的主机)
- [实测验证约定](#实测验证约定)
- [扩展结构化的查找表](#扩展结构化的查找表)
- [凭证文件安全性](#凭证文件安全性)
- [TUI](#tui)

## 包布局

```text
main.go               Entrypoint: sets build info, then internal/cli.Execute()
internal/cli/         Cobra commands (package cli), one file per command group:
                      chat.go, models.go, usage.go, validate.go, accounts_cli.go,
                      coding_cli.go, coding_mcp_cli.go, ...; common.go holds the
                      shared plumbing, root.go the global flags
pkg/client/           The Go library — no CLI/TUI dependency, stdlib-only
                      (no third-party imports); layout below
pkg/observe/          OpenTelemetry Hook implementation (otel.go; depends on
                      go.opentelemetry.io/otel); separate package so pkg/client
                      stays stdlib-only
internal/accounts/    Multi-account credential store (~/.config/zai-client/accounts.json —
                      dir kept as zai-client for upgrade compat)
internal/atomicfile/  Atomic temp-file-then-rename writes (Write, WriteWithBackup)
                      used by every credential store and tool-config writer
internal/coding/      GLM Coding Plan credential store, the tool registry (one
                      tool_*.go file per tool), and Z.AI's MCP servers
internal/fileinput/   FileOrURL: a URL passes through, a local path is base64-encoded —
                      shared by `ocr parse`, `chat create --image/--video/--file`,
                      and the TUI media tab
internal/modelview/   Presentation helpers for model metadata (token counts, prices,
                      capability names) — shared by the CLI and the TUI
internal/usageview/   Presentation helpers for quota and usage (time windows, heat maps,
                      quota summaries, peak-hours logic) — shared by the CLI and the
                      TUI so their output can never drift
internal/tui/         Bubble Tea terminal UI: a root model plus one subpackage per
                      tab and a few shared subpackages
```

`pkg/client` 是核心的公开包，并且被刻意设计为**仅依赖标准库**——零第三方导入——
这样任何人都可以依赖这个 client，而无需把 OTel SDK、MCP SDK 或其他沉重的
可观测性/集成依赖一并拖进来。可观测性的接缝是一个仅依赖标准库的 `Hook` 接口
（见 [库使用指南——可观测性钩子](library-guide.md#可观测性钩子)）；具体实现位于
`pkg/observe`（OpenTelemetry）以及未来的其他包中，每个包都有自己的依赖集合，
这样用户只为自己导入的东西付出代价。`internal/` 下的所有内容都是实现细节，
编译器会禁止外部代码导入它们，因此 CLI/TUI 层可以自由重构。CLI 和 TUI 都是
`pkg/client` 的轻量调用方。`go install github.com/SamyRai/go-z-ai@latest` 会把
根目录的 `main.go` 构建为 `go-z-ai` 二进制。

### pkg/client 文件归属

每个关注点都恰好有一个归属文件，因此对它的改动只需编辑一个文件：

| 文件 | 负责 |
|---|---|
| `client.go` | `Client`、`NewClient`，以及各服务的访问器（`Chat()`、`Models()` ……） |
| `config.go` | `Config`、它的默认值（`DefaultTimeout`、`DefaultMaxRetries`、`DefaultRetryDelay`）、校验、默认 HTTP transport，以及 `NewClientFromEnv` |
| `region.go` | **每一个 URL。** `Region`、各网关主机、API 根路径，以及 `Region.*BaseURL` 方法；`ParseRegion` |
| `transport.go` | 唯一的请求路径（见下文）：`apiRequest`、`open`、`do`、`doRaw`、multipart 编码 |
| `retry.go` | 可重试性判定、退避、`Retry-After` 解析 |
| `envelope.go` | `Envelope[T]`（monitor 和 biz API 的 `{code, msg, success, data}` 包装）以及 `fetchEnvelope` |
| `stream.go` | 同步 SSE 迭代器（`streamSSE`、`scanSSE`） |
| `errors.go` | `APIError`、错误码表、对未知错误码按 HTTP 状态码分类、带内（in-band）错误 |
| `hook.go` | `Hook` 接口以及每次尝试的调用辅助函数 |
| `models_catalog.go` | **全部模型知识：** ID、上下文、价格、能力、推理强度，以及 `Default*Model` 常量 |
| `models.go` | `ModelsService`、`ModelDetails`、`Pricing`（包括 `Pricing.Cost`） |
| `chat_types.go` | Chat 的 wire 类型：`Message`、`ChatRequest`、`ThinkingConfig`、`ResponseFormat`、tools、`ChatResponse`、`StreamChunk`、`Usage` |
| `chat.go` | `ChatService`：`Create`、`CreateAsync`、`Stream`，以及请求校验（推理强度、工具规则） |
| `chat_tools.go` | `RunWithTools`，即工具调用循环 |
| `content.go` | 多模态消息编码（把 `Message.Images`/`Videos`/`Files` 转为 content parts） |
| `toolschema.go` | `SanitizeToolSchemas`，即 GLM 的解析器所需的 JSON Schema 改写，对 chat、Anthropic 和 Responses 的 tools 一视同仁地应用（`sanitizeTools`） |
| `anthropic.go` | `AnthropicService`（Messages API），包括它的流解码器 |
| `responses_types.go`, `responses.go` | Responses 协议的 wire 类型（Codex 所使用的协议）；`ResponsesService`：`Create`、`Stream`、校验、流解码器 |
| `quota.go`, `timezone.go` | Coding-plan 的配额与用量（monitor API）；服务器时区处理 |
| `account.go` | `AccountService`（biz API）：`Balance`、`Subscriptions` |
| `detection.go`, `status.go` | `DetectionService`：`DetectAccountType`、`CheckAccountStatus` |
| `async.go` | 异步任务类型以及 `GetAsyncResult`/`WaitForResult`，由 chat、image 和 video 共用 |
| `images.go`, `videos.go`, `audio.go`, `voice.go` | 媒体服务 |
| `files.go`, `batch.go`, `fileparser.go`, `layout.go` | 文件、批量任务、文档解析、版式解析（OCR） |
| `tools.go`, `agents.go` | 网页搜索/reader/tokenizer；agents |
| `embeddings.go`, `moderations.go`, `rerank.go` | 检索与审核服务 |
| `version.go` | `Version()` 以及默认的 `User-Agent` |

`testdata/cassettes/` 存放实测验证测试所回放的录制交互。

## CLI 约定

需要 API 客户端的命令处理器注册为 `RunE: runWithClient(runX)`，并把已解析好的
`*client.Client` 作为第三个参数——`runWithClient`（位于 `internal/cli/common.go`）
通过 `getClient` 只解析一次，这样各处理器就不必各自重复"解析并校验"的前置逻辑。
凭证优先级本身位于 `resolveConfig`（flag → `--account` →
`ZAI_API_KEY` → 当前激活账户），在 `credentials_test.go` 中有单元测试覆盖。
同一个函数还负责解析区域（`--region` → `ZAI_REGION` → 账户存储的区域）以及其他
全局设置。

输出格式是统一的：每个打印结果的命令都通过 `addFormatFlag` 注册共享的
`--format` flag，并通过 `emit(cmd, v, textFn)` 渲染——后者在 `--format json`
时输出美观的 JSON，否则运行可读性更好的 `textFn`。进度信息（`progressf`）
写到 stderr，以保证 stdout 始终是合法的 JSON。

## 请求门面

每个服务方法最终都会汇聚到同一条请求路径——服务永远不会自己构造
`http.Request` 或 `http.Client`。服务把自己的调用描述为一个 `apiRequest`
（方法、路径、请求体，以及可选的 base URL、API key、额外的头、一个 multipart
表单和钩子归属），然后把它交给下面这些 `Client` 入口之一，它们全都建立在
`open` 之上：

```text
doRequest(ctx, method, path, body, result)  // the common case: do() with the defaults
do(ctx, apiRequest, result)                 // open, then JSON-decode the body into result
doRaw(ctx, apiRequest)                      // open, then return the raw body (audio, file downloads)
streamSSE(ctx, c, apiRequest, decode)       // open, then decode the body incrementally (see Streaming)
        every one of them goes through Client.open(ctx, apiRequest) → *attempt
```

`open` 为每个端点负责鉴权、重试/退避、错误解析以及可观测性钩子的生命周期。
`apiRequest` 上留空的路由字段会回退到客户端的配置：没有 base URL 就用
`Config.BaseURL`，没有 API key 就用 `Config.APIKey`。需要不同主机的服务
（Agents、monitor、biz、Anthropic）会用某个 `Region` 方法来设置 `baseURL`；
需要不同凭证的服务（Embeddings、Moderations）会设置 `apiKey`——它绝不会绕过
`open`。

**钩子配对。** `open` 返回一个 `attempt`：即打开的 2xx 响应，加上产生它的那次
尝试的钩子状态。每次尝试都会触发 `OnRequest`，然后恰好触发一个终结钩子——
`OnResponse` 或 `OnError`。一次即将被重试的失败尝试会在退避之前触发 `OnError`，
而 `open` 的调用方则用 `succeed` 或 `fail` 来结束成功的那次尝试。因此在
`OnRequest` 中开启的 span 总是能够被结束。

**Multipart 上传**（文件、音频转写、文档解析）是设置了 `form` 的 `apiRequest`。
它们同样经过 `open`，但绝不重试：在瞬时失败时重新上传文件应由调用方决定，而不是
一个安全的默认行为。

**monitor/biz 信封。** 配额（monitor）和账户（biz）API 把每个响应都包在
`Envelope[T]` 里，并在 HTTP 200 的响应体内报告业务失败。`fetchEnvelope` 会解码
信封，并把失败转换为 `*APIError`（当信封的 code 对应某个 HTTP 状态码时，按它
分类），所以返回 `Envelope` 的服务已经成功了，调用方只需读取 `Data`。

## 流式

`stream.go` 把 server-sent events 实现为一个 Go 迭代器。`Chat().Stream` 和
`Anthropic().Stream` 返回 `iter.Seq2[T, error]`；`streamSSE` 在迭代器内部同步地
运行整个流——没有生产者 goroutine，也没有 channel——所以跳出 `range` 循环就会
停止读取并关闭响应体，而取消 context 会中止一次被阻塞的读取。`scanSSE` 是共享的
行级解析器；每种协议各自提供一个小型解码器（`chat.go` 中的
`decodeChatStream`，`anthropic.go` 中的 `decodeAnthropicStream`）。

以带内（in-band）方式送达的错误——OpenAI 风格的 `{"error": ...}` 数据块或
Anthropic 的 `event: error`——会以一个 `*APIError` 结束该流，它由与 HTTP 错误相同
的分类器构建（`errors.go`）。钩子通过 `OnStreamChunk` 看到每一个解码出来的数据块；
这次尝试在正常结束时以 `OnResponse` 收尾，在流中途失败或提前 break
（`context.Canceled`）时以 `OnError` 收尾。

## 重试与超时设计

- `Config.Timeout` 限定的是拨号 + TLS 握手 + 等待响应头的时间——故意**不**用
  整个 `http.Client.Timeout`，因为后者会在生成过程中途截断一次长时间的
  `Stream` 读取。
- 一次失败的尝试在可重试时会被重试：传输层失败（服务器根本没有应答），或被标记
  为可重试的 `*APIError`——HTTP 429 和 5xx，外加 `errors.go` 标记为瞬时的业务
  错误码。表中不认识的错误码仅按 HTTP 状态码分类，因此只有限流和服务端故障才会
  被重试；鉴权、配额和参数错误不会。
- 退避是从 `Config.RetryDelay` 起的指数退避，带抖动（最高 25%）。存在
  `Retry-After` 头（秒数或 HTTP 日期）时以它为准。任何单次延迟都以 30 秒为上限。
  重试循环受 `Config.MaxRetries` 限制（默认 3；`-1` 表示禁用重试）。每次重试前
  都会先检查 `ctx.Err()`，这样被取消的 context 会立即中止，而不是先睡完一次
  退避再退出。
- 流式只重试*连接*尝试——一旦 SSE 流真正开始，流中途的失败会直接上报给调用方，
  而不是被静默重试（无法知道调用方已经消费了多少响应内容）。

## 为什么有些服务会访问不同的主机

`Region`（`region.go`）是网关 URL 的唯一归属者：`pkg/client` 中的每一个 base URL
——以及 `internal/coding` 中的每一个配置写入器——都由 `Region.Host` 加上某一个
API 根路径派生而来。这些根是 `Region.PaaSBaseURL`、`CodingBaseURL`、
`AnthropicBaseURL`、`ResponsesBaseURL`、`MonitorBaseURL`、`BizBaseURL`、
`AgentsBaseURL` 和 `MCPServerURL(name)`；`ConsoleURL` 是 Web 控制台，
`BaseURLFor(AccountType)` 则按账户类型选出 chat 的根。包级常量
（`DefaultBaseURL`、`BigModelBaseURL` ……）是两个网关的相同取值。

| 服务 | Base URL | 原因 |
|---|---|---|
| Chat、Models、Images、Videos、Audio、Voice、Files、Batch、Tools、Rerank …… | `Config.BaseURL`——默认为 `Region.PaaSBaseURL()`（`/api/paas/v4`）。Coding Plan 的 key 通过设置 `Config.BaseURL` 指向 `Region.CodingBaseURL()`（`/api/coding/paas/v4`）（CLI 根据账户类型这样做） | 通用情况。显式设置的 `Config.BaseURL` 永远优先。 |
| Embeddings、Moderations | `BigModelBaseURL`（`open.bigmodel.cn`），使用 `Config.ChinaAPIKey` 鉴权 | 仅在中国平台有文档；对于试过的模型，`api.z.ai` 会返回 1211 Unknown Model。实测验证发现，普通的 z.ai 密钥在两个平台上鉴权表现一致，因此 `ChinaAPIKey` 默认会回退到 `APIKey`。 |
| Anthropic Messages | `Region.AnthropicBaseURL()` | 独立的 Anthropic 兼容根。 |
| Responses（Codex 协议） | `Region.ResponsesBaseURL()`——`/api/v1` | 独立的 OpenAI Responses 协议根，文档面向 GLM Coding Plan（docs.z.ai/devpack/tool/codex）。 |
| 配额/用量（monitor）、账户（biz）、账户类型检测 | `Region.MonitorBaseURL()`、`Region.BizBaseURL()` | 按区域划分，因此中国的 key 能访问它自己所在区域的端点。 |
| Agents | `Region.AgentsBaseURL()`——裸的 `/api` 根，没有 `/paas/v4` | 实测验证——把 `/v1/agents` 嵌套在 chat-completions 的 base 下会返回 404。 |

### 区域网关选择（Config.Region）

Z.AI 通过两个区域网关提供同一套 GLM 模型家族：国际主机 `api.z.ai` 和中国大陆镜像
`open.bigmodel.cn`。`Config.Region`（`RegionGlobal`，默认值，或 `RegionChina`）
为每一个区域相关的服务选择主机——monitor、biz、agents、Anthropic Messages、
Responses 和账户类型检测——并且它是 `Config.BaseURL` 为空时的默认值，因此
`RegionChina` 也会把 chat 移到 `open.bigmodel.cn`。显式设置的 `Config.BaseURL`
对 chat/PaaS 根仍然优先。Embeddings 和 Moderations 始终使用
`BigModelBaseURL`。

从 CLI 上，`--region {global,china}` 或 `ZAI_REGION`（别名：`cn`、`bigmodel`、
`west`）会选择每一个端点，而 `--base-url` 只覆盖 chat/PaaS 根。未知的取值会回退
到 global 而不是报错，因此一次拼写错误永远不会阻塞一条与之无关的命令。已存储的
账户会记录其 key 签发时所在的区域；`accounts add` 会检测它，并由
`Account.ResolvedBaseURL` 根据账户类型和区域推导出 chat 的根，这样一把 key 就
不会被指向错误的端点。

中国镜像侧的 monitor/biz/agents 主机，是通过把 `api.z.ai` 的路径布局镜像到
`open.bigmodel.cn` 上建模出来的，并在 `region.go` 中标记为 `NOT VERIFIED LIVE`
——中国平台已经实测验证会在 `/models` 和 `/chat/completions` 上提供相同的 OpenAPI
表面（见 `BigModelBaseURL`），但中国侧的 monitor/biz/agents 主机还没有被任何
cassette 录制到。如果你持有一把有权限的中国密钥，可以用 `ZAI_RECORD=1` 把它们
录制下来。

## 实测验证约定

Z.AI 自家的 SDK 和文档之间有时彼此不一致，有时甚至与线上 API 实际返回的结果
不一致（某个文档标注为可选的端点，缺了就 400；某个错误被嵌在 200 响应体里；
同一个字段在两个官方 SDK 里类型不同）。与其只信单一来源，本项目里的新服务都会
对真实 API 调用做一次校验，并把这次交互录制为 [go-vcr](https://github.com/dnaeon/go-vcr)
cassette（`pkg/client/testdata/cassettes/`），在 `ModeReplayOnly` 下回放，这样
测试套件永远不会真的触网。

`pkg/client` 里有两个文件承担实测验证工作，职责各不相同：**`live_replay_test.go`**
持有 `Test*Live` 测试，它们把已提交的 cassette 作为冻结的发现来回放（权限门槛、
"200 里嵌着失败"的怪现象、单把密钥跨主机通用的论断）；它是"已经确认了什么、
为什么重要"的运行日志。**`live_verify_test.go`** 持有 `TestVerify*` 录制脚手架——
每个测试在你用 `ZAI_RECORD=1` 录到一条新的成功路径 cassette 之前都会 SKIP，
录到之后再回放；它是"还差一次真实录制"的待办清单（见
[路线图](roadmap.md)）。

如果你要扩展某个服务，在新增 cassette 之前请先阅读
[Contributing § the live-verification convention](../../CONTRIBUTING.md)。

## 扩展结构化的查找表

本仓库里有几种类型把一小撮封闭的、API 定义的取值通过一张表加上一个查找函数
映射成人类可读的元数据，而不是一串 `if`/`switch` 条件。最清晰的例子是
`pkg/client/models_catalog.go` 的 `modelsCatalog`：每一行是一个
`ModelCatalogEntry`，携带该模型的上下文大小、输出上限、价格、`Capabilities`
（`CapText`、`CapVision`、`CapVideo`、`CapFile`、`CapThinking` ……）以及
`ReasoningEfforts`。相关的行共享它们的能力集合和强度列表（`capsReasoning`、
`capsMultimodal`、`effortsGLM53`），这样它们就不会彼此漂移。

```go
{
    ID: "glm-5.3-flash", Family: "GLM-5", Tier: "flash", Name: "GLM-5.3-Flash",
    Capabilities: capsMultimodal, ReasoningEfforts: effortsGLM53,
    ContextSize: ctx1M, MaxOutput: out128K, Pricing: usd(0.15, 0.03, 0.50),
    ...
},
```

当 Z.AI 新增一个模型时，改动是增量的（追加一行）且局部的；表中没有的任何内容
仍然会从 `/models` 中出现，只是数据稀疏，而不是硬失败。查找先按精确 ID，再按
已收录 ID 的带日期快照（`findCatalogEntry`），并且 `ModelDetails.HasCapability`
是判定某项能力的唯一地方，因此 CLI、TUI 和库中的过滤器彼此一致。同样的形状也
出现在 `internal/coding` 中——`Tools` 注册表（`tools.go`，每个工具一个
`tool_*.go`）、`MCPServers` 注册表（`mcp.go`）以及 `Plans` 列表（`plans.go`）。
当你为某个既有概念新增一个被识别的取值时，优先采用这种形状，而不是再加一条条件
分支。

`modelsCatalog` 同时也是对一个数据可得性缺口的修复：Z.AI 的 `/models` 端点只
返回 OpenAI 式的裸 `{id, object, created, owned_by}` 结构，所以如果不做补充，
每个 `Context`/`Pricing`/`Capabilities` 单元格都会渲染为 `-`/`0`。
`ModelsService.List` 会让每个解码出来的模型经过 `enrichModel`，它会叠加目录中的
字段，但当两者都存在时**以线上 API 的值为准**——所以哪天 Z.AI 开始在 `/models`
里发送 `max_context` 或 `pricing`，真实数字就会接管，无需改动代码。价格和上下文
数字是从 <https://docs.z.ai/guides/overview/pricing> 以及各模型页面抄录的，需要
定期手动刷新（这些数字变化时 API 不会给出任何信号）；文件头部带有"最后验证"的
日期。同一文件中的 `Default*Model` 常量（`DefaultModel`、`DefaultFastModel`、
`DefaultVisionModel`、`DefaultOCRModel`、`DefaultASRModel`、`DefaultTTSModel`）
是 CLI、TUI 和编码工具配置写入器所采用的默认值，因此一次目录刷新会同时更新它们
全部。

## 凭证文件安全性

`internal/accounts`（多账户存储）和 `internal/coding`（GLM Coding Plan 凭证存储，
以及它写入的每一个第三方工具配置——Claude Code、Codex、OpenCode、Crush、
Factory Droid）都通过 `internal/atomicfile` 写文件：先在同一目录写一个临时文件，
再 rename，绝不会直接就地写入。写入中途崩溃或被 kill 会保持原文件完好无损，而
不是被截断。符号链接的配置会先被解析，因此链接本身得以保留。

其中有几个是被合并写入的*其他程序*的真实配置文件，而不是本项目完全自有的文件，
所以 `internal/coding` 用 `atomicfile.WriteWithBackup` 写它们：在第一次覆盖之前，
它会把现有文件复制为 `<path>.zai.bak`，并且绝不替换该备份，而 `configfile.go`
只编辑它自己拥有的键，保留其余一切。它按文件扩展名读写 JSON 或 TOML（Codex 的
`config.toml`）；TOML 的重写会丢掉注释，官方助手也是如此，而备份保留着原文件。
包含密钥的文件以 `0600` 创建；目录以 `0700` 创建。

## TUI

`internal/tui` 是一个 [Bubble Tea](https://github.com/charmbracelet/bubbletea)
程序（v2），每个子包对应一个标签页（`chat`、`models`、`usage`、`accounts`、
`coding`、`media`、`tools`）。每个标签页都是一个独立的 `tea.Model`，拥有自己的
`Update`/`View`；根 model 在它们之间分发。长时间运行的工作（一次 API 调用、一次
文件系统操作）会被包进一个 `tea.Cmd` 闭包里，由 Bubble Tea 在自己的 goroutine
上运行——从 `tea.Cmd` 中调用的代码不能假设包级可变状态没有竞争（例如修改
`http.DefaultClient`）；`internal/coding/validator.go` 正是出于这个原因，在每次
调用时都构建一个全新的 `client.Client`。

### 文件与子包

| 路径 | 职责 |
|---|---|
| `tui.go` | `Run(Config)`：构建 session 和根 model，并运行程序 |
| `session.go` | `session`：各屏幕共享的状态——API 客户端（在激活账户变化时重建，通过原子指针读取，这样 `tea.Cmd` goroutine 也能使用它）、各个存储，以及已存储的 coding 套餐 |
| `root.go` | `rootModel`：持有 chrome 和布局，路由消息，处理全局按键和浮层，并把主体委托给当前激活的屏幕 |
| `screens.go` | 屏幕可以实现、根 model 会探测的可选接口（`streamer`、`inputCapturer`、`helpProvider`、`refresher`、`chatModelSetter`/`chatModelGetter`） |
| `header.go` | 顶部 header 行及其徽标 |
| `tabs.go` | `tab` 枚举、标签页名称，以及标签栏的渲染和鼠标命中检测 |
| `toast.go` | 状态行 toast，以及 `describeErr`（把 API 错误类别映射为消息和严重程度） |
| `overlay.go`, `helpoverlay.go`, `keys.go` | 浮层合成以及面板/模型选择器/帮助的接线；帮助浮层；全局按键映射 |
| `accounts/`, `chat/`, `coding/`, `media/`, `models/`, `tools/`, `usage/` | 每个标签页一个。`chat/stream.go` 把 client 的流迭代器适配到消息循环；`models/detail.go` 和 `usage/heatmap.go` 存放纯渲染函数 |
| `formtab/` | Media 和 Tools 标签页共用的屏幕：一排单输入的请求表单、同一时刻只有一个在途请求（用 `esc` 取消），以及一个可滚动的结果面板。各标签页只提供它们自己的 `Form` |
| `modelpicker/` | chat 模型选择器浮层（打开时获取、可过滤的列表） |
| `palette/` | `ctrl+p` 命令面板浮层 |
| `uimsg/` | 共享的消息类型，让各屏幕无需导入根 model 就能与之通信 |
| `uistyle/` | 共享的 lipgloss 样式词汇表以及亮色/暗色调色板 |

### 共享基础设施

chrome（header、标签栏、面板、状态行、帮助栏）由 `internal/tui/root.go` 中的根
model 持有；每个屏幕只渲染自己的主体。一些横切的部分放在屏幕之外，这样就能在
没有导入环的情况下共享：

#### 顶部 header

header（`internal/tui/header.go`）是单独一行：左侧是 `go-z-ai` 应用徽标，右侧是
一组右对齐的上下文徽标，两者之间有一个间隔把徽标推到右边缘。这些徽标让每个标签页
都能一眼看到状态：

- **account** —— 当前激活账户的名称（未设置账户时是带警告样式的 `none`，让缺少
  凭证的状态一目了然）。
- **type** —— `pay-as-you-go` / `coding-plan`，因为它决定了哪一类端点族和哪些
  功能（例如 Usage 标签页的 monitor 端点）可用。
- **plan** —— 来自 coding 存储的 `Global` / `China`，之所以展示它，是因为它会在
  无声无息间改变 Coding 标签页和 coding-agent 集成所访问的端点。仅在已配置套餐时
  显示。
- **model** —— Chat 标签页当前选中的模型。对每个标签页都有用，因为用户常常在
  撰写聊天的同时查看 Usage 或 Models。

在窄终端上，header 会逐步丢弃徽标：先丢 plan 和 type（保留 account + model），
再丢 model（只剩 account），这样应用名称和最要紧的信息始终可见。徽标通过
`uistyle.RenderBadge(label, value, valueStyle)` 构建——一个弱化的标签配上一个按
角色着色的值。

标签条的上方和下方各有一行间隔，让 chrome 有些喘息的空间；`chromeRows` 在计算
内部面板尺寸时会把这两行间隔都算进去。

- **`internal/tui/uistyle`** —— lipgloss 样式词汇表和一套亮色/暗色双调色板。根
  model 会对照终端的实际背景（`tea.BackgroundColorMsg`）解析调色板，并通过
  `uistyle.SetDark` 重建每一个共享样式，因此整个应用会跟随终端主题。样式是在
  主题变化时被重新赋值的包级 `var`；各屏幕在渲染时重新读取它们。
- **`internal/tui/uimsg`** —— 共享的消息类型（`Err`、`Status`、`Routed`、
  `CloseOverlay`、`OpenModelPicker`、`AccountChanged`、`PlanChanged`），让各屏幕
  无需导入根 model 就能与之通信。`Routed` 把一个异步结果连同其发起的标签页一起
  包装起来（`uimsg.Route` 用一个 `tea.Cmd` 构建它），这样在某个标签页中发起的
  结果，即使用户已切换标签页，也仍会落到原来的那个标签页。
- **浮层** —— 根 model 上单独一个 `overlay tea.Model` 槽位，通过
  `lipgloss.Place` 合成在当前激活屏幕之上（`internal/tui/overlay.go` 中的
  `placeOverlay`）。打开时它独占按键；resize、背景色和 routed 消息仍会流向下方的
  屏幕，这样浮层关闭时它们依然保持正确的布局。帮助浮层（`?` / `f1`）、命令面板
  （`ctrl+p`）和 chat 模型选择器（在 chat 标签页按 `ctrl+o`，或面板中的
  "Switch chat model"）都是由根 model 持有的浮层。

### 按键绑定

全局（在任何屏幕看到按键之前，由根 model 处理）：

| 按键 | 操作 |
|----------------|--------------------------------------------------|
| `tab` / `shift+tab` | 下一个 / 上一个标签页 |
| `?` / `f1` | 切换帮助浮层（所有绑定）；当焦点在文本输入框时，`?` 会被当作输入字符 |
| `ctrl+p` | 命令面板（模糊搜索应用级操作） |
| `ctrl+c` | 退出（或取消正在进行的 chat 流） |
| 鼠标点击标签栏 | 切换标签页 |
| 鼠标滚轮 | 滚动当前激活的视口 / 列表 |

各屏幕自己的绑定会显示在底部的帮助栏和帮助浮层中。

### 响应式布局

Models 和 Usage 标签页采用三档宽度：

- **≥ 100 列** —— 双栏布局（Models：表格 + 实时预览；Usage：配额与热力图并排）。
- **70–99 列** —— 单栏，完整的表格/热力图行。
- **< 70 列** —— 紧凑：Models 表格去掉 CAPS 列并缩短价格表头；Usage 热力图收缩为
  每个分区一行的摘要；标签栏切换为仅含标签页名称的紧凑形式。

当低于 **60×22** 时，根 model 只渲染一条居中的"Terminal too small"消息，要求至少
60×22，而不是让 chrome 互相重叠。各屏幕仍会收到 `WindowSizeMsg` 并为自己的尺寸
设定下限，所以一旦窗口重新大过阈值，应用就会以正确的布局恢复，无需任何额外
工作。
