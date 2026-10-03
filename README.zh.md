# go-z-ai

面向 Z.AI（智谱 AI / BigModel）平台的 Go **CLI**、**库** 与 **TUI**——把所有
GLM 模型能力收进一个工具，并附带 `@z_ai/coding-helper` 的 Go 移植版，用于将
Claude Code、Codex、OpenCode、Crush 和 Factory Droid 接入你的 GLM Coding Plan。

[English](README.md) | **简体中文** | [Русский](README.ru.md) | [Deutsch](README.de.md) | [Татарча](README.tt.md) | [Türkçe](README.tr.md)

[![CI](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SamyRai/go-z-ai.svg)](https://pkg.go.dev/github.com/SamyRai/go-z-ai)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/SamyRai/go-z-ai?label=openssf%20scorecard)](https://securityscorecards.dev/viewer/?uri=github.com/SamyRai/go-z-ai)
[![Latest release](https://img.shields.io/github/v/release/SamyRai/go-z-ai)](https://github.com/SamyRai/go-z-ai/releases)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## 快速示例

```bash
# 1. 配置（以下任一方式均可——环境变量、.env 文件，或 --config <file>）
export ZAI_API_KEY=your_api_key_here
# 或者：cp .env.example .env，然后编辑 .env

# 2. 使用 CLI
go-z-ai chat create "用一段话解释 goroutine" --stream
```

```go
// ……或者导入库——无需 CLI。
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func main() {
	c, err := client.NewClientFromEnv() // 读取 ZAI_API_KEY
	if err != nil {
		log.Fatal(err)
	}
	req := client.ChatRequest{
		Model:    client.DefaultModel,
		Messages: []client.Message{{Role: "user", Content: "Explain goroutines in one paragraph"}},
	}
	for chunk, err := range c.Chat().Stream(context.Background(), req) {
		if err != nil {
			log.Fatal(err)
		}
		if len(chunk.Choices) > 0 {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}
	fmt.Println()
}
```

更多可运行的示例——工具调用、结构化输出、视觉输入、异步图像轮询、Anthropic
`/v1/messages` 端点——位于 [`examples/`](examples/)。

## 功能

- **聊天**——流式（库中为 Go 迭代器）、推理控制（`reasoning_effort`、思考开关）、
  函数 / 工具调用（另有内置的 `web_search`、`retrieval` 和 `mcp` 工具类型）、
  在提示词中给出 schema 的 JSON 对象输出、多模态输入（图像、视频、文件），以及
  **Anthropic 兼容的 `/v1/messages`** 端点（Claude Code 接入 GLM Coding Plan
  时正是访问它）。
- **模型**——精选目录（上下文窗口、最大输出、价格、能力、可接受的推理强度），
  位于 `go-z-ai models` 和 `client.CatalogEntry` 之后；默认模型见[下文](#默认模型)。
- **媒体**——图像生成、视频生成（始终为异步，可选择错峰排队）、音频转写、TTS
  以及 GLM-TTS 声音克隆。
- **文档理解**——版面 OCR、手写 OCR，以及用于 RAG 预处理的文档解析。
- **检索**——embeddings、重排序、内置 web search / web reader / tokenizer 工具。
- **内容审核**——通过国内平台端点提供内容审核。Embeddings 与内容审核由
  `open.bigmodel.cn` 提供，可能受账户权限限制；详见
  [路线图与已知限制](docs/zh/roadmap.md)。
- **Agent**——Z.AI 的专用 Agent（翻译、幻灯片 / 海报生成、视频特效）。
- **批量任务与文件**——用于聊天补全的 JSONL 批量任务，以及文件上传 / 列出 / 下载 / 删除。
- **GLM Coding Plan**——账户类型检测、基于额度（credit）的配额 / 用量监控、
  多账户管理，以及通过 `go-z-ai coding` 把 Claude Code、Codex、OpenCode、Crush
  和 Factory Droid 接入你的订阅，并注册 Z.AI 的四个官方 MCP 服务器（Vision、
  网络搜索、网页读取、Zread）。
- **DX**——全屏终端 UI（`go-z-ai tui`：聊天、模型、用量、账户、编码、媒体和工具
  标签页）、一个 `--region` 开关（`api.z.ai` ↔ `open.bigmodel.cn`）即可选定所有
  端点、自动重试（退避 + 抖动，并遵循 `Retry-After`），以及强类型的 `APIError`
  （映射 Z.AI 错误码，未知错误码回退为 HTTP 状态码）。

### 默认模型

调用方应使用 `pkg/client` 中的常量来选取默认值，而不是硬编码模型 ID：

| 常量 | 模型 | 用途 |
|---|---|---|
| `client.DefaultModel` | `glm-5.3` | 聊天、Anthropic 端点、tokenizer。1M 上下文，推理强度 `low`/`high`/`max` |
| `client.DefaultFastModel`, `client.DefaultVisionModel` | `glm-5.3-flash` | 快速、低成本档位；原生多模态（图像、视频、文件输入） |
| `client.DefaultOCRModel` | `glm-ocr` | `ocr`、`Layout().Parse` |
| `client.DefaultASRModel` | `glm-asr-2512` | `audio transcribe`（最长 30 秒的音频片段） |
| `client.DefaultTTSModel` | `glm-tts` | `audio speech` |
| `client.ModelGLMImage`, `client.ModelCogView4` | `glm-image`（默认）、`cogview-4-250304` | `image generate` |
| `client.VideoModels` | `cogvideox-3`（默认）、`viduq1-*`、`vidu2-*` | `video generate` |

目录还涵盖 `glm-5.3-flashx`、`glm-5.2`、GLM-4.x 系列以及免费档
（`go-z-ai models free`）。它是精选快照（最近更新于 2026-10-02）——
`go-z-ai models list` 会显示上下文、最大输出、价格、能力和推理强度，实时
`/models` 的值优先于目录中的值。

## 安装

```bash
go install github.com/SamyRai/go-z-ai@latest
```

这会在 `$GOPATH/bin` 下生成一个名为 `go-z-ai` 的二进制。

```bash
# 可选简写别名: ln -s "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai"
```

需要 Go 1.26.4+ 以及一个 [Z.AI API key](https://z.ai/manage-apikey/apikey-list)。
从源码构建、首次鉴权、故障排查请见 **[快速开始 →](docs/zh/getting-started.md)**

## 作为 CLI 使用

单个 `go-z-ai` 二进制覆盖全部能力。每条命令都支持 `--help`；下面是快速导览：

```bash
go-z-ai chat create "..." --stream          # 聊天（流式、推理强度、工具调用、图像 / 视频 / 文件输入、JSON 输出）
go-z-ai anthropic messages "..." --stream   # Anthropic 兼容的 /v1/messages
go-z-ai responses create "..." --stream     # OpenAI Responses 协议（/api/v1，Codex 所使用）
go-z-ai image|video|audio|voice ...         # 图像 / 视频生成、音频转写、TTS、声音克隆
go-z-ai ocr|parser ...                      # OCR + 文档解析
go-z-ai embeddings|rerank|moderations ...   # 检索 + 内容审核
go-z-ai models list                         # 模型目录：上下文、价格、能力、推理强度
go-z-ai account detect|status               # key 类型（coding plan / 按量付费）、区域、健康状态
go-z-ai accounts add|use|quota|usage ...    # 多账户 + GLM Coding Plan 监控
go-z-ai usage quota                         # 当前 key 的 GLM Coding Plan 配额窗口
go-z-ai coding auth|load|doctor|mcp ...     # 把 Claude Code / Codex / OpenCode / Crush / Factory Droid 接入 GLM Coding Plan
go-z-ai tui                                 # 全屏终端 UI（包含上述全部能力）
go-z-ai validate                            # 确认你的 key 可用（一次免费请求）
```

大多数会输出结果的命令都支持 `--format text|json`（JSON 输出到 stdout，进度信息
输出到 stderr，方便用管道接 `jq`）。根级 `--region global|china` 参数（环境变量
`ZAI_REGION`）会选定所有端点；`--base-url` 仅覆盖聊天 / PaaS 根地址。

### 编码工具

`go-z-ai coding` 是 Z.AI `@z_ai/coding-helper` 的 Go 移植版。支持的工具：
`claude-code`、`codex`、`opencode`、`crush` 和 `factory-droid`。

```bash
go-z-ai coding auth glm_coding_plan_global <key>   # 校验并保存套餐 key（或 glm_coding_plan_china）
go-z-ai coding load claude-code                    # 写入该工具的配置（另有：codex, opencode, crush, factory-droid）
go-z-ai coding mcp add claude-code                 # 注册官方 MCP 服务器（用 --server 选择）
go-z-ai coding doctor                              # 健康检查；出现问题时以非零状态码退出
```

- 对于 Claude Code，haiku 档映射到 `client.DefaultFastModel`，sonnet / opus 档
  映射到 `client.DefaultModel`，并带有 Claude Code 的 `[1m]` 1M 上下文后缀。
  `--haiku`、`--sonnet`、`--opus`、`--no-model-mapping`、`--auto-compact-window`、
  `--max-thinking-tokens` 和 `--max-output-tokens` 可在 `coding load` 与
  `coding auth` 上调整该映射。
- `coding mcp add|remove <tool>` 会注册 Z.AI 的四个官方 MCP 服务器：
  `zai-mcp-server`（Vision；通过 `npx` 在本地运行，需要 Node.js）、
  `web-search-prime`、`web-reader` 和 `zread`（托管服务，使用套餐 key 鉴权）。
  默认注册该工具支持的全部服务器——Codex 只支持 Vision（openai/codex#14793）；
  `coding mcp status` 会显示各工具当前已有的配置。MCP 调用会消耗套餐配额。
- 对于 Codex，`coding load codex` 会把一个使用 Responses 协议（`/api/v1`）的
  ZAI provider 写入 `~/.codex/config.toml`，并把模型的元数据写入
  `~/.codex/models.json`。

→ 完整命令列表：**[CLI 参考](docs/zh/cli-reference.md)**

## 作为 Go 库使用

`pkg/client` 就是这个库（仅依赖标准库）；`pkg/observe` 是可选的 OpenTelemetry
hook 适配器。`internal/` 下的所有内容都属于实现细节。重试、超时、区域网关选择
与错误映射都被集中处理——各 service 从不自行构造 `http.Client` 或发起裸请求。

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

// 从环境变量读取：ZAI_API_KEY、ZAI_API_BASE_URL、ZAI_REGION、
// ZAI_CHINA_API_KEY、ZAI_MONITOR_TIMEZONE。
c, err := client.NewClientFromEnv()

// 或者显式配置：
c, err = client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    // 可选：Region、BaseURL、Timeout、MaxRetries、RetryDelay、ChinaAPIKey、
    // UserAgent、Hooks
})
```

各 service 均遵循 `c.<Service>().<Method>(ctx, …)` 模式：

| 访问器 | 覆盖范围 |
|---|---|
| `c.Chat()` | `Create`、`Stream`（迭代器）、`CreateAsync`、`RunWithTools` |
| `c.Anthropic()` | Anthropic 协议的 `/v1/messages`（`Create`、`Stream`） |
| `c.Responses()` | `/api/v1` 上的 OpenAI Responses 协议——Codex 所用的接口（`Create`、`Stream`） |
| `c.Models()` | 列出、查询，文本 / 视觉 / 免费模型过滤 |
| `c.Images()` / `c.Videos()` | 图像（`Generate`、`GenerateAsync`）、视频（始终异步） |
| `c.Audio()` / `c.Voice()` | 音频转写、TTS、声音克隆 |
| `c.Layout()` / `c.FileParser()` | OCR + 面向 RAG 的文档转文本 |
| `c.Files()` / `c.Batch()` | 文件上传、批量任务 |
| `c.Agents()` | Z.AI 专用 Agent |
| `c.Embeddings()` / `c.Rerank()` / `c.Moderations()` | 检索 + 内容审核 |
| `c.Tools()` | WebSearch、WebReader、Tokenize |
| `c.Quota()` / `c.Account()` / `c.Detection()` | GLM Coding Plan 配额与用量、账户信息、账户类型检测 |
| `c.GetAsyncResult()` / `c.WaitForResult()` | 异步任务的共享轮询 |

值得了解的几点：

- 默认值来自模型目录：`client.DefaultModel`、`client.DefaultFastModel`、
  `client.DefaultVisionModel`，以及 OCR / ASR / TTS 相关常量。
  `ChatRequest.ReasoningEffort` 会按模型可接受的档位校验（GLM-5.3 为
  `client.EffortLow`、`EffortHigh`、`EffortMax`）；`client.CatalogEntry(model)`
  和 `Pricing.Cost(usage)` 可获取上下文大小与价格。
- `Chat().Stream`、`Anthropic().Stream` 和 `Responses().Stream` 返回
  `iter.Seq2` 迭代器。跳出循环会关闭该流，而流内的错误数据块（chunk）会以
  `*APIError` 结束该流。
- Z.AI 没有 `json_schema` 响应格式。要获得结构化输出，请设置
  `ResponseFormat: client.JSONObjectFormat()`，并把
  `client.JSONSchemaPrompt(schema)` 放进 system prompt。
- `Config.Region`（`client.RegionGlobal` 或 `client.RegionChina`）决定所有 URL；
  `Config.BaseURL` 默认为 `Region.PaaSBaseURL()`。

→ 带示例的完整 API：**[库使用指南](docs/zh/library-guide.md)**
→ 生成的参考文档：[pkg.go.dev](https://pkg.go.dev/github.com/SamyRai/go-z-ai)

## 配置

凭据按以下优先级解析（最高优先者胜出）：

| 方式 | 适用场景 |
|---|---|
| `--api-key <key>` 参数 | 一次性调用、脚本、CI |
| `--account <name>` 参数 | 在[已存储的账户](docs/zh/accounts-and-quota.md)之间切换 |
| `ZAI_API_KEY` 环境变量（或 `.env` 文件） | 日常本地 shell 使用 |
| 账户库的活动账户 | 在执行 `go-z-ai accounts use <name>` 之后 |

`.env` 文件是最常见的做法——复制带注释的模板然后编辑即可：

```bash
cp .env.example .env
# 或者指向任意文件：go-z-ai --config /path/to/config ...
```

```dotenv
ZAI_API_KEY=your_api_key_here
# ZAI_API_BASE_URL=https://api.z.ai/api/paas/v4     # 覆盖聊天端点
# ZAI_REGION=china                                   # 如果你的 key 是在 open.bigmodel.cn 上签发的；会选定所有端点
# ZAI_CHINA_API_KEY=...                              # 单独的 bigmodel.cn 凭据（embeddings / 内容审核）
# ZAI_MONITOR_TIMEZONE=UTC                           # 配额 / 用量 API 所用时区（默认 UTC+8）
```

库中的 `client.NewClientFromEnv()` 会从进程环境变量中读取同样的变量（它不会加载
`.env` 文件）。

→ 完整参考（多账户、区域网关、配额窗口）：
**[账户与配额](docs/zh/accounts-and-quota.md)**

## 文档

**[完整文档索引 →](docs/zh/README.md)**

| | |
|---|---|
| [快速开始](docs/zh/getting-started.md) | [CLI 参考](docs/zh/cli-reference.md) |
| [账户与配额](docs/zh/accounts-and-quota.md) | [编码工具](docs/zh/coding-tools.md) |
| [库使用指南](docs/zh/library-guide.md) | [错误处理](docs/zh/error-handling.md) |
| [架构](docs/zh/architecture.md) | [路线图与已知限制](docs/zh/roadmap.md) |
| [贡献指南](CONTRIBUTING.md) | [安全策略](SECURITY.md) |
| [行为准则](CODE_OF_CONDUCT.md) | [更新日志](CHANGELOG.md) |

## 与官方 SDK 的对比

Z.AI / 智谱为 **Python**（[zai-org/z-ai-sdk-python](https://github.com/zai-org/z-ai-sdk-python)，
PyPI 包名 `zai-sdk`）、**Node**（[MetaGLM/zhipuai-sdk-nodejs-v4](https://github.com/MetaGLM/zhipuai-sdk-nodejs-v4)）
和 **Java**（[MetaGLM/zhipuai-sdk-java-v4](https://github.com/MetaGLM/zhipuai-sdk-java-v4)）
提供了官方 SDK。目前**没有官方 Go SDK**——`go-z-ai` 填补了这一空白，并在同一套
API 之上叠加了 CLI、TUI、区域网关切换（`api.z.ai` ↔ `open.bigmodel.cn`）以及 GLM
Coding Plan 多账户管理。

> ℹ️ 仓库根目录下的 `zai-claude-config.json` 是一个**模板**，其中是占位值
>（`"your-zai-api-key-here"`），用于说明 `go-z-ai coding load claude-code`
> 会写入哪些设置。程序不会读取它，它不是真实配置，也不携带任何凭据；如需当前的
> 模型映射，请运行 `go-z-ai coding load claude-code`。
>
> ⚠️ **使用政策。** Z.AI 的 coding 端点仅限「官方支持的工具」，禁止基于 SDK 的
> 访问；详见 [Coding Tools — Compliance](docs/en/coding-tools.md#compliance--usage-policy-)。
> `go-z-ai` 在每个请求中发送标识性的 `User-Agent` 头，其 `coding` 子命令用于接入
> 官方支持的工具。在获得明确列入之前，从自定义集成直接通过 `pkg/client` 访问
> coding 端点，风险自负。

## 贡献指南

请见 [CONTRIBUTING.md](CONTRIBUTING.md)。如果你要新增或修改某个 service，请特别
留意本项目的"实测验证"约定（录制真实 API 调用形成 cassette，而不是手写 fixture）。

## 许可证

Apache License 2.0——见 [LICENSE](LICENSE)。

## 支持

- **Z.AI API 文档**：<https://docs.z.ai>
- **问题反馈**：[GitHub Issues](https://github.com/SamyRai/go-z-ai/issues)
- **安全**：见 [SECURITY.md](SECURITY.md)——请不要把安全漏洞作为公开 issue 提交。
