# 文档

**English** | **简体中文** | [Русский](../ru/README.md)

> 各语言版本位于 `docs/<lang>/`。英文是权威来源，其它语言版本可能落后于它。
> 改动行为的 PR 请先更新 `docs/en/`。

## 面向用户

使用 `go-z-ai` 命令行工具。

| 文档 | 内容 |
|---|---|
| [快速开始](getting-started.md) | 安装、鉴权、首批命令 |
| [CLI 参考](cli-reference.md) | 每条命令，按功能区域组织 |
| [账户与配额](accounts-and-quota.md) | 多账户、配额 / 用量监控、区域网关（api.z.ai / open.bigmodel.cn） |
| [编码工具](coding-tools.md) | 把 Claude Code / OpenCode / Crush / Factory Droid / Cursor 接入你的 GLM Coding Plan |

## 面向开发者

把 `pkg/client` 作为 Go 库使用，或为本仓库贡献代码。

| 文档 | 内容 |
|---|---|
| [库使用指南](library-guide.md) | 每个服务，附示例——流式、函数 / 工具调用、结构化输出、异步轮询、`Region` 旋钮 |
| [错误处理](error-handling.md) | `APIError`、完整的错误码表、重试行为 |
| [架构](architecture.md) | 包布局、请求门面、区域网关选择、实测验证约定 |
| [路线图与已知限制](roadmap.md) | 哪些尚未验证、未实现，或属于已知 bug——适合作为首个贡献 |
| [贡献指南](../../CONTRIBUTING.md) | 开 PR 前必读——含实测验证 / cassette 约定 |
| [安全策略](../../SECURITY.md) | 如何报告漏洞 |
| [更新日志](../../CHANGELOG.md) | 何时发布了什么 |

## 仓库设置

| 文件 | 内容 |
|---|---|
| [.env.example](../../.env.example) | `.env` 的带注释模板（`ZAI_API_KEY`、`ZAI_API_BASE_URL`、`ZAI_REGION`、多账户指针） |
| [仓库设置清单](../../.github/SETUP.md) | 一次性的 GitHub 配置——分支保护规则集、Dependabot、CodeQL、密钥扫描 |

## 快速链接

- 主 [README](../../README.md)——项目概览、一行安装
- [pkg.go.dev 参考](https://pkg.go.dev/github.com/SamyRai/go-z-ai)——生成的 Go API 文档
- [Issues](https://github.com/SamyRai/go-z-ai/issues) · [Discussions](https://github.com/SamyRai/go-z-ai/discussions)
