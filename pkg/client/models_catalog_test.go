package client

import (
	"slices"
	"strings"
	"testing"
)

// enrichModel's overlay contract, table-driven. Each case checks one rule of
// the enrichment precedence (live API wins → catalog fills the rest → no
// fabrication for unknown IDs → snapshot matching).
func TestEnrichModel(t *testing.T) {
	glm46 := func() *ModelCatalogEntry {
		e := findCatalogEntry("glm-4.6")
		if e == nil {
			t.Fatal("expected glm-4.6 in catalog")
		}
		return e
	}()

	cases := []struct {
		name string
		raw  ModelDetails
		// checks run on the enriched result
		checks func(t *testing.T, m ModelDetails)
	}{
		{
			name: "bare API shape gets fully enriched",
			raw:  ModelDetails{ID: "glm-4.6", OwnedBy: "z-ai"},
			checks: func(t *testing.T, m ModelDetails) {
				if m.ContextSize != glm46.ContextSize {
					t.Errorf("ContextSize = %d, want %d (catalog)", m.ContextSize, glm46.ContextSize)
				}
				if m.MaxOutput != glm46.MaxOutput {
					t.Errorf("MaxOutput = %d, want %d (catalog)", m.MaxOutput, glm46.MaxOutput)
				}
				if m.Pricing == nil || m.Pricing.Input != glm46.Pricing.Input {
					t.Errorf("Pricing.Input = %v, want %v", m.Pricing, glm46.Pricing)
				}
				if !m.HasCapability(CapThinking) {
					t.Errorf("expected thinking capability, got %v", m.Capabilities)
				}
				if m.Name != glm46.Name {
					t.Errorf("Name = %q, want %q", m.Name, glm46.Name)
				}
				if m.Family != "GLM-4" || m.Tier != "flagship" {
					t.Errorf("Family/Tier = %q/%q", m.Family, m.Tier)
				}
			},
		},
		{
			name: "live API context overrides catalog",
			raw:  ModelDetails{ID: "glm-4.6", ContextSize: 999_999},
			checks: func(t *testing.T, m ModelDetails) {
				if m.ContextSize != 999_999 {
					t.Errorf("live ContextSize should win, got %d", m.ContextSize)
				}
				// catalog-only fields still come through
				if m.MaxOutput != glm46.MaxOutput {
					t.Errorf("MaxOutput should still come from catalog, got %d", m.MaxOutput)
				}
			},
		},
		{
			name: "live API pricing overrides catalog",
			raw:  ModelDetails{ID: "glm-4.6", Pricing: &Pricing{Input: 9.99, Output: 9.99, Unit: "USD/1M"}},
			checks: func(t *testing.T, m ModelDetails) {
				if m.Pricing.Input != 9.99 {
					t.Errorf("live Pricing should win, got %v", m.Pricing)
				}
			},
		},
		{
			name: "unknown ID passes through with no fabricated data",
			raw:  ModelDetails{ID: "no-such-model", OwnedBy: "z-ai"},
			checks: func(t *testing.T, m ModelDetails) {
				if m.ContextSize != 0 || m.Pricing != nil || len(m.Capabilities) != 0 {
					t.Errorf("unknown ID should not be enriched, got %+v", m)
				}
				if m.IsFree() {
					t.Error("unknown model with nil Pricing must not be free")
				}
			},
		},
		{
			name: "versioned snapshot ID resolves via prefix match",
			raw:  ModelDetails{ID: "glm-4.6-2025-07-09"},
			checks: func(t *testing.T, m ModelDetails) {
				if m.ContextSize != glm46.ContextSize {
					t.Errorf("expected prefix match to glm-4.6, got ContextSize %d", m.ContextSize)
				}
				if m.Name != glm46.Name {
					t.Errorf("expected name %q, got %q", glm46.Name, m.Name)
				}
			},
		},
		{
			name: "unknown variant does not inherit a base model's metadata",
			raw:  ModelDetails{ID: "glm-5.3-prime"},
			checks: func(t *testing.T, m ModelDetails) {
				if m.Pricing != nil || m.ContextSize != 0 {
					t.Errorf("glm-5.3-prime must not borrow glm-5.3 pricing/context, got %+v", m)
				}
			},
		},
		{
			name: "IDs match case-insensitively",
			raw:  ModelDetails{ID: "GLM-5.3"},
			checks: func(t *testing.T, m ModelDetails) {
				if m.Name != "GLM-5.3" || m.ContextSize == 0 {
					t.Errorf("expected GLM-5.3 to resolve to glm-5.3, got %+v", m)
				}
			},
		},
		{
			name: "empty ID is a no-op",
			raw:  ModelDetails{ID: ""},
			checks: func(t *testing.T, m ModelDetails) {
				if m.ContextSize != 0 || m.Pricing != nil {
					t.Errorf("empty ID should not be enriched, got %+v", m)
				}
			},
		},
		{
			name: "live OwnedBy is preserved, not overwritten",
			raw:  ModelDetails{ID: "glm-4.6"}, // OwnedBy empty
			checks: func(t *testing.T, m ModelDetails) {
				if m.OwnedBy != "z-ai" {
					t.Errorf("empty OwnedBy should fall back to z-ai, got %q", m.OwnedBy)
				}
			},
		},
		{
			name: "Name/Description/efforts filled from the catalog",
			raw:  ModelDetails{ID: DefaultModel},
			checks: func(t *testing.T, m ModelDetails) {
				if m.Name == "" || m.Description == "" || len(m.ReasoningEfforts) == 0 {
					t.Errorf("expected catalog name/description/efforts, got %+v", m)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.checks(t, enrichModel(tc.raw))
		})
	}
}

// IsFree must treat nil Pricing as "unknown" (not free) — this is the fix
// for the polarity bug where the TUI's Free filter used to show every model.
func TestIsFreePolarity(t *testing.T) {
	cases := []struct {
		name string
		m    ModelDetails
		want bool
	}{
		{"nil pricing is not free", ModelDetails{ID: "x"}, false},
		{"explicit zero pricing is free", ModelDetails{ID: "x", Pricing: &Pricing{}}, true},
		{"non-zero input is not free", ModelDetails{ID: "x", Pricing: &Pricing{Input: 0.01}}, false},
		{"non-zero output is not free", ModelDetails{ID: "x", Pricing: &Pricing{Output: 0.01}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.m.IsFree(); got != tc.want {
				t.Errorf("IsFree = %v, want %v", got, tc.want)
			}
		})
	}
}

// findCatalogEntry must not mutate the catalog — enrichment returns copies.
func TestEnrichDoesNotMutateCatalog(t *testing.T) {
	p := findCatalogEntry("glm-4.6").Pricing
	originalInput := p.Input

	m := enrichModel(ModelDetails{ID: "glm-4.6"})
	m.Pricing.Input = 1234.5 // mutate the enriched copy

	if p.Input != originalInput {
		t.Errorf("enrichment must copy Pricing: catalog now has Input=%v (was %v)", p.Input, originalInput)
	}
}

// HasCapability is case-sensitive and exact (no substring matching) so
// "text" never matches "contexts".
func TestHasCapabilityExact(t *testing.T) {
	m := ModelDetails{ID: "x", Capabilities: []string{CapText, CapVision}}
	if !m.HasCapability(CapText) {
		t.Error("expected text capability")
	}
	if !m.HasCapability(CapVision) {
		t.Error("expected vision capability")
	}
	if m.HasCapability(CapCode) {
		t.Error("did not expect code capability")
	}
	if m.HasCapability("TEXT") { // case-sensitive
		t.Error("capability match should be case-sensitive")
	}
}

// Every catalog row must be internally consistent: unique lowercase IDs, a
// name, known capability codes, and efforts drawn from AllEfforts. The
// recommended-model constants must point at cataloged models.
func TestCatalogIntegrity(t *testing.T) {
	known := []string{CapText, CapVision, CapVideo, CapFile, CapThinking, CapTools, CapCode, CapOCR, CapAudio}
	seen := map[string]bool{}
	for _, e := range modelsCatalog {
		if e.ID == "" || e.ID != strings.ToLower(e.ID) || seen[e.ID] {
			t.Errorf("bad or duplicate ID %q", e.ID)
		}
		seen[e.ID] = true
		if e.Name == "" || e.Family == "" || e.Pricing == nil {
			t.Errorf("%s: missing name/family/pricing", e.ID)
		}
		for _, c := range e.Capabilities {
			if !slices.Contains(known, c) {
				t.Errorf("%s: unknown capability %q", e.ID, c)
			}
		}
		for _, eff := range e.ReasoningEfforts {
			if !slices.Contains(AllEfforts, eff) {
				t.Errorf("%s: unknown effort %q", e.ID, eff)
			}
		}
	}
	for _, id := range []string{DefaultModel, DefaultFastModel, DefaultVisionModel, DefaultOCRModel} {
		if !seen[id] {
			t.Errorf("recommended model %q is not cataloged", id)
		}
	}
	if e, _ := CatalogEntry(DefaultVisionModel); !slices.Contains(e.Capabilities, CapVision) {
		t.Errorf("DefaultVisionModel %q lacks the vision capability", DefaultVisionModel)
	}
}

// Pricing.Cost bills cached prompt tokens at the cached rate.
func TestPricingCost(t *testing.T) {
	p := Pricing{Input: 1.0, Cached: 0.2, Output: 4.0}
	u := Usage{PromptTokens: 1_000_000, CompletionTokens: 500_000, PromptTokensDetails: &PromptTokensDetail{CachedTokens: 500_000}}
	if got, want := p.Cost(u), 0.5+0.1+2.0; got < want-1e-9 || got > want+1e-9 {
		t.Errorf("Cost = %v, want %v", got, want)
	}
}
