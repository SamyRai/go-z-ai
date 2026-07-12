# Roadmap & Known Limitations

## Known bug

**`GetAccountStatus`'s insufficient-balance branch is unreachable.**
`GetAccountStatus` calls `TestBalance` first, which already intercepts the
1113 (insufficient balance) case and returns its own cleaned-up message — by
the time `GetAccountStatus` inspects the resulting error string, the "1113"
marker its own classification looks for is already gone, so that branch can
never match. Net effect: this case falls through to
`APIAccessible=false` instead of the intended `APIAccessible=true,
HasBalance=false`. Current (wrong) behavior is locked in by
`TestGetAccountStatusInsufficientBalanceViaTestBalanceShortcut` in
`pkg/client/usage_test.go` so it doesn't get accidentally "fixed" into a
third behavior — the actual fix needs a design call: either stop
`TestBalance` from transforming the message before `GetAccountStatus`
classifies it, or have `GetAccountStatus` inspect the underlying `*APIError`
via `errors.As` instead of string-matching a message it doesn't fully
control. [Contributions welcome](../CONTRIBUTING.md).

## Unverified live

A few services are implemented from Z.AI's documented OpenAPI spec but their
*success* response shape hasn't been confirmed against a real successful
call (only the request shape and error paths have) — the account used for
development has no PAYG balance/entitlement for these. If you have an
account that can reach a real success response for any of these and hit a
shape mismatch, please [open an issue](https://github.com/SamyRai/go-z-ai/issues)
or a PR with a recorded cassette:

- Agents `Invoke`'s success-path response shape (`Choices`/`Usage`)
- Embeddings and Moderations' actual output (entitlement-gated on every
  account tested so far — see [Accounts & Quota](accounts-and-quota.md))
- Voice `Clone`/`Delete` (`List` is confirmed live and working)
- Batch and Files endpoints generally

## Not implemented

- **Anthropic-compatible endpoint wrapper** — Z.AI also exposes
  `/api/anthropic`; `pkg/coding` already points third-party tools at it, but
  there's no typed Go client for it the way there is for the OpenAI-style
  `/api/paas/v4` surface.
- **Request/response logging and metrics collection** — no built-in
  instrumentation hooks yet.
- **Performance benchmarks** — deferred until a real bottleneck is measured;
  no known hot path currently justifies one (see the `golang-performance`
  guidance this repo follows: profile before optimizing).

## Deliberately not implemented

- **Assistant API** — confirmed deprecated. Z.AI's own live OpenAPI spec
  (`docs.bigmodel.cn/openapi/openapi.json`) marks every Assistant path
  `"deprecated": true`, and calling it from `api.z.ai` times out entirely
  rather than erroring. Building a client for a sunset API isn't worth the
  maintenance surface — if Z.AI ever un-deprecates it, the spec above has
  the full request/response schemas ready to transcribe.
