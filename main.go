package main

import "github.com/SamyRai/go-z-ai/internal/cli"

// Build-time variables, populated by GoReleaser ldflags (-X main.version=...).
// Defaults are used for `go install`/`go build` from source.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.SetBuildInfo(version, commit, date)
	cli.Execute()
}
