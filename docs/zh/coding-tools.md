# 编码工具（GLM Coding Plan）

`go-z-ai coding` 用于把第三方编码助手配置为使用你的 GLM
Coding Plan，而不是它们默认的供应商。它是 Z.AI 官方
`@z_ai/coding-helper`（"chelper"）CLI
（[文档](https://docs.z.ai/devpack/extension/coding-tool-helper)）的 Go 移植版，
共享同一个凭证文件，因此两者可以互换使用。

## 支持的工具

| Tool | ID（别名） | 写入的配置文件 |
|---|---|---|
| Claude Code | `claude-code` (`claude`) | `~/.claude/settings.json`（外加 `~/.claude.json`：引导标志和 MCP 服务器） |
| Codex | `codex` | `~/.codex/config.toml`（外加 `~/.codex/models.json`：模型元数据） |
| OpenCode | `opencode` | `~/.config/opencode/opencode.json` |
| Crush | `crush` | `~/.config/crush/crush.json` |
| Factory Droid | `factory-droid` (`droid`, `factory`) | `~/.factory/settings.json`（MCP 服务器位于 `~/.factory/mcp.json`） |

路径均位于你的主目录下。运行 `go-z-ai coding tools` 可以查看安装状态（该工具的
二进制文件是否在 `PATH` 上：`claude`、`codex`、`opencode`、`crush`、`droid`），
以及本机上解析出的精确路径。

不再支持 Cursor（早期版本会写入 Cursor 的设置文件）。Z.AI 把 Cursor 记录为仅限
GUI 的配置方式：OpenAI 协议、你的 key，以及把套餐的 coding 端点作为 base URL
（[文档](https://docs.z.ai/devpack/tool/cursor)）。

### 每个工具的配置会得到什么

`coding load <tool>` 会把套餐合并进该工具自己的配置，并保持无关设置不变。

- **Claude Code**：在 `~/.claude/settings.json` 中写入一个 `env` 块，包含 key、
  套餐的 Anthropic 端点、层级映射，以及
  [Claude Code：模型映射](#claude-code模型映射) 下列出的其他变量。在
  `~/.claude.json` 中，除非已有设置，否则会设置 `hasCompletedOnboarding: true`。
  它还会从 `env` 块中移除过期的 `ANTHROPIC_API_KEY`。手动配置的等价做法见
  [Claude Code 文档](https://docs.z.ai/devpack/tool/claude)。
- **Codex**：在 `~/.codex/config.toml` 中写入 `model_provider = "ZAI"`、
  `model = <client.DefaultModel>`、`model_reasoning_effort`（该模型所接受的最强
  等级，GLM-5.3 为 `max`）、`model_catalog_json = "~/.codex/models.json"`，以及
  一个 `[model_providers.ZAI]` 表，其中以套餐的 Responses 端点作为 `base_url`，
  以 key 作为 `experimental_bearer_token`，并设置 `wire_api = "responses"`。
  在 `~/.codex/models.json` 中，它会写入来自目录的该模型元数据（推理等级、
  上下文窗口、输入模态），替换同一模型的旧条目并保留其他条目。这是 Z.AI 文档
  和官方助手所采用的格式。
  （[文档](https://docs.z.ai/devpack/tool/codex)）
- **OpenCode**：一个以 OpenCode 内置套餐 provider 命名的 provider 条目，即
  `zai-coding-plan`（Global 套餐）或 `zhipuai-coding-plan`（China 套餐），其中
  包含 `options.apiKey`；另一个套餐的条目会被移除。它会设置 `$schema`，并且只在
  `model` 和 `small_model` 未设置或已指向某个套餐 provider 时，才把它们设为
  `<provider>/<client.DefaultModel>` 和 `<provider>/<client.DefaultFastModel>`，
  这样你自己的选择得以保留。（[文档](https://docs.z.ai/devpack/tool/opencode)）
- **Crush**：`providers.zai`，其中 `id: "zai"`、`name: "ZAI Provider"`，以套餐
  的 coding 端点作为 `base_url`，外加 `api_key`。不会写入模型；请在 Crush 的 UI
  中选择。（[文档](https://docs.z.ai/devpack/tool/crush)）
- **Factory Droid**：针对 `client.DefaultModel` 的两个 `customModels` 条目，一个
  走 Anthropic 协议（`provider: "anthropic"`，套餐的 Anthropic 端点），一个走
  OpenAI 协议（`provider: "generic-chat-completion-api"`，coding 端点）。
  `maxOutputTokens` 来自模型目录。显示名称形如
  `GLM-5.3 [GLM Coding Plan Global] - Anthropic`；任何显示名称包含
  `GLM Coding Plan` 的现有条目都会被替换（这也涵盖官方助手写入的条目），其他
  条目会保留。
  （[文档](https://docs.z.ai/devpack/tool/droid)）

配置文件是如何写入的：

- API key 会以明文写入各工具的配置，这是那些工具所期望的。文件以 `0600` 权限
  原子写入。
- 在配置文件第一次被覆盖之前，会在它旁边保留一份副本，命名为
  `<file>.zai.bak`。已有的备份绝不会被替换。对于符号链接的配置，会穿透链接写入，
  因此 dotfile 管理器可以继续正常工作。
- 配置会以 JSON 读取并重写（带缩进，键已排序）；`.toml` 文件（Codex）则以 TOML
  处理。TOML 的重写会丢掉注释，官方助手也是如此；`.zai.bak` 备份保留着原文件。
  无法解析的配置会让命令以解析错误失败；它不会被覆盖。
- 对未安装的工具执行 `coding load` 仍会写入其配置。

## 套餐

| Plan identifier | 网关 | OpenAI 兼容端点 | Anthropic 端点 | Responses 端点（Codex） |
|---|---|---|---|---|
| `glm_coding_plan_global` | `https://api.z.ai` | `https://api.z.ai/api/coding/paas/v4` | `https://api.z.ai/api/anthropic` | `https://api.z.ai/api/v1` |
| `glm_coding_plan_china` | `https://open.bigmodel.cn` | `https://open.bigmodel.cn/api/coding/paas/v4` | `https://open.bigmodel.cn/api/anthropic` | `https://open.bigmodel.cn/api/v1` |

选择与你 GLM Coding Plan 订阅所在区域匹配的那一个即可。套餐决定了 `coding`
写入的所有内容所属的区域（工具端点、托管 MCP 的 URL、Vision 服务器的模式）。
所有 URL 都来自 `pkg/client/region.go` 中的 `client.Region`。全局的 `--region`、
`--api-key` 和 `--account` flag 不适用于 `coding`；请改用套餐以及
`--key`/`--plan`。

## 快速开始

```bash
# 1. Store and validate your GLM Coding Plan key (one-time)
go-z-ai coding auth glm_coding_plan_global YOUR_KEY

# 2. Load it into a tool
go-z-ai coding load claude-code
# tool IDs: claude-code, codex, opencode, crush, factory-droid
# aliases also work: claude, droid, factory

# 3. Check everything's wired up
go-z-ai coding status
go-z-ai coding doctor
```

凭证存放在 `~/.chelper/config.yaml`（与官方 Node 助手字节兼容；键为 `lang`、
`plan`、`api_key`）——`coding auth` 会写入这里一次，之后 `coding load` 在没有传入
`--key`/`--plan` 覆盖时，会为每个工具从这里读取。

如果想让某个工具停止使用 Z.AI，但又不丢失已存储的凭证：

```bash
go-z-ai coding unload claude-code
```

这只会移除 `load` 添加的 Z.AI 专属字段（Claude Code 受管理的 `env` 变量、Codex
的 `ZAI` provider 以及——只要它仍是当前生效的 provider——其顶层模型设置、
OpenCode 的套餐 provider 和套餐的 `model`/`small_model`、Crush 的
`providers.zai`、Droid 的套餐 `customModels` 条目）；它不会动你现有配置文件的
其余部分。Codex 的 `models.json` 会保留，与官方助手的做法一致。它不会移除 MCP
服务器（请使用 `coding mcp remove`），Claude Code 的 `hasCompletedOnboarding`
也会保留。

如果该工具的配置中没有 Z.AI 套餐，`unload` 会输出
`<Tool> is not configured for a Z.AI plan; nothing to remove.` 并以 0 退出，因此
在脚本中反复运行是安全的。只有当工具的配置指向某个套餐端点（Claude Code 的
`ANTHROPIC_BASE_URL`、Codex 的 `model_providers.ZAI.base_url`、Crush 的
`providers.zai.base_url`）或包含某个套餐条目（OpenCode、Droid）时，才算作已配置；
为其他 provider 设置的 key 不会被触碰。

TUI 的 coding 标签页（`go-z-ai tui`）提供同样的操作：`a` 鉴权、`l` 加载、`u`
卸载、`m` 注册该工具所支持的 MCP 服务器、`r` 重新扫描。它会以下面的默认 Claude
Code 调优配置进行加载。

## Claude Code：模型映射

`load` 会写入 Z.AI 为 Claude Code 所记录的变量，其中包括通过
`ANTHROPIC_DEFAULT_*_MODEL` 把 Claude Code 的各模型层级映射到 GLM 模型
（[文档](https://docs.z.ai/devpack/tool/claude)）：

| Env var | 值 |
|---|---|
| `ANTHROPIC_AUTH_TOKEN` | 你的 key（以 `Authorization: Bearer` 发送，这是 Z.AI 的 Anthropic 端点所期望的；不使用 `ANTHROPIC_API_KEY`） |
| `ANTHROPIC_BASE_URL` | 套餐的 Anthropic 端点 |
| `API_TIMEOUT_MS` | `3000000` |
| `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC` | `1` |
| `ANTHROPIC_DEFAULT_HAIKU_MODEL` | `client.DefaultFastModel` + `[1m]`（`glm-5.3-flash[1m]`） |
| `ANTHROPIC_DEFAULT_SONNET_MODEL` | `client.DefaultModel` + `[1m]`（`glm-5.3[1m]`） |
| `ANTHROPIC_DEFAULT_OPUS_MODEL` | `client.DefaultModel` + `[1m]`（`glm-5.3[1m]`） |
| `CLAUDE_CODE_AUTO_COMPACT_WINDOW` | 主模型的上下文大小，1048576（可用下面的 flag 设为你自己的值） |
| `MAX_THINKING_TOKENS`, `CLAUDE_CODE_MAX_OUTPUT_TOKENS` | 仅在你传入对应 flag 时才设置 |

模型 ID 来自 `pkg/client/models_catalog.go`，因此刷新目录就会更新 `load` 写入的
内容。`[1m]` 后缀是 Claude Code 的一个约定，用来让某个模型启用其 100 万 token
的上下文；它只应出现在这些变量中，绝不能出现在 API 请求里。`load` 会为目录中
上下文达到 100 万 token 或更多的每个模型添加它，这涵盖了两个默认模型，官方助手
也是这么做的。通过 `--haiku`/`--sonnet`/`--opus` 传入的值会原样写入，所以对于
100 万上下文的模型，请自己加上 `[1m]`。

Z.AI 的各个页面对这个映射的说法并不完全一致：
`docs.z.ai/devpack/tool/claude` 把它拆开（haiku 用 flash，sonnet 和 opus 用旗舰
模型，如上所示），而 `docs.z.ai/devpack/latest-model` 则对三个层级都显示 flash
模型。官方助手（0.1.1）同样对三个层级都写入 flash 模型。可以覆盖任意层级，或完全
退出这套映射：

```bash
# Use the flash model for every tier
go-z-ai coding load claude-code \
  --sonnet 'glm-5.3-flash[1m]' --opus 'glm-5.3-flash[1m]'

# Leave Claude Code's model selection alone
go-z-ai coding load claude-code --no-model-mapping
```

调优 flag，均为可选：

| Flag | 作用 |
|---|---|
| `--haiku`, `--sonnet`, `--opus` | 覆盖该层级的模型 ID |
| `--no-model-mapping` | 省略三个 `ANTHROPIC_DEFAULT_*_MODEL` 变量 |
| `--auto-compact-window int` | 设置 `CLAUDE_CODE_AUTO_COMPACT_WINDOW`；默认为主模型的上下文；如果你固定使用上下文更小的模型，可以调低（例如 128000）；`0` 表示省略该变量 |
| `--max-thinking-tokens int` | 设置 `MAX_THINKING_TOKENS`（扩展思考的预算）；`0`/省略 = 不设置 |
| `--max-output-tokens int` | 设置 `CLAUDE_CODE_MAX_OUTPUT_TOKENS`；`0`/省略 = 不设置 |

这些 flag 只存在于 `coding load` 和 `coding auth` 上，并且在写入工具配置时才
生效：即 `coding load <tool>` 和 `coding auth reload <tool>`。
`coding auth <plan> <key>` 只会存储凭证，并忽略它们。它们对 Claude Code 以外的
工具没有任何作用。每次 `load` 都会在写入前移除它所管理的变量，因此上一次运行时
的调优不会被沿用：请每次都重新传入这些 flag。

## Key 管理

```bash
go-z-ai coding auth glm_coding_plan_global YOUR_KEY   # validate and store
go-z-ai coding auth revoke              # clear the stored key, keep the plan choice
go-z-ai coding auth reload <tool>       # re-push stored creds into a tool
go-z-ai coding load <tool> --key OTHER_KEY --plan glm_coding_plan_china  # one-off override
```

`--plan` 和 `--key` 存在于 `coding load` 和 `coding mcp add` 上；各自独立地回退到
已存储的值。如果既没有已存储的凭证，也没有覆盖项，命令会以
`no credentials configured` 失败。

默认情况下，`coding auth` 在存储新 key 前会先对 API 进行校验：对套餐 coding
端点发起一次真实的 `GET /models`（30 秒超时）。401 会被报告为被拒绝
（`Z.AI rejected the key (401)`）；网络错误会被报告为校验失败。如果你想离线存储
key（例如为尚未联网测试过的机器编写脚本），可以用 `--no-validate` 跳过校验。

## MCP 服务器

官方 `@z_ai/coding-helper` 有一个"管理 MCP 服务"的步骤。`coding mcp` 对 Z.AI
官方的四个 GLM Coding Plan MCP 服务器做同样的事，把它们注册到你正在使用的
工具中：

| Server ID | 类型 | 作用 |
|---|---|---|
| `zai-mcp-server` | 本地，`npx -y @z_ai/mcp-server` | Vision：截图 OCR、错误截图诊断、UI 转产物与 UI 差异检查、图表和图形理解、图像和视频分析（[文档](https://docs.z.ai/devpack/mcp/vision-mcp-server)） |
| `web-search-prime` | 托管，streamable HTTP | 网页搜索（[文档](https://docs.z.ai/devpack/mcp/search-mcp-server)） |
| `web-reader` | 托管，streamable HTTP | 抓取并阅读网页（[文档](https://docs.z.ai/devpack/mcp/reader-mcp-server)） |
| `zread` | 托管，streamable HTTP | 读取和搜索 GitHub 仓库（[文档](https://docs.z.ai/devpack/mcp/zread-mcp-server)） |

```bash
go-z-ai coding mcp add claude-code              # all four, using the stored credentials
go-z-ai coding mcp add opencode --server web-search-prime --server zread
go-z-ai coding mcp add crush --plan glm_coding_plan_china --key OTHER_KEY
go-z-ai coding mcp remove claude-code --server zread
go-z-ai coding mcp status                       # which tools have which servers
```

`--server` 接受服务器 ID，可重复使用或以逗号分隔；默认是该工具支持的每一个
服务器，未知的 ID 会报错，并列出有效的 ID。如果条目已存在，`add` 会再次写入该
条目，从而刷新 key。`remove` 只会删除所指明的官方条目，文件中的其他 MCP 服务器
一概不动。`mcp status` 会读取每个工具的 MCP 文件，并列出在其中找到的官方服务器
ID。

**Codex 只接受 Vision 服务器。** Codex 的 streamable-HTTP MCP 客户端把 Z.AI 托管
服务器对 `notifications/initialized` 返回的空应答视为致命错误
（[openai/codex#14793](https://github.com/openai/codex/issues/14793)），所以官方
助手在 Codex 上会隐藏它们，`coding mcp add codex` 也同样如此：默认注册 Vision，
并以该原因拒绝托管服务器，不写入任何内容。

托管服务器的地址是 `https://<gateway>/api/mcp/<name>/mcp`，其中 `<name>` 为
`web_search_prime`、`web_reader`、`zread` 之一，使用
`Authorization: Bearer <your key>` 鉴权。Vision 服务器在本地运行，会以
`Z_AI_API_KEY` 获得 key，另外还有 `Z_AI_MODE`：global 套餐为 `ZAI`，China 套餐为
`ZHIPU`。这个开关决定了 Vision 服务器调用哪个平台，所以中国的 key 若搭配 `ZAI`
就会打到 `api.z.ai`。

对这些服务器的调用会消耗你套餐的配额。在 credit 套餐上，Web Search、Web Reader
和 Zread 每次调用消耗 1.2 credit（[文档](https://docs.z.ai/devpack/overview)）；
见 [账户与配额](accounts-and-quota.md)。

**Vision 服务器需要 Node.js。** 它通过 `npx` 运行。Z.AI 自家文档推荐 Node.js
22+，不过 npm 包只声明了 18+ 的要求。如果 `PATH` 上找不到 `npx` 而你又选择了
Vision 服务器，`coding mcp add` 会发出警告（不会中断），`coding doctor` 和
`coding mcp status` 也会打印同样的提示，因为只要 Node.js 一可用，该配置就是
有效的。

**MCP 配置文件往往与你的 GLM 凭证不是同一个文件。** 五个工具中有两个把 MCP
服务器存放在与 provider/API 设置不同的文件里，并且每个工具都有自己的条目结构：

| Tool | 凭证配置 | MCP 配置 | 条目结构 |
|---|---|---|---|
| Claude Code | `~/.claude/settings.json` | `~/.claude.json`，`mcpServers` 键 | `{type: "stdio", command, args, env}` / `{type: "http", url, headers}` |
| Codex | `~/.codex/config.toml` | 同一文件，`mcp_servers` 表 | `{type: "local", command, args, env}`；不支持托管服务器 |
| OpenCode | `opencode.json` | 同一文件，`mcp` 键 | `{type: "local", command: [...], environment}` / `{type: "remote", url, headers}` |
| Crush | `crush.json` | 同一文件，`mcp` 键 | `{type: "stdio", ...}` / `{type: "http", ...}` |
| Factory Droid | `~/.factory/settings.json` | `~/.factory/mcp.json`，`mcpServers` 键 | 同 Claude Code，外加 `disabled: false` |

## 状态与 JSON 输出

```bash
go-z-ai coding status          # stored plan, whether a key is stored, every tool
go-z-ai coding tools           # IDs, commands, install status, config paths
go-z-ai coding mcp status      # official MCP servers per tool
go-z-ai coding status --format json
```

`status`、`tools` 和 `mcp status` 接受 `--format json`（默认 `text`）。`tools` 和
`mcp status` 会打印一个数组，每个工具一个对象；`status` 会打印
`{"credentials": {"plan": ..., "key_stored": true}, "tools": [...]}`——无论这里还是
`doctor`，都不会输出 key 的任何部分。每个工具对象都有
`id`、`name`、`command`、`installed`、`config_path`、`configured`，以及（存在时）
`plan`、`model_map`（Claude Code 的层级映射）、`mcp_servers` 和 `error`（无法读取
的配置，例如格式错误的 JSON；一个工具读取失败不会妨碍其他工具被报告）。

```text
Stored credentials
==================
  Plan: GLM Coding Plan (Global)
  Key:  stored

Coding tools
============
  Claude Code    installed       Z.AI · GLM Coding Plan (Global) · MCP: zai-mcp-server, web-search-prime, web-reader, zread
                 models: haiku=glm-5.3-flash[1m] sonnet=glm-5.3[1m] opus=glm-5.3[1m]
  Codex          installed       Z.AI · GLM Coding Plan (Global) · MCP: zai-mcp-server
  OpenCode       not installed   native config
  Crush          not installed   native config
  Factory Droid  not installed   native config
```

`native config` 表示该工具的配置中没有 Z.AI 套餐。

## Doctor

```bash
go-z-ai coding doctor
```

检查是否已存储套餐和 key、每个受支持工具的配置能否被读取，以及 `PATH` 上是否至少
安装了一个受支持的工具；它还会提示 `npx` 对 Vision MCP 服务器是否可用。每条
发现占一行（`✓` 正常、`⚠` 有问题、`ℹ` 提示）。当某些功能不工作时，这是很好的
第一步。

`doctor` 在发现任何问题时会以状态 1 退出，并输出 `N problem(s) found`：没有已存储
的凭证、某个工具的配置无法解析，或 `PATH` 上没有任何受支持的工具。缺少 `npx` 只是
一条提示，不会改变退出状态，因此 `doctor` 可用作 CI 或环境预置时的检查。

## 合规与使用政策 ⚠️

Z.AI 的 coding 端点（`/api/coding/paas/v4`）受
[使用政策](https://docs.z.ai/devpack/usage-policy)限制，仅限"官方支持的工具"
使用，并明确禁止"基于 SDK 的访问"。三次违规将导致账户被封禁。在服务器端，
无法识别身份的第三方客户端与被禁止的访问无从区分
（[pi#4187](https://github.com/earendil-works/pi/issues/4187)）。
[套餐概览](https://docs.z.ai/devpack/overview)说的是同一件事：该套餐面向官方
支持的工具。

`go-z-ai` 通过两种方式缓解这一问题：

1. **`coding` 子命令接入的是 Z.AI 有文档记载的工具**（Claude Code、Codex、
   OpenCode、Crush、Factory Droid 在 `docs.z.ai/devpack/tool/` 下各有一个页面），
   把它们接入各自的原生配置格式——与 Z.AI 自己的 `@z_ai/coding-helper` 所做的
   事情相同。这种接入本身就是受支持的路径；`go-z-ai` 只是用一个 Go 二进制文件
   把它自动化。哪些工具在政策的受支持名单上，由 Z.AI 说了算；请查看政策页面
   获取当前名单。
2. **`go-z-ai` 自己发出的每个请求都带有一个用于标识身份的
   `User-Agent: go-z-ai/<version>` 头**（需要自有标识的下游应用、代理和 MCP
   服务器可通过 `Config.UserAgent` 覆盖它）。这是把 go-z-ai 与匿名/被禁止的访问
   区分开来的最低限度的规范做法。

这样做**并不能**达到的效果：

- 它**不会**让 go-z-ai 成为"官方支持的工具"——只有 Z.AI 才能授予该身份。在此之前，
  使用 `coding` 子命令接入受支持的工具是合规路径；而在自定义集成中直接使用
  `pkg/client` 访问 `/api/coding/paas/v4`，风险由用户自行承担。
- 它**不会**伪装或规避检测。这个头诚实地标识了客户端；目标是做一个好公民，而
  不是隐藏自己。

如果你在 `pkg/client` 之上构建下游工具、代理或 MCP 服务器，请设置一个独特的
`Config.UserAgent`（例如 `"my-tool/1.0 (go-z-ai)"`），这样 Z.AI 就能把你的流量
单独识别出来，你也就继承了 go-z-ai 的好公民默认做法，而不是削弱它。
