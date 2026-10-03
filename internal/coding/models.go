package coding

import (
	"strconv"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// The plan's models come from the client's recommended-model constants, so a
// catalog refresh updates every tool config this package writes.
var (
	// MainModel is the flagship model tools default to.
	MainModel = client.DefaultModel
	// FastModel is the fast tier (Claude Code's haiku slot, OpenCode's
	// small_model).
	FastModel = client.DefaultFastModel
)

// ModelMap maps Claude Code's haiku/sonnet/opus tiers to model IDs via the
// ANTHROPIC_DEFAULT_*_MODEL env vars.
type ModelMap struct {
	Haiku  string `json:"haiku"`
	Sonnet string `json:"sonnet"`
	Opus   string `json:"opus"`
}

// DefaultModelMap is Z.AI's documented Claude Code mapping
// (docs.z.ai/devpack/tool/claude): haiku → the fast model, sonnet/opus → the
// flagship, each with Claude Code's "[1m]" suffix when the model has a 1M
// context.
func DefaultModelMap() *ModelMap {
	return &ModelMap{Haiku: ClaudeModelID(FastModel), Sonnet: ClaudeModelID(MainModel), Opus: ClaudeModelID(MainModel)}
}

// oneMillion is the context size from which Claude Code expects the "[1m]"
// opt-in suffix.
const oneMillion = 1_000_000

// ClaudeModelID formats a model ID for Claude Code's model env vars: models
// with a context window of at least 1M tokens get the "[1m]" suffix, as the
// official helper writes them. It is a Claude Code convention — never send
// it to the API.
func ClaudeModelID(model string) string {
	if e, ok := client.CatalogEntry(model); ok && e.ContextSize >= oneMillion {
		return model + "[1m]"
	}
	return model
}

// ClaudeOptions tunes the Claude Code config. Zero values omit the
// corresponding env var.
type ClaudeOptions struct {
	ModelMap          *ModelMap // ANTHROPIC_DEFAULT_*_MODEL tier mapping
	AutoCompactWindow int       // CLAUDE_CODE_AUTO_COMPACT_WINDOW
	MaxThinkingTokens int       // MAX_THINKING_TOKENS extended-thinking budget
	MaxOutputTokens   int       // CLAUDE_CODE_MAX_OUTPUT_TOKENS
}

// DefaultClaudeOptions is the recommended Claude Code tuning: the documented
// model mapping plus an auto-compact window matching the flagship's context.
func DefaultClaudeOptions() ClaudeOptions {
	opts := ClaudeOptions{ModelMap: DefaultModelMap()}
	if e, ok := client.CatalogEntry(MainModel); ok {
		opts.AutoCompactWindow = e.ContextSize
	}
	return opts
}

// env returns the env vars these options set, keyed by name.
func (o ClaudeOptions) env() map[string]any {
	env := map[string]any{}
	if o.ModelMap != nil {
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = o.ModelMap.Haiku
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = o.ModelMap.Sonnet
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = o.ModelMap.Opus
	}
	for name, v := range map[string]int{
		"CLAUDE_CODE_AUTO_COMPACT_WINDOW": o.AutoCompactWindow,
		"MAX_THINKING_TOKENS":             o.MaxThinkingTokens,
		"CLAUDE_CODE_MAX_OUTPUT_TOKENS":   o.MaxOutputTokens,
	} {
		if v > 0 {
			env[name] = strconv.Itoa(v)
		}
	}
	return env
}

// maxOutput returns a model's output cap from the catalog, or fallback.
func maxOutput(model string, fallback int) int {
	if e, ok := client.CatalogEntry(model); ok && e.MaxOutput > 0 {
		return e.MaxOutput
	}
	return fallback
}

// displayModelName returns a model's catalog display name, or its ID.
func displayModelName(model string) string {
	if e, ok := client.CatalogEntry(model); ok && e.Name != "" {
		return e.Name
	}
	return model
}
