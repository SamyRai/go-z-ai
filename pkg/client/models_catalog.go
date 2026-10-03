package client

import (
	"slices"
	"strings"
	"time"
	"unicode"
)

// This file holds the curated Z.AI model catalog — the single source of truth
// for model knowledge the /models endpoint does not return (context window,
// pricing, capabilities, accepted reasoning-effort levels, descriptions) and
// for the recommended default models every caller (CLI, TUI, coding-tool
// config writers) uses. It follows the "structured lookup table" pattern (see
// docs/en/architecture.md): append a row to add a model, no new conditionals.
//
// SOURCE & VERIFICATION (2026-10-02)
//
// Prices: https://docs.z.ai/guides/overview/pricing (USD per 1M tokens; cached
// input storage is "limited-time free" everywhere, so it is not modeled).
// Context/output limits: docs.z.ai model pages, cross-checked against the
// Hugging Face config.json of the open-weight releases (1M = 1,048,576;
// 200K = 202,752; 128K = 131,072; 96K = 98,304). reasoning_effort levels:
// docs.z.ai chat-completion reference and the GLM-5.2/5.3 guides.
//
// The catalog is a SNAPSHOT: Z.AI changes prices and adds models without
// notice, and /models gives no signal when either happens. Unknown models
// still appear in listings with sparse data, and enrichModel lets live API
// values win over catalog values. When refreshing: re-check the pricing page,
// update changed rows, bump the date above, add rows for new models.

// Recommended models. Callers pick defaults from here rather than
// hard-coding model IDs.
const (
	// DefaultModel is the flagship text model for reasoning, coding, and
	// agentic work.
	DefaultModel = "glm-5.3"
	// DefaultFastModel is the fast, low-cost tier — also natively multimodal.
	DefaultFastModel = "glm-5.3-flash"
	// DefaultVisionModel accepts image, video, and file input.
	DefaultVisionModel = DefaultFastModel
	// DefaultOCRModel serves LayoutService.Parse (/layout_parsing).
	DefaultOCRModel = "glm-ocr"
	// DefaultASRModel serves AudioService.Transcribe.
	DefaultASRModel = "glm-asr-2512"
	// DefaultTTSModel serves AudioService.Speech.
	DefaultTTSModel = "glm-tts"
)

// Capability codes used by the catalog and ModelDetails.HasCapability.
const (
	CapText     = "text"     // chat / completions
	CapVision   = "vision"   // image input
	CapVideo    = "video"    // video input
	CapFile     = "file"     // document (file_url) input
	CapThinking = "thinking" // reasoning (reasoning_content)
	CapTools    = "tools"    // function calling
	CapCode     = "code"     // tuned for code generation / agentic coding
	CapOCR      = "ocr"      // document / text extraction
	CapAudio    = "audio"    // speech input (transcription)
)

// ModelCatalogEntry is one curated model row. Zero-valued fields mean
// "unknown" (rendered as `-`), never "free" or "no context": IsFree requires
// an explicit zero-cost Pricing.
type ModelCatalogEntry struct {
	// ID is the canonical model identifier as sent in requests.
	ID string
	// Family groups related variants ("GLM-5", "GLM-4"); Tier is a short
	// label ("flagship", "flash", "flashx", "air", "turbo", "vision", ...).
	Family, Tier string
	// Capabilities is the set of Cap* codes the model supports.
	Capabilities []string
	// ReasoningEfforts lists the ChatRequest.ReasoningEffort levels the model
	// accepts; empty means the model does not take the parameter.
	ReasoningEfforts []string
	// ContextSize is the max input context in tokens; MaxOutput the max
	// generated tokens. Zero means unknown.
	ContextSize, MaxOutput int
	// Pricing is per 1M tokens (USD); nil means unknown, an all-zero value
	// means free.
	Pricing *Pricing
	// Name is the display name; Description a one-line blurb.
	Name, Description string
	// Created is the release date (Unix seconds), best effort.
	Created int64
}

// Shared capability sets and effort lists, so related rows can't drift.
var (
	capsReasoning  = []string{CapText, CapThinking, CapTools, CapCode}
	capsMultimodal = []string{CapText, CapVision, CapVideo, CapFile, CapThinking, CapTools, CapCode}
	capsVision     = []string{CapText, CapVision, CapVideo, CapFile, CapThinking, CapTools}
	effortsGLM53   = []string{EffortLow, EffortHigh, EffortMax}
)

// Context and output limits (tokens).
const (
	ctx1M   = 1_048_576
	ctx200K = 202_752
	ctx128K = 131_072
	ctx64K  = 65_536
	out128K = 131_072
	out96K  = 98_304
	out32K  = 32_768
	out16K  = 16_384
)

func usd(in, cached, out float64) *Pricing {
	return &Pricing{Input: in, Cached: cached, Output: out, Unit: "USD/1M"}
}

// modelsCatalog is the curated catalog. Lookup is by exact ID, then by dated
// snapshot of an ID (see findCatalogEntry).
var modelsCatalog = []ModelCatalogEntry{
	// --- GLM-5 family ---
	{
		ID: "glm-5.3", Family: "GLM-5", Tier: "flagship", Name: "GLM-5.3",
		Capabilities: capsReasoning, ReasoningEfforts: effortsGLM53,
		ContextSize: ctx1M, MaxOutput: out128K, Pricing: usd(1.40, 0.26, 4.40),
		Description: "Flagship model for coding and long-horizon agentic work; always reasons (effort low/high/max).",
		Created:     1_786_665_600,
	},
	{
		ID: "glm-5.3-flash", Family: "GLM-5", Tier: "flash", Name: "GLM-5.3-Flash",
		Capabilities: capsMultimodal, ReasoningEfforts: effortsGLM53,
		ContextSize: ctx1M, MaxOutput: out128K, Pricing: usd(0.15, 0.03, 0.50),
		Description: "Fast, low-cost, natively multimodal GLM-5.3 (image, video, and file input).",
		Created:     1_787_702_400,
	},
	{
		ID: "glm-5.3-flashx", Family: "GLM-5", Tier: "flashx", Name: "GLM-5.3-FlashX",
		Capabilities: capsMultimodal, ReasoningEfforts: effortsGLM53,
		ContextSize: ctx1M, MaxOutput: out128K, Pricing: usd(0.37, 0.075, 1.25),
		Description: "Higher-throughput serving of GLM-5.3-Flash.",
		Created:     1_789_689_600,
	},
	{
		ID: "glm-5.2", Family: "GLM-5", Tier: "flagship", Name: "GLM-5.2",
		Capabilities: capsReasoning, ReasoningEfforts: AllEfforts,
		ContextSize: ctx1M, MaxOutput: out128K, Pricing: usd(1.40, 0.26, 4.40),
		Description: "Previous flagship; superseded by GLM-5.3.",
		Created:     1_781_625_600,
	},
	{
		ID: "glm-5.1", Family: "GLM-5", Tier: "flagship", Name: "GLM-5.1",
		Capabilities: capsReasoning,
		ContextSize:  ctx200K, MaxOutput: out128K, Pricing: usd(1.40, 0.26, 4.40),
		Description: "Earlier GLM-5 flagship; superseded by GLM-5.3.",
		Created:     1_774_620_000,
	},
	{
		ID: "glm-5", Family: "GLM-5", Tier: "flagship", Name: "GLM-5",
		Capabilities: capsReasoning,
		ContextSize:  ctx200K, MaxOutput: out128K, Pricing: usd(1.00, 0.20, 3.20),
		Description: "First GLM-5 release.",
		Created:     1_770_739_200,
	},
	{
		ID: "glm-5-turbo", Family: "GLM-5", Tier: "turbo", Name: "GLM-5-Turbo",
		Capabilities: []string{CapText, CapTools, CapCode},
		ContextSize:  ctx200K, MaxOutput: out128K, Pricing: usd(1.20, 0.24, 4.00),
		Description: "GLM-5 tuned for long tool-call chains in agent frameworks; no longer on the pricing page.",
		Created:     1_773_504_000,
	},
	{
		ID: "glm-5v-turbo", Family: "GLM-5", Tier: "vision", Name: "GLM-5V-Turbo",
		Capabilities: capsVision,
		ContextSize:  ctx200K, MaxOutput: out128K, Pricing: usd(1.20, 0.24, 4.00),
		Description: "GLM-5 vision model for multimodal coding agents; no longer on the pricing page.",
		Created:     1_775_001_600,
	},

	// --- GLM-4 family ---
	{
		ID: "glm-4.7", Family: "GLM-4", Tier: "flagship", Name: "GLM-4.7",
		Capabilities: capsReasoning,
		ContextSize:  ctx200K, MaxOutput: out128K, Pricing: usd(0.60, 0.11, 2.20),
		Description: "Last GLM-4 flagship; strong reasoning and coding.",
		Created:     1_766_332_800,
	},
	{
		ID: "glm-4.7-flashx", Family: "GLM-4", Tier: "flashx", Name: "GLM-4.7-FlashX",
		Capabilities: []string{CapText, CapThinking, CapTools},
		ContextSize:  ctx200K, MaxOutput: out128K, Pricing: usd(0.07, 0.01, 0.40),
		Description: "Cheapest paid text model; higher concurrency than GLM-4.7-Flash.",
		Created:     1_768_780_800,
	},
	{
		ID: "glm-4.7-flash", Family: "GLM-4", Tier: "flash", Name: "GLM-4.7-Flash",
		Capabilities: []string{CapText, CapThinking, CapTools},
		ContextSize:  ctx200K, MaxOutput: out128K, Pricing: usd(0, 0, 0),
		Description: "Free lightweight model (rate-limited).",
		Created:     1_768_780_800,
	},
	{
		ID: "glm-4.6", Family: "GLM-4", Tier: "flagship", Name: "GLM-4.6",
		Capabilities: capsReasoning,
		ContextSize:  ctx200K, MaxOutput: out128K, Pricing: usd(0.60, 0.11, 2.20),
		Description: "Reasoning and coding model with 200K context.",
		Created:     1_759_276_800,
	},
	{
		ID: "glm-4.6v", Family: "GLM-4", Tier: "vision", Name: "GLM-4.6V",
		Capabilities: capsVision,
		ContextSize:  ctx128K, MaxOutput: out32K, Pricing: usd(0.30, 0.05, 0.90),
		Description: "Vision model with native multimodal function calling.",
		Created:     1_765_152_000,
	},
	{
		ID: "glm-4.6v-flashx", Family: "GLM-4", Tier: "vision", Name: "GLM-4.6V-FlashX",
		Capabilities: capsVision,
		ContextSize:  ctx128K, MaxOutput: out32K, Pricing: usd(0.04, 0.004, 0.40),
		Description: "Low-cost GLM-4.6V variant.",
		Created:     1_765_152_000,
	},
	{
		ID: "glm-4.6v-flash", Family: "GLM-4", Tier: "vision", Name: "GLM-4.6V-Flash",
		Capabilities: capsVision,
		ContextSize:  ctx128K, MaxOutput: out32K, Pricing: usd(0, 0, 0),
		Description: "Free lightweight vision model (rate-limited).",
		Created:     1_765_152_000,
	},
	{
		ID: "glm-4.5", Family: "GLM-4", Tier: "flagship", Name: "GLM-4.5",
		Capabilities: capsReasoning,
		ContextSize:  ctx128K, MaxOutput: out96K, Pricing: usd(0.60, 0.11, 2.20),
		Description: "Hybrid reasoning model; predecessor to GLM-4.6.",
		Created:     1_753_632_000,
	},
	{
		ID: "glm-4.5-x", Family: "GLM-4", Tier: "x", Name: "GLM-4.5-X",
		Capabilities: capsReasoning,
		ContextSize:  ctx128K, MaxOutput: out96K, Pricing: usd(2.20, 0.45, 8.90),
		Description: "High-speed serving of GLM-4.5.",
		Created:     1_753_632_000,
	},
	{
		ID: "glm-4.5-air", Family: "GLM-4", Tier: "air", Name: "GLM-4.5-Air",
		Capabilities: capsReasoning,
		ContextSize:  ctx128K, MaxOutput: out96K, Pricing: usd(0.20, 0.03, 1.10),
		Description: "Lightweight MoE variant of GLM-4.5.",
		Created:     1_753_632_000,
	},
	{
		ID: "glm-4.5-airx", Family: "GLM-4", Tier: "air", Name: "GLM-4.5-AirX",
		Capabilities: capsReasoning,
		ContextSize:  ctx128K, MaxOutput: out96K, Pricing: usd(1.10, 0.22, 4.50),
		Description: "High-speed serving of GLM-4.5-Air.",
		Created:     1_753_632_000,
	},
	{
		ID: "glm-4.5-flash", Family: "GLM-4", Tier: "flash", Name: "GLM-4.5-Flash",
		Capabilities: []string{CapText, CapThinking, CapTools},
		ContextSize:  ctx128K, MaxOutput: out96K, Pricing: usd(0, 0, 0),
		Description: "Free model, retirement announced — prefer GLM-4.7-Flash.",
		Created:     1_753_632_000,
	},
	{
		ID: "glm-4.5v", Family: "GLM-4", Tier: "vision", Name: "GLM-4.5V",
		Capabilities: []string{CapText, CapVision, CapVideo, CapThinking, CapTools},
		ContextSize:  ctx64K, MaxOutput: out16K, Pricing: usd(0.60, 0.11, 1.80),
		Description: "Earlier vision model; superseded by GLM-4.6V.",
		Created:     1_754_870_400,
	},
	{
		ID: "glm-4-32b-0414-128k", Family: "GLM-4", Tier: "compact", Name: "GLM-4-32B-0414-128K",
		Capabilities: []string{CapText, CapTools},
		ContextSize:  ctx128K, MaxOutput: out16K, Pricing: usd(0.10, 0, 0.10),
		Description: "Cost-effective 32B dense model.",
		Created:     1_744_588_800,
	},

	// --- Specialized (token-priced, not served by chat completions) ---
	{
		ID: DefaultOCRModel, Family: "GLM", Tier: "ocr", Name: "GLM-OCR",
		Capabilities: []string{CapVision, CapOCR},
		Pricing:      usd(0.03, 0, 0.03),
		Description:  "Document layout parsing / OCR for images and PDFs (LayoutService).",
	},
	{
		ID: DefaultASRModel, Family: "GLM", Tier: "asr", Name: "GLM-ASR-2512",
		Capabilities: []string{CapAudio},
		Pricing:      usd(0.03, 0, 0),
		Description:  "Speech-to-text for clips up to 30 s (AudioService.Transcribe).",
		Created:      1_765_238_400,
	},
}

// findCatalogEntry resolves a model ID to its catalog entry, or nil. Match
// order: exact ID (case-insensitive — the API accepts "GLM-5.3"), then a
// dated snapshot of a cataloged ID ("glm-4.6-2025-07-09" → glm-4.6). Only
// snapshot suffixes that start with a digit match, so an unknown variant such
// as "glm-5.3-prime" never inherits glm-5.3's pricing.
func findCatalogEntry(id string) *ModelCatalogEntry {
	id = strings.ToLower(id)
	if id == "" {
		return nil
	}
	for i := range modelsCatalog {
		if modelsCatalog[i].ID == id {
			return &modelsCatalog[i]
		}
	}
	var best *ModelCatalogEntry
	for i := range modelsCatalog {
		e := &modelsCatalog[i]
		suffix, ok := strings.CutPrefix(id, e.ID+"-")
		if !ok || suffix == "" || !unicode.IsDigit(rune(suffix[0])) {
			continue
		}
		if best == nil || len(e.ID) > len(best.ID) {
			best = e
		}
	}
	return best
}

// CatalogEntry returns a copy of the catalog entry for model, if known.
func CatalogEntry(model string) (ModelCatalogEntry, bool) {
	e := findCatalogEntry(model)
	if e == nil {
		return ModelCatalogEntry{}, false
	}
	return e.clone(), true
}

// clone returns a deep copy, so callers can never mutate the catalog.
func (e ModelCatalogEntry) clone() ModelCatalogEntry {
	e.Capabilities = slices.Clone(e.Capabilities)
	e.ReasoningEfforts = slices.Clone(e.ReasoningEfforts)
	if e.Pricing != nil {
		p := *e.Pricing
		e.Pricing = &p
	}
	return e
}

// enrichModel returns raw with catalog metadata filled into the fields the
// API left empty. Live API values always win; a model without a catalog
// entry is returned unchanged (nothing is invented).
func enrichModel(raw ModelDetails) ModelDetails {
	found := findCatalogEntry(raw.ID)
	if found == nil {
		return raw
	}
	entry := found.clone()
	out := raw
	if out.Name == "" {
		out.Name = entry.Name
	}
	if out.Description == "" {
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
	if len(out.ReasoningEfforts) == 0 {
		out.ReasoningEfforts = entry.ReasoningEfforts
	}
	if out.Pricing == nil {
		out.Pricing = entry.Pricing
	}
	if out.Created == 0 {
		out.Created = entry.Created
	}
	if out.OwnedBy == "" {
		out.OwnedBy = "z-ai"
	}
	return out
}

// HasCapability reports whether m advertises capability c (a Cap* code) —
// the one place a capability is determined, so the CLI, TUI, and library
// filters can never drift apart. An uncataloged model reports false.
func (m ModelDetails) HasCapability(c string) bool {
	return slices.Contains(m.Capabilities, c)
}

// IsFree reports whether m is genuinely free: a known Pricing whose input
// and output rates are both zero. Unknown (nil) pricing is not free.
func (m ModelDetails) IsFree() bool {
	return m.Pricing != nil && m.Pricing.Input == 0 && m.Pricing.Output == 0
}

// CreatedTime returns the model's release time, or the zero time if unknown.
func (m ModelDetails) CreatedTime() time.Time {
	if m.Created <= 0 {
		return time.Time{}
	}
	return time.Unix(m.Created, 0)
}
