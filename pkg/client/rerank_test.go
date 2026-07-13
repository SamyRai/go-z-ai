package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Create posts to /rerank and defaults Model to "rerank" when unset.
func TestRerankCreateDefaultsModel(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"id":"rr-1","created":123,"request_id":"req-1","results":[{"document":"doc A","index":0,"relevance_score":0.98}],"usage":{"prompt_tokens":10,"total_tokens":10}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.Rerank().Create(context.Background(), RerankRequest{
		Query:     "query text",
		Documents: []string{"doc A", "doc B"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotPath != "/rerank" {
		t.Errorf("expected path /rerank, got %q", gotPath)
	}
	if !strings.Contains(gotBody, `"model":"rerank"`) {
		t.Errorf("expected default model=rerank in request body, got: %s", gotBody)
	}
	if len(resp.Results) != 1 || resp.Results[0].RelevanceScore != 0.98 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// Missing query or documents must fail before any request is sent.
func TestRerankCreateValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Rerank().Create(context.Background(), RerankRequest{Documents: []string{"a"}}); err == nil {
		t.Error("expected error for missing query")
	}
	if _, err := c.Rerank().Create(context.Background(), RerankRequest{Query: "q"}); err == nil {
		t.Error("expected error for missing documents")
	}
}
