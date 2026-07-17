# Roadmap & Known Limitations

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
- **Anthropic Messages** (`AnthropicService`, `POST /api/anthropic/v1/messages`)
  — routing, the `anthropic-version` header, and Bearer auth are confirmed
  reaching the live endpoint (a bogus key returns a clean HTTP 401, not a
  404/timeout), but the *success*-path response body (`content` blocks,
  `stop_reason`, `usage`) is modeled from Anthropic's documented shape and not
  yet parsed from a real entitled call here. In particular, whether GLM returns
  reasoning as Anthropic `thinking` blocks or in an OpenAI-style
  `reasoning_content` field (the claude-code-router#1133 case) is unconfirmed —
  `AnthropicResponse.Thinking()` reads both, but which one the endpoint actually
  populates needs a live capture.
- **Tool-schema compatibility rewriting** — the set of JSON-Schema constructs
  GLM's parser rejects with HTTP 500 (`anyOf`/`oneOf`/`allOf`/`$ref`) is drawn
  from community bug reports (e.g.
  [claude-code-router#1474](https://github.com/musistudio/claude-code-router/issues/1474)),
  not yet reproduced against a live account here. The rewrite itself is
  fully unit-tested and inert on already-flat schemas; if you can record a
  live cassette that pins down exactly which constructs 500 (and which the
  flattened output makes pass), that would upgrade this from "documented
  behavior" to "live-verified." See `pkg/client/toolschema.go`.

## Not implemented

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
