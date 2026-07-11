package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Generate is always async: it returns a task handle to poll, not a video.
func TestVideosGenerate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/videos/generations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"id":"vid-1","task_status":"PROCESSING"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.Videos().Generate(context.Background(), VideoGenerationRequest{Model: "cogvideox-3", Prompt: "a dog running"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp.ID != "vid-1" || resp.TaskStatus != TaskStatusProcessing {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// A request needs either a prompt or at least one image_url.
func TestVideosGenerateValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Videos().Generate(context.Background(), VideoGenerationRequest{Prompt: "x"}); err == nil {
		t.Fatal("expected error for missing model")
	}
	if _, err := c.Videos().Generate(context.Background(), VideoGenerationRequest{Model: "cogvideox-3"}); err == nil {
		t.Fatal("expected error for missing prompt and image_url")
	}
	// image_url alone (no prompt) must be accepted for image-to-video models.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"id":"vid-2","task_status":"PROCESSING"}`)
	}))
	defer srv.Close()
	c2 := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	if _, err := c2.Videos().Generate(context.Background(), VideoGenerationRequest{Model: "viduq1-image", ImageURL: []string{"https://example.com/x.png"}}); err != nil {
		t.Fatalf("expected image_url-only request to be accepted: %v", err)
	}
}
