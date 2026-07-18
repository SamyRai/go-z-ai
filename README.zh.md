# Z.AI API 客户端

[English](README.md) | **简体中文** | [Русский](README.ru.md) | [Deutsch](README.de.md) | [Татарча](README.tt.md) | [Türkçe](README.tr.md)

[![CI](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/SamyRai/go-z-ai/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/SamyRai/go-z-ai.svg)](https://pkg.go.dev/github.com/SamyRai/go-z-ai)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/SamyRai/go-z-ai?label=openssf%20scorecard)](https://securityscorecards.dev/view.html?uri=github.com/SamyRai/go-z-ai)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

一个用于 Z.AI（智谱 AI / BigModel）API 平台的 Go 命令行工具和客户端库：
聊天补全、模型、图像、视频、音频、向量、内容审核、重排序、Agent、
批量任务、文件解析、GLM Coding Plan 账户与配额管理，以及 `@z_ai/coding-helper`
的 Go 移植版——用于把 Claude Code、OpenCode、Crush、Factory Droid 和 Cursor
接入你的 GLM Coding Plan。

## 安装

```bash
go install github.com/SamyRai/go-z-ai@latest
```

需要 Go 1.26.4+ 和一个 [Z.AI API Key](https://z.ai/manage-apikey)。
从源码构建、首次鉴权、故障排查请见 **[快速开始 →](docs/zh/getting-started.md)**

## 快速示例

```bash
export ZAI_API_KEY=your_api_key_here
zai-client chat create "用一段话解释 goroutine" --stream
```

```go
import "github.com/SamyRai/go-z-ai/pkg/client"

c, _ := client.NewClientFromEnv()
resp, _ := c.Chat().Create(ctx, client.ChatRequest{
    Model:    "glm-5.2",
    Messages: []client.Message{{Role: "user", Content: "用一段话解释 goroutine"}},
})
fmt.Println(resp.Choices[0].Message.Content)
```

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

## 功能覆盖

聊天（流式、结构化输出、深度思考、函数调用、视觉输入），Anthropic 兼容的
`/v1/messages` 端点，模型、图像、视频、音频（转写 + TTS + 声音克隆），OCR 与
文档解析，向量、内容审核、重排序、Agent、文件、批量任务，GLM Coding Plan
用量 / 配额 / 多账户管理，以及全屏终端 UI（`zai-client tui`）。完整命令列表见
[CLI 参考](docs/zh/cli-reference.md)，Go API 见 [库使用指南](docs/zh/library-guide.md)。

## 与官方 SDK 的对比

Z.AI / 智谱的官方 SDK 覆盖 **Python**（[zai-org/z-ai-sdk-python](https://github.com/zai-org/z-ai-sdk-python)）、
**Node**（[MetaGLM/zhipuai-sdk-nodejs-v4](https://github.com/MetaGLM/zhipuai-sdk-nodejs-v4)）和
**Java**——目前**没有官方 Go SDK**。本仓库填补了这一空白，并在同一套 API 之上
提供了 CLI、TUI、区域网关切换（`api.z.ai` 与 `open.bigmodel.cn`）以及 GLM
Coding Plan 多账户管理。

## 贡献

请见 [CONTRIBUTING.md](CONTRIBUTING.md)。如果你要新增或修改某个 service，
请特别注意本项目的"实测录制"约定（录制真实 API 调用形成 cassette，而不是手写
fixture）。

## 许可证

Apache License 2.0——见 [LICENSE](LICENSE)。

## 支持

- **Z.AI API 文档**：<https://docs.z.ai>
- **问题反馈**：[GitHub Issues](https://github.com/SamyRai/go-z-ai/issues)
- **安全**：见 [SECURITY.md](SECURITY.md)——请不要把安全漏洞作为公开 issue 提交。
