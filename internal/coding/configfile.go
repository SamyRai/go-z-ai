package coding

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"

	"github.com/SamyRai/go-z-ai/internal/atomicfile"
)

// These helpers edit another program's config in place: unknown fields are
// preserved (mirroring the official helper's object spreads), and every write
// goes through atomicfile.WriteWithBackup. The format follows the file
// extension — TOML for .toml, JSON otherwise. Rewriting drops comments, as
// the official helper's TOML round trip does; the one-time .zai.bak backup
// keeps the original.

// configFormat decodes and encodes one config file format.
type configFormat struct {
	unmarshal func([]byte, any) error
	marshal   func(map[string]any) ([]byte, error)
}

var (
	jsonFormat = configFormat{
		unmarshal: json.Unmarshal,
		marshal:   func(m map[string]any) ([]byte, error) { return json.MarshalIndent(m, "", "  ") },
	}
	tomlFormat = configFormat{
		unmarshal: toml.Unmarshal,
		marshal:   func(m map[string]any) ([]byte, error) { return toml.Marshal(m) },
	}
)

// formatOf picks the format for path by its extension.
func formatOf(path string) configFormat {
	if filepath.Ext(path) == ".toml" {
		return tomlFormat
	}
	return jsonFormat
}

// readConfigMap reads path as an object; a missing file is an empty object.
func readConfigMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	m := map[string]any{}
	if err := formatOf(path).unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

// writeConfigMap writes m to path (0600, atomically, keeping a one-time
// backup).
func writeConfigMap(path string, m map[string]any) error {
	data, err := formatOf(path).marshal(m)
	if err != nil {
		return err
	}
	return atomicfile.WriteWithBackup(path, data, 0o600)
}

// editConfigMap applies edit to the object stored at path and writes it back.
func editConfigMap(path string, edit func(m map[string]any)) error {
	m, err := readConfigMap(path)
	if err != nil {
		return err
	}
	edit(m)
	return writeConfigMap(path, m)
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
