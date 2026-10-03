# CLI 参考

每条命令都支持 `--help`，其中是权威且始终最新的 flag 列表
（`go-z-ai <command> --help`、`go-z-ai <command> <subcommand> --help`）。
本页是一次有组织的概览；如果本页与 `--help` 出现不一致，请以 `--help`
为准。

## 目录

- [全局 flag](#全局-flag)
- [聊天补全](#聊天补全)
- [模型](#模型)
- [账户、用量与配额](#账户用量与配额)
- [编码工具（GLM Coding Plan）](#编码工具glm-coding-plan)
- [文件与批量任务](#文件与批量任务)
- [媒体生成](#媒体生成)
- [文档解析与 OCR](#文档解析与-ocr)
- [检索辅助](#检索辅助)
- [内容审核](#内容审核)
- [Agents](#agents)
- [工具（网页搜索、reader、tokenizer）](#工具网页搜索readertokenizer)
- [Anthropic 兼容端点](#anthropic-兼容端点)
- [Responses 端点（Codex 协议）](#responses-端点codex-协议)
- [终端 UI](#终端-ui)

## 全局 flag

以下 flag 适用于每条命令：

| Flag | Description |
|---|---|
| `--api-key string` | Z.AI API Key（或 `ZAI_API_KEY` 环境变量） |
| `--account string` | 按名称为本次命令使用一个已存储的账户（见 [账户与配额](accounts-and-quota.md)） |
| `--region string` | 区域网关：`global`（api.z.ai，默认）或 `china`（open.bigmodel.cn）。别名 `cn`、`bigmodel`、`west`。或使用 `ZAI_REGION` 环境变量。未知值回退到 global。 |
| `--base-url string` | Chat/PaaS API 根地址（或 `ZAI_API_BASE_URL` 环境变量）。默认：所选区域的根地址，例如 `https://api.z.ai/api/paas/v4` |
| `--china-api-key string` | 用于 Embeddings/Moderations 的 open.bigmodel.cn key（或 `ZAI_CHINA_API_KEY`；未设置时回退到 `--api-key`） |
| `--monitor-timezone string` | quota/usage（monitor）API 所采用的时区（或 `ZAI_MONITOR_TIMEZONE`；默认 CST/UTC+8）。接受 IANA 名称、`UTC`，或形如 `+8` 的偏移量 |
| `--config string` | 配置文件（默认：`.env`） |
| `-v`, `--version` | 打印版本并退出。发布版本（GoReleaser ldflags）打印标签；开发版本打印 `dev`，并附带提交和构建日期。 |

大多数会产生结果的命令都接受 `--format text|json`（`models` 命令把文本模式叫作
`table`；`embeddings` 和 `moderations` 默认使用 `json`）。少数只执行动作的命令
没有 `--format`：`accounts add|use|remove`、`coding auth|load|unload|doctor`、
`coding mcp add|remove`、`audio speech`、`files download` 和 `validate`。
进度和状态信息会输出到 stderr，因此使用 `--format json` 时，stdout 始终是有效的
JSON，可以直接管道给 `jq`。

`--region`（或 `ZAI_REGION`）会为 CLI 访问的每个端点选择网关：chat 及其他
PaaS 服务、Anthropic 兼容接口、配额/用量、账户（biz）、agents，以及账户类型
检测。当你的 key 是在 `open.bigmodel.cn` 上签发时，请把它设为 `china`。
`--base-url` 只覆盖 chat/PaaS 根地址，并且在该处优先于区域默认值。
Embeddings 和 Moderations 始终使用 `open.bigmodel.cn`。当 `--region` 和
`ZAI_REGION` 都未设置时，会采用已存储账户的区域。参见
[账户与配额 § 区域网关](accounts-and-quota.md#区域网关apizai--openbigmodelcn)。

## 聊天补全

```bash
go-z-ai chat create <message> [flags]
go-z-ai chat async-result <task-id>
```

`chat create` 是主要入口。采样设置（`--temperature`、`--top-p`、
`--max-tokens`）在保持为 `0` 时默认使用模型自身的取值。

| Flag | Purpose |
|---|---|
| `--model string` | 默认：目录中的默认 chat 模型（`client.DefaultModel`，当前为 `glm-5.3`） |
| `--system string` | 系统消息 |
| `--stream` | 逐 token 的流式输出。配合 `--format json` 时，每行一个数据块 |
| `--async` | 提交后不等结果；用 `chat async-result <task-id>` 轮询 |
| `--temperature float`, `--top-p float`, `--max-tokens int` | 采样控制（`0` = 模型默认值） |
| `--do-sample` | 是否采样（默认 `true`）；`--do-sample=false` 即贪心解码 |
| `--stop strings` | 停止序列（API 只认其中一个） |
| `--thinking string` | `enabled` 或 `disabled`（GLM-5.3 模型始终会推理） |
| `--effort string` | 推理强度：`max\|xhigh\|high\|medium\|low\|minimal\|none`。会按模型在目录中的条目校验：GLM-5.3 接受 `low`、`high`、`max`；GLM-5.2 接受所有等级 |
| `--show-reasoning` | 打印推理过程（文本模式下输出到 stderr） |
| `--json` | 要求返回 JSON 对象形式的响应 |
| `--json-schema string` | 返回符合某个 schema 的 JSON 对象响应：`@file.json` 或内联 JSON。Z.AI 没有 `json_schema` 响应格式，因此该 schema 会被加进系统消息 |
| `--tool string` | 函数调用工具声明：`@tools.json` 或内联 JSON 数组 |
| `--tool-stream` | 增量地流式输出工具调用参数（GLM-4.6+） |
| `--image string`, `--video string`, `--file string`（可重复） | 附加图片、视频或文档：一个 URL，或 `@path` 指向本地文件（base64 编码） |
| `--format text\|json` | 输出格式 |

```bash
go-z-ai chat create "Summarize this in 3 bullets" --stream
go-z-ai chat create "Plan a migration" --effort max --show-reasoning
go-z-ai chat create "Extract fields" --json-schema @schema.json --format json
go-z-ai chat create "Describe this" --image @photo.jpg --model glm-5.3-flash
```

附件需要具备对应能力的模型（在 `go-z-ai models list` 中为 `video`、`file`、
`vision`）。目录中的默认 chat 模型仅支持文本；`glm-5.3-flash`
（`client.DefaultVisionModel`）原生支持多模态（图像、视频、文件输入）。

工具调用只会被打印，不会被 CLI 执行——关于 Go 端 `RunWithTools`
自动执行循环，见 [库使用指南 § 函数调用](library-guide.md#函数调用)。

`chat async-result`、`image status` 和 `video status` 共用同一个处理程序：
在文本模式下，任务状态输出到 stderr，结果（消息、图片 URL 或视频 URL）输出到
stdout；`--format json` 则打印完整的任务结果。

> **视觉 + 工具调用可能返回 HTTP 401。** 社区反馈（例如
> [claude-code-router#1491](https://github.com/musistudio/claude-code-router/issues/1491)）
> 表明，在某些 GLM 配置下，把视觉模型（在 `glm-4.6v`/`glm-4.5v` 上使用
> `--image`）与函数调用工具（`--tool`）放在同一请求里会被以 401 拒绝——
> 一个已鉴权的 key 也只是在这种组合下失败。如果你遇到这种情况，请拆分
> 工作：用视觉模型处理图片轮次，用文本模型处理工具调用轮次，而不是把图片和
> 工具一起发送。此处尚未在真实账户上复现。

## 模型

```bash
go-z-ai models list
go-z-ai models get <model-id>
go-z-ai models text | vision | free
```

`/models` 端点只返回裸 ID；上下文大小、最大输出、价格（每 100 万 token 的美元
价格）、能力（`text`、`vision`、`video`、`file`、`thinking`、`tools`、`code`、
`ocr`、`audio`）以及可接受的推理强度，都来自 `pkg/client/models_catalog.go` 中
精心维护的目录。目录中不认识的模型仍会出现，未知的值以 `-` 显示。`models get`
还会显示推理强度等级。

## 账户、用量与配额

详细说明见 [账户与配额](accounts-and-quota.md)。快速参考：

```bash
go-z-ai accounts add <name> --api-key <key> [--type coding_plan|pay_as_you_go] [--region global|china] [--base-url-override URL] [--force]
go-z-ai accounts list [--format json] [--reveal]   # keys masked by default; --reveal for export
go-z-ai accounts use <name>
go-z-ai accounts show [name] [--format json] [--reveal]
go-z-ai accounts current                            # shorthand for 'accounts show' (active account)
go-z-ai accounts quota [--only name...]
go-z-ai accounts usage [--days N] [--today] [--metric model|tool|both] [--only name...]
go-z-ai accounts remove <name> [--yes]

go-z-ai account detect                              # type and region of the current key (free)
go-z-ai account status [--watch 5m]                 # can the key authenticate and spend?
go-z-ai account balance                             # pay-as-you-go wallet balance
go-z-ai account subscriptions                       # GLM Coding Plan subscriptions

go-z-ai usage quota                                 # Coding Plan quota windows, reset times, pace
go-z-ai validate                                    # free request (list models) to check the key
```

- 除非给出了 `--type`，`accounts add` 会通过一次免费探测来检测账户类型和区域。
  `--region` 表示该 key 是在哪个网关上签发的；启用检测时，它是探测的起点。
  使用 `--type` 时不做任何检测，因此对于中国的 key，请自己传入 `--region china`。
- `accounts quota` 和 `accounts usage` 覆盖所有已存储的账户（可用 `--only`
  限制，可重复）；pay-as-you-go 账户会被跳过，因为 monitor 端点仅适用于
  coding-plan。`accounts usage` 在 8 天及以内按小时分桶，在 9 天及以上按天分桶
  （`--days` 默认为 14）。
- `account status` 通过 quota 端点免费检查 Coding Plan 的 key。pay-as-you-go 的
  key 需要一次最小的计费请求（Z.AI 没有针对它们的余额查询 API），而且
  `--watch` 会在每次重新检查时计费。
- `account balance` 和 `account subscriptions` 使用 biz 端点，它们尚未在真实账户上
  验证过（见 [路线图](roadmap.md)）。
- `usage quota` 会显示每个 credit 窗口及其重置时间和节奏，外加一条高峰时段提示
  （周一至周五 14:00–18:00 UTC+8）。`usage` 没有其他子命令；若要同时查看多个
  账户，请使用 `accounts quota` 和 `accounts usage`。

## 编码工具（GLM Coding Plan）

把 Claude Code、Codex、OpenCode、Crush 或 Factory Droid 接到你的 GLM
Coding Plan。完整流程见 [编码工具](coding-tools.md)。

```bash
go-z-ai coding auth <plan> <key> [--no-validate]   # validate + store a credential
go-z-ai coding auth revoke
go-z-ai coding auth reload <tool>     # re-push stored creds into a tool's config
go-z-ai coding load <tool> [--plan ...] [--key ...]   # write it into a tool's config
go-z-ai coding unload <tool>          # says so and succeeds if the tool isn't configured
go-z-ai coding status [--format json]
go-z-ai coding tools [--format json]  # list supported tools + install status
go-z-ai coding doctor                 # health check; exits non-zero on problems

go-z-ai coding mcp add <tool> [--server id]... [--plan ...] [--key ...]
go-z-ai coding mcp remove <tool> [--server id]...
go-z-ai coding mcp status [--format json]
```

工具 ID：`claude-code`、`codex`、`opencode`、`crush`、`factory-droid`
（别名 `claude`、`droid`、`factory`）。套餐：`glm_coding_plan_global`、
`glm_coding_plan_china`。

`coding load` 和 `coding auth` 接受 Claude Code 的调优 flag：`--haiku`、
`--sonnet`、`--opus`（覆盖某一档位的模型）、`--no-model-mapping`、
`--auto-compact-window`、`--max-thinking-tokens`、`--max-output-tokens`。

`coding mcp` 会注册 Z.AI 官方的四个 MCP 服务器；`--server`（可重复）用于选择
其中的几个，默认是该工具支持的每一个服务器。Codex 只接受本地的 Vision
服务器：它的 streamable-HTTP MCP 客户端会拒绝托管的那几个（openai/codex#14793），
所以请求其中任何一个都会失败。对这些服务器的调用会消耗套餐配额。

| Server ID | 作用 |
|---|---|
| `zai-mcp-server` | Vision：截图、图表、UI、图像和视频。通过 `npx` 在本地运行，因此需要 Node.js |
| `web-search-prime` | 网页搜索（托管） |
| `web-reader` | 抓取并阅读网页（托管） |
| `zread` | 读取和搜索 GitHub 仓库（托管） |

托管的服务器使用套餐 key 向该套餐所在区域进行鉴权。

## 文件与批量任务

```bash
go-z-ai files upload <file> [--purpose batch|user_data|agent|file-extract|code-interpreter|voice-clone-input]
go-z-ai files list [--purpose ...]
go-z-ai files delete <file-id>
go-z-ai files download <file-id> <output-path>

go-z-ai batch create <input-file-id> [--endpoint ...] [--auto-delete-input] [--metadata key=value]...
go-z-ai batch status <batch-id>
go-z-ai batch list [--after ...] [--limit N]
go-z-ai batch cancel <batch-id>
```

`files upload` 默认使用 `--purpose batch`。批量任务会异步处理来自一个 JSONL
文件的大量聊天补全或 embedding 请求——先上传文件，再用返回的 file ID 创建批量
任务。`--auto-delete-input` 会在批量任务结束时删除输入文件；`--metadata` 可重复
使用。`batch status` 会打印输出文件和错误文件的 ID，你可以用 `files download`
获取它们。

## 媒体生成

```bash
# Images — default model glm-image (cogview-4-250304 also supported)
go-z-ai image generate <prompt> [--model glm-image|cogview-4-250304] [--size ...] [--quality hd|standard] [--async]
go-z-ai image status <id>
# --quality: hd is the default (~20s); standard is faster (~5-10s).
# --async is supported by glm-image only.

# Video — always async (cogvideox-3 | viduq1-text | viduq1-image | viduq1-start-end | vidu2-image | vidu2-start-end | vidu2-reference)
go-z-ai video generate --prompt "..." [--model ...] [--image URL]... [--size ...] [--aspect-ratio ...] \
    [--duration N] [--audio] [--off-peak]
go-z-ai video status <id>
# --off-peak queues the task for off-peak processing at a lower price.
# Model-specific: --fps and --quality (cogvideox-3), --style (viduq1-text),
# --movement (Vidu), --aspect-ratio (Vidu text/reference). Default model cogvideox-3.

# Audio
go-z-ai audio transcribe <file> [--model ...] [--hotword word]... [--prompt ...]   # .wav/.mp3, <=25MB, <=30s
go-z-ai audio speech <text> <output-path> [--voice ...] [--speed N] [--audio-format wav|pcm]

# Voice cloning (pairs with audio speech --voice)
go-z-ai voice clone <voice-name> <sample-file-id> <preview-text>
go-z-ai voice list [--name ...] [--type OFFICIAL|PRIVATE]
go-z-ai voice delete <voice-id>
```

`audio transcribe` 默认使用目录中的 ASR 模型（`client.DefaultASRModel`），
`audio speech` 使用 TTS 模型（`client.DefaultTTSModel`）。`--voice` 接受系统音色
（`tongtong`（默认）、`chuichui`、`xiaochen`、`jam`、`kazi`、`douji`、`luodo`）
或克隆音色的 ID；`--speed` 取值 0.5–2。`voice clone` 所用的样本文件必须已经通过
`files upload --purpose voice-clone-input` 上传。

## 文档解析与 OCR

```bash
# Layout OCR (glm-ocr) — image/PDF into Markdown
go-z-ai ocr parse <file-or-url> [--start-page N] [--end-page N]
go-z-ai ocr handwriting <file> [--probability] [--language ...]

# Document parser (RAG/retrieval preprocessing) — a separate product from OCR
go-z-ai parser parse <file> <file-type>              # synchronous
go-z-ai parser create <file> <tool-type> <file-type> # async: lite|expert|prime
go-z-ai parser result <task-id> <format>              # text|download_link
```

`parser` 与 `ocr` 解决的是不同问题：OCR 从图片中提取版式/文本；而
parser 专为把文档转成可供 RAG 使用的文本而设计，并支持更多工具档位。

## 检索辅助

```bash
go-z-ai embeddings create <text> [--model embedding-3|embedding-2] [--dimensions N]
go-z-ai rerank <query> <documents...> [--top-n N]
```

Embeddings 会路由到 `open.bigmodel.cn`——关于原因以及对鉴权的影响，见
[账户与配额 § 区域网关](accounts-and-quota.md#区域网关apizai--openbigmodelcn)。
`--dimensions` 仅适用于 `embedding-3`（256、512、1024 或 2048）。Rerank 使用
chat/PaaS 根地址（`--base-url`，或 `--region` 的默认值）；它并不固定指向中国
平台主机。

## 内容审核

```bash
go-z-ai moderations check <text>
```

会路由到 `open.bigmodel.cn`——注意事项同上面的 Embeddings。

## Agents

```bash
go-z-ai agents invoke <agent-id> <message> [--source-lang ...] [--target-lang ...] [--var key=value]...
go-z-ai agents async-result <agent-id> <async-id> [--conversation-id ID] [--var key=value]...
```

调用 Z.AI 的专用 agents（翻译、幻灯片/海报生成、视频特效模板）。`--var` 用于设置
agent 的自定义变量，可重复使用；`--source-lang` 和 `--target-lang` 是
`source_lang` 和 `target_lang` 这两个变量的简写。`invoke` 会把会话 ID 打印到
stderr；请通过 `--conversation-id` 把它传回 `async-result`。注意：即便调用在
业务层面失败（例如余额不足），Agents API 也会返回 HTTP 200——CLI 会从响应体中
识别出该失败，并将其作为命令错误报告。

## 工具（网页搜索、reader、tokenizer）

```bash
go-z-ai tools web-search <query> [--engine ...] [--count N] [--recency ...] [--domain ...] [--content-size medium|high]
go-z-ai tools web-reader <url> [--no-images]
go-z-ai tools tokenizer <text> [--model ...]
```

`web-search` 的 flag：

| Flag | Purpose |
|---|---|
| `--engine string` | `search-prime`（默认）、`search_std`、`search_pro`、`search_pro_sogou`、`search_pro_quark` |
| `--count int` | 结果数量，1–50（默认 10） |
| `--recency string` | `oneDay`、`oneWeek`、`oneMonth`、`oneYear`、`noLimit` |
| `--domain string` | 将结果限制在某一个域名 |
| `--content-size string` | 结果内容的大小：`medium` 或 `high` |

`tokenizer` 统计单条用户消息的 token 数；`--model` 默认为目录中的默认 chat
模型。

## Anthropic 兼容端点

```bash
go-z-ai anthropic messages <prompt> [--model ...] [--max-tokens 1024] \
    [--system ...] [--temperature ...] [--thinking-budget N] [--stream]
```

调用 Z.AI 的 Anthropic 协议端点（所选 `--region` 上的
`/api/anthropic/v1/messages`）——即 GLM Coding Plan 让 Claude Code 指向的同一个
端点——而不是 OpenAI 风格的 `chat create`。`--model` 默认为目录中的默认 chat
模型；Messages API 要求提供 `--max-tokens`，默认值为 1024。打印消息文本（或用
`--stream` 流式输出文本增量；配合 `--format json` 时每行一个事件）；
`--thinking-budget N` 启用扩展思考并把推理过程打印到 stderr。Go API 见
[库使用指南](library-guide.md#anthropic-兼容的-messages-api)。

## Responses 端点（Codex 协议）

```bash
go-z-ai responses create <prompt> [--model ...] [--instructions ...] \
    [--effort low|high|max] [--max-output-tokens N] [--stream] [--show-reasoning]
```

调用 Z.AI 的 OpenAI Responses 协议端点（所选 `--region` 上的
`/api/v1/responses`）——即 Codex 所配置的接口，文档面向 GLM Coding Plan。
`--model` 默认为目录中的默认 chat 模型，`--effort` 会按它来校验。打印输出文本
（工具调用以及在使用 `--show-reasoning` 时的推理过程输出到 stderr）；`--stream`
打印文本增量，配合 `--format json` 时则每行一个事件。Go API 见
[库使用指南](library-guide.md)。

## 终端 UI

```bash
go-z-ai tui
```

启动一个全屏终端 UI，包含 Chat、Models、Usage、Accounts、Coding、Media、
Tools 等标签页——在同一个交互会话里提供与上面 CLI 命令相同的功能。按 `?` 或
`F1` 查看按键绑定。
