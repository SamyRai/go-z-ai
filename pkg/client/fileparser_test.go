package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Create posts multipart to /files/parser/create and returns a task ID to poll.
func TestFileParserCreate(t *testing.T) {
	var gotPath, gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		writeJSON(w, http.StatusOK, `{"success":true,"message":"任务创建成功","task_id":"task-1"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.FileParser().Create(context.Background(), FileParserRequest{
		FileName: "report.pdf",
		FileData: []byte("pdf-bytes"),
		ToolType: FileParserToolLite,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotPath != "/files/parser/create" {
		t.Errorf("expected path /files/parser/create, got %q", gotPath)
	}
	if !strings.HasPrefix(gotContentType, "multipart/form-data") {
		t.Errorf("expected multipart content-type, got %q", gotContentType)
	}
	if resp.TaskID != "task-1" || !resp.Success {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// Missing file data or tool_type must fail before any request is sent.
func TestFileParserCreateValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.FileParser().Create(context.Background(), FileParserRequest{ToolType: FileParserToolLite}); err == nil {
		t.Error("expected error for missing file data")
	}
	if _, err := c.FileParser().Create(context.Background(), FileParserRequest{FileData: []byte("x")}); err == nil {
		t.Error("expected error for missing tool_type")
	}
}

// Sync posts multipart to /files/parser/sync and returns the parsed result directly.
func TestFileParserSync(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(w, http.StatusOK, `{"status":"succeeded","message":"ok","task_id":"task-2","content":"parsed text"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.FileParser().Sync(context.Background(), FileParserRequest{
		FileName: "report.pdf",
		FileData: []byte("pdf-bytes"),
		ToolType: FileParserToolPrimeSync,
		FileType: "PDF",
	})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if gotPath != "/files/parser/sync" {
		t.Errorf("expected path /files/parser/sync, got %q", gotPath)
	}
	if resp.Status != FileParserStatusSucceeded || resp.Content != "parsed text" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// Sync must reject a missing FileType before sending any request — a real
// call without it returns a malformed, silently-empty success response
// instead of a usable error (see Sync's doc comment).
func TestFileParserSyncRequiresFileType(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	_, err := c.FileParser().Sync(context.Background(), FileParserRequest{
		FileName: "report.pdf",
		FileData: []byte("pdf-bytes"),
		ToolType: FileParserToolPrimeSync,
	})
	if err == nil {
		t.Error("expected error for missing file_type")
	}
}

// Result fetches GET /files/parser/result/{taskId}/{formatType}.
func TestFileParserResult(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(w, http.StatusOK, `{"status":"succeeded","message":"ok","task_id":"task-1","content":"parsed text"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.FileParser().Result(context.Background(), "task-1", FileParserFormatText)
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if gotPath != "/files/parser/result/task-1/text" {
		t.Errorf("expected path /files/parser/result/task-1/text, got %q", gotPath)
	}
	if resp.Content != "parsed text" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// Missing task ID or format type must fail before any request is sent.
func TestFileParserResultValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.FileParser().Result(context.Background(), "", FileParserFormatText); err == nil {
		t.Error("expected error for missing task id")
	}
	if _, err := c.FileParser().Result(context.Background(), "task-1", ""); err == nil {
		t.Error("expected error for missing format type")
	}
}
