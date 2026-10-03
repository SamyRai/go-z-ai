# 账户与配额

## 多账户

无需每次切换密钥都手动编辑 `.env`，注册一次具名账户后即可在它们之间切换：

```bash
go-z-ai accounts add personal --api-key sk-...          # type and region auto-detected
go-z-ai accounts add work --api-key sk-... --type coding_plan --region china

go-z-ai accounts list
go-z-ai accounts use personal        # sets the default for future commands
go-z-ai accounts show                # shows the active account
go-z-ai accounts remove work --yes
```

账户信息存储在 `$XDG_CONFIG_HOME/zai-client/accounts.json`（或
`~/.config/zai-client/accounts.json`），以 `0600` 权限原子写入。（目录有意
命名为 `zai-client` 而非 `go-z-ai`，这样从重命名之前的旧版本升级时，已有安装
仍能保留它们的 `accounts.json`。）

**类型与区域自动检测：** `accounts add` 会探测 coding-plan 专属的
monitor/quota 端点（一次免费调用，不消耗 token）：先在你通过 `--region`
指定的区域上探测（默认 global），再到另一个网关上探测，因为中国签发的密钥
只会在 `open.bigmodel.cn` 上应答。得到结构良好的 quota 响应，就意味着该区域
上是 `coding_plan`。如果两个网关都没能识别出订阅，但至少有一个网关作了应答，
则该密钥是 `pay_as_you_go` —— 这是一种排除法的推断，并非正向确认，因为不存在
专门针对 pay-as-you-go 密钥的端点。如果两个网关都无法连通，`add` 会直接失败，
而不是瞎猜。传入 `--type` 可跳过探测；此时区域取自 `--region`。
`go-z-ai account detect` 会对当前密钥运行同样的探测，且不存储任何内容。

**解析顺序** —— 关于 `--api-key`、`--account`、环境变量以及已存储的激活账户
之间的完整优先级列表，请见 [快速开始](getting-started.md#2-鉴权)。

## 检查密钥

```bash
go-z-ai account detect          # type (coding plan / pay-as-you-go), region, plan tier — free
go-z-ai account status          # does the key work and can it spend right now?
go-z-ai account status --watch 5m
go-z-ai account balance         # pay-as-you-go wallet
go-z-ai account subscriptions   # GLM Coding Plan subscriptions
```

对 coding-plan 密钥而言，`account status` 是一次免费的配额检查。Z.AI 没有
针对 pay-as-you-go 密钥的余额查询调用，所以对这类密钥，它会发送一次计费的、
单 token 的补全请求 —— 而 `--watch` 会在每个间隔重复这次请求，所以请把间隔
设得长一些。`account balance` 和 `account subscriptions` 读取的是 biz API，
Z.AI 并未为其提供文档；它们**尚未实测验证**（见 [路线图](roadmap.md)）。
余额以该区域的计费币种表示（api.z.ai 为 USD，open.bigmodel.cn 为 CNY）。

## 配额与用量监控

GLM Coding Plan 的配额按窗口划分，账户具有哪些窗口取决于其套餐世代：

| 套餐 | 窗口 | 计量 |
|---|---|---|
| 基于 credit 的套餐（2026-07-30 起发售） | 5 小时、每周（`CREDIT_LIMIT`） | Credit（额度）。模型用量和 MCP 工具调用从同一份 credit 中扣除；Web Search、Web Reader 和 Zread 每次调用消耗 1.2 credit（[文档](https://docs.z.ai/devpack/overview)） |
| 旧版 token 套餐 | 5 小时、每周（`TOKENS_LIMIT`） | Token，仅以百分比形式报告 |
| 旧版 token 套餐 | 每月（`TIME_LIMIT`） | MCP 工具调用，附带按工具的明细 |

5 小时窗口在消耗发生五小时后刷新；API 会报告每个窗口的重置时间。请把窗口类型
视为一个开放集合：客户端遇到未知类型时会以通用方式渲染，而不是报错失败。

```bash
go-z-ai accounts quota                    # across all stored accounts
go-z-ai accounts usage --days 14          # token/tool usage heat map
go-z-ai accounts usage --today            # shorthand for --days 1
go-z-ai usage quota                       # the current key only
go-z-ai account status [--watch 5m]       # can the key spend right now?
```

`usage quota` 输出示例（`accounts quota` 会为每个账户打印同样的内容）：

```text
📊 GLM Coding Plan (pro)

• 5-hour credit window — 62% used
  620 / 1.0K used — 380 left
  resets Oct 12 11:14 CEST (in 2h 14m)
  62% used at 55% of window elapsed — on pace to run out ~32m before reset
  ⚡ peak hours until 12:00 CEST: credits bill at the full rate (off-peak costs 50%)

• weekly credit window — 41% used
  4.1K / 10.0K used — 5.9K left
  resets Oct 18 07:00 CEST (in 5d)
  41% used at 15% of window elapsed — on pace to run out ~4d before reset
  ⚡ peak hours until 12:00 CEST: credits bill at the full rate (off-peak costs 50%)
```

**Pace**（节奏）行回答了"我是不是消耗得太快？"这一问题 —— 它根据窗口*自身*
报告的用量以及窗口已经流逝的比例进行外推，让你在真正撞墙*之前*就发现自己是否
会提前耗尽。这是基于 API 返回数值的直线推算。

### 高峰时段

对所有用户而言，高峰时段都是周一至周五 UTC+8 的 14:00–18:00，与其所在时区
无关（[文档](https://docs.z.ai/devpack/overview)）。quota API 会应用高峰定价，
却不会把它暴露出来，所以高峰期间 CLI 和 TUI 会额外加上一条提示。高峰期的
成本取决于套餐：

- **基于 credit 的套餐**：高峰期用量按标准 credit 费率计费，非高峰期用量按 50% 计费。
- **旧版 token 套餐**：高峰期 GLM-5.3 按 3× 计（非高峰 1×），GLM-5.3-Flash
  按 1.2× 计（非高峰 0.4×）。

已公布的促销活动会让每个小时都按非高峰费率计费；当前这一档的时间是
2026-09-25 至 2026-10-07（UTC+8）。这些规则位于
`internal/usageview/peak.go`，截至 2026-10-02。

### 时区

所有时间都以**你的本地时区**渲染。重置时间是绝对时间戳，因此无需转换。不过
用量端点交换的是不带时区的时间字符串，使用的是服务器自己的墙上时钟，即
**UTC+8**，所以**用量热力图**（`accounts usage`）会把它的窗口和桶区间转换为
你的本地时间；当服务器时区与你的本地时区不同时，会附上一行指明服务器时区的
说明。请求的范围在发送前也会按服务器时区来界定，因此 `--today` 取到的是
*你的*今天，而不是被时区差异错位的一段时间。

如果某次实际抓包显示你的账户所在的服务器时区不同，可以用
`--monitor-timezone` 或 `ZAI_MONITOR_TIMEZONE` 覆盖这一假设（接受 IANA 名称，
如 `Asia/Shanghai`、`UTC`，或形如 `+8` 的偏移量）。

并非每个账户都显示全部窗口 —— 套餐层级和账户类型都会影响哪些窗口适用
（`accounts show <name>` 会报告账户类型；`pay_as_you_go` 账户会被
`accounts quota`/`accounts usage` 完全跳过，因为 coding-plan monitor 端点并不
适用于它们）。

### 为什么"用量 API 不存在"是错的

如果你在网上（或在本仓库的 git 历史里）看到旧的说法称 Z.AI 没有用量/配额
API —— 这只对在隔离环境中测试的通用 `/api/paas/v4` 接口成立。coding-plan 的
monitor 端点（`/monitor/usage/quota/limit`、`/monitor/usage/model-usage`、
`/monitor/usage/tool-usage`）是真实存在的 —— 它们正是 Z.AI 自家用量插件所调用
的接口 —— 也是 `pkg/client` 的 `QuotaService` 所构建的基础。如果
`accounts quota` 对某个账户没有任何返回，请先检查它的类型（`pay_as_you_go`
账户确实没有这些数据），再假设 API 出了问题。

## 区域网关（api.z.ai / open.bigmodel.cn）

Z.AI 通过两个区域网关提供同一套 GLM 模型家族：国际主机 `api.z.ai`（默认）和
中国大陆镜像 `open.bigmodel.cn`。有两个相互独立的因素决定某次调用落在哪个
主机上：

**1. Embeddings 和 Moderations 始终路由到 `open.bigmodel.cn`**——它们是代码里
唯一被固定指向中国主机的服务（`pkg/client/embeddings.go` 和
`pkg/client/moderations.go` 都指向 `BigModelBaseURL`）。`--china-api-key` /
`ZAI_CHINA_API_KEY` 是此处的凭据开关；它是可选的，因为普通的 `ZAI_API_KEY`
在两个平台上鉴权方式完全相同（相同的 `/models` 目录、相同的计费层级错误 ——
已实测验证），所以回退才是常见情况。仅当你持有独立的、仅限 bigmodel.cn
使用的凭据时，才需要单独设置中国密钥。Rerank 和 Voice 使用默认的
`--base-url`（默认即 `api.z.ai`）——它们仅在中国平台有文档，但客户端并不会
强制把它们路由过去。

**2. 其余一切都跟随 `--region china`（或 `ZAI_REGION=china`）**：chat、
Anthropic 和 Responses API、配额/用量、账户（余额、订阅）、agents，以及账户类型
检测，都会移到 `open.bigmodel.cn`；否则它们使用 `api.z.ai`。这正是中国密钥
需要的开关 —— 没有它，这些调用会打到 `api.z.ai`，而中国签发的密钥可能会鉴权
失败或被错误分类。已存储的账户会记住它的区域，所以 `accounts use` 也会一并
切换区域。`--base-url` 仍然只覆盖 chat/PaaS 根地址（对 Coding Plan 账户而言，
它是该套餐的 coding 根地址），而 Embeddings/Moderations 的主机永远不变。
别名：`cn`、`bigmodel`、`west`；未知值会回退到 global。

你能否从中国平台文档中提到的服务（Embeddings、Moderations、Rerank、Voice）拿到真实结果，
取决于你账户的**套餐权益**，而非你用哪把密钥。GLM Coding Plan 账户的模型目录
仅含 chat —— 用该账户调用这些服务会在任一平台上返回 `400 Unknown Model`
（错误码 1211）。这是预期行为，不是 bug：通过 `go-z-ai models list` 查看
你账户目录中实际包含的内容。

中国镜像上 monitor/biz/agents/detection 的主机镜像了 `api.z.ai` 的路径布局，
但此处**尚未实测验证** —— `open.bigmodel.cn` 已实测验证可为 `/models` 和
`/chat/completions` 提供相同的 OpenAPI 接口，但中国侧的 monitor/biz/agents
路径尚未被任何 cassette 录制。见 [路线图](roadmap.md)。

## 错误码

`APIError`（见 [错误处理](error-handling.md)）对客户端已知的每一个 Z.AI
错误码都做了归类。围绕配额，你最常碰到的几个：

| 码 | 含义 | 可重试 |
|---|---|---|
| 1113 | 余额不足 / 无资源包 | 否 —— 充值或切换账户 |
| 1308 | 当前窗口已达用量上限 | 否 —— 等待重置 |
| 1316 / 1317 | 5 小时 / 7 天窗口已用尽，且没有余额用于额外用量 | 否 —— 等待重置或充值 |
| 1211 | 未知模型 | 否 —— 通常是权益门槛，见上文 |
| 1302 | 已达速率限制 | 是 —— 客户端已通过带退避的方式重试 |

完整的表格以及如何在你自己的代码中根据 `APIError.Category` 进行分支处理，请见
[错误处理](error-handling.md)。
