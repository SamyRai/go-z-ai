package fileinput

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestFileOrURLPassesThroughHTTP(t *testing.T) {
	for _, u := range []string{"http://example.com/x.png", "https://example.com/y.pdf"} {
		got, err := FileOrURL(u)
		if err != nil {
			t.Fatalf("FileOrURL(%q): %v", u, err)
		}
		if got != u {
			t.Errorf("expected URL to pass through unchanged, got %q", got)
		}
	}
}

func TestFileOrURLBase64EncodesLocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.png")
	if err := os.WriteFile(path, []byte("hello-bytes"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := FileOrURL(path)
	if err != nil {
		t.Fatalf("FileOrURL: %v", err)
	}
	if want := base64.StdEncoding.EncodeToString([]byte("hello-bytes")); got != want {
		t.Errorf("expected base64 %q, got %q", want, got)
	}
}

func TestFileOrURLMissingFileErrors(t *testing.T) {
	if _, err := FileOrURL("/nonexistent/path/x.png"); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}
