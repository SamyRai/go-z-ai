// Package fileinput resolves a user-supplied "file or URL" argument into the
// value the layout/OCR API expects: an http(s) URL is passed through verbatim,
// while a local path is read and base64-encoded. The CLI (`ocr parse`) and the
// TUI media tab both take this same kind of argument, so the rule lives here
// once instead of being copied into each.
package fileinput

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// FileOrURL returns target unchanged when it is an http(s) URL, otherwise reads
// the local file at target and returns its base64 encoding.
func FileOrURL(target string) (string, error) {
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return target, nil
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", target, err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
