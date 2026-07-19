# Z.AI API 客户端

面向 Z.AI（智谱 AI / BigModel）平台的 Go **CLI**、**库** 与 **TUI**——把所有
GLM 模型能力收进一个工具，并附带 `@z_ai/coding-helper` 的 Go 移植版，用于将
Claude Code、OpenCode、Crush、Factory Droid 和 Cursor 接入你的 GLM Coding Plan。

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
zai-client chat create "用一段话解释 goroutine" --stream
```

```go
// ……或者导入库——无需 CLI。
import "github.com/SamyRai/go-z-ai/pkg/client"

c, _ := client.NewClientFromEnv()
resp, _ := c.Chat().Create(ctx, client.ChatRequest{
    Model:    "glm-5.2",
    Messages: []client.Message{{Role: "user", Content: "用一段话解释 goroutine"}},
})
fmt.Println(resp.Choices[0].Message.Content)
```

更多可运行的示例——流式、异步图像轮询、Anthropic `/v1/messages` 端点——位于
[`examples/`](examples/)。

## 功能

- **聊天**——流式、结构化输出（JSON Schema）、深度思考、函数 / 工具调用、
  视觉输入（`glm-4.6v`/`glm-4.5v`），以及 **Anthropic 兼容的 `/v1/messages`**
  端点（Claude Code 与 Cursor 接入 GLM Coding Plan 时正是访问它）。
- **媒体**——图像生成、视频生成（始终为异步）、音频转写、TTS 以及 GLM-TTS 声音克隆。
- **文档理解**——版面 OCR、手写 OCR，以及用于 RAG 预处理的文档解析。
- **检索**——embeddings、重排序、内置 web search / web reader / tokenizer 工具。
- **内容审核**——通过国内平台端点提供内容审核。
- **Agent**——Z.AI 的专用 Agent（翻译、幻灯片 / 海报生成、视频特效）。
- **批量任务与文件**——用于聊天补全的 JSONL 批量任务，以及文件上传 / 列出 / 下载。
- **GLM Coding Plan**——配额 / 用量监控、多账户管理，以及通过 `zai-client coding`
  把 Claude Code、OpenCode、Crush、Factory Droid 和 Cursor 接入你的订阅。
- **DX**——全屏终端 UI（`zai-client tui`）、区域网关切换（`api.z.ai` ↔
  `open.bigmodel.cn`）、自动重试（退避 + 抖动），以及强类型的 `APIError`（涵盖
  每一个 Z.AI 错误码）。

## 安装

```bash
go install github.com/SamyRai/go-z-ai@latest
```

这会在 `$GOPATH/bin` 下生成一个名为 `go-z-ai` 的二进制。下面的示例都使用更短的
名字 **`zai-client`**——做软链或重命名即可：

```bash
ln -s "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai-client"
# 或者：mv "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai-client"
```

需要 Go 1.26.4+ 以及一个 [Z.AI API key](https://z.ai/manage-apikey/apikey-list)。
从源码构建、首次鉴权、故障排查请见 **[快速开始 →](docs/zh/getting-started.md)**

## 作为 CLI 使用

单个 `zai-client` 二进制覆盖全部能力。每条命令都支持 `--help`；下面是快速导览：

```bash
zai-client chat create "..." --stream          # 聊天（流式、工具调用、视觉输入、结构化输出）
zai-client anthropic messages "..." --stream   # Anthropic 兼容的 /v1/messages
zai-client image|video|audio|voice ...         # 图像 / 视频生成、音频转写、TTS、声音克隆
zai-client ocr|parser ...                      # OCR + 文档解析
zai-client embeddings|rerank|moderations ...   # 检索 + 内容审核
zai-client models list                         # 模型目录 + 定价
zai-client accounts add|use|quota|usage ...    # 多账户 + GLM Coding Plan 监控
zai-client coding auth|load|doctor|mcp ...     # 把 Claude Code / Cursor / 等接入 GLM Coding Plan
zai-client tui                                 # 全屏终端 UI（包含上述全部能力）
zai-client validate                            # 用一次真实调用验证你的 key 可用
```

所有会产生结果的命令都支持 `--format text|json`（JSON 输出到 stdout，进度信息输出
到 stderr，方便用管道接 `jq`）。

→ 完整命令列表：**[CLI 参考](docs/zh/cli-reference.md)**

## 作为 Go 库使用

`pkg/client` 是唯一可公开导入的包；`internal/` 下的所有内容都属于实现细节。
重试、超时、区域网关选择与错误映射都被集中处理——各 service 从不自行构造
`http.Client` 或发起裸请求。

```bash
go get github.com/SamyRai/go-z-ai
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

c, err := client.NewClient(client.Config{
    APIKey: os.Getenv("ZAI_API_KEY"),
    // 可选：BaseURL、Timeout、MaxRetries、RetryDelay、ChinaAPIKey、Region
})
```

各 service 均遵循 `c.<Service>().<Method>(ctx, …)` 模式：

| 访问器 | 覆盖范围 |
|---|---|
| `c.Chat()` | 补全、流式、异步、`RunWithTools` |
| `c.Anthropic()` | Anthropic 协议的 `/v1/messages`（Create、CreateStream） |
| `c.Models()` | 列出、查询，文本 / 视觉 / 免费模型过滤 |
| `c.Images()` / `c.Videos()` | 图像（同步 / 异步）、视频（始终异步） |
| `c.Audio()` / `c.Voice()` | 音频转写、TTS、声音克隆 |
| `c.Layout()` / `c.FileParser()` | OCR + 面向 RAG 的文档转文本 |
| `c.Files()` / `c.Batch()` | 文件上传、批量任务 |
| `c.Agents()` | Z.AI 专用 Agent |
| `c.Embeddings()` / `c.Rerank()` / `c.Moderations()` | 检索 + 内容审核 |
| `c.Tools()` | WebSearch、WebReader、Tokenize |
| `c.Usage()` / `c.Quota()` / `c.Account()` / `c.Detection()` | GLM Coding Plan 监控 |
| `c.GetAsyncResult()` / `c.WaitForResult()` | 异步任务的共享轮询 |

→ 带示例的完整 API：**[库使用指南](docs/zh/library-guide.md)**
→ 生成的参考文档：[pkg.go.dev](https://pkg.go.dev/github.com/SamyRai/go-z-ai)

## 配置

提供凭据的三种方式，按以下优先级解析（最高优先者胜出）：

| 方式 | 适用场景 |
|---|---|
| `--api-key <key>` 参数 | 一次性调用、脚本、CI |
| `--account <name>` 参数 | 在[已存储的账户](docs/zh/accounts-and-quota.md)之间切换 |
| `ZAI_API_KEY` 环境变量（或 `.env` 文件） | 日常本地 shell 使用 |
| 账户库的活动账户 | 在执行 `zai-client accounts use <name>` 之后 |

`.env` 文件是最常见的做法——复制带注释的模板然后编辑即可：

```bash
cp .env.example .env
# 或者指向任意文件：zai-client --config /path/to/config ...
```

```dotenv
ZAI_API_KEY=your_api_key_here
# ZAI_API_BASE_URL=https://api.z.ai/api/paas/v4     # 覆盖聊天端点
# ZAI_REGION=china                                   # 如果你的 key 是在 open.bigmodel.cn 上签发的
# ZAI_CHINA_API_KEY=...                              # 单独的 bigmodel.cn 凭据
# ZAI_ENV=production
```

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
>（`"your-zai-api-key-here"`），供 `zai-client coding load claude-code` 使用。
> 它不是真实配置，也不携带任何凭据。

## 贡献指南

请见 [CONTRIBUTING.md](CONTRIBUTING.md)。如果你要新增或修改某个 service，请特别
留意本项目的"实测验证"约定（录制真实 API 调用形成 cassette，而不是手写 fixture）。

## 许可证

Apache License 2.0——见 [LICENSE](LICENSE)。

## 支持

- **Z.AI API 文档**：<https://docs.z.ai>
- **问题反馈**：[GitHub Issues](https://github.com/SamyRai/go-z-ai/issues)
- **安全**：见 [SECURITY.md](SECURITY.md)——请不要把安全漏洞作为公开 issue 提交。
