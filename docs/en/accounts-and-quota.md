# Accounts & Quota

## Multiple accounts

Instead of hand-editing `.env` every time you switch keys, register named
accounts once and switch between them:

```bash
go-z-ai accounts add personal --api-key sk-...          # type and region auto-detected
go-z-ai accounts add work --api-key sk-... --type coding_plan --region china

go-z-ai accounts list
go-z-ai accounts use personal        # sets the default for future commands
go-z-ai accounts show                # shows the active account
go-z-ai accounts remove work --yes
```

Accounts are stored at `$XDG_CONFIG_HOME/zai-client/accounts.json` (or
`~/.config/zai-client/accounts.json`), written atomically with `0600`
permissions. (The directory is intentionally `zai-client`, not `go-z-ai`, so
existing installs keep their `accounts.json` on upgrade from the pre-rename
name.)

**Type and region auto-detection:** `accounts add` probes the
coding-plan-only monitor/quota endpoint, a free call with no token cost: first
on the `--region` you gave (global by default), then on the other gateway,
since a China-issued key answers only on `open.bigmodel.cn`. A well-formed
quota response means `coding_plan` on that region. If neither gateway
identifies a subscription but at least one answered, the key is
`pay_as_you_go` — an inference by elimination, not a positive confirmation,
since no endpoint is specific to pay-as-you-go keys. If neither gateway can be
reached, `add` fails rather than guessing. Pass `--type` to skip the probe;
the region then comes from `--region`. `go-z-ai account detect` runs the same
probe for the current key without storing anything.

**Resolution order** — see [Getting Started](getting-started.md#2-authenticate)
for the full priority list across `--api-key`, `--account`, env vars, and the
active stored account.

## Checking a key

```bash
go-z-ai account detect          # type (coding plan / pay-as-you-go), region, plan tier — free
go-z-ai account status          # does the key work and can it spend right now?
go-z-ai account status --watch 5m
go-z-ai account balance         # pay-as-you-go wallet
go-z-ai account subscriptions   # GLM Coding Plan subscriptions
```

`account status` is a free quota check for a coding-plan key. Z.AI has no
balance call for pay-as-you-go keys, so for those it sends one billed
one-token completion — and `--watch` repeats it at every interval, so keep
the interval long. `account balance` and `account subscriptions` read the
biz API, which Z.AI does not document; they are **NOT VERIFIED LIVE** (see
[Roadmap](roadmap.md)). Balances are in the region's billing currency (USD on
api.z.ai, CNY on open.bigmodel.cn).

## Quota and usage monitoring

GLM Coding Plan quota comes in windows, and which ones an account has depends
on its plan generation:

| Plan | Window | Meters |
|---|---|---|
| Credit-based (sold since 2026-07-30) | 5-hour, weekly (`CREDIT_LIMIT`) | Credits. Model usage and MCP tool calls draw from the same credits; Web Search, Web Reader, and Zread cost 1.2 credits per call ([docs](https://docs.z.ai/devpack/overview)) |
| Legacy token plans | 5-hour, weekly (`TOKENS_LIMIT`) | Tokens, reported as a percentage only |
| Legacy token plans | monthly (`TIME_LIMIT`) | MCP tool calls, with a per-tool breakdown |

The 5-hour window refreshes five hours after consumption; the API reports
each window's reset time. Treat the window types as an open set: the client
renders an unknown type generically instead of failing.

```bash
go-z-ai accounts quota                    # across all stored accounts
go-z-ai accounts usage --days 14          # token/tool usage heat map
go-z-ai accounts usage --today            # shorthand for --days 1
go-z-ai usage quota                       # the current key only
go-z-ai account status [--watch 5m]       # can the key spend right now?
```

Example `usage quota` output (`accounts quota` prints the same per account):

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

The **pace** line answers "am I burning too fast?" — it extrapolates the
window's *own* reported usage against how much of the window has elapsed, so
you find out you're on track to run out early *before* you actually hit the
wall. It's straight-line math on the numbers the API returns.

### Peak hours

Peak hours are Monday–Friday 14:00–18:00 UTC+8 for every user, whatever
their time zone ([docs](https://docs.z.ai/devpack/overview)). The quota API
applies peak pricing without exposing it, so the CLI and TUI add a notice
while peak is on. What peak costs depends on the plan:

- **Credit-based plans** bill peak usage at the standard credit rate and
  off-peak usage at 50%.
- **Legacy token plans** count GLM-5.3 at 3× during peak (1× off-peak) and
  GLM-5.3-Flash at 1.2× (0.4× off-peak).

Announced promotions bill every hour at the off-peak rate; the current one
runs 2026-09-25 to 2026-10-07 (UTC+8). These rules live in
`internal/usageview/peak.go`, as of 2026-10-02.

### Timezones

All times render in **your local timezone**. Reset times are absolute
timestamps, so they need no conversion. The usage endpoints, though, exchange
zoneless time strings in the server's own wall clock, **UTC+8**, so the
**usage heat map** (`accounts usage`) converts its windows and bucket spans
to your local time, with a one-line note naming the server zone when it
differs. The requested range is also framed in the server's zone before
sending, so `--today` fetches *your* today, not a slice shifted by the zone
difference.

If a live capture ever shows a different server zone for your account, override
the assumption with `--monitor-timezone` or `ZAI_MONITOR_TIMEZONE` (accepts IANA
names like `Asia/Shanghai`, `UTC`, or offsets like `+8`).

Not every account shows every window — plan tier and account type both affect
which windows apply (`accounts show <name>` reports the account's type;
`pay_as_you_go` accounts are skipped by `accounts quota`/`accounts usage`
entirely, since the coding-plan monitor endpoint doesn't apply to them).

### Why "usage doesn't exist" is wrong

If you find older notes online (or in this repo's git history) claiming Z.AI
has no usage/quota API — that was true only of the general
`/api/paas/v4` surface tested in isolation. The coding-plan monitor endpoints
(`/monitor/usage/quota/limit`, `/monitor/usage/model-usage`,
`/monitor/usage/tool-usage`) are real — they are what Z.AI's own usage plugin
calls — and what `pkg/client`'s `QuotaService` is built on. If `accounts quota` returns
nothing for an account, check its type first (`pay_as_you_go` accounts
genuinely don't have this data) before assuming the API is broken.

## Regional gateways (api.z.ai / open.bigmodel.cn)

Z.AI serves the same GLM model family from two regional gateways: the
international host `api.z.ai` (the default) and the China-mainland mirror
`open.bigmodel.cn`. Two separate concerns decide which host a given call
lands on:

**1. Embeddings and Moderations always route to `open.bigmodel.cn`** —
they're the only services pinned to the China host in code
(`pkg/client/embeddings.go`, `pkg/client/moderations.go` both target
`BigModelBaseURL`). `--china-api-key` /
`ZAI_CHINA_API_KEY` is the credential knob here; it's optional because a
regular `ZAI_API_KEY` authenticates identically on both platforms (same
`/models` catalog, same billing-level errors — live-verified), so the
fallback is the common case. Set a separate China key only if you hold a
distinct bigmodel.cn-only credential. Rerank and Voice use the default
`--base-url` (`api.z.ai` by default) — they're documented only on the
China platform but the client doesn't force-route them.

**2. Everything else follows `--region china` (or `ZAI_REGION=china`)**:
chat, the Anthropic and Responses APIs, quota/usage, account (balance,
subscriptions), agents, and account-type detection all move to
`open.bigmodel.cn`; otherwise they use `api.z.ai`. This is the knob a China
key needs — without it, those calls hit `api.z.ai`, where a China-issued key
can fail auth or get mis-classified. A stored account remembers its region,
so `accounts use` switches it too. `--base-url` still overrides the chat/PaaS
root alone (for a Coding Plan account it is the plan's coding root), and the
Embeddings/Moderations host never changes. Aliases: `cn`, `bigmodel`, `west`;
an unknown value falls back to global.

Whether you get real results from the China-documented services (Embeddings,
Moderations, Rerank, Voice) depends on your account's **plan entitlement**,
not which key you use. A GLM Coding Plan account's model catalog is
chat-only — calling those with that account returns `400 Unknown Model`
(error code 1211) on either platform. That's expected, not a bug: check
`go-z-ai models list` to see what's actually in your account's catalog.

The China mirror hosts for monitor/biz/agents/detection mirror the
`api.z.ai` path layout but are **NOT VERIFIED LIVE** here — `open.bigmodel.cn`
is live-verified to serve the same OpenAPI surface for `/models` and
`/chat/completions`, but the monitor/biz/agents paths on the China side
haven't been captured by a cassette yet. See [Roadmap](roadmap.md).

## Error codes

`APIError` (see [Error Handling](error-handling.md)) categorizes every Z.AI
error code the client knows. The ones you'll most often hit around quota:

| Code | Meaning | Retriable |
|---|---|---|
| 1113 | Insufficient balance / no resource package | No — recharge or switch account |
| 1308 | Usage limit reached for the current window | No — wait for reset |
| 1316 / 1317 | 5-hour / 7-day window used up and no balance for extra usage | No — wait for reset or recharge |
| 1211 | Unknown model | No — usually an entitlement gate, see above |
| 1302 | Rate limit reached | Yes — the client already retries this with backoff |

See [Error Handling](error-handling.md) for the complete table and how to
branch on `APIError.Category` in your own code.
