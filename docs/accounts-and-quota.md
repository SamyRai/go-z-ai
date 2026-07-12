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
pay-as-you-go keys. Pass `--type` explicitly to skip the probe.

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

• weekly token window
  Usage: 41%
  Resets: 2026-07-17 09:17:18 CEST (in 5d 22h)

• monthly MCP tools quota
  Usage: 574/1000 (57%) — 426 remaining
  Resets: 2026-07-26 09:17:18 CEST (in 18d 6h)
  By tool:
    - search-prime: 470
    - web-reader: 97
    - zread: 7
```

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

## China platform key

Embeddings, Moderations, Rerank, and Voice route to `open.bigmodel.cn` —
the only platform that documents those endpoints (`api.z.ai`'s doc index
doesn't mention them). `--china-api-key` / `ZAI_CHINA_API_KEY` is optional:
confirmed live that a regular `ZAI_API_KEY` authenticates identically on both
platforms (same `/models` catalog, same billing-level errors), so the
fallback is the common case. Set a separate China key only if you hold a
distinct bigmodel.cn-only credential.

Whether you get real results from those endpoints depends on your account's
**plan entitlement**, not which key you use. A GLM Coding Plan account's
model catalog is chat-only — calling Embeddings/Moderations/Rerank/Voice with
that account returns `400 Unknown Model` (error code 1211) on either
platform. That's expected, not a bug: check `zai-client models list` to see
what's actually in your account's catalog.

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
