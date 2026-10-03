// Package fileinput resolves a user-supplied "file or URL" argument into the
// form an API expects: an http(s) URL passes through verbatim, while a local
// path is read and base64-encoded — raw for the layout/OCR API, or as a data:
// URI for chat attachments. The CLI and the TUI take the same kind of
// argument, so the rule lives here once.
package fileinput

import (
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// IsURL reports whether target is an http(s) URL.
func IsURL(target string) bool {
	return strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://")
}

// FileOrURL returns target unchanged when it is a URL, otherwise the base64
// encoding of the local file at target.
func FileOrURL(target string) (string, error) {
	if IsURL(target) {
		return target, nil
	}
	data, err := read(target)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// URLOrDataURI returns target unchanged when it is a URL, otherwise the local
// file at target (an optional leading "@" is accepted) as a base64 data: URI
// whose MIME type comes from the extension, falling back to fallbackType.
func URLOrDataURI(target, fallbackType string) (string, error) {
	if IsURL(target) {
		return target, nil
	}
	path := strings.TrimPrefix(target, "@")
	data, err := read(path)
	if err != nil {
		return "", err
	}
	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		mimeType = fallbackType
	}
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data)), nil
}

func read(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	return data, nil
}
