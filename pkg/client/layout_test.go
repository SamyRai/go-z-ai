package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Parse defaults Model to glm-ocr when the caller doesn't set one, and
// returns the recognized Markdown.
func TestLayoutParseDefaultsModel(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/layout_parsing" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"id":"1","md_results":"# Hello"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.Layout().Parse(context.Background(), LayoutParsingRequest{File: "aGVsbG8="})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if resp.MDResults != "# Hello" {
		t.Fatalf("unexpected md_results: %q", resp.MDResults)
	}
	if !strings.Contains(gotBody, `"model":"glm-ocr"`) {
		t.Fatalf("expected default model glm-ocr in request body, got: %s", gotBody)
	}
}

// File is required.
func TestLayoutParseValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Layout().Parse(context.Background(), LayoutParsingRequest{}); err == nil {
		t.Fatal("expected error for missing file")
	}
}
