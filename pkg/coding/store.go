package coding

import (
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// StoredConfig is the ~/.chelper/config.yaml schema. Field names and YAML keys
// match @z_ai/coding-helper so the Go client and the Node helper share one file.
type StoredConfig struct {
	Lang   string `yaml:"lang,omitempty"`
	Plan   string `yaml:"plan,omitempty"`
	APIKey string `yaml:"api_key,omitempty"`
}

// Store reads and writes the chelper credential/config file under
// <home>/.chelper/config.yaml.
type Store struct {
	home string
}

// NewStore returns a Store rooted at the user's home directory.
func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &Store{home: home}, nil
}

// newStoreAt returns a Store rooted at base (for tests).
func newStoreAt(base string) *Store {
	return &Store{home: base}
}

// Path returns the absolute config file path.
func (s *Store) Path() string {
	return filepath.Join(s.home, ".chelper", "config.yaml")
}

// Load reads the config, returning defaults when absent. A missing file is not
// an error (first run).
func (s *Store) Load() (*StoredConfig, error) {
	c := &StoredConfig{Lang: "en_US"}
	data, err := os.ReadFile(s.Path())
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}
	if c.Lang == "" {
		c.Lang = "en_US"
	}
	return c, nil
}

// Save writes the config, creating the .chelper directory. The file holds an
// API key, so it is created 0600.
func (s *Store) Save(c *StoredConfig) error {
	if err := os.MkdirAll(filepath.Dir(s.Path()), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(s.Path(), data, 0o600)
}

// SetPlan records the plan choice.
func (s *Store) SetPlan(plan string) error {
	c, err := s.Load()
	if err != nil {
		return err
	}
	c.Plan = plan
	return s.Save(c)
}

// SetAPIKey records the API key.
func (s *Store) SetAPIKey(key string) error {
	c, err := s.Load()
	if err != nil {
		return err
	}
	c.APIKey = key
	return s.Save(c)
}

// RevokeAPIKey removes the stored key but keeps the plan choice (matching the
// helper's revokeApiKey, which only clears api_key).
func (s *Store) RevokeAPIKey() error {
	c, err := s.Load()
	if err != nil {
		return err
	}
	c.APIKey = ""
	return s.Save(c)
}
