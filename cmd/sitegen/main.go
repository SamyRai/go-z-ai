// Command sitegen renders the go-z-ai static HTML site.
//
// The site is composed of:
//   - One landing page per locale, generated from each root README.
//   - Doc pages for full-docs locales (en/ru/zh) from docs/<lang>/*.md.
//   - Meta pages (CHANGELOG, CONTRIBUTING, SECURITY, CODE_OF_CONDUCT) at root.
//   - Dynamic content on the landing page: latest release, stars, contributors,
//     recent commits — pulled live from the GitHub API at build time.
//
// Output is pure static HTML/CSS/SVG with no JS framework and no web fonts.
// Designed to be deployed to GitHub Pages via .github/workflows/pages.yml.
//
// Usage:
//
//	go run ./cmd/sitegen [-out site] [-owner SamyRai] [-repo go-z-ai]
//	                     [-offline]
//
// Or via the Makefile:
//
//	make site         # generate into ./site
//	make site-serve   # generate + serve on http://localhost:8000
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/SamyRai/go-z-ai/internal/sitegen"
)

func main() {
	var (
		out      = flag.String("out", "site", "output directory")
		owner    = flag.String("owner", "SamyRai", "GitHub owner")
		repo     = flag.String("repo", "go-z-ai", "GitHub repo name")
		module   = flag.String("module", "github.com/SamyRai/go-z-ai", "Go module path")
		name     = flag.String("name", "go-z-ai", "project display name")
		tagline  = flag.String("tagline", "A Go CLI, library, and TUI for the Z.AI (Zhipu AI) API", "one-line tagline")
		rootDir  = flag.String("root", ".", "repo root (for reading markdown sources)")
		offline  = flag.Bool("offline", false, "skip GitHub API calls (no live data)")
	)
	flag.Parse()

	if err := sitegen.Run(context.Background(), sitegen.Options{
		OutDir:       *out,
		Owner:        *owner,
		Repo:         *repo,
		Module:       *module,
		Name:         *name,
		Tagline:      *tagline,
		RootDir:      *rootDir,
		SkipNetwork:  *offline,
	}); err != nil {
		log.Fatalf("sitegen: %v", err)
	}
	wd, _ := os.Getwd()
	log.Printf("site generated → %s/%s", wd, *out)
}
