package client

import (
	"strings"
	"time"
)

// This file holds the curated Z.AI model catalog — the single source of truth
// for the metadata the /models endpoint does not return (context window,
// pricing, capabilities, descriptions). It follows the codebase's documented
// "structured lookup table" pattern (see docs/en/architecture.md, "Extending a
// structured lookup table"): append a row to add a model, no new conditionals.
//
// SOURCE & VERIFICATION
//
// Pricing and context figures are transcribed from Z.AI's official pricing
// page: https://docs.z.ai/guides/overview/pricing  (verified 2026-07-19).
// All prices are USD per 1M tokens.
//
// This catalog is a SNAPSHOT. Z.AI changes pricing periodically and adds new
// models without notice, and the /models endpoint gives no signal when either
// happens. Mitigations baked in here:
//
//   - Unknown models from /models still appear in listings; they just render
//     with sparse data (`-` for unknown cells) rather than being hidden or
//     guessed at.
//   - enrichModel overlays catalog values, but LIVE API VALUES ALWAYS WIN
//     when both are present — so the day Z.AI starts sending max_context or
//     pricing in /models, the real numbers take over automatically.
//
// When refreshing: re-fetch the pricing page, update the rows whose numbers
// changed, bump the "verified" date in the comment above, and add a row for
// any model /models reports that isn't cataloged yet.

// Well-known capability codes used across the catalog and the HasCapability
// predicate. Adding a new capability is additive: append a constant and use
// it in catalog entries + HasCapability callsites.
const (
	CapText     = "text"     // ordinary chat / completions
	CapVision   = "vision"   // accepts image inputs
	CapThinking = "thinking" // hybrid reasoning mode (thinking on/off)
	CapTools    = "tools"    // function / tool calling
	CapCode     = "code"     // optimized for code generation / agentic coding
	CapOCR      = "ocr"      // document / text extraction from images
)

// ModelCatalogEntry is one curated model row. Zero-valued fields mean
// "unknown" (rendered as `-`), never "free" or "no context" — IsFree requires
// an explicit zero-cost Pricing pointer to avoid the polarity bug that
// previously made every model appear free when Pricing was nil.
type ModelCatalogEntry struct {
	// ID is the canonical model identifier as returned by /models.
	ID string
	// Aliases are alternative IDs that should resolve to this entry (e.g.
	// older names, regional variants). Matched only if ID doesn't match.
	Aliases []string
	// Family groups related variants ("GLM-5", "GLM-4.6"); Tier is a short
	// label ("flagship", "fast", "air", "vision", "ocr").
	Family, Tier string
	// Capabilities is the set of CapXxx codes this model supports.
	Capabilities []string
	// ContextSize is the max input context in tokens; MaxOutput is the max
	// generated tokens. Zero means unknown.
	ContextSize, MaxOutput int
	// Pricing is per 1M tokens (USD). Non-nil only when at least one price
	// is known. For truly free models, set a non-nil Pricing with all-zero
	// fields — IsFree reads that as free.
	Pricing *Pricing
	// Name is the human-readable display name; Description is a one-line
	// blurb. Both populate the enriched ModelDetails.
	Name, Description string
	// Created is the release epoch (Unix seconds), best-effort. Used to
	// populate ModelDetails.Created when /models omits it.
	Created int64
}

// modelsCatalog is the curated catalog. Order is loose; lookup is by exact
// ID, then alias, then prefix (see findCatalogEntry).
//
// Pricing source: https://docs.z.ai/guides/overview/pricing (verified 2026-07-19).
// Context sources: docs.z.ai model pages + z.ai blog posts for each family.
var modelsCatalog = []ModelCatalogEntry{
	// --- GLM-5 family (flagship line) ---
	{
		ID:           "glm-5.2",
		Family:       "GLM-5",
		Tier:         "flagship",
		Capabilities: []string{CapText, CapThinking, CapTools, CapCode},
		ContextSize:  128_000,
		Pricing:      &Pricing{Input: 1.40, Output: 4.40, Cached: 0.28, Unit: "USD/1M"},
		Name:         "GLM-5.2",
		Description:  "Latest flagship GLM model; top-tier reasoning, coding, and agentic performance.",
		Created:      1_781_625_600,
	},
	{
		ID:           "glm-5.1",
		Family:       "GLM-5",
		Tier:         "flagship",
		Capabilities: []string{CapText, CapThinking, CapTools, CapCode},
		ContextSize:  128_000,
		Pricing:      &Pricing{Input: 1.40, Output: 4.40, Cached: 0.28, Unit: "USD/1M"},
		Name:         "GLM-5.1",
		Description:  "Flagship GLM model; strong reasoning and tool use.",
		Created:      1_774_620_000,
	},
	{
		ID:           "glm-5",
		Family:       "GLM-5",
		Tier:         "flagship",
		Capabilities: []string{CapText, CapThinking, CapTools, CapCode},
		ContextSize:  128_000,
		Pricing:      &Pricing{Input: 1.00, Output: 3.20, Cached: 0.20, Unit: "USD/1M"},
		Name:         "GLM-5",
		Description:  "Flagship-tier model at lower cost than mainstream frontier alternatives.",
		Created:      1_770_739_200,
	},
	{
		ID:           "glm-5-turbo",
		Family:       "GLM-5",
		Tier:         "fast",
		Capabilities: []string{CapText, CapTools},
		ContextSize:  128_000,
		Pricing:      &Pricing{Input: 1.20, Output: 4.00, Unit: "USD/1M"},
		Name:         "GLM-5 Turbo",
		Description:  "Higher-throughput, lower-latency GLM-5 variant for production workloads.",
		Created:      1_773_504_000,
	},
	{
		ID:           "glm-5-flashx",
		Aliases:      []string{"glm-5-flash"},
		Family:       "GLM-5",
		Tier:         "fast",
		Capabilities: []string{CapText, CapTools},
		ContextSize:  128_000,
		Pricing:      &Pricing{Input: 0.10, Output: 0.30, Cached: 0.02, Unit: "USD/1M"},
		Name:         "GLM-5 FlashX",
		Description:  "Ultra-fast, ultra-cheap FlashX variant — good for high-volume simple tasks.",
	},
	{
		ID:           "glm-5v",
		Family:       "GLM-5",
		Tier:         "vision",
		Capabilities: []string{CapText, CapVision, CapThinking, CapTools},
		ContextSize:  128_000,
		Pricing:      &Pricing{Input: 1.40, Output: 4.40, Cached: 0.28, Unit: "USD/1M"},
		Name:         "GLM-5V",
		Description:  "Vision-capable GLM-5; accepts images alongside text for multimodal reasoning.",
	},

	// --- GLM-4.x family ---
	{
		ID:           "glm-4.7",
		Family:       "GLM-4",
		Tier:         "flagship",
		Capabilities: []string{CapText, CapThinking, CapTools, CapCode},
		ContextSize:  128_000,
		Pricing:      &Pricing{Input: 0.60, Output: 2.20, Cached: 0.11, Unit: "USD/1M"},
		Name:         "GLM-4.7",
		Description:  "Newer GLM-4 family release; competitive reasoning and coding.",
		Created:      1_766_332_800,
	},
	{
		ID:           "glm-4.6",
		Family:       "GLM-4",
		Tier:         "flagship",
		Capabilities: []string{CapText, CapThinking, CapTools, CapCode},
		ContextSize:  200_000,
		MaxOutput:    128_000,
		Pricing:      &Pricing{Input: 0.60, Output: 2.20, Cached: 0.11, Unit: "USD/1M"},
		Name:         "GLM-4.6",
		Description:  "Advanced reasoning and coding model with 200K context; built for agentic workflows.",
		Created:      1_759_276_800,
	},
	{
		ID:           "glm-4.6v",
		Family:       "GLM-4",
		Tier:         "vision",
		Capabilities: []string{CapText, CapVision, CapThinking, CapTools},
		ContextSize:  64_000,
		Pricing:      &Pricing{Input: 0.60, Output: 2.20, Cached: 0.11, Unit: "USD/1M"},
		Name:         "GLM-4.6V",
		Description:  "Vision-capable variant of GLM-4.6; multimodal understanding.",
	},
	{
		ID:           "glm-4.5",
		Family:       "GLM-4",
		Tier:         "flagship",
		Capabilities: []string{CapText, CapThinking, CapTools, CapCode},
		ContextSize:  200_000,
		MaxOutput:    96_000,
		Pricing:      &Pricing{Input: 0.60, Output: 2.20, Cached: 0.11, Unit: "USD/1M"},
		Name:         "GLM-4.5",
		Description:  "Hybrid reasoning model with 200K context; predecessor to GLM-4.6.",
		Created:      1_753_632_000,
	},
	{
		ID:           "glm-4.5-air",
		Family:       "GLM-4",
		Tier:         "air",
		Capabilities: []string{CapText, CapTools, CapCode},
		ContextSize:  200_000,
		Pricing:      &Pricing{Input: 0.20, Output: 1.10, Cached: 0.03, Unit: "USD/1M"},
		Name:         "GLM-4.5 Air",
		Description:  "Lightweight MoE variant of GLM-4.5; optimized for tool use and front-end coding at low cost.",
	},
	{
		ID:           "glm-4.5v",
		Family:       "GLM-4",
		Tier:         "vision",
		Capabilities: []string{CapText, CapVision, CapThinking, CapTools},
		ContextSize:  64_000,
		Pricing:      &Pricing{Input: 0.60, Output: 2.20, Cached: 0.11, Unit: "USD/1M"},
		Name:         "GLM-4.5V",
		Description:  "Vision-capable variant of GLM-4.5; multimodal understanding.",
	},
	{
		ID:           "glm-4.5-airx",
		Family:       "GLM-4",
		Tier:         "air",
		Capabilities: []string{CapText, CapTools},
		ContextSize:  128_000,
		Pricing:      &Pricing{Input: 0.30, Output: 1.50, Unit: "USD/1M"},
		Name:         "GLM-4.5 AirX",
		Description:  "Higher-throughput Air variant for production traffic.",
	},

	// --- OCR ---
	{
		ID:           "glm-ocr",
		Family:       "GLM",
		Tier:         "ocr",
		Capabilities: []string{CapVision, CapOCR, CapText},
		Pricing:      &Pricing{Input: 0.50, Unit: "USD/1M"}, // image-priced in practice; token rate is approximate
		Name:         "GLM-OCR",
		Description:  "Document / text extraction from images; returns structured text.",
	},
}

// findCatalogEntry resolves a model ID to its catalog entry, or returns nil if
// none matches. Match order: exact ID, then alias, then (as a last resort for
// versioned variants like `glm-4.6-2025-07-09`) the longest ID prefix that
// starts with a catalog entry's ID. Prefix matching never invents data — it
// only reuses an existing entry's metadata for an unrecognized versioned ID.
func findCatalogEntry(id string) *ModelCatalogEntry {
	if id == "" {
		return nil
	}
	for i := range modelsCatalog {
		if modelsCatalog[i].ID == id {
			return &modelsCatalog[i]
		}
	}
	for i := range modelsCatalog {
		for _, a := range modelsCatalog[i].Aliases {
			if a == id {
				return &modelsCatalog[i]
			}
		}
	}
	// Prefix fallback: a versioned snapshot ID like "glm-4.6-2025-07-09"
	// should still resolve to the glm-4.6 entry. Walk all entries and pick
	// the longest matching ID so a more-specific entry wins over a shorter
	// prefix.
	var best *ModelCatalogEntry
	bestLen := 0
	for i := range modelsCatalog {
		cid := modelsCatalog[i].ID
		if len(id) > len(cid)+1 && strings.HasPrefix(id, cid+"-") && len(cid) > bestLen {
			best = &modelsCatalog[i]
			bestLen = len(cid)
		}
	}
	return best
}

// enrichModel returns a copy of raw with catalog metadata overlaid for the
// fields raw leaves empty. Live API values always win:
//
//   - Name: kept if non-empty, else catalog Name (via CatalogName).
//   - Description: same.
//   - ContextSize: kept if non-zero, else catalog value.
//   - MaxOutput, Family, Tier, Capabilities: always taken from catalog
//     (the API doesn't send these today; if it ever does, they'll be added
//     to the wire shape and this logic updated then).
//   - Pricing: kept if non-nil, else catalog value. A live nil Pricing is
//     treated as "unknown" (not "free") — see IsFree.
//   - Created: kept if non-zero, else catalog value.
//
// If no catalog entry matches, raw is returned unchanged (no fabrication).
func enrichModel(raw ModelDetails) ModelDetails {
	entry := findCatalogEntry(raw.ID)
	if entry == nil {
		return raw
	}
	out := raw

	// Name/Description: prefer live API, fall back to catalog. Stored in the
	// dedicated Catalog* fields so future wire-decoded Name/Description are
	// distinguishable and take precedence at the presentation layer.
	out.CatalogName = entry.Name
	out.CatalogDescription = entry.Description
	if out.Name == "" && entry.Name != "" {
		out.Name = entry.Name
	}
	if out.Description == "" && entry.Description != "" {
		out.Description = entry.Description
	}

	if out.ContextSize == 0 {
		out.ContextSize = entry.ContextSize
	}
	if out.MaxOutput == 0 {
		out.MaxOutput = entry.MaxOutput
	}
	if out.Family == "" {
		out.Family = entry.Family
	}
	if out.Tier == "" {
		out.Tier = entry.Tier
	}
	if len(out.Capabilities) == 0 {
		out.Capabilities = entry.Capabilities
	}
	if out.Pricing == nil && entry.Pricing != nil {
		p := *entry.Pricing // copy so callers can't mutate the catalog
		out.Pricing = &p
	}
	if out.Created == 0 {
		out.Created = entry.Created
	}
	if out.OwnedBy == "" {
		out.OwnedBy = "z-ai"
	}
	return out
}

// HasCapability reports whether m advertises capability c (one of the CapXxx
// constants). This replaces every prior substring-based vision/text heuristic
// — there is now exactly one place a capability is determined, and the TUI /
// CLI / client all read it, so the Vision/Text filters can never drift apart.
// A model with no catalog entry has no known capabilities and reports false
// for everything (it still appears under the All filter).
func (m ModelDetails) HasCapability(c string) bool {
	for _, cap := range m.Capabilities {
		if cap == c {
			return true
		}
	}
	return false
}

// IsFree reports whether m is genuinely free — a non-nil Pricing whose input
// and output rates are both zero. A nil Pricing means "unknown" and is NOT
// treated as free, which is the fix for the polarity bug where the TUI's Free
// filter used to show every model (because Pricing was always nil in
// production) while the CLI's showed none.
func (m ModelDetails) IsFree() bool {
	return m.Pricing != nil && m.Pricing.Input == 0 && m.Pricing.Output == 0
}

// CreatedTime returns the model's release time, or the zero time if Created
// is unset. Convenience so callers don't each repeat the epoch conversion.
func (m ModelDetails) CreatedTime() time.Time {
	if m.Created <= 0 {
		return time.Time{}
	}
	return time.Unix(m.Created, 0)
}
