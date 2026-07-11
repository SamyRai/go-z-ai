package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A synchronous generation request returns the image URL(s) from the
// response body.
func TestImagesGenerate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/generations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"created":1,"data":[{"url":"https://example.com/img.png"}]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.Images().Generate(context.Background(), ImageGenerationRequest{Model: "glm-image", Prompt: "a cat"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].URL != "https://example.com/img.png" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// GenerateAsync hits the /async prefix and returns a task handle, not a
// finished image.
func TestImagesGenerateAsync(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/async/images/generations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"id":"task-1","task_status":"PROCESSING"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.Images().GenerateAsync(context.Background(), ImageGenerationRequest{Model: "glm-image", Prompt: "a cat"})
	if err != nil {
		t.Fatalf("GenerateAsync: %v", err)
	}
	if resp.ID != "task-1" || resp.TaskStatus != TaskStatusProcessing {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// Missing required fields must fail before any request is sent.
func TestImagesGenerateValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Images().Generate(context.Background(), ImageGenerationRequest{Prompt: "x"}); err == nil {
		t.Fatal("expected error for missing model")
	}
	if _, err := c.Images().Generate(context.Background(), ImageGenerationRequest{Model: "glm-image"}); err == nil {
		t.Fatal("expected error for missing prompt")
	}
}

// A non-200 response must surface as a structured APIError, proving Generate
// goes through the shared retry/error-parsing transport, not a bare client.
func TestImagesGenerateAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusBadRequest, `{"error":{"code":"-1","message":"bad prompt"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	_, err := c.Images().Generate(context.Background(), ImageGenerationRequest{Model: "glm-image", Prompt: "x"})
	if _, ok := errors.AsType[*APIError](err); !ok {
		t.Fatalf("expected *APIError, got %T (%v)", err, err)
	}
}
