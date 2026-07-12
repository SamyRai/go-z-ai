package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Create posts to /batches and defaults CompletionWindow to "24h" when the
// caller doesn't set one.
func TestBatchCreateDefaultsCompletionWindow(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/batches" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"id":"batch-1","object":"batch","endpoint":"/v4/chat/completions","input_file_id":"file-1","completion_window":"24h","status":"validating","created_at":123}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	b, err := c.Batch().Create(context.Background(), BatchCreateRequest{
		InputFileID: "file-1",
		Endpoint:    BatchEndpointChatCompletions,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if b.ID != "batch-1" || b.Status != BatchStatusValidating {
		t.Fatalf("unexpected response: %+v", b)
	}
	if !strings.Contains(gotBody, `"completion_window":"24h"`) {
		t.Errorf("expected default completion_window=24h in request body, got: %s", gotBody)
	}
}

// Missing input_file_id or endpoint must fail before any request is sent.
func TestBatchCreateValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Batch().Create(context.Background(), BatchCreateRequest{Endpoint: BatchEndpointChatCompletions}); err == nil {
		t.Error("expected error for missing input_file_id")
	}
	if _, err := c.Batch().Create(context.Background(), BatchCreateRequest{InputFileID: "file-1"}); err == nil {
		t.Error("expected error for missing endpoint")
	}
}

// Retrieve fetches a batch's current state by ID.
func TestBatchRetrieve(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/batches/batch-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"id":"batch-1","object":"batch","status":"completed","output_file_id":"file-out","request_counts":{"completed":10,"failed":0,"total":10}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	b, err := c.Batch().Retrieve(context.Background(), "batch-1")
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if !b.IsTerminal() {
		t.Error("expected a completed batch to be terminal")
	}
	if b.OutputFileID != "file-out" {
		t.Errorf("expected output_file_id file-out, got %q", b.OutputFileID)
	}
	if b.RequestCounts == nil || b.RequestCounts.Total != 10 {
		t.Errorf("unexpected request_counts: %+v", b.RequestCounts)
	}
}

// A non-terminal batch (in_progress) must report IsTerminal()==false.
func TestBatchIsTerminal(t *testing.T) {
	for status, want := range map[string]bool{
		BatchStatusValidating: false,
		BatchStatusInProgress: false,
		BatchStatusFinalizing: false,
		BatchStatusCompleted:  true,
		BatchStatusFailed:     true,
		BatchStatusExpired:    true,
		BatchStatusCancelling: false,
		BatchStatusCancelled:  true,
	} {
		b := &Batch{Status: status}
		if got := b.IsTerminal(); got != want {
			t.Errorf("status %q: IsTerminal() = %v, want %v", status, got, want)
		}
	}
}

// List encodes after/limit as query parameters.
func TestBatchList(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/batches" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		writeJSON(w, http.StatusOK, `{"object":"list","data":[{"id":"batch-1"}],"has_more":true}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	list, err := c.Batch().List(context.Background(), "batch-0", 5)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Data) != 1 || !list.HasMore {
		t.Fatalf("unexpected response: %+v", list)
	}
	if gotQuery != "after=batch-0&limit=5" {
		t.Errorf("unexpected query: %q", gotQuery)
	}
}

// Cancel posts to /batches/{id}/cancel.
func TestBatchCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/batches/batch-1/cancel" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"id":"batch-1","status":"cancelling"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	b, err := c.Batch().Cancel(context.Background(), "batch-1")
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if b.Status != BatchStatusCancelling {
		t.Errorf("unexpected status: %q", b.Status)
	}
}
