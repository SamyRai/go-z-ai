package coding

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/SamyRai/go-z-ai/internal/atomicfile"
)

// These helpers edit another program's JSON config in place: unknown fields
// are preserved (mirroring the official helper's object spreads), and every
// write goes through atomicfile.WriteWithBackup.

// readJSONMap reads path as a JSON object; a missing file is an empty object.
func readJSONMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	m := map[string]any{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

// writeJSONMap writes m to path (0600, atomically, keeping a one-time backup).
func writeJSONMap(path string, m map[string]any) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.WriteWithBackup(path, data, 0o600)
}

// editJSONMap applies edit to the object stored at path and writes it back.
func editJSONMap(path string, edit func(m map[string]any)) error {
	m, err := readJSONMap(path)
	if err != nil {
		return err
	}
	edit(m)
	return writeJSONMap(path, m)
}

// objectField returns the child object at key, creating an empty one if it
// is missing or not an object.
func objectField(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	child := map[string]any{}
	m[key] = child
	return child
}

// deleteFromObject removes names from the child object at key, dropping the
// child entirely once it is empty.
func deleteFromObject(m map[string]any, key string, names ...string) {
	child, ok := m[key].(map[string]any)
	if !ok {
		return
	}
	for _, n := range names {
		delete(child, n)
	}
	if len(child) == 0 {
		delete(m, key)
	}
}
