# Accounts & Quota

## Multiple accounts

Instead of hand-editing `.env` every time you switch keys, register named
accounts once and switch between them:

```bash
zai-client accounts add personal --api-key sk-...          # type auto-detected
zai-client accounts add work --api-key sk-... --type coding_plan

zai-client accounts list
zai-client accounts use personal        # sets the default for future commands
zai-client accounts show                # shows the active account
zai-client accounts remove work --yes
```

Accounts are stored at `$XDG_CONFIG_HOME/zai-client/accounts.json` (or
`~/.config/zai-client/accounts.json`), written atomically with `0600`
permissions.

**Type auto-detection:** `accounts add` probes the coding-plan-only
monitor/quota endpoint with a single free (no token cost) call. A successful,
well-formed response means `coding_plan`; anything else falls back to
`pay_as_you_go` — this is an inference by elimination, not a positive
confirmation, since no endpoint is known to exist that's specific to
pay-as-you-go keys. Pass `--type` explicitly to skip the probe. The probe
respects `--region`: a `glm_coding_plan_china` key probes
`open.bigmodel.cn`'s coding endpoint instead of `api.z.ai`'s, so it classifies
against the right platform (see [Regional gateways](#regional-gateways-apiza--openbigmodelcn)
below).

**Resolution order** — see [Getting Started](getting-started.md#2-authenticate)
for the full priority list across `--api-key`, `--account`, env vars, and the
active stored account.

## Quota and usage monitoring

GLM Coding Plan accounts have three independent quota windows:

| Window | Tracks | Type |
|---|---|---|
| 5-hour rolling | API request tokens | Rolling — usage in the last 5 hours, not a fixed period |
| Weekly rolling | API request tokens | Rolling — usage in the last 7 days |
| Monthly | MCP tool calls (web search, web-reader, zread) | Fixed calendar-month reset |

```bash
zai-client accounts quota                    # across all stored accounts
zai-client accounts usage --days 14          # token/tool usage heat map
zai-client accounts usage --today            # shorthand for --days 1
zai-client usage quota                       # single active account
zai-client usage check --watch               # alert when usage crosses 80%
```

Example `accounts quota` output:

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

The **Pace** line (token windows only) answers "am I burning too fast?" — it
extrapolates the window's *own* reported usage against how much of the window
has elapsed, so you find out you're on track to run out early *before* you
actually hit the wall. It's straight-line math on the numbers the API returns
(no assumption about peak/off-peak pricing).

Not every account shows every window — plan tier and account type both affect
which windows apply (`accounts show <name>` reports the account's type;
`pay_as_you_go` accounts are skipped by `accounts quota`/`accounts usage`
entirely, since the coding-plan monitor endpoint doesn't apply to them).

### Why "usage doesn't exist" is wrong

If you find older notes online (or in this repo's git history) claiming Z.AI
has no usage/quota API — that was true only of the general
`/api/paas/v4` surface tested in isolation. The coding-plan monitor endpoints
(`/monitor/usage/quota/limit`, `/monitor/usage/model-usage`,
`/monitor/usage/tool-usage`) are real, documented, and what `pkg/client`'s
`QuotaService`/`UsageService` are built on. If `accounts quota` returns
nothing for an account, check its type first (`pay_as_you_go` accounts
genuinely don't have this data) before assuming the API is broken.

## Regional gateways (api.z.ai / open.bigmodel.cn)

Z.AI serves the same GLM model family from two regional gateways: the
international host `api.z.ai` (the default) and the China-mainland mirror
`open.bigmodel.cn`. Two separate concerns decide which host a given call
lands on:

**1. Embeddings / Moderations / Rerank / Voice always route to
`open.bigmodel.cn`** — it's the only platform that documents those endpoints
(`api.z.ai`'s doc index doesn't mention them). `--china-api-key` /
`ZAI_CHINA_API_KEY` is the credential knob here; it's optional because a
regular `ZAI_API_KEY` authenticates identically on both platforms (same
`/models` catalog, same billing-level errors — live-verified), so the
fallback is the common case. Set a separate China key only if you hold a
distinct bigmodel.cn-only credential.

**2. monitor / biz / agents / detection route to `open.bigmodel.cn` when
`--region china` (or `ZAI_REGION=china`)** is set; otherwise they route to
`api.z.ai`. This is the knob a `glm_coding_plan_china` user needs so quota /
usage, account info, agents, and account-type detection land on the right
host — without it, those calls hit `api.z.ai` and a China-issued key can fail
auth or get mis-classified. `--region` does **not** change the chat base URL
(use `--base-url` for that) or the Embeddings/Moderations host. Aliases:
`cn`, `bigmodel`, `west`; an unknown value falls back to global.

Whether you get real results from the China-only services (Embeddings,
Moderations, Rerank, Voice) depends on your account's **plan entitlement**,
not which key you use. A GLM Coding Plan account's model catalog is
chat-only — calling those with that account returns `400 Unknown Model`
(error code 1211) on either platform. That's expected, not a bug: check
`zai-client models list` to see what's actually in your account's catalog.

The China mirror hosts for monitor/biz/agents/detection mirror the
`api.z.ai` path layout but are **NOT VERIFIED LIVE** here — `open.bigmodel.cn`
is live-verified to serve the same OpenAPI surface for `/models` and
`/chat/completions`, but the monitor/biz/agents paths on the China side
haven't been captured by a cassette yet. See [Roadmap](roadmap.md).

## Error codes

`APIError` (see [Error Handling](error-handling.md)) categorizes every Z.AI
error code Client code has seen. The two most common ones you'll hit around
quota:

| Code | Meaning | Retriable |
|---|---|---|
| 1113 | Insufficient balance / no resource package | No — recharge or switch account |
| 1308 | Usage limit reached for the current window | No — wait for reset |
| 1211 | Unknown model | No — usually an entitlement gate, see above |
| 1302 | Rate limit reached | Yes — the client already retries this with backoff |

See [Error Handling](error-handling.md) for the complete table and how to
branch on `APIError.Category` in your own code.
