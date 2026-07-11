# Z.AI API Client

A comprehensive, production-ready Go client for the Z.AI (Zhipu AI) API platform with CLI tools for managing chat completions, models, usage, billing, and account operations.

## Features

- **Complete API Coverage**: Support for chat completions, models, usage monitoring, and billing
- **Streaming**: Token-by-token streaming via SSE (`chat create --stream`, `Chat().CreateStream`)
- **Structured Output**: `json_schema` response format (`--json-schema`)
- **Function Calling**: tool declarations in the CLI + a Go `RunWithTools` auto-executing loop
- **Deep Thinking**: `thinking`/`effort` controls (`--thinking`, `--effort`)
- **Resilient Transport**: automatic retry with exponential backoff, jitter, and `Retry-After` on 429/5xx/network errors
- **Clean Architecture**: SRP-compliant design with separate services and reusable components
- **CLI Interface**: Rich command-line interface with multiple output formats
- **Configuration Management**: Flexible configuration via environment variables, flags, and config files
- **Error Handling**: Structured errors with categories, retriable flags, and user-friendly messages
- **Type Safety**: Strongly typed request/response structures
- **Caching**: Built-in model caching for improved performance
- **Coding Tool Setup**: configure Claude Code, OpenCode, Crush, and Factory Droid to use your GLM Coding Plan — a Go port of `@z_ai/coding-helper` (`pkg/coding`)

## Installation

### Prerequisites
- Go 1.26.4 or higher
- Z.AI API key ([Get one here](https://z.ai/manage-apikey))

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd zai-api

# Build the CLI
go build -o zai-client .

# Or install directly
go install .
```

## Configuration

Set your API key using one of these methods:

### 1. Environment Variables (Recommended)
```bash
export ZAI_API_KEY=your_api_key_here
export ZAI_API_BASE_URL=https://api.z.ai/api/paas/v4  # Optional
```

### 2. .env File
```bash
cp .env.example .env
# Edit .env with your API key
```

### 3. Command Line Flags
```bash
zai-client --api-key your_key --base-url https://api.z.ai/api/paas/v4
```

### 4. Multiple Accounts

If you switch between more than one Z.AI account, use the `accounts` command
instead of hand-editing `.env`. Each account declares an explicit type
(`coding_plan` or `pay_as_you_go`, auto-detected via a free probe unless
`--type` is passed), which the CLI uses to derive the correct base URL —
closing the class of bug where a key and base URL get mismatched by hand.
Credentials are stored in `~/.config/zai-client/accounts.json` (0600).

```bash
zai-client accounts add work --api-key your_key       # type auto-detected
zai-client accounts add personal --api-key other_key
zai-client accounts list                              # NAME  TYPE  BASE URL  API KEY  ACTIVE
zai-client accounts use work                           # switch the active account
zai-client accounts current                            # show the active account's details
zai-client accounts quota                               # check usage/reset times across ALL accounts
zai-client accounts usage                                # token/tool usage heat map, all accounts, last 14 days
zai-client accounts usage --today                        # today only, hourly detail
zai-client --account personal usage quota               # one-off: run any command against a specific account
zai-client accounts remove personal --yes               # required if removing the active account
```

`--api-key`/`--base-url` flags and `ZAI_API_KEY`/`ZAI_API_BASE_URL` env vars
still take precedence over the accounts store, so this is purely additive —
existing single-account `.env` setups keep working unchanged.

`accounts usage` renders a terminal heat map (`░▒▓█`, scaled per row) of
token usage per model and call counts per tool (web search/reader/zread),
bucketed daily by default or hourly for an 8-day-or-less window (`--today`,
or `--days` 1-8). The underlying API doesn't expose a request-by-request
log — only aggregate counts per time bucket — so this is activity-over-time,
not a request history.

## Coding Tool Configuration (GLM Coding Plan)

`zai-client coding` is a Go port of Z.AI's official [`@z_ai/coding-helper`](https://www.npmjs.com/package/@z_ai/coding-helper) (`npx @z_ai/coding-helper`). It stores your GLM Coding Plan credential and loads it into supported coding tools in each tool's native config format. The credential store at `~/.chelper/config.yaml` is shared/compatible with the Node helper.

Supported tools: **Claude Code, OpenCode, Crush, Factory Droid**. Plans: `glm_coding_plan_global` (api.z.ai) and `glm_coding_plan_china` (open.bigmodel.cn).

```bash
# Store + validate your key (plan global|china)
zai-client coding auth glm_coding_plan_global <your-api-key>
zai-client coding auth glm_coding_plan_global <key> --no-validate   # skip network check

# Load the stored credential into a coding tool (writes its native config)
zai-client coding load claude-code     # alias: claude
zai-client coding load opencode
zai-client coding load crush
zai-client coding load factory-droid

# Reload stored creds into a tool (same as load)
zai-client coding auth reload claude

# Remove Z.AI config from a tool, or revoke the stored key
zai-client coding unload claude-code
zai-client coding auth revoke

# Inspect state
zai-client coding status     # stored creds + per-tool detection
zai-client coding tools      # supported tools, install status, config paths
zai-client coding doctor     # health check
```

What each tool gets:

| Tool | Config file | What is written |
|---|---|---|
| Claude Code | `~/.claude/settings.json` (+ `hasCompletedOnboarding` in `~/.claude.json`) | `env`: `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_BASE_URL`, `API_TIMEOUT_MS`, `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC`, plus model mapping + auto-compact window (see below) |
| OpenCode | `~/.config/opencode/opencode.json` | provider `zai-coding-plan`/`zhipuai-coding-plan` with `apiKey`; `model`, `small_model` |
| Crush | `~/.config/crush/crush.json` | `providers.zai` (base_url + api_key) |
| Factory Droid | `~/.factory/settings.json` | two `customModels` (Anthropic + OpenAI protocols) |

### Claude Code model mapping & tuning

The official `@z_ai/coding-helper` writes only the four base env vars above and relies on Z.AI's endpoint to default the model. Claude Code actually supports more knobs, so `zai-client coding load claude-code` goes further and applies (by default) Z.AI's **documented recommended model mapping** and a large **auto-compact window** — giving you explicit control over cost, speed, and context use:

| Env var | Default | Purpose |
|---|---|---|
| `ANTHROPIC_DEFAULT_HAIKU_MODEL` | `glm-4.5-air` | fast/cheap tier |
| `ANTHROPIC_DEFAULT_SONNET_MODEL` | `glm-4.7` | balanced tier |
| `ANTHROPIC_DEFAULT_OPUS_MODEL` | `glm-4.7` | strongest tier |
| `CLAUDE_CODE_AUTO_COMPACT_WINDOW` | `1000000` | push auto-compaction out for large-context GLM models |

Mapping defaults come from Z.AI's docs ([scenario-example/develop-tools/claude](https://docs.z.ai/scenario-example/develop-tools/claude) → "How to Switch the Model"). Override any tier or the window, or disable the extras:

```bash
# Custom tiers
zai-client coding load claude-code --haiku glm-4.5-flash --sonnet glm-4.6 --opus glm-5.2

# 128K-context model → smaller window
zai-client coding load claude-code --auto-compact-window 128000

# Extended-thinking + output budgets
zai-client coding load claude-code --max-thinking-tokens 8000 --max-output-tokens 65536

# Match @z_ai/coding-helper exactly (base env vars only, no model mapping or window)
zai-client coding load claude-code --no-model-mapping --auto-compact-window 0
```

> **Why `ANTHROPIC_AUTH_TOKEN` not `ANTHROPIC_API_KEY`:** Claude Code's env reference maps `ANTHROPIC_AUTH_TOKEN` → `Authorization: Bearer` and `ANTHROPIC_API_KEY` → `X-Api-Key`. Z.AI's Anthropic endpoint authenticates via `Bearer`, so `AUTH_TOKEN` is the correct one (the official helper agrees). Earlier builds of this tool wrote `ANTHROPIC_API_KEY` + model mapping — corrected.

## CLI Usage

### Validate API Configuration

```bash
# Test your API key
zai-client validate

# With specific API key
zai-client validate --api-key your_key
```

### Chat Completions

#### Simple Chat
```bash
# Basic chat completion
zai-client chat simple glm-5.2 "Hello, how are you?"

# With system message
zai-client chat create "Explain quantum computing" \
  --model glm-5.2 \
  --temperature 0.7 \
  --max-tokens 2000 \
  --system "You are a physics expert"

# JSON output
zai-client chat create "What is AI?" --format json
```

#### Advanced Chat

```bash
# Stream the response token-by-token
zai-client chat create "Write a haiku about the sea" --model glm-5.2 --stream

# Deep thinking with a reasoning effort
zai-client chat create "Prove there are infinitely many primes" \
  --thinking enabled --effort high --show-reasoning

# Structured output (json_schema) from a file
zai-client chat create "Extract the user details" \
  --json-schema @schema.json --schema-name user --schema-strict

# Structured output from an inline schema
zai-client chat create "Give me a person" \
  --json-schema '{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}}}'

# Declare function-calling tools; the model's tool_calls are printed (not executed)
zai-client chat create "What's the weather in Paris?" --tool @tools.json

# Stop sequences and nucleus sampling
zai-client chat create "List three fruits" --stop "[" --top-p 0.9
```

Streaming prints content deltas to stdout (JSONL chunks with `--format json`);
reasoning goes to stderr under `--show-reasoning`. The CLI declares tools and
shows any `tool_calls` the model returns — for an auto-executing loop, use the
Go `RunWithTools` helper.

### Models

#### List Models
```bash
# List all models
zai-client models list

# With pricing information
zai-client models list --pricing

# JSON output
zai-client models list --format json

# List text-only models
zai-client models text

# List vision models
zai-client models vision

# List free models
zai-client models free
```

#### Get Model Details
```bash
zai-client models get glm-5.2
zai-client models get glm-4.7 --format json
```

### Usage & Quota

#### Check Quota
```bash
# Get current quota and usage
zai-client usage quota

# JSON output
zai-client usage quota --format json
```

#### Usage Summary
```bash
# Comprehensive usage summary
zai-client usage summary

# JSON output
zai-client usage summary --format json
```

#### Account Information
```bash
# Get account details
zai-client usage account

# JSON output
zai-client usage account --format json
```

#### Billing Information
```bash
# Get billing details
zai-client usage billing

# JSON output
zai-client usage billing --format json
```

#### Check Quota Status
```bash
# Check if quota is low
zai-client usage check

# Watch mode (check every minute)
zai-client usage check --watch
```

## Go Client Library

### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "zai-api-client/pkg/client"
)

func main() {
    // Create client from environment
    apiClient, err := client.NewClientFromEnv()
    if err != nil {
        log.Fatal(err)
    }

    // Create chat completion
    messages := []client.Message{
        {Role: "user", Content: "Hello!"},
    }
    
    response, err := apiClient.Chat().CreateSimple("glm-5.2", "Hello!", messages)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response.Choices[0].Message.Content)
}
```

### Advanced Usage

```go
// Create client with custom configuration
config := client.Config{
    APIKey:  "your_api_key",
    BaseURL: "https://api.z.ai/api/paas/v4",
    Timeout: 30 * time.Second,
}

apiClient, err := client.NewClient(config)
if err != nil {
    log.Fatal(err)
}

// Advanced chat completion with parameters
req := client.ChatRequest{
    Model: "glm-5.2",
    Messages: []client.Message{
        {Role: "system", Content: "You are a helpful assistant"},
        {Role: "user", Content: "Explain Go programming"},
    },
    Temperature: 0.8,
    MaxTokens:   4096,
    Thinking: &client.ThinkingConfig{
        Type:   "enabled",
        Effort: "high",
    },
}

response, err := apiClient.Chat().Create(req)
if err != nil {
    log.Fatal(err)
}

// Check usage
quota, err := apiClient.Usage().GetQuota()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Used: %d/%d tokens\n", quota.UsedQuota, quota.TotalQuota)

// List models
models, err := apiClient.Models().List()
if err != nil {
    log.Fatal(err)
}

for _, model := range models.Models {
    fmt.Printf("%s: %s\n", model.ID, model.Name)
}
```

### Streaming, Tools & Structured Output

```go
// Streaming: receive one delta at a time over SSE
err = apiClient.Chat().CreateStream(ctx, req, func(ch client.StreamChunk) error {
    if len(ch.Choices) > 0 {
        fmt.Print(ch.Choices[0].Delta.Content)
    }
    return nil // return non-nil to abort
})

// Structured output via json_schema
req.ResponseFormat = client.NewJSONSchemaFormat(
    "person",
    json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"integer"}}}`),
    true, // strict
)

// Auto-executing tool-calling loop: the SDK dispatches each tool_call to your
// function and re-requests until the model returns a final answer.
req.Tools = []client.Tool{{Type: "function", Function: &client.FunctionDef{Name: "get_weather"}}}
resp, err := apiClient.Chat().RunWithTools(ctx, req, func(name, args string) (string, error) {
    return callMyTool(name, args)
})

// Resilient transport: tune retry (defaults 3 retries, 200ms base; -1 disables).
config := client.Config{APIKey: "your_api_key", MaxRetries: 5, RetryDelay: 100 * time.Millisecond}
```

Transient failures (429, 5xx, network errors with a *retriable* API code) are
retried automatically with exponential backoff, jitter, and `Retry-After` honor;
non-retriable codes (e.g. quota-exhausted `1308`, auth `1308/1001`) fail fast.

## Architecture

### Project Structure
```
zai-api/
├── main.go                # Entry point + root cobra command
├── chat.go                # Chat CLI (stream, schema, tools, thinking)
├── models.go              # Model CLI commands
├── usage.go               # Usage/quota CLI commands
├── accounts_cli.go        # Multi-account management CLI
├── coding_cli.go          # GLM Coding Plan credentials & coding-tool config CLI
├── tools_cli.go           # Tools CLI
├── tui_cli.go             # Interactive terminal UI entry point ("zai-client tui")
├── common.go              # Shared CLI helpers (getClient, outputJSON, maskAPIKey)
├── pkg/
│   ├── client/            # API client package (SDK)
│   │   ├── client.go      # HTTP client, retry/backoff
│   │   ├── chat.go        # ChatService (Create, CreateStream + SSE)
│   │   ├── chat_tools.go  # RunWithTools tool-calling loop
│   │   ├── types.go       # Request/response/streaming types
│   │   ├── errors.go      # Structured API error mapping
│   │   ├── models.go      # ModelsService
│   │   ├── usage.go       # UsageService
│   │   ├── quota.go       # Quota + tool-usage monitoring
│   │   ├── account.go     # AccountService
│   │   ├── tools.go       # Web search/reader/tokenizer tools
│   │   └── detection.go   # Endpoint / key-type detection
│   ├── accounts/          # Multi-account credential store
│   ├── coding/            # GLM Coding Plan credentials + per-tool config writers
│   │                      #   (Claude Code, OpenCode, Crush, Factory Droid, Cursor)
│   ├── usageview/         # Shared usage/quota rendering helpers (CLI + TUI)
│   └── tui/                # Bubble Tea v2 interactive terminal UI
├── .env / .env.example
└── README.md
```

### Design Principles

- **Single Responsibility**: Each service handles one aspect of the API
- **Reusability**: Services can be used independently or together
- **Type Safety**: Strong typing for requests and responses
- **Error Handling**: Comprehensive error handling with context
- **Configuration**: Flexible configuration management
- **Testing**: Testable components with clear interfaces

## API Services

### ChatService
Handles all chat completion operations:
- `Create()` - Create chat completion with full parameters
- `CreateSimple()` - Simple chat with defaults
- `CreateStream(ctx, req, onChunk)` - Streaming completion via SSE (one delta per callback)
- `RunWithTools(ctx, req, exec)` / `RunWithToolsLimit(...)` - Auto-executing tool-calling loop

### ModelsService
Manages model information:
- `List()` - List all available models
- `Get()` - Get specific model details
- `GetTextModels()` - List text-only models
- `GetVisionModels()` - List vision-capable models
- `GetFreeModels()` - List free models
- `RefreshCache()` - Refresh model cache

### UsageService
Monitors usage and quotas:
- `GetQuota()` - Get current quota information
- `GetAccountInfo()` - Get account details
- `GetBillingInfo()` - Get billing information
- `GetUsageSummary()` - Get comprehensive summary
- `IsQuotaLow()` - Check if quota is running low
- `TimeUntilReset()` - Get time until quota reset

## Error Handling

The client provides detailed error messages for common issues:

- `ErrInvalidAPIKey` - Invalid or missing API key
- `ErrInvalidModel` - Model not found or invalid
- `ErrRateLimitExceeded` - Rate limit exceeded
- `ErrQuotaExceeded` - Quota exceeded
- `ErrUnauthorized` - Authorization failed
- `ErrNetworkError` - Network connectivity issues

## Configuration Priority

1. Command line flags (highest priority)
2. Environment variables
3. .env file
4. Default values (lowest priority)

## Examples

### Complete Chat Example
```bash
# Set API key
export ZAI_API_KEY=your_key

# Validate configuration
zai-client validate

# Create chat completion
zai-client chat create "Write a Go function to reverse a string" \
  --model glm-5.2 \
  --temperature 0.5 \
  --max-tokens 1000 \
  --system "You are a Go programming expert"

# Check usage
zai-client usage quota
```

### Monitor Quota Script
```bash
#!/bin/bash
# Monitor quota and alert if low

while true; do
  if zai-client usage check | grep -q "WARNING"; then
    echo "⚠️  Quota is running low!"
    # Send notification
  fi
  sleep 300  # Check every 5 minutes
done
```

## Contributing

1. Follow Go best practices and idioms
2. Maintain SRP in all components
3. Add tests for new functionality
4. Update documentation as needed
5. Use meaningful commit messages

## License

[Specify your license]

## Support

- **Documentation**: [https://docs.z.ai](https://docs.z.ai)
- **API Reference**: [https://docs.z.ai/api-reference/introduction](https://docs.z.ai/api-reference/introduction)
- **Issues**: [GitHub Issues](https://github.com/zai-org/z-ai-sdk-python/issues)

## Roadmap

- [x] Streaming chat completions
- [x] Function calling support (Go `RunWithTools` loop; CLI tool declaration)
- [x] Structured output (`json_schema`)
- [x] Deep thinking controls
- [x] Retry logic with exponential backoff + `Retry-After`
- [x] Test suite for the client (errors, retry, streaming, tools, types)
- [x] Coding-tool configuration (Claude Code, OpenCode, Crush, Factory Droid) via `pkg/coding` — port of `@z_ai/coding-helper`
- [ ] Image generation support (CogView-4)
- [ ] Video generation support (CogVideoX-3, Vidu)
- [ ] Multimodal message content (vision models)
- [ ] Batch request handling
- [ ] Performance benchmarks
- [ ] Request/response logging
- [ ] Metrics collection

See [`SPRINT_PLAN.md`](./SPRINT_PLAN.md) and [`todo.md`](./todo.md) for the
Sprint B plan (generation, multimodal, correctness) and known limitations.