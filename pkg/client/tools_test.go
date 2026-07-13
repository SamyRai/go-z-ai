package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// WebSearch posts to /web_search (on Config.BaseURL, not a separate
// gateway) and always sends search_intent explicitly, since the API
// declares it required even though false is its zero value.
func TestToolsWebSearch(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"id":"ws-1","created":123,"request_id":"req-1","search_result":[{"title":"Example","link":"https://example.com","content":"hi"}]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.Tools().WebSearch(context.Background(), WebSearchRequest{
		SearchQuery:  "golang",
		SearchEngine: SearchEnginePro,
	})
	if err != nil {
		t.Fatalf("WebSearch: %v", err)
	}
	if gotPath != "/web_search" {
		t.Errorf("expected path /web_search, got %q", gotPath)
	}
	if !strings.Contains(gotBody, `"search_intent":false`) {
		t.Errorf("expected search_intent explicitly present (required field), got: %s", gotBody)
	}
	if len(resp.SearchResult) != 1 || resp.SearchResult[0].Link != "https://example.com" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// Missing search_query or search_engine must fail before any request is sent.
func TestToolsWebSearchValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Tools().WebSearch(context.Background(), WebSearchRequest{SearchEngine: SearchEnginePro}); err == nil {
		t.Error("expected error for missing search_query")
	}
	if _, err := c.Tools().WebSearch(context.Background(), WebSearchRequest{SearchQuery: "x"}); err == nil {
		t.Error("expected error for missing search_engine")
	}
}

// WebReader posts to /reader; an explicit RetainImages=false must survive
// on the wire (not get dropped by omitempty, since the API's default is true).
func TestToolsWebReaderRetainImagesFalse(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"id":"r-1","reader_result":{"content":"body","title":"T","url":"https://example.com"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	retain := false
	resp, err := c.Tools().WebReader(context.Background(), WebReaderRequest{URL: "https://example.com", RetainImages: &retain})
	if err != nil {
		t.Fatalf("WebReader: %v", err)
	}
	if gotPath != "/reader" {
		t.Errorf("expected path /reader, got %q", gotPath)
	}
	if !strings.Contains(gotBody, `"retain_images":false`) {
		t.Errorf("expected explicit retain_images:false to survive omitempty, got: %s", gotBody)
	}
	if resp.ReaderResult == nil || resp.ReaderResult.Content != "body" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// Omitting RetainImages entirely must omit the field (letting the API's
// own default of true apply), not send a false zero value.
func TestToolsWebReaderRetainImagesOmittedByDefault(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"id":"r-2","reader_result":{"content":"body"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	if _, err := c.Tools().WebReader(context.Background(), WebReaderRequest{URL: "https://example.com"}); err != nil {
		t.Fatalf("WebReader: %v", err)
	}
	if strings.Contains(gotBody, "retain_images") {
		t.Errorf("expected retain_images omitted when unset, got: %s", gotBody)
	}
}

// Missing url must fail before any request is sent.
func TestToolsWebReaderValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Tools().WebReader(context.Background(), WebReaderRequest{}); err == nil {
		t.Error("expected error for missing url")
	}
}

// Tokenize posts to /tokenizer with a chat-shaped request (model + messages).
func TestToolsTokenize(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"id":"t-1","usage":{"prompt_tokens":5,"total_tokens":5}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.Tools().Tokenize(context.Background(), TokenizerRequest{
		Model:    "glm-4.6",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Tokenize: %v", err)
	}
	if gotPath != "/tokenizer" {
		t.Errorf("expected path /tokenizer, got %q", gotPath)
	}
	if !strings.Contains(gotBody, `"messages":[{"role":"user","content":"hello"}]`) {
		t.Errorf("expected chat-shaped messages in request body, got: %s", gotBody)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// Missing model or messages must fail before any request is sent.
func TestToolsTokenizeValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Tools().Tokenize(context.Background(), TokenizerRequest{Messages: []Message{{Role: "user", Content: "hi"}}}); err == nil {
		t.Error("expected error for missing model")
	}
	if _, err := c.Tools().Tokenize(context.Background(), TokenizerRequest{Model: "glm-4.6"}); err == nil {
		t.Error("expected error for missing messages")
	}
}
