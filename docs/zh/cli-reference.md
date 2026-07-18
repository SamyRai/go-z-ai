# CLI 参考

每条命令都支持 `--help`，其中是权威且始终最新的 flag 列表
（`zai-client <command> --help`、`zai-client <command> <subcommand> --help`）。
本页是一次有组织的概览；如果本页与 `--help` 出现不一致，请以 `--help`
为准。

## 全局 flag

以下 flag 适用于每条命令：

| Flag | Description |
|---|---|
| `--api-key string` | Z.AI API Key（或 `ZAI_API_KEY` 环境变量） |
| `--base-url string` | API base URL（默认：`https://api.z.ai/api/paas/v4`） |
| `--account string` | 按名称为本次命令指定一个已存储的账户（见 [账户与配额](accounts-and-quota.md)） |
| `--china-api-key string` | 用于 embeddings/moderations 的 open.bigmodel.cn key（或 `ZAI_CHINA_API_KEY`；未设置时回退到 `--api-key`） |
| `--region string` | 用于 monitor/biz/agents/detection 的区域网关：`global`（api.z.ai，默认）或 `china`（open.bigmodel.cn）。别名 `cn`、`bigmodel`、`west`。或使用 `ZAI_REGION` 环境变量。不会覆盖 `--base-url`。未知值回退到 global。 |
| `--config string` | 配置文件（默认：`.env`） |

每条会产生结果的命令都接受 `--format text\|json`（少数命令在 payload
面向机器时默认使用 `json`——例如 `embeddings`、`moderations`）。在
`json` 模式下，进度/状态信息会输出到 stderr，因此 stdout 始终是有效的
JSON，可以直接管道给 `jq`。

当你的 key 是在 `open.bigmodel.cn` 上签发时，请设置 `--region china`
（或 `ZAI_REGION=china`），这样配额/用量、账户信息、agents 以及账户类型
检测才会落到对应的主机上。它**不会**改变聊天补全的 base URL（那需要用
`--base-url`），也不会改变 embeddings/moderations 的主机（始终是 China
平台）。参见 [账户与配额 § 区域网关](accounts-and-quota.md#regional-gateways-apiza--openbigmodelcn)。

## 聊天补全

```bash
zai-client chat create [message] [flags]
zai-client chat simple [model] [message]
zai-client chat async-result [task-id]
```

`chat create` 是主要入口：

| Flag | Purpose |
|---|---|
| `--model string` | 默认 `glm-5.2` |
| `--stream` | 逐 token 的流式输出 |
| `--async` | 提交后不等结果；用 `chat async-result <task-id>` 轮询 |
| `--temperature float`, `--top-p float`, `--max-tokens int` | 采样控制 |
| `--system string` | 系统消息 |
| `--thinking string`, `--effort string` | 深度思考模式与努力等级（`max\|xhigh\|high\|medium\|low\|minimal\|none`；`xhigh`→`max` 仅 GLM-5.2 支持） |
| `--show-reasoning` | 将 `reasoning_content` 打印到 stderr |
| `--json-schema string` | 结构化输出：`@file.json` 或内联 JSON |
| `--tool string` | 函数调用工具声明：`@tools.json` 或内联 JSON 数组 |
| `--image string`（可重复） | 附加一张图片：一个 URL，或 `@path` 指向本地文件（base64 编码）。需要视觉模型（`glm-4.6v`/`glm-4.5v`） |
| `--stop strings` | 停止序列（可重复，最多 4 个） |
| `--format text\|json` | 输出格式 |

```bash
zai-client chat create "Summarize this in 3 bullets" --model glm-5.2 --stream
zai-client chat create "Describe this" --image @photo.jpg --model glm-4.6v
zai-client chat create "Extract fields" --json-schema @schema.json
```

工具调用只会被打印，不会被 CLI 执行——关于 Go 端 `RunWithTools`
自动执行循环，见 [库使用指南 § 函数调用](library-guide.md#function-calling)。

> **视觉 + 工具调用可能返回 HTTP 401。** 社区反馈（例如
> [claude-code-router#1491](https://github.com/musistudio/claude-code-router/issues/1491)）
> 表明，在某些 GLM 配置下，把视觉模型（在 `glm-4.6v`/`glm-4.5v` 上使用
> `--image`）与函数调用工具（`--tool`）放在同一请求里会被以 401 拒绝——
> 一个已鉴权的 key 也只是在这种组合下失败。如果你遇到这种情况，请拆分
> 工作：用视觉模型处理图片轮次，用文本模型（`glm-5.2`）处理工具调用轮次，
> 而不是把图片和工具一起发送。此处尚未在真实账户上实测复现——见
> [路线图](roadmap.md)。

## 模型

```bash
zai-client models list [--pricing]
zai-client models get [model-id]
zai-client models text | vision | free
```

## 账户、用量与配额

详细说明见 [账户与配额](accounts-and-quota.md)。快速参考：

```bash
zai-client accounts add <name> --api-key <key> [--type coding_plan|pay_as_you_go]
zai-client accounts list [--format json] [--reveal]   # keys masked by default; --reveal for export
zai-client accounts use <name>
zai-client accounts show [name] [--format json] [--reveal]
zai-client accounts quota [--only name...]
zai-client accounts usage [--days N] [--today] [--metric model|tool|both]
zai-client accounts remove <name> [--yes]

zai-client usage quota | summary | account | billing | check [--watch] | detect
zai-client account info | status
zai-client validate
```

## 编码工具（GLM Coding Plan）

把 Claude Code、OpenCode、Crush、Factory Droid 或 Cursor 接到你的 GLM
Coding Plan。完整流程见 [编码工具](coding-tools.md)。

```bash
zai-client coding auth <plan> <key>      # validate + store a credential
zai-client coding auth revoke
zai-client coding load <tool>            # write it into a tool's config
zai-client coding unload <tool>
zai-client coding status
zai-client coding tools                  # list supported tools + install status
zai-client coding doctor                 # health check

zai-client coding mcp add <tool>         # register Z.AI's Vision MCP server
zai-client coding mcp status
zai-client coding mcp remove <tool>
```

## 文件与批量任务

```bash
zai-client files upload <file> [--purpose batch|code-interpreter|agent|voice-clone-input]
zai-client files list [--purpose ...]
zai-client files delete <file-id>
zai-client files download <file-id> <output-path>

zai-client batch create <input-file-id> [--endpoint ...]
zai-client batch status <batch-id>
zai-client batch list [--after ...] [--limit N]
zai-client batch cancel <batch-id>
```

批量任务会异步处理来自一个 JSONL 文件的大量聊天补全请求——先上传文件，
再用返回的 file ID 创建批量任务。

## 媒体生成

```bash
# Images (glm-image | cogview-4-250304)
zai-client image generate <prompt> [--model ...] [--size ...] [--quality hd|standard] [--async]
zai-client image status <id>

# Video — always async (cogvideox-3 | viduq1-text | viduq1-image | vidu2-image | ...)
zai-client video generate --prompt "..." [--model ...] [--duration N] [--aspect-ratio ...]
zai-client video status <id>

# Audio
zai-client audio transcribe <file>                       # glm-asr, .wav/.mp3, <=25MB, <=30s
zai-client audio speech <text> <output-path> [--voice ...] [--speed N] [--format wav|pcm]

# Voice cloning (pairs with audio speech --voice)
zai-client voice clone <voice-name> <sample-file-id> <preview-text>
zai-client voice list [--name ...] [--type OFFICIAL|PRIVATE]
zai-client voice delete <voice-id>
```

## 文档解析与 OCR

```bash
# Layout OCR — image/PDF into Markdown
zai-client ocr parse <file-or-url> [--start-page N] [--end-page N]
zai-client ocr handwriting <file> [--probability]

# Document parser (RAG/retrieval preprocessing) — a separate product from OCR
zai-client parser parse <file> <file-type>              # synchronous
zai-client parser create <file> <tool-type> <file-type> # async: lite|expert|prime
zai-client parser result <task-id> <format>              # text|download_link
```

`parser` 与 `ocr` 解决的是不同问题：OCR 从图片中提取版式/文本；而
parser 专为把文档转成可供 RAG 使用的文本而设计，并支持更多工具档位。

## 检索辅助

```bash
zai-client embeddings create <text> [--model embedding-3|embedding-2] [--dimensions N]
zai-client rerank <query> <documents...> [--top-n N]
```

embeddings 会路由到 `open.bigmodel.cn`——关于原因以及对鉴权的影响，见
[账户与配额 § 区域网关](accounts-and-quota.md#regional-gateways-apiza--openbigmodelcn)。

## 内容审核

```bash
zai-client moderations check <text>
```

同样会路由到 `open.bigmodel.cn`——注意事项同上面的 embeddings。

## Agents

```bash
zai-client agents invoke <agent-id> <message> [--source-lang ...] [--target-lang ...]
zai-client agents async-result <agent-id> <async-id>
```

调用 Z.AI 的专用 agents（翻译、幻灯片/海报生成、视频特效模板）。注意：
即便调用在业务层面失败（例如余额不足），Agents API 也会返回 HTTP 200——
CLI 会从响应体中报告该失败，而不是作为命令错误抛出。

## 工具（网页搜索、reader、tokenizer）

```bash
zai-client tools web-search <query> [--engine ...] [--count N]
zai-client tools web-reader <url> [--no-images]
zai-client tools tokenizer <text> [--model ...]
```

## Anthropic 兼容端点

```bash
zai-client anthropic messages <prompt> [--model glm-4.6] [--max-tokens 1024] \
    [--system ...] [--temperature ...] [--thinking-budget N] [--stream]
```

调用 Z.AI 的 Anthropic 协议端点（`/api/anthropic/v1/messages`）——即
GLM Coding Plan 让 Claude Code 指向的同一个端点——而不是 OpenAI 风格的
`chat create`。打印消息文本（或用 `--stream` 流式输出文本增量）；
`--thinking-budget N` 启用扩展思考并把推理过程打印到 stderr。Go API 见
[库使用指南](library-guide.md#anthropic-compatible-messages-api)。

## 终端 UI

```bash
zai-client tui
```

启动一个全屏终端 UI，包含 Chat、Models、Usage、Accounts、Coding、Media、
Tools 等标签页——在同一个交互会话里提供与上面 CLI 命令相同的功能。
