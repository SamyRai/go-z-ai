# 架构

## 目录

- [包布局](#包布局)
- [CLI 约定](#cli-约定)
- [请求门面](#请求门面)
- [重试与超时设计](#重试与超时设计)
- [为什么有些服务会访问不同的主机](#为什么有些服务会访问不同的主机)
- [实测验证约定](#实测验证约定)
- [扩展结构化的查找表](#扩展结构化的查找表)
- [凭证文件安全性](#凭证文件安全性)
- [TUI](#tui)

## 包布局

```text
main.go               一个五行的入口：package main → internal/cli.Execute()
internal/cli/         CLI 命令（package cli），每个命令组一个文件：
                      chat.go、accounts_cli.go、coding_cli.go ……
pkg/client/           Go 库——每个 API 服务一个文件，不依赖 CLI/TUI
internal/accounts/    多账户凭证存储（~/.config/go-z-ai/accounts.json）
internal/coding/      GLM Coding Plan 凭证存储 + 各工具的配置写入器
internal/usageview/   纯展示辅助函数（时间窗口、热力图、格式化）——
                      由 CLI 和 TUI 共享，保证两者输出永远不会漂移
internal/tui/         Bubble Tea 终端 UI，每个标签页一个子包
internal/fileinput/   FileOrURL：URL 直接透传，本地路径会被 base64 编码——
                      由 `ocr parse` 和 TUI 媒体标签页共享
```

`pkg/client` 对任何 CLI 或 TUI 特定的东西都没有依赖——它被设计为可独立导入
（参见 [库使用指南](library-guide.md)），并且是**唯一**的公开包。`internal/`
下的所有内容都是实现细节，编译器会禁止外部代码导入它们，因此 CLI/TUI 层可以
自由重构。CLI 和 TUI 都是 `pkg/client` 的轻量调用方。`go install
github.com/SamyRai/go-z-ai@latest` 仍然会把根目录的 `main.go` 构建为
`go-z-ai` 二进制。

## CLI 约定

需要 API 客户端的命令处理器注册为 `RunE: runWithClient(runX)`，并把已解析好的
`*client.Client` 作为第三个参数——`runWithClient`（位于 `internal/cli/common.go`）
通过 `getClient` 只解析一次，这样各处理器就不必各自重复"解析并校验"的前置逻辑。
凭证优先级本身位于 `resolveConfig`（flag → `--account` →
`ZAI_API_KEY`/`KEY` → 当前激活账户），在 `credentials_test.go` 中有单元测试覆盖。

输出格式是统一的：每个打印结果的命令都通过 `addFormatFlag` 注册共享的
`--format` flag，并通过 `emit(cmd, v, textFn)` 渲染——后者在 `--format json`
时输出美观的 JSON，否则运行可读性更好的 `textFn`。支持 JSON 的命令的进度信息
都写到 stderr，以保证 stdout 始终是合法的 JSON。

## 请求门面

每个服务方法最终都会汇聚到三个 `Client` 方法之一——服务永远不会自己构造
`http.Client` 或直接发起原始请求：

```text
doRequest(ctx, method, endpoint, body, result)
  → doRequestBase(ctx, baseURL, method, endpoint, body, result)   // 用于非默认 base URL
    → doRequestBaseKey(ctx, baseURL, apiKey, method, endpoint, body, result)  // 用于换用不同的凭证
```

正是这一层把重试/退避、错误解析和鉴权集中到了一处。需要不同 base URL（Agents）
或不同凭证（Embeddings/Moderations，使用 `ChinaAPIKey`）的服务会调用更下层
的链路——它绝不会绕过这条链路。

`sendMultipart` 是 multipart/form-data 上传（文件、音频转写、文档解析）的并行
路径——它**不**进行重试，因为在瞬时失败时重新上传文件应由调用方决定，而不是
一个安全的默认行为。

## 重试与超时设计

- `Config.Timeout` 限定的是拨号 + TLS 握手 + 等待响应头的时间——故意**不**用
  整个 `http.Client.Timeout`，因为后者会在生成过程中途截断一次长时间的
  `CreateStream` SSE 读取。
- 重试适用于 429/5xx/网络错误，采用指数退避、抖动（最高 25%）并支持
  `Retry-After` 响应头，最多重试到 `Config.MaxRetries`（默认 3）。每次重试前
  都会先检查 `ctx.Err()`，这样被取消的 context 会立即中止，而不是先睡完一次
  退避再退出。
- 流式（`CreateStream`）只重试*连接*尝试——一旦 SSE 流真正开始，流中途的失败
  会直接上报给调用方，而不是被静默重试（无法知道调用方已经消费了多少响应内容）。

## 为什么有些服务会访问不同的主机

| 服务 | Base URL | 原因 |
|---|---|---|
| Embeddings、Moderations | `open.bigmodel.cn` | 唯一为这些端点提供文档的平台——`api.z.ai` 的文档索引两者都未提及。实测验证发现，普通的 z.ai 密钥在两个平台上鉴权表现一致，因此 `Config.ChinaAPIKey` 默认会回退到 `Config.APIKey`。 |
| Agents | 默认 `https://api.z.ai/api`（裸根路径，没有 `/paas/v4`）；当 `Config.Region = RegionChina` 时为 `https://open.bigmodel.cn/api` | 实测验证——把 `/v1/agents` 嵌套在 chat-completions 的 base 下会返回 404。 |
| 其他所有 | `Config.BaseURL`（`/api/paas/v4`，对于 GLM Coding Plan 账户则为 `/api/coding/paas/v4`） | 通用情况。 |

### 区域网关选择（Config.Region）

Z.AI 通过两个区域网关提供同一套 GLM 模型家族：国际主机 `api.z.ai` 和中国大陆镜像
`open.bigmodel.cn`。大多数服务通过 `Config.BaseURL`（聊天、文件、工具等）或固定
常量（Embeddings/Moderations 始终用中国主机）选择自己的主机。有四个服务——
**monitor**（配额/用量）、**biz**（账户信息）、**agents** 和**账户类型检测**——
过去被硬编码到 `api.z.ai`，导致一把 `glm_coding_plan_china` 密钥无法访问本区域的
monitor/usage 端点（并且会被 `accounts add`/`account detect` 误判）。

`Config.Region`（`RegionGlobal`，默认值，或 `RegionChina`）只为这四个区域相关的
服务选择主机。它**不会**覆盖 `Config.BaseURL`（聊天接口）或 Embeddings/Moderations
的主机。从 CLI 上，使用 `--region {global,china}` 或 `ZAI_REGION`（别名：`cn`、
`bigmodel`、`west`）。未知的取值会回退到 global 而不是报错，因此一次拼写错误
永远不会阻塞一条与之无关的命令。

中国镜像侧的 monitor/biz/agents/检测主机，是通过把 `api.z.ai` 的路径布局镜像到
`open.bigmodel.cn` 上建模出来的，并标记为 `NOT VERIFIED LIVE`——中国平台已经
实测验证会在 `/models` 和 `/chat/completions` 上提供相同的 OpenAPI 表面
（见 `BigModelBaseURL`），但中国侧的 monitor/biz/agents 主机还没有被任何
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
录到之后再回放；它是"还差一次真实录制"的待办清单。

如果你要扩展某个服务，在新增 cassette 之前请先阅读
[Contributing § the live-verification convention](../../CONTRIBUTING.md)。

## 扩展结构化的查找表

本仓库里有几种类型把一小撮封闭的、API 定义的取值通过一张配置表加上一个查找函数
映射成人类可读的元数据，而不是一串 `if`/`switch` 条件——`pkg/client/quota.go`
里的 `quotaWindowConfigs`/`findWindowConfig` 是最清晰的例子：

```go
var quotaWindowConfigs = []QuotaWindowConfig{
    {Type: QuotaTypeTokensLimit, UnitCode: UnitCodeHourly, Number: 5, Description: "5-hour rolling token window"},
    {Type: QuotaTypeTokensLimit, UnitCode: UnitCodeWeekly, Number: 1, Description: "weekly token window"},
    {Type: QuotaTypeTimeLimit, UnitCode: UnitCodeMonthly, Number: 1, Description: "monthly MCP tools quota"},
}
```

当 Z.AI 新增一种窗口类型时，改动是增量的（追加一行）且局部的，对于表中尚未收录
的取值会回退到一段通用的兜底描述，而不是硬失败。同样的模式也出现在模型分类
（`pkg/client/models.go` 的 `visionModelMarkers`——单一事实来源，使
`isTextModel`/`isVisionModel` 永远不会自相矛盾）以及账户类型到端点的解析
（`internal/coding/plans.go`）中。当你为某个既有概念新增一个被识别的取值时，
优先采用这种形状，而不是再加一条条件分支。

## 凭证文件安全性

`internal/accounts`（多账户存储）和 `internal/coding`（GLM Coding Plan 凭证存储，
以及它写入的每一个第三方工具配置——Claude Code、OpenCode、Crush、Factory Droid、
Cursor）都通过"临时文件原子写入后再 rename"的方式写文件，绝不会直接就地写入。
写入中途崩溃或被 kill 会保持原文件完好无损，而不是被截断——这一点对本项目尤为
重要，因为其中几个文件是*其他程序*的真实配置文件，只是被合并写入，而不是本项目
完全自有的文件。包含密钥的文件以 `0600` 创建；目录以 `0700` 创建。

## TUI

`internal/tui` 是一个 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 程序，
每个子包对应一个标签页（`chat`、`models`、`usage`、`accounts`、`coding`、
`media`、`tools`）。每个标签页都是一个独立的 `tea.Model`，拥有自己的
`Update`/`View`；`internal/tui/root.go` 在它们之间分发。长时间运行的工作
（一次 API 调用、一次文件系统操作）会被包进一个 `tea.Cmd` 闭包里，由 Bubble Tea
在自己的 goroutine 上运行——从 `tea.Cmd` 中调用的代码不能假设包级可变状态没有
竞争（我们曾经在这方面踩过一次坑，就是 `http.DefaultClient.Timeout`；具体细节
见 `internal/coding/validator.go` 文档注释中的修复说明）。
