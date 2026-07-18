# 账户与配额

## 多账户

无需每次切换密钥都手动编辑 `.env`，注册一次具名账户后即可在它们之间切换：

```bash
zai-client accounts add personal --api-key sk-...          # 类型自动检测
zai-client accounts add work --api-key sk-... --type coding_plan

zai-client accounts list
zai-client accounts use personal        # 为后续命令设置默认账户
zai-client accounts show                # 显示当前激活的账户
zai-client accounts remove work --yes
```

账户信息存储在 `$XDG_CONFIG_HOME/zai-client/accounts.json`（或
`~/.config/zai-client/accounts.json`），以 `0600` 权限原子写入。

**类型自动检测：** `accounts add` 会通过一次免费（不消耗 token）调用去探测
coding-plan 专属的 monitor/quota 端点。成功且结构良好的响应意味着
`coding_plan`；其他情况则回退到 `pay_as_you_go` —— 这是一种排除法的推断，
并非正向确认，因为目前已知不存在专门针对 pay-as-you-go 密钥的端点。传入
`--type` 可显式跳过此探测。该探测遵循 `--region`：`glm_coding_plan_china`
密钥会探测 `open.bigmodel.cn` 的 coding 端点而非 `api.z.ai` 的端点，从而能在
正确的平台上完成分类（见下文 [区域网关](#regional-gateways-apiza--openbigmodelcn)）。

**解析顺序** —— 关于 `--api-key`、`--account`、环境变量以及已存储的激活账户
之间的完整优先级列表，请见 [快速开始](getting-started.md#2-authenticate)。

## 配额与用量监控

GLM Coding Plan 账户有三个相互独立的配额窗口：

| 窗口 | 追踪对象 | 类型 |
|---|---|---|
| 5 小时滚动 | API 请求 token | 滚动 —— 过去 5 小时的用量，并非固定周期 |
| 每周滚动 | API 请求 token | 滚动 —— 过去 7 天的用量 |
| 每月 | MCP 工具调用（web search、web-reader、zread） | 固定的自然月重置 |

```bash
zai-client accounts quota                    # 跨所有已存储账户
zai-client accounts usage --days 14          # token/工具用量热力图
zai-client accounts usage --today            # --days 1 的简写
zai-client usage quota                       # 单个激活账户
zai-client usage check --watch               # 当用量超过 80% 时告警
```

`accounts quota` 输出示例：

```
📊 GLM Coding Plan Usage (PRO tier)

• 5-hour rolling token window
  Usage: 62%
  Resets: 2026-07-11 09:30:00 CEST (in 2h 14m)
  Pace: 62% used at 55% of window elapsed — on pace to run out ~24m before reset

• weekly token window
  Usage: 41%
  Resets: 2026-07-17 09:17:18 CEST (in 5d 22h)
  Pace: 41% used at 15% of window elapsed — on pace to run out ~4d before reset

• monthly MCP tools quota
  Usage: 574/1000 (57%) — 426 remaining
  Resets: 2026-07-26 09:17:18 CEST (in 18d 6h)
  By tool:
    - search-prime: 470
    - web-reader: 97
    - zread: 7
```

**Pace**（节奏）行（仅 token 窗口）回答了"我是不是消耗得太快？"这一问题 ——
它根据窗口*自身*报告的用量以及窗口已经流逝的比例进行外推，让你在真正撞墙
*之前*就发现自己是否会提前耗尽。这是基于 API 返回数值的直线推算（不假设
高峰/非高峰定价）。

并非每个账户都显示全部窗口 —— 套餐层级和账户类型都会影响哪些窗口适用
（`accounts show <name>` 会报告账户类型；`pay_as_you_go` 账户会被
`accounts quota`/`accounts usage` 完全跳过，因为 coding-plan monitor 端点并不
适用于它们）。

### 为什么"用量 API 不存在"是错的

如果你在网上（或在本仓库的 git 历史里）看到旧的说法称 Z.AI 没有用量/配额
API —— 这只对在隔离环境中测试的通用 `/api/paas/v4` 接口成立。coding-plan 的
monitor 端点（`/monitor/usage/quota/limit`、`/monitor/usage/model-usage`、
`/monitor/usage/tool-usage`）是真实存在、有文档支撑的，也是 `pkg/client` 的
`QuotaService`/`UsageService` 所构建的基础。如果 `accounts quota` 对某个账户
没有任何返回，请先检查它的类型（`pay_as_you_go` 账户确实没有这些数据），再假设
API 出了问题。

## 区域网关（api.z.ai / open.bigmodel.cn）

Z.AI 通过两个区域网关提供同一套 GLM 模型家族：国际主机 `api.z.ai`（默认）和
中国大陆镜像 `open.bigmodel.cn`。有两个相互独立的因素决定某次调用落在哪个
主机上：

**1. Embeddings 和 Moderations 始终路由到 `open.bigmodel.cn`**——它们是代码里
唯一被固定指向中国主机的服务（`pkg/client/embeddings.go` 和
`pkg/client/moderations.go` 都调用 `doRequestBaseKey(BigModelBaseURL, …)`）。
`--china-api-key` / `ZAI_CHINA_API_KEY` 是此处的凭据开关；它是可选的，因为普通的
`ZAI_API_KEY` 在两个平台上鉴权方式完全相同（相同的 `/models` 目录、相同的
计费层级错误 —— 已实测验证），所以回退才是常见情况。仅当你持有独立的、仅限
bigmodel.cn 使用的凭据时，才需要单独设置中国密钥。Rerank 和 Voice 使用默认的
`--base-url`（默认即 `api.z.ai`）——它们仅在中国平台有文档，但客户端并不会
强制把它们路由过去。

**2. 当设置了 `--region china`（或 `ZAI_REGION=china`）时，monitor / biz /
agents / detection 会路由到 `open.bigmodel.cn`**；否则它们会路由到
`api.z.ai`。这正是 `glm_coding_plan_china` 用户需要的开关，能让配额/用量、
账户信息、agents 以及账户类型检测都落在正确的主机上 —— 否则这些调用会打到
`api.z.ai`，而中国签发的密钥可能会鉴权失败或被错误分类。`--region`
**不会**改变 chat 的 base URL（为此请使用 `--base-url`）或
Embeddings/Moderations 的主机。别名：`cn`、`bigmodel`、`west`；未知值会回退到
global。

你能否从中国平台文档中提到的服务（Embeddings、Moderations、Rerank、Voice）拿到真实结果，
取决于你账户的**套餐权益**，而非你用哪把密钥。GLM Coding Plan 账户的模型目录
仅含 chat —— 用该账户调用这些服务会在任一平台上返回 `400 Unknown Model`
（错误码 1211）。这是预期行为，不是 bug：通过 `zai-client models list` 查看
你账户目录中实际包含的内容。

中国镜像上 monitor/biz/agents/detection 的主机镜像了 `api.z.ai` 的路径布局，
但此处**尚未实测验证** —— `open.bigmodel.cn` 已实测验证可为 `/models` 和
`/chat/completions` 提供相同的 OpenAPI 接口，但中国侧的 monitor/biz/agents
路径尚未被任何 cassette 录制。见 [路线图](roadmap.md)。

## 错误码

`APIError`（见 [错误处理](error-handling.md)）对客户端代码见过的每一个 Z.AI
错误码都做了归类。围绕配额，你最常碰到的几个：

| 码 | 含义 | 可重试 |
|---|---|---|
| 1113 | 余额不足 / 无资源包 | 否 —— 充值或切换账户 |
| 1308 | 当前窗口已达用量上限 | 否 —— 等待重置 |
| 1211 | 未知模型 | 否 —— 通常是权益门槛，见上文 |
| 1302 | 已达速率限制 | 是 —— 客户端已通过带退避的方式重试 |

完整的表格以及如何在你自己的代码中根据 `APIError.Category` 进行分支处理，请见
[错误处理](error-handling.md)。
