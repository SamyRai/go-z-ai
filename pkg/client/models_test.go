package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// HasCapability is now the single source of truth for whether a model is
// vision-capable, text-capable, etc. — replacing the two parallel substring
// heuristics (visionModelMarkers in the client, a different list in the TUI)
// that used to drift apart. This test pins the catalog's capability claims so
// a future catalog edit that drops CapVision from a vision model fails loudly.
func TestCatalogCapabilities(t *testing.T) {
	vision := []string{"glm-5v", "glm-4.6v", "glm-4.5v", "glm-ocr"}
	for _, id := range vision {
		m := enrichModel(ModelDetails{ID: id})
		if !m.HasCapability(CapVision) {
			t.Errorf("expected %q to be vision-capable after enrichment, capabilities=%v", id, m.Capabilities)
		}
	}
	text := []string{"glm-4.6", "glm-4.5-air", "glm-5", "glm-5-turbo", "glm-5.2"}
	for _, id := range text {
		m := enrichModel(ModelDetails{ID: id})
		if !m.HasCapability(CapText) {
			t.Errorf("expected %q to be text-capable after enrichment, capabilities=%v", id, m.Capabilities)
		}
		if m.HasCapability(CapVision) {
			t.Errorf("expected %q to NOT be vision-capable, capabilities=%v", id, m.Capabilities)
		}
	}
}

// A model with no catalog entry must not crash, must not fabricate
// capabilities, and must not be reported as free — the polarity bug that
// previously made every unknown model appear free.
func TestUncatalogedModelIsSafe(t *testing.T) {
	m := enrichModel(ModelDetails{ID: "totally-unknown-model"})
	if m.HasCapability(CapVision) || m.HasCapability(CapText) {
		t.Errorf("uncataloged model should have no capabilities, got %v", m.Capabilities)
	}
	if m.IsFree() {
		t.Error("uncataloged model with nil Pricing must not be reported as free")
	}
	if m.ContextSize != 0 || m.Pricing != nil {
		t.Errorf("uncataloged model should have no fabricated data, got context=%d pricing=%v", m.ContextSize, m.Pricing)
	}
}

const modelsListBody = `{"object":"list","data":[
	{"id":"glm-4.6","owned_by":"z-ai"},
	{"id":"glm-4.5v","owned_by":"z-ai"},
	{"id":"glm-free-model","owned_by":"z-ai","pricing":{"prompt":0,"completion":0}},
	{"id":"glm-4.7","owned_by":"z-ai","pricing":{"prompt":0.01,"completion":0.02}}
]}`

// List fetches and caches models; a second call must not hit the network again.
func TestModelsListCaches(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		writeJSON(w, http.StatusOK, modelsListBody)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	if _, err := c.Models().List(context.Background()); err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, err := c.Models().List(context.Background()); err != nil {
		t.Fatalf("List (cached): %v", err)
	}
	if requests != 1 {
		t.Errorf("expected 1 request (second List should hit cache), got %d", requests)
	}
}

// RefreshCache clears the cache so the next List re-fetches.
func TestModelsRefreshCache(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		writeJSON(w, http.StatusOK, modelsListBody)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	if _, err := c.Models().List(context.Background()); err != nil {
		t.Fatalf("List: %v", err)
	}
	if err := c.Models().RefreshCache(context.Background()); err != nil {
		t.Fatalf("RefreshCache: %v", err)
	}
	if requests != 2 {
		t.Errorf("expected 2 requests (RefreshCache forces a re-fetch), got %d", requests)
	}
}

// Get returns the matching model, or an error when not found or modelID is empty.
func TestModelsGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, modelsListBody)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	m, err := c.Models().Get(context.Background(), "glm-4.6")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if m.ID != "glm-4.6" {
		t.Errorf("unexpected model: %+v", m)
	}

	if _, err := c.Models().Get(context.Background(), "does-not-exist"); err == nil {
		t.Error("expected error for unknown model ID")
	}
	if _, err := c.Models().Get(context.Background(), ""); err == nil {
		t.Error("expected error for empty model ID")
	}
}

// GetTextModels/GetVisionModels/GetFreeModels filter the full catalog correctly.
func TestModelsFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, modelsListBody)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})

	text, err := c.Models().GetTextModels(context.Background())
	if err != nil {
		t.Fatalf("GetTextModels: %v", err)
	}
	if len(text) != 3 {
		t.Errorf("expected 3 text models, got %d: %+v", len(text), text)
	}

	vision, err := c.Models().GetVisionModels(context.Background())
	if err != nil {
		t.Fatalf("GetVisionModels: %v", err)
	}
	if len(vision) != 1 || vision[0].ID != "glm-4.5v" {
		t.Errorf("expected 1 vision model (glm-4.5v), got %+v", vision)
	}

	free, err := c.Models().GetFreeModels(context.Background())
	if err != nil {
		t.Fatalf("GetFreeModels: %v", err)
	}
	if len(free) != 1 || free[0].ID != "glm-free-model" {
		t.Errorf("expected 1 free model (glm-free-model), got %+v", free)
	}
}
