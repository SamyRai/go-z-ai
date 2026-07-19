package client

import "fmt"

// Version is the semantic version of the go-z-ai library/CLI. It is populated
// at build time by GoReleaser ldflags
// (-X github.com/SamyRai/go-z-ai/pkg/client.version=x.y.z) and otherwise
// defaults to "dev" for `go install`/`go build` from source.
//
// Library users can read this for feature detection, logging, or telemetry.
// The CLI's --version flag surfaces the same value (main.go wires its own
// copy via ldflags for legacy reasons; both receive identical values at
// release time).
var version = "dev"

// Version returns the build-time version string (e.g. "0.2.0") or "dev" when
// built from source without ldflags.
func Version() string { return version }

// userAgent is the full User-Agent header value sent on every request. It is
// computed once at package init. RFC 7231 §5.3.3 form: product/token "("
// optional-comment ")". The comment carries the version for server-side
// identification without leaking platform details.
//
// Why this exists: Z.AI's coding-endpoint usage policy restricts access to
// "officially supported tools" and treats unidentified SDK-style clients as
// policy violations (three violations = account ban; see docs/en/coding-tools.md
// and the usage policy at https://docs.z.ai/devpack/usage-policy). Sending an
// identifying User-Agent is the minimum hygiene that distinguishes go-z-ai
// from prohibited access — it is the literal first ask in any partnership
// conversation with Z.ai ("I ship identifying headers and want to be a good
// citizen; how do I get on the supported list?").
var userAgent = fmt.Sprintf("go-z-ai/%s", version)

// UserAgent returns the User-Agent header value the client sends on every
// request. Exposed so callers (proxies, MCP servers, tests) can reuse the
// same identifier when issuing their own requests.
func UserAgent() string { return userAgent }
