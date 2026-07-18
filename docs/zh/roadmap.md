# 路线图与已知限制

这里记录还未收尾的事项、原因，以及让它收尾的确切动作。当某条事项被一个已提交的
cassette 或一次已合并的改动解决后，它就会从本页移除——这是 verify-first（先验证）
约定的工作清单（缘由见
[贡献指南 § the live-verification convention](../../CONTRIBUTING.md) 和
[架构 § the live-verification convention](architecture.md#the-live-verification-convention)）。

> **自行录制 cassette：** 下方每一条"未实测"事项都对应一个 `TestVerify*`
> 测试，在你用 `ZAI_RECORD=1` 录制一份成功路径 cassette 之前它会一直 SKIP。
> 测试框架在保存前会把 `Authorization` 改写为 `Bearer REDACTED`——提交前用
> `grep "Bearer " pkg/client/testdata/cassettes/<name>.yaml` 确认一下。
>
> ```sh
> ZAI_RECORD=1 ZAI_API_KEY=<real-key> go test -run TestVerify<Name> ./pkg/client
> ```

## 未实测

分两类：**服务**——其成功路径的响应结构尚未被录制；以及 **2026-07-18 sprint 中新增的字段**——
它们与文档一致，但尚未被某个 cassette 固定下来。

### 仍需成功路径 cassette 的服务

目前使用的开发账户没有这些服务的 PAYG 余额 / 权益，因此只验证了请求结构和错误路径。
一份录制到真实成功响应的 cassette 就能让每条事项收尾。

- **Anthropic Messages**（`TestVerifyAnthropicMessages`）——路由、`anthropic-version`
  header 以及 Bearer 鉴权都已确认（一个假 key 会返回干净的 401，而不是 404 / 超时）。
  cassette 能给出定论的问题是：GLM 是把推理内容作为 Anthropic 的 `thinking` block
  暴露，还是作为 OpenAI 风格的 `reasoning_content` 字段？
  （[claude-code-router#1133](https://github.com/musistudio/claude-code-router/issues/1133)）
- **Embeddings**（`TestVerifyEmbeddings`）——目前在所有测试过的账户上都返回
  `400 Unknown Model`（code 1211）；这是权益门槛，不是路由 bug
  （见 [账户与配额](accounts-and-quota.md)）。
- **Moderations**（`TestVerifyModerations`）——同样的 1211 权益门槛。
- **Agents `Invoke` 的成功结构**（`TestVerifyAgentsInvoke`）——目前只有失败信封
  （`ID`/`AgentID`/`Status`/`Error`）经过实测确认，通过
  `testdata/cassettes/agents_invoke.yaml`（一个 200 状态码内嵌失败的响应）。
  `Choices`/`Usage` 的成功结构仅依据文档建模。
- **Voice `Clone` / `Delete`**（`TestVerifyVoiceClone`、`TestVerifyVoiceDelete`）
  ——`Voice List` 已经实测确认；clone / delete 需要一段已上传的样音和一个真实的克隆
  voice ID 才能录制。clone 需要 `ZAI_VOICE_SAMPLE_FILE_ID` + `ZAI_VOICE_NAME`；
  delete 需要 `ZAI_VOICE_ID`。
- **Batch 和 Files 端点整体**——还没有专门的 `TestVerify*` 脚手架；需要一个有权益的
  PAYG 账户才能录制。

### 2026-07-18 sprint 中新增的字段，等待一份 cassette

这些字段被加进了 `pkg/client/types.go` / `chat.go`，用以匹配当前 docs.z.ai 的
chat-completion 规格。它们都是增量字段并已通过单元测试，但在 cassette 固定其确切线上
结构之前**未经实测验证**。每一条都已有对应的 `TestVerify*` 测试待录制。

- **`ChatRequest.StreamToolCall`**（`TestVerifyChatStreamToolCall`）——GLM-4.6+
  流式工具调用 delta。cassette 应展示 tool-call delta 分散在多个 SSE chunk 中通过
  `StreamDelta.ToolCalls` 到达。
- **`Tool` 在 `function` / `retrieval` / `web_search` 之间的判别**
  （`NewFunctionTool` / `NewRetrievalTool` / `NewWebSearchTool`）——规格列出了全部
  三种类型；目前只有 `function` 被确认。`web_search` 的 payload 结构
  （`{"search_query":[...]}`）遵循官方 Python SDK 的示例。
- **`ChatResponse.WebSearch`**（`TestVerifyChatWebSearchResponse`）——当某个
  `web_search` 工具触发时返回的顶层 `web_search` 数组。条目结构复用了 `tools.go`
  里的 `WebSearchResult`（独立 web-search 工具的版本已实测验证）；作为顶层数组
  出现的位置是依据文档建模的。
- **`ThinkingConfig.Effort = "xhigh"`**——加入了经过校验的枚举
  （`xhigh`→`max`，仅 GLM-5.2）。没有专门的测试；任何 thinking + xhigh 的 cassette
  都会覆盖它。
- **`FinishReason*` 常量**（`sensitive`、`model_context_window_exceeded`、
  `network_error`）——来自文档；目前还没有 cassette 复现这些终止路径。
- **客户端的工具名正则**（`^[A-Za-z0-9_-]{1,64}$`）以及 **128 个函数的上限**——
  这些是服务端文档化的规则，我们在本地强制执行；尚未确认它们是否就是服务端的精确
  拒绝准则。
- **面向 monitor/biz/agents/detection 的中国区域网关**——`RegionChina` 把
  quota/usage/account/agents/detection 路由到 `open.bigmodel.cn`。`/models` 和
  `/chat/completions` 已在中国 host 上实测验证；monitor/biz/agents 路径是通过镜像
  `api.z.ai` 的布局建模的，需要用一把有权益的中国 key 录制一份 cassette 来确认。

### 较早的遗留问题（暂无专门测试）

- **Tool-schema 兼容性改写**——GLM 解析器对哪些 JSON-Schema 构造返回 HTTP 500
  （`anyOf`/`oneOf`/`allOf`/`$ref`）这一集合，来自社区 bug 反馈
  （[claude-code-router#1474](https://github.com/musistudio/claude-code-router/issues/1474)），
  并未在此对真实账户复现。改写本身已通过完整的单元测试，并且对已经是平铺结构的
  schema 完全无效；如果有一份 cassette 精确地固定下哪些构造会 500（以及改写后的
  输出让其中哪些能通过），就能把这条从"文档化的行为"升级为"已实测验证"。见
  `pkg/client/toolschema.go`。

## 未实现

- **请求/响应日志与指标采集**——目前没有内置的 instrumentation hook。
- **性能基准**——推迟到真正测量出瓶颈时再做；目前没有任何已知的热点路径值得为其
  建立基准（先 profile，再优化）。

## 故意不实现

- **Assistant API**——已确认被弃用。Z.AI 自己线上的 OpenAPI 规格
  （`docs.bigmodel.cn/openapi/openapi.json`）把每一条 Assistant 路径都标记为
  `"deprecated": true`，并且从 `api.z.ai` 调用它会完全超时，而不是返回错误。
  为一个已下线的 API 编写客户端，在维护成本上不划算——如果 Z.AI 哪天重新启用它，
  上面的规格里就有完整的请求/响应 schema 可以直接转录。
