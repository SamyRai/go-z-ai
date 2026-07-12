package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// isTextModel and isVisionModel must be logical opposites for every ID —
// they used to each hardcode their own copy of the vision-model list,
// which could silently drift apart (e.g. a new vision model added to only
// one list would make both functions return true for it). Now isTextModel
// is defined as !isVisionModel, so this is a regression test against that
// class of bug reappearing, not just a snapshot of current behavior.
func TestIsTextVisionModelAreOpposites(t *testing.T) {
	ids := []string{
		"glm-4.6", "glm-4.5-air", "glm-5", "glm-5-turbo",
		"glm-5v", "glm-4.6v", "glm-4.5v", "glm-ocr",
		"glm-5v-turbo", "some-glm-ocr-variant",
	}
	for _, id := range ids {
		if isTextModel(id) == isVisionModel(id) {
			t.Errorf("isTextModel(%q) = %v, isVisionModel(%q) = %v — must be opposites", id, isTextModel(id), id, isVisionModel(id))
		}
	}
}

func TestIsVisionModel(t *testing.T) {
	vision := []string{"glm-5v", "glm-4.6v", "glm-4.5v", "glm-ocr", "glm-5v-turbo"}
	for _, id := range vision {
		if !isVisionModel(id) {
			t.Errorf("expected %q to be a vision model", id)
		}
	}
	text := []string{"glm-4.6", "glm-4.5-air", "glm-5", "glm-5-turbo"}
	for _, id := range text {
		if isVisionModel(id) {
			t.Errorf("expected %q to not be a vision model", id)
		}
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
