package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A bare http(s):// URL passes through unchanged.
func TestResolveImageArgURL(t *testing.T) {
	got, err := resolveImageArg("https://example.com/cat.png")
	if err != nil {
		t.Fatalf("resolveImageArg: %v", err)
	}
	if got != "https://example.com/cat.png" {
		t.Errorf("expected URL to pass through unchanged, got %q", got)
	}
}

// An @path argument reads the local file and base64-encodes it as a data:
// URI, guessing the MIME type from the extension.
func TestResolveImageArgLocalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "photo.png")
	if err := os.WriteFile(path, []byte("fake-png-bytes"), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	got, err := resolveImageArg("@" + path)
	if err != nil {
		t.Fatalf("resolveImageArg: %v", err)
	}
	if !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Errorf("expected a data:image/png;base64 URI, got %q", got)
	}
	if strings.Contains(got, "fake-png-bytes") {
		t.Error("expected the bytes to be base64-encoded, not embedded raw")
	}
}

// An unrecognized extension falls back to image/jpeg (the API's default).
func TestResolveImageArgUnknownExtensionFallsBackToJPEG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "photo.unknownext")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	got, err := resolveImageArg("@" + path)
	if err != nil {
		t.Fatalf("resolveImageArg: %v", err)
	}
	if !strings.HasPrefix(got, "data:image/jpeg;base64,") {
		t.Errorf("expected fallback to image/jpeg, got %q", got)
	}
}

// A missing local file must error, not silently produce an empty image.
func TestResolveImageArgMissingFile(t *testing.T) {
	if _, err := resolveImageArg("@/nonexistent/path/photo.png"); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}
