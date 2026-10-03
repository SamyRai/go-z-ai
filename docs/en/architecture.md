# Architecture

## Contents

- [Package layout](#package-layout)
- [CLI conventions](#cli-conventions)
- [The request facade](#the-request-facade)
- [Streaming](#streaming)
- [Retry and timeout design](#retry-and-timeout-design)
- [Why some services hit a different host](#why-some-services-hit-a-different-host)
- [The live-verification convention](#the-live-verification-convention)
- [Extending a structured lookup table](#extending-a-structured-lookup-table)
- [Credential file safety](#credential-file-safety)
- [The TUI](#the-tui)

## Package layout

```text
main.go               Entrypoint: sets build info, then internal/cli.Execute()
internal/cli/         Cobra commands (package cli), one file per command group:
                      chat.go, models.go, usage.go, validate.go, accounts_cli.go,
                      coding_cli.go, coding_mcp_cli.go, ...; common.go holds the
                      shared plumbing, root.go the global flags
pkg/client/           The Go library — no CLI/TUI dependency, stdlib-only
                      (no third-party imports); layout below
pkg/observe/          OpenTelemetry Hook implementation (otel.go; depends on
                      go.opentelemetry.io/otel); separate package so pkg/client
                      stays stdlib-only
internal/accounts/    Multi-account credential store (~/.config/zai-client/accounts.json —
                      dir kept as zai-client for upgrade compat)
internal/atomicfile/  Atomic temp-file-then-rename writes (Write, WriteWithBackup)
                      used by every credential store and tool-config writer
internal/coding/      GLM Coding Plan credential store, the tool registry (one
                      tool_*.go file per tool), and Z.AI's MCP servers
internal/fileinput/   FileOrURL: a URL passes through, a local path is base64-encoded —
                      shared by `ocr parse`, `chat create --image/--video/--file`,
                      and the TUI media tab
internal/modelview/   Presentation helpers for model metadata (token counts, prices,
                      capability names) — shared by the CLI and the TUI
internal/usageview/   Presentation helpers for quota and usage (time windows, heat maps,
                      quota summaries, peak-hours logic) — shared by the CLI and the
                      TUI so their output can never drift
internal/tui/         Bubble Tea terminal UI: a root model plus one subpackage per
                      tab and a few shared subpackages
```

`pkg/client` is the core public package and is deliberately **stdlib-only** —
zero third-party imports — so anyone can depend on the client without dragging
the OTel SDK, MCP SDK, or other heavy observability/integration deps. The
observability seam is a stdlib-only `Hook` interface (see
[Library Guide — Observability hooks](library-guide.md#observability-hooks));
concrete implementations live in `pkg/observe` (OpenTelemetry) and future
packages, each with its own dependency set so users only pay for what they
import. Everything under `internal/` is implementation the compiler forbids
outside code from importing, so the CLI/TUI layers can be refactored freely.
The CLI and TUI are both thin callers of `pkg/client`.
`go install github.com/SamyRai/go-z-ai@latest` builds the root `main.go` into
the `go-z-ai` binary.

### pkg/client file ownership

Each concern has exactly one owning file, so a change to it is a one-file edit:

| File | Owns |
|---|---|
| `client.go` | `Client`, `NewClient`, and the per-service accessors (`Chat()`, `Models()`, ...) |
| `config.go` | `Config`, its defaults (`DefaultTimeout`, `DefaultMaxRetries`, `DefaultRetryDelay`), validation, the default HTTP transport, and `NewClientFromEnv` |
| `region.go` | **Every URL.** `Region`, the gateway hosts, the API-root paths, and the `Region.*BaseURL` methods; `ParseRegion` |
| `transport.go` | The single request path (see below): `apiRequest`, `open`, `do`, `doRaw`, multipart encoding |
| `retry.go` | Retriability, backoff, `Retry-After` parsing |
| `envelope.go` | `Envelope[T]`, the `{code, msg, success, data}` wrapper of the monitor and biz APIs, and `fetchEnvelope` |
| `stream.go` | The synchronous SSE iterator (`streamSSE`, `scanSSE`) |
| `errors.go` | `APIError`, the error-code table, classification of unknown codes by HTTP status, in-band stream errors |
| `hook.go` | The `Hook` interface and the per-attempt call helpers |
| `models_catalog.go` | **All model knowledge:** IDs, contexts, prices, capabilities, reasoning efforts, and the `Default*Model` constants |
| `models.go` | `ModelsService`, `ModelDetails`, `Pricing` (including `Pricing.Cost`) |
| `chat_types.go` | Chat wire types: `Message`, `ChatRequest`, `ThinkingConfig`, `ResponseFormat`, tools, `ChatResponse`, `StreamChunk`, `Usage` |
| `chat.go` | `ChatService`: `Create`, `CreateAsync`, `Stream`, request validation (effort, tool rules) |
| `chat_tools.go` | `RunWithTools`, the tool-calling loop |
| `content.go` | Multimodal message encoding (`Message.Images`/`Videos`/`Files` to content parts) |
| `toolschema.go` | `SanitizeToolSchemas`, the JSON Schema rewrite GLM's parser needs, applied to chat, Anthropic, and Responses tools alike (`sanitizeTools`) |
| `anthropic.go` | `AnthropicService` (Messages API), including its stream decoder |
| `responses_types.go`, `responses.go` | Responses-protocol wire types (what Codex speaks); `ResponsesService`: `Create`, `Stream`, validation, stream decoder |
| `quota.go`, `timezone.go` | Coding-plan quota and usage (monitor API); the server-timezone handling |
| `account.go` | `AccountService` (biz API): `Balance`, `Subscriptions` |
| `detection.go`, `status.go` | `DetectionService`: `DetectAccountType`, `CheckAccountStatus` |
| `async.go` | Async task types and `GetAsyncResult`/`WaitForResult`, shared by chat, image, and video |
| `images.go`, `videos.go`, `audio.go`, `voice.go` | Media services |
| `files.go`, `batch.go`, `fileparser.go`, `layout.go` | Files, batch jobs, document parsing, layout parsing (OCR) |
| `tools.go`, `agents.go` | Web search/reader/tokenizer; agents |
| `embeddings.go`, `moderations.go`, `rerank.go` | Retrieval and moderation services |
| `version.go` | `Version()` and the default `User-Agent` |

`testdata/cassettes/` holds the recorded interactions the live-verification
tests replay.

## CLI conventions

Command handlers that need an API client are registered as
`RunE: runWithClient(runX)` and take the resolved `*client.Client` as a third
parameter — `runWithClient` (in `internal/cli/common.go`) resolves it once via
`getClient`, so the handlers don't each repeat the resolve-and-check preamble.
Credential precedence itself lives in `resolveConfig` (flag → `--account` →
`ZAI_API_KEY` → active account), unit-tested in `credentials_test.go`. The
same function resolves the region (`--region` → `ZAI_REGION` → the account's
stored region) and the other global settings.

Output format is uniform: every command that prints a result registers the
shared `--format` flag via `addFormatFlag` and renders through `emit(cmd, v,
textFn)`, which emits pretty JSON for `--format json` and otherwise runs the
human-readable `textFn`. Progress chatter (`progressf`) goes to stderr so
stdout stays valid JSON.

## The request facade

Every service method funnels through one request path — services never build
their own `http.Request` or `http.Client`. A service describes its call as an
`apiRequest` (method, path, body, and optionally a base URL, API key, extra
headers, a multipart form, and hook attribution) and hands it to one of these
`Client` entry points, all of which sit on `open`:

```text
doRequest(ctx, method, path, body, result)  // the common case: do() with the defaults
do(ctx, apiRequest, result)                 // open, then JSON-decode the body into result
doRaw(ctx, apiRequest)                      // open, then return the raw body (audio, file downloads)
streamSSE(ctx, c, apiRequest, decode)       // open, then decode the body incrementally (see Streaming)
        every one of them goes through Client.open(ctx, apiRequest) → *attempt
```

`open` owns authentication, retry/backoff, error parsing, and the
observability-hook lifecycle for every endpoint. Routing fields left empty on
an `apiRequest` fall back to the client's configuration: no base URL means
`Config.BaseURL`, no API key means `Config.APIKey`. A service that needs a
different host (Agents, monitor, biz, Anthropic) sets `baseURL` from a
`Region` method; one that needs a different credential (Embeddings,
Moderations) sets `apiKey` — it never bypasses `open`.

**Hook pairing.** `open` returns an `attempt`: the open 2xx response plus the
hook state of the attempt that produced it. Every attempt fires `OnRequest`
and then exactly one terminal hook — `OnResponse` or `OnError`. A failed
attempt that is about to be retried fires `OnError` before the backoff, and
the caller of `open` finishes the successful attempt with `succeed` or `fail`.
A span started in `OnRequest` can therefore always be ended.

**Multipart uploads** (files, audio transcription, document parsing) are an
`apiRequest` with `form` set. They go through `open` too but are never
retried: re-uploading a file on a transient failure is a caller decision, not
a safe default.

**The monitor/biz envelope.** The quota (monitor) and account (biz) APIs wrap
every response in `Envelope[T]` and report business failures inside an HTTP
200 body. `fetchEnvelope` decodes the envelope and turns a failure into an
`*APIError` (classified by the envelope code when it mirrors an HTTP status),
so a service that returns an `Envelope` has already succeeded and callers only
read `Data`.

## Streaming

`stream.go` implements server-sent events as a Go iterator. `Chat().Stream`
and `Anthropic().Stream` return `iter.Seq2[T, error]`; `streamSSE` runs the
whole stream synchronously inside the iterator — no producer goroutine, no
channel — so breaking out of the `range` loop stops reading and closes the
response body, and cancelling the context aborts a blocked read.
`scanSSE` is the shared line-level parser; each protocol supplies a small
decoder (`decodeChatStream` in `chat.go`, `decodeAnthropicStream` in
`anthropic.go`).

An error delivered in-band — an OpenAI-style `{"error": ...}` chunk or an
Anthropic `event: error` — ends the stream with an `*APIError`, built by the
same classifier as HTTP errors (`errors.go`). Hooks see every decoded chunk
through `OnStreamChunk`; the attempt ends with `OnResponse` on a clean end, or
`OnError` on a mid-stream failure or an early break (`context.Canceled`).

## Retry and timeout design

- `Config.Timeout` bounds dial + TLS handshake + waiting for response
  headers — deliberately **not** the whole `http.Client.Timeout`, which would
  truncate a long-running `Stream` read partway through a generation.
- A failed attempt is retried when it is retriable: a transport failure (the
  server never answered), or an `*APIError` flagged retriable — HTTP 429 and
  5xx, plus the business codes `errors.go` marks as transient. Error codes the
  table doesn't know are classified by HTTP status alone, so only throttling
  and server-side failures are ever retried; auth, quota, and parameter errors
  are not.
- Backoff is exponential from `Config.RetryDelay` with jitter (up to 25%).
  A `Retry-After` header (delay-seconds or an HTTP date) wins when present.
  Any single delay is capped at 30 seconds. The retry loop is bounded by
  `Config.MaxRetries` (default 3; `-1` disables retries). Every retry checks
  `ctx.Err()` first, so a cancelled context aborts immediately instead of
  sleeping through a backoff.
- Streaming retries only the *connection* attempt — once the SSE stream has
  actually started, a mid-stream failure is surfaced to the caller, not
  silently retried (there's no way to know how much of the response the
  caller already consumed).

## Why some services hit a different host

`Region` (`region.go`) is the single owner of the gateway URLs: every base URL
in `pkg/client` — and every config writer in `internal/coding` — is derived
from `Region.Host` plus one API-root path. The roots are `Region.PaaSBaseURL`,
`CodingBaseURL`, `AnthropicBaseURL`, `ResponsesBaseURL`, `MonitorBaseURL`,
`BizBaseURL`, `AgentsBaseURL`, and `MCPServerURL(name)`; `ConsoleURL` is the web console and
`BaseURLFor(AccountType)` picks the chat root for an account type. The
package-level constants (`DefaultBaseURL`, `BigModelBaseURL`, ...) are the
same values for the two gateways.

| Service | Base URL | Why |
|---|---|---|
| Chat, Models, Images, Videos, Audio, Voice, Files, Batch, Tools, Rerank, ... | `Config.BaseURL` — `Region.PaaSBaseURL()` by default (`/api/paas/v4`). A Coding Plan key is pointed at `Region.CodingBaseURL()` (`/api/coding/paas/v4`) by setting `Config.BaseURL` (the CLI does this from the account type) | The general case. An explicit `Config.BaseURL` always wins. |
| Embeddings, Moderations | `BigModelBaseURL` (`open.bigmodel.cn`), authenticated with `Config.ChinaAPIKey` | Documented only on the China platform; `api.z.ai` answers 1211 Unknown Model for the models tried. Live-verified that a regular z.ai key authenticates identically on both platforms, so `ChinaAPIKey` falls back to `APIKey` by default. |
| Anthropic Messages | `Region.AnthropicBaseURL()` | Separate Anthropic-compatible root. |
| Responses (Codex protocol) | `Region.ResponsesBaseURL()` — `/api/v1` | Separate OpenAI Responses-protocol root, documented for the GLM Coding Plan (docs.z.ai/devpack/tool/codex). |
| Quota/usage (monitor), account (biz), account-type detection | `Region.MonitorBaseURL()`, `Region.BizBaseURL()` | Region-scoped, so a China key reaches its own region's endpoints. |
| Agents | `Region.AgentsBaseURL()` — the bare `/api` root, no `/paas/v4` | Verified live — nesting `/v1/agents` under the chat-completions base 404s. |

### Regional gateway selection (Config.Region)

Z.AI serves the same GLM model family from two regional gateways: the
international host `api.z.ai` and the China-mainland mirror
`open.bigmodel.cn`. `Config.Region` (`RegionGlobal`, the default, or
`RegionChina`) selects the host for every region-scoped service — monitor,
biz, agents, Anthropic Messages, Responses, and account-type detection — and it is the
default for `Config.BaseURL` when that is empty, so `RegionChina` also moves
chat to `open.bigmodel.cn`. An explicit `Config.BaseURL` still wins for the
chat/PaaS root. Embeddings and Moderations always use `BigModelBaseURL`.

From the CLI, `--region {global,china}` or `ZAI_REGION` (aliases: `cn`,
`bigmodel`, `west`) selects every endpoint, and `--base-url` overrides only the
chat/PaaS root. An unknown value falls back to global rather than erroring, so
a typo never blocks an unrelated command. Stored accounts record the region
their key was issued in; `accounts add` detects it, and `Account.ResolvedBaseURL`
derives the chat root from the account type and region so a key can't be
pointed at the wrong endpoint.

The China mirror hosts for monitor/biz/agents are modeled by mirroring the
`api.z.ai` path layout on `open.bigmodel.cn` and marked `NOT VERIFIED LIVE` in
`region.go` — the China platform is live-verified to serve the same OpenAPI
surface for `/models` and `/chat/completions` (see `BigModelBaseURL`), but the
monitor/biz/agents hosts on the China side have not been captured by a
cassette yet. Pin them with `ZAI_RECORD=1` if you hold an entitled China key.

## The live-verification convention

Z.AI's own SDKs and docs sometimes disagree with each other, and sometimes
with what the live API actually returns (an endpoint documented as optional
that 400s without it; an error embedded in a 200 response body; a field typed
differently across two official SDKs). Rather than trust a single source,
new services here are checked against a real API call and the interaction is
recorded as a [go-vcr](https://github.com/dnaeon/go-vcr) cassette
(`pkg/client/testdata/cassettes/`), replayed in `ModeReplayOnly` so the test
suite never touches the network.

Two files in `pkg/client` carry the live-verification work, with distinct
roles: **`live_replay_test.go`** holds the `Test*Live` tests that replay the
committed cassettes as frozen findings (entitlement gates, the
200-with-embedded-failure quirk, the single-key-across-hosts claim); it is the
running log of what's been confirmed and why it mattered. **`live_verify_test.go`**
holds the `TestVerify*` recording harness — each test SKIPS until you capture a
new success-path cassette with `ZAI_RECORD=1`, then replays it; it's the
to-do list of shapes still pending a real capture (see the
[Roadmap](roadmap.md)).

If you're extending a service, see
[Contributing § the live-verification convention](../../CONTRIBUTING.md) before
you add a new cassette.

## Extending a structured lookup table

Several types in this codebase map a small, closed set of API-defined values
to human-readable metadata via a table plus a lookup function, rather than a
chain of `if`/`switch` conditionals. The clearest example is
`pkg/client/models_catalog.go`'s `modelsCatalog`: each row is a
`ModelCatalogEntry` carrying the model's context size, output cap, pricing,
`Capabilities` (`CapText`, `CapVision`, `CapVideo`, `CapFile`, `CapThinking`,
...), and `ReasoningEfforts`. Related rows share their capability sets and
effort lists (`capsReasoning`, `capsMultimodal`, `effortsGLM53`) so they can't
drift.

```go
{
    ID: "glm-5.3-flash", Family: "GLM-5", Tier: "flash", Name: "GLM-5.3-Flash",
    Capabilities: capsMultimodal, ReasoningEfforts: effortsGLM53,
    ContextSize: ctx1M, MaxOutput: out128K, Pricing: usd(0.15, 0.03, 0.50),
    ...
},
```

When Z.AI adds a model, the change is additive (append a row) and localized;
anything not in the table still appears from `/models` with sparse data rather
than a hard failure. Lookup is by exact ID, then by dated snapshot of a
cataloged ID (`findCatalogEntry`), and `ModelDetails.HasCapability` is the one
place a capability is decided, so the CLI, TUI, and library filters agree.
The same shape shows up in `internal/coding` — the `Tools` registry
(`tools.go`, one `tool_*.go` per tool), the `MCPServers` registry (`mcp.go`),
and the `Plans` list (`plans.go`). Prefer this shape over adding another
conditional branch when you're adding a new recognized value to an existing
concept.

`modelsCatalog` is also the fix for a data-availability gap: Z.AI's `/models`
endpoint returns only the OpenAI-bare `{id, object, created, owned_by}`
shape, so without enrichment every `Context`/`Pricing`/`Capabilities` cell
renders as `-`/`0`. `ModelsService.List` runs each decoded model through
`enrichModel`, which overlays catalog fields but lets **live API values
win** when both are present — so the day Z.AI starts sending `max_context`
or `pricing` in `/models`, the real numbers take over with no code change.
Pricing and context figures are transcribed from
<https://docs.z.ai/guides/overview/pricing> and the model pages, and need
periodic manual refresh (the API gives no signal when they change); the file
header carries the "last verified" date. The `Default*Model` constants in the
same file (`DefaultModel`, `DefaultFastModel`, `DefaultVisionModel`,
`DefaultOCRModel`, `DefaultASRModel`, `DefaultTTSModel`) are what the CLI, the
TUI, and the coding-tool config writers default to, so a catalog refresh
updates all of them at once.

## Credential file safety

Both `internal/accounts` (the multi-account store) and `internal/coding` (the GLM
Coding Plan credential store, plus every third-party tool config it writes —
Claude Code, Codex, OpenCode, Crush, Factory Droid) write through `internal/atomicfile`:
a temp file in the same directory, then a rename, never a direct in-place
write. A crash or kill mid-write leaves the original file untouched instead of
truncated. A symlinked config is resolved first so the link itself survives.

Several of these are *other programs'* real config files being merged into,
not files this project owns outright, so `internal/coding` writes them with
`atomicfile.WriteWithBackup`: before the first overwrite it copies the
existing file to `<path>.zai.bak` and never replaces that backup, and
`configfile.go` edits only the keys it owns, preserving everything else. It
reads and writes JSON or TOML by file extension (Codex's `config.toml`); a
TOML rewrite drops comments, as the official helper's does, and the backup
keeps the original.
Files containing a key are created `0600`; directories `0700`.

## The TUI

`internal/tui` is a [Bubble Tea](https://github.com/charmbracelet/bubbletea)
program (v2) with one tab per subpackage (`chat`, `models`, `usage`,
`accounts`, `coding`, `media`, `tools`). Each tab is an independent
`tea.Model` with its own `Update`/`View`; the root model dispatches between
them. Long-running work (an API call, a filesystem operation) is wrapped in a
`tea.Cmd` closure, which Bubble Tea runs on its own goroutine — code called
from a `tea.Cmd` must not rely on package-level mutable state being
uncontended (for example, mutating `http.DefaultClient`);
`internal/coding/validator.go` builds a fresh `client.Client` per call for
this reason.

### Files and subpackages

| Path | Responsibility |
|---|---|
| `tui.go` | `Run(Config)`: builds the session and the root model, runs the program |
| `session.go` | `session`: state the screens share — the API client (rebuilt when the active account changes, read through an atomic pointer so `tea.Cmd` goroutines can use it), the stores, and the stored coding plan |
| `root.go` | `rootModel`: owns the chrome and layout, routes messages, handles global keys and overlays, delegates the body to the active screen |
| `screens.go` | The optional interfaces a screen can implement and the root probes for (`streamer`, `inputCapturer`, `helpProvider`, `refresher`, `chatModelSetter`/`chatModelGetter`) |
| `header.go` | The top header line and its badges |
| `tabs.go` | The `tab` enum, tab names, and the tab-bar rendering and mouse hit-testing |
| `toast.go` | Status-line toasts and `describeErr` (API error category to message and severity) |
| `overlay.go`, `helpoverlay.go`, `keys.go` | Overlay compositing and the palette/model-picker/help wiring; the help overlay; the global key map |
| `accounts/`, `chat/`, `coding/`, `media/`, `models/`, `tools/`, `usage/` | One tab each. `chat/stream.go` adapts the client's stream iterator to the message loop; `models/detail.go` and `usage/heatmap.go` hold the pure renderers |
| `formtab/` | The screen shared by the Media and Tools tabs: a row of single-input request forms, one in-flight request at a time (cancel with `esc`), and a scrollable result pane. The tabs supply only their `Form`s |
| `modelpicker/` | The chat model-picker overlay (filterable list fetched on open) |
| `palette/` | The `ctrl+p` command palette overlay |
| `uimsg/` | Shared message types so screens can talk to the root without importing it |
| `uistyle/` | The shared lipgloss style vocabulary and the light/dark palette |

### Shared infrastructure

The chrome (header, tab bar, panel, status line, help bar) is owned by the
root model in `internal/tui/root.go`; each screen renders only its body. A
few cross-cutting pieces live outside the screens so they can be shared
without an import cycle:

#### Top header

The header (`internal/tui/header.go`) is a single line: the `go-z-ai` app
badge on the left, a right-aligned cluster of context badges, and a spacer
between them that pushes the badges to the right edge. The badges give every
tab at-a-glance state:

- **account** — the active account's name (or a warn-styled `none` when no
  account is set, so the missing-credentials state is unmissable).
- **type** — `pay-as-you-go` / `coding-plan`, since that determines which
  endpoint family and which features (e.g. the Usage tab's monitor
  endpoints) are available.
- **plan** — `Global` / `China` from the coding store, surfaced because it
  silently changes the endpoint the Coding tab and coding-agent
  integrations talk to. Only shown when a plan is configured.
- **model** — the Chat tab's currently-selected model. Useful on every tab
  because users often check Usage or Models while composing a chat.

On narrow terminals the header drops badges progressively: first plan and
type (keeping account + model), then the model (account only), so the app
name and the essentials stay visible. Badges are built via
`uistyle.RenderBadge(label, value, valueStyle)` — a muted label paired with a
role-colored value.

A spacer line sits above and below the tab strip so the chrome has breathing
room; `chromeRows` accounts for both spacers when sizing the inner panel.

- **`internal/tui/uistyle`** — the lipgloss style vocabulary and a dual
  light/dark palette. The root model resolves the palette against the
  terminal's actual background (`tea.BackgroundColorMsg`) and rebuilds every
  shared style via `uistyle.SetDark`, so the whole app follows the terminal
  theme. Styles are package `var`s reassigned on theme change; screens read
  them fresh at render time.
- **`internal/tui/uimsg`** — shared message types (`Err`, `Status`, `Routed`,
  `CloseOverlay`, `OpenModelPicker`, `AccountChanged`, `PlanChanged`) so
  screens can talk to the root without importing it. `Routed` wraps an async
  result with its originating tab (`uimsg.Route` builds one from a
  `tea.Cmd`) so a result started in one tab still lands there after the user
  switches.
- **Overlays** — a single `overlay tea.Model` slot on the root, composited
  above the active screen via `lipgloss.Place` (`placeOverlay` in
  `internal/tui/overlay.go`). While open it owns keypresses; resize,
  background-color, and routed msgs still flow to the screens beneath so they
  stay correctly laid out when the overlay closes. The help overlay (`?` /
  `f1`), the command palette (`ctrl+p`), and the chat model picker
  (`ctrl+o` from the chat tab, or "Switch chat model" in the palette) are all
  root-owned overlays.

### Key bindings

Global (handled by the root before any screen sees the key):

| Key            | Action                                           |
|----------------|--------------------------------------------------|
| `tab` / `shift+tab` | next / previous tab                          |
| `?` / `f1`     | toggle the help overlay (all bindings); `?` is typed into a focused text field instead |
| `ctrl+p`       | command palette (fuzzy-search app-wide actions)  |
| `ctrl+c`       | quit (or cancel an in-flight chat stream)        |
| mouse click on tab bar | switch tab                               |
| mouse wheel    | scroll the active viewport / list                |

Per-screen bindings are surfaced in the help bar at the bottom and in the
help overlay.

### Responsive layout

The Models and Usage tabs use three width tiers:

- **≥ 100 cols** — two-column layouts (Models: table + live preview; Usage:
  quota + heatmap side by side).
- **70–99 cols** — single column, full table/heatmap rows.
- **< 70 cols** — compact: the Models table drops the CAPS column and
  shortens price headers; the Usage heatmap collapses to one-line-per-section
  summaries; the tab bar switches to a compact form of bare tab names.

Below **60×22** the root renders only a centered "Terminal too small" message
asking for at least 60×22 instead of overlapping chrome. Screens still
receive `WindowSizeMsg` and floor their own dimensions, so growing back past
the threshold resumes a correctly-laid-out app with no extra work.
