# 编码工具（GLM Coding Plan）

`go-z-ai coding` 用于把第三方编码助手配置为使用你的 GLM
Coding Plan，而不是它们默认的供应商。它是 Z.AI 官方
`@z_ai/coding-helper`（"chelper"）CLI 的 Go 移植版，共享同一个凭证文件，因此
两者可以互换使用。

## 支持的工具

| Tool | 写入的配置文件 |
|---|---|
| Claude Code | `~/.claude/settings.json`（外加 `~/.claude.json` 的引导标志） |
| OpenCode | `~/.config/opencode/opencode.json` |
| Crush | `~/.config/crush/crush.json` |
| Factory Droid | `~/.factory/settings.json` |
| Cursor | 随操作系统而异——macOS 上为 `~/Library/Application Support/Cursor/User/settings.json`，其他平台为 `~/.cursor/settings.json`（或 `~/.config/Cursor/User/settings.json`） |

运行 `go-z-ai coding tools` 可以查看本机上的安装状态以及解析出的精确路径。

## 套餐

| Plan identifier | Endpoint |
|---|---|
| `glm_coding_plan_global` | `https://api.z.ai` |
| `glm_coding_plan_china` | `https://open.bigmodel.cn` |

选择与你 GLM Coding Plan 套餐所在区域匹配的那一个即可。

## 快速开始

```bash
# 1. 存储并校验你的 GLM Coding Plan key（一次性）
go-z-ai coding auth glm_coding_plan_global YOUR_KEY

# 2. 把它加载到某个工具中
go-z-ai coding load claude-code
# 工具 ID：claude-code, opencode, crush, factory-droid, cursor
# 也支持别名：claude, droid, factory

# 3. 检查是否一切就绪
go-z-ai coding status
go-z-ai coding doctor
```

凭证存放在 `~/.chelper/config.yaml`（与官方 Node 助手字节兼容）——`coding auth`
会写入这里一次，之后 `coding load` 在没有传入 `--key`/`--plan` 覆盖时，会为每个
工具从这里读取。

如果想让某个工具停止使用 Z.AI，但又不丢失已存储的凭证：

```bash
go-z-ai coding unload claude-code
```

这只会移除它添加的 Z.AI 专属字段，不会动你现有配置文件的其余部分。

## Claude Code：模型映射

官方助手只会设置 `ANTHROPIC_AUTH_TOKEN`、`ANTHROPIC_BASE_URL`、
`API_TIMEOUT_MS` 和 `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`。本客户端默认
更进一步，还会通过 `ANTHROPIC_DEFAULT_*_MODEL` 把 Claude Code 的各模型层级映射
到具体的 GLM 模型，与
[Z.AI 文档中的推荐配置](https://docs.z.ai/scenario-example/develop-tools/claude)
保持一致：

| Claude tier | 默认 GLM 模型 |
|---|---|
| haiku | `glm-4.5-air` |
| sonnet | `glm-4.7` |
| opus | `glm-4.7` |

可以覆盖任意层级，或完全停用模型映射：

```bash
go-z-ai coding auth glm_coding_plan_global YOUR_KEY \
  --sonnet glm-5.2 --opus glm-5.2

go-z-ai coding auth glm_coding_plan_global YOUR_KEY --no-model-mapping
```

以下 flag 也可配置，均为可选（值为 0 或省略则不设置对应环境变量）：

| Flag | 设置的环境变量 | 使用场景 |
|---|---|---|
| `--auto-compact-window int` | `CLAUDE_CODE_AUTO_COMPACT_WINDOW` | 默认为 1,000,000（GLM-5.2 的上下文长度）；如果你固定使用 128K 上下文的模型，可以调低（例如 128000） |
| `--max-thinking-tokens int` | `MAX_THINKING_TOKENS` | 扩展思考的预算 |
| `--max-output-tokens int` | `CLAUDE_CODE_MAX_OUTPUT_TOKENS` | 输出上限 |

这些 flag 在 `coding` 命令上是持久化的，因此对 `auth`、`load` 和 `reload`
的作用方式相同。

## Key 管理

```bash
go-z-ai coding auth revoke              # 清除已存储的 key，保留所选套餐
go-z-ai coding auth reload <tool>       # 重新把已存储的凭证推送到某个工具
go-z-ai coding load <tool> --key OTHER_KEY --plan glm_coding_plan_china  # 一次性覆盖
```

默认情况下，`coding auth` 在存储新 key 前会先对 API 进行校验（发起一次真实的
`/models` 调用）。如果你想离线存储 key（例如为尚未联网测试过的机器编写脚本），
可以用 `--no-validate` 跳过这一步。

## Vision MCP Server

官方 `@z_ai/coding-helper` 向导中有一个"管理 MCP 服务"的步骤，本客户端此前并未
复刻：Z.AI 自带一个
[Vision MCP Server](https://docs.z.ai/devpack/mcp/vision-mcp-server)
（`@z_ai/mcp-server`）——截图 OCR、错误截图诊断、图表/图形理解，以及基于
GLM-4.6V 的通用图像/视频分析，通过 `npx` 按需启动。`coding mcp` 会把它注册到你
正在使用的工具中：

```bash
go-z-ai coding mcp add claude-code     # 使用已存储的 API Key
go-z-ai coding mcp add crush --key OTHER_KEY
go-z-ai coding mcp status              # 查看哪些工具已配置
go-z-ai coding mcp remove claude-code
```

**需要 Node.js。** 该 server 本身通过 `npx -y @z_ai/mcp-server` 运行——Z.AI 自家
文档目前推荐 Node.js 22+，不过 npm 包只声明了 18+ 的要求。如果 `PATH` 上找不到
`npx`，`coding mcp add`/`doctor` 会发出警告（不会中断），因为只要 Node.js 一可用，
该配置就是有效的。

**MCP 配置文件往往与你的 GLM 凭证不是同一个文件。** 五个工具中有两个把 MCP
服务存放在与 provider/API 设置不同的文件里：

| Tool | 凭证配置 | MCP 配置 |
|---|---|---|
| Claude Code | `~/.claude/settings.json` | `~/.claude.json` |
| OpenCode | `opencode.json` | 同一文件，`mcp` 键 |
| Crush | `crush.json` | 同一文件，`mcp` 键 |
| Factory Droid | `~/.factory/settings.json` | `~/.factory/mcp.json` |
| Cursor | 随操作系统而异的 `settings.json` | 同目录下的同级 `mcp.json` |

Z.AI 自家文档并未把 Cursor 明确列为受支持的 Vision MCP 客户端——这里使用的是
Cursor 文档中对任意 server 通用的 MCP 结构，理论上应可用，但尚未经 Z.AI
针对该 server 确认。

## Doctor

```bash
go-z-ai coding doctor
```

检查内容：是否已存储凭证、凭证格式是否正确、`PATH` 上安装了哪些受支持的工具，
以及这些工具中哪些已经配置好 Z.AI（如已注册则包括 Vision MCP Server）。出问题时
这是很好的第一步。
