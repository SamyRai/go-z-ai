package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Upload sends the file bytes and purpose as multipart fields and returns
// the resulting FileObject.
func TestFilesUpload(t *testing.T) {
	var gotPurpose, gotFileName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if fh := r.MultipartForm.File["file"]; len(fh) == 1 {
			gotFileName = fh[0].Filename
		}
		gotPurpose = r.FormValue("purpose")
		writeJSON(w, http.StatusOK, `{"id":"file-1","bytes":4,"created_at":123,"filename":"batch.jsonl","object":"file","purpose":"batch","status":"processed"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 3}) // retries must not apply to uploads
	f, err := c.Files().Upload(context.Background(), "batch.jsonl", []byte("data"), FilePurposeBatch)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if f.ID != "file-1" || f.Status != "processed" {
		t.Fatalf("unexpected response: %+v", f)
	}
	if gotFileName != "batch.jsonl" {
		t.Errorf("expected filename batch.jsonl, got %q", gotFileName)
	}
	if gotPurpose != "batch" {
		t.Errorf("expected purpose batch, got %q", gotPurpose)
	}
}

// Empty data, empty filename, and empty purpose must all fail validation
// before any request is sent.
func TestFilesUploadValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Files().Upload(context.Background(), "f.txt", nil, FilePurposeBatch); err == nil {
		t.Error("expected error for empty data")
	}
	if _, err := c.Files().Upload(context.Background(), "", []byte("x"), FilePurposeBatch); err == nil {
		t.Error("expected error for empty filename")
	}
	if _, err := c.Files().Upload(context.Background(), "f.txt", []byte("x"), ""); err == nil {
		t.Error("expected error for empty purpose")
	}
}

// List encodes the purpose filter as a query parameter and returns the
// paginated file list.
func TestFilesList(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotQuery = r.URL.RawQuery
		writeJSON(w, http.StatusOK, `{"object":"list","data":[{"id":"file-1","purpose":"batch"}],"has_more":false}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	list, err := c.Files().List(context.Background(), FilePurposeBatch)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Data) != 1 || list.Data[0].ID != "file-1" {
		t.Fatalf("unexpected response: %+v", list)
	}
	if gotQuery != "purpose=batch" {
		t.Errorf("expected query purpose=batch, got %q", gotQuery)
	}
}

// Delete hits DELETE /files/{id} and returns the deletion confirmation.
func TestFilesDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files/file-1" || r.Method != http.MethodDelete {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"id":"file-1","deleted":true,"object":"file"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.Files().Delete(context.Background(), "file-1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !resp.Deleted {
		t.Errorf("expected deleted=true, got %+v", resp)
	}
}

// Content downloads the raw file bytes from /files/{id}/content.
func TestFilesContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files/file-1/content" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte("raw file bytes"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	data, err := c.Files().Content(context.Background(), "file-1")
	if err != nil {
		t.Fatalf("Content: %v", err)
	}
	if string(data) != "raw file bytes" {
		t.Errorf("unexpected content: %q", data)
	}
}
