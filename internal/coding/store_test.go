package coding

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestStoreRoundTrip(t *testing.T) {
	s := newStoreAt(t.TempDir())
	if err := s.SetPlan(PlanGlobal); err != nil {
		t.Fatalf("SetPlan: %v", err)
	}
	if err := s.SetAPIKey("secret-key"); err != nil {
		t.Fatalf("SetAPIKey: %v", err)
	}

	c, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Plan != PlanGlobal || c.APIKey != "secret-key" || c.Lang != "en_US" {
		t.Fatalf("unexpected config: %+v", c)
	}
}

func TestStoreRevokeKeepsPlan(t *testing.T) {
	s := newStoreAt(t.TempDir())
	_ = s.SetPlan(PlanChina)
	_ = s.SetAPIKey("k")
	if err := s.RevokeAPIKey(); err != nil {
		t.Fatalf("RevokeAPIKey: %v", err)
	}
	c, _ := s.Load()
	if c.APIKey != "" {
		t.Errorf("api_key should be cleared, got %q", c.APIKey)
	}
	if c.Plan != PlanChina {
		t.Errorf("plan should be retained, got %q", c.Plan)
	}
}

// The on-disk file must use the same YAML keys as @z_ai/coding-helper so the
// Node tool and Go client interoperate, and be 0600 since it holds a key.
func TestStoreInteropFormatAndPerms(t *testing.T) {
	s := newStoreAt(t.TempDir())
	_ = s.SetAPIKey("xyz")

	info, err := os.Stat(s.Path())
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("perms = %o, want 0600", info.Mode().Perm())
	}

	raw, _ := os.ReadFile(s.Path())
	var node yaml.Node
	if err := yaml.Unmarshal(raw, &node); err != nil {
		// Fall back to plain text checks if the shape differs.
		if !strings.Contains(string(raw), "api_key: xyz") {
			t.Errorf("expected 'api_key: xyz' in YAML, got:\n%s", raw)
		}
	}
	if !strings.Contains(string(raw), "api_key:") {
		t.Errorf("expected 'api_key:' key in YAML, got:\n%s", raw)
	}
}

func TestStoreMissingFileIsFirstRun(t *testing.T) {
	s := newStoreAt(t.TempDir())
	c, err := s.Load()
	if err != nil {
		t.Fatalf("Load on first run should not error: %v", err)
	}
	if c.Lang != "en_US" {
		t.Errorf("default lang = %q, want en_US", c.Lang)
	}
	if _, err := os.Stat(filepath.Join(t.TempDir(), ".chelper")); !os.IsNotExist(err) {
		t.Error("Load should not create directories")
	}
}
