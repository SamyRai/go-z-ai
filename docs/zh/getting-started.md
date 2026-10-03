# 快速开始

本指南让你在几分钟内从零开始跑通第一条 `go-z-ai` 命令。

## 1. 安装

**前置条件：** Go 1.26.4+，一个 Z.AI API Key（[在此创建](https://z.ai/manage-apikey/apikey-list)）。

```bash
go install github.com/SamyRai/go-z-ai@latest
```

安装后的二进制名为 `go-z-ai`。可选简写别名：

```bash
ln -s "$(go env GOPATH)/bin/go-z-ai" "$(go env GOPATH)/bin/zai"
```

或者直接从源码构建：

```bash
git clone https://github.com/SamyRai/go-z-ai.git
cd go-z-ai
go build -o go-z-ai .
```

无论你走哪条路径，都请确认 `go-z-ai` 能被解析到、并且位于你的 `PATH` 上：

```bash
go-z-ai --version
```

本指南余下部分假设这个二进制文件叫做 `go-z-ai`。

## 2. 鉴权

按你习惯的方式挑一种即可。它们的解析优先级如下（高者优先）：

| 方式 | 何时使用 |
|---|---|
| `--api-key` 标志 | 一次性调用、脚本、CI |
| `--account <name>` 标志 | 你注册了多个账户（见 [Accounts & Quota](accounts-and-quota.md)） |
| `ZAI_API_KEY` 环境变量（或当前目录下的 `.env` 文件） | 日常本地 shell 使用——最常见的情况 |
| 账户仓库的活动账户 | 你已经运行过 `accounts use <name>`，并希望它默认生效 |

只有一个 key 时，最快的路径：

```bash
export ZAI_API_KEY=your_api_key_here
go-z-ai validate
```

`validate` 会发起一次免费请求（列出模型），在继续之前确认 key 可用。

会读取以下环境变量（每个都有对应的 flag）：

| 变量 | Flag | 用途 |
|---|---|---|
| `ZAI_API_KEY` | `--api-key` | 你的 Z.AI API Key |
| `ZAI_REGION` | `--region` | 区域网关：`global`（api.z.ai，默认）或 `china`（open.bigmodel.cn） |
| `ZAI_API_BASE_URL` | `--base-url` | 仅覆盖 chat/PaaS API 根地址（默认：所选区域的根地址） |
| `ZAI_CHINA_API_KEY` | `--china-api-key` | 用于 Embeddings/Moderations 的独立 open.bigmodel.cn key（回退到 `ZAI_API_KEY`） |

另外还会读取 `ZAI_MONITOR_TIMEZONE`（`--monitor-timezone`）；它只对配额 / 用量输出
有影响。

如果你的 key 是在 Z.AI 中国平台（`open.bigmodel.cn`）签发的，请设置
`--region china`（或 `ZAI_REGION=china`）。区域会决定每个端点所用的主机——chat、
配额 / 用量、账户、agents 以及账户类型检测——所以如果不设置，这些调用会打到
`api.z.ai`，而中国签发的 key 可能在鉴权时失败。`accounts add` 在你注册 key 时会
替你检测区域（如果使用了 `--type`，则会跳过检测，此时请自己加上
`--region china`）。Embeddings 和 Moderations 始终使用 `open.bigmodel.cn`。完整情况见
[Accounts & Quota § Regional gateways](accounts-and-quota.md#区域网关apizai--openbigmodelcn)。

## 3. 你的第一批命令

```bash
# See what models you have access to (context, prices, capabilities from the catalog)
go-z-ai models list

# Send a chat completion (uses the catalog's default chat model)
go-z-ai chat create "Explain goroutines in one paragraph"

# Stream the response token-by-token
go-z-ai chat create "Write a haiku about Go" --stream

# Detect whether the key is Coding Plan or pay-as-you-go, and on which gateway (free)
go-z-ai account detect

# Check that the key can spend (free for Coding Plan keys; one minimal billed
# request for pay-as-you-go keys), then your quota (Coding Plan accounts only)
go-z-ai account status
go-z-ai usage quota
```

大多数命令都可以加上 `--format json` 来获得机器可读的输出；进度信息输出到
stderr，因此 stdout 保持干净。

接下来：

- **完整命令参考：** [CLI Reference](cli-reference.md)
- **多账户 / 配额监控：** [Accounts & Quota](accounts-and-quota.md)
- **把 Claude Code / OpenCode / Crush / Factory Droid 接入你的 GLM Coding Plan：** [Coding Tools](coding-tools.md)
- **把本项目当作 Go 库使用，而不是 CLI：** [Library Guide](library-guide.md)
- **全屏终端 UI**（把 chat、models、usage、accounts、coding、media、tools 这些标签页放到一个界面里）：`go-z-ai tui`

## 故障排查

**"API key is required"**——上面四种方式都没有解析到 key。
再检查一下 `echo $ZAI_API_KEY`，或者显式传入 `--api-key` 来确认。

**"invalid API key" / HTTP 401**——找到了 key，但 Z.AI 拒绝了它。
去 [z.ai/manage-apikey](https://z.ai/manage-apikey/apikey-list) 重新生成。

**在 `embeddings`/`moderations`/`rerank`/`voice` 上出现 "Unknown Model"（错误 1211）**——
这几乎总是账户的权益门槛，而不是 bug：你的账户套餐目录里不包含该模型。运行
`go-z-ai models list` 来看看你的 key 实际能访问哪些模型。完整解释见
[Accounts & Quota](accounts-and-quota.md)。

**其它问题**——[提一个 issue](https://github.com/SamyRai/go-z-ai/issues)，
附上具体命令和错误输出（记得给 key 打码）。
