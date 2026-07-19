// Package accounts persists multiple named Z.AI credentials and tracks which
// one is active, so the CLI can switch between accounts without hand-editing
// .env.
package accounts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// Account is one named Z.AI credential.
type Account struct {
	Name            string             `json:"name"`
	APIKey          string             `json:"api_key"`
	Type            client.AccountType `json:"type"`
	BaseURLOverride string             `json:"base_url_override,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	LastUsedAt      time.Time          `json:"last_used_at"` // zero value means never used
}

// ResolvedBaseURL returns the base URL to use for this account. If
// BaseURLOverride is set it always wins; otherwise the URL is derived from
// Type so a stored account can never point a key at the wrong endpoint.
func (a Account) ResolvedBaseURL() (string, error) {
	if a.BaseURLOverride != "" {
		return a.BaseURLOverride, nil
	}
	switch a.Type {
	case client.AccountTypeCodingPlan:
		return client.CodingBaseURL, nil
	case client.AccountTypePayAsYouGo:
		return client.ProdBaseURL, nil
	default:
		return "", fmt.Errorf("account %q has unrecognized type %q; set --base-url-override or re-add with --type", a.Name, a.Type)
	}
}

// SupportsMonitorEndpoints reports whether the coding-plan monitor endpoints
// (quota/limit, model-usage, tool-usage) are expected to work for this
// account's type.
func (a Account) SupportsMonitorEndpoints() bool {
	return a.Type == client.AccountTypeCodingPlan
}

// Store is the on-disk shape of the accounts config file.
type Store struct {
	Active   string             `json:"active,omitempty"`
	Accounts map[string]Account `json:"accounts"`
}

// ConfigPath resolves where the accounts store lives: $ZAI_ACCOUNTS_FILE if
// set (mainly for tests), else $XDG_CONFIG_HOME/zai-client/accounts.json,
// else ~/.config/zai-client/accounts.json.
func ConfigPath() (string, error) {
	if p := os.Getenv("ZAI_ACCOUNTS_FILE"); p != "" {
		return p, nil
	}

	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}

	// Directory is intentionally "zai-client" (not "go-z-ai"): the binary was
	// renamed, but existing installs keep their accounts.json on upgrade. Do
	// not "fix" this without a migration path.
	return filepath.Join(base, "zai-client", "accounts.json"), nil
}

// Load reads the accounts store. A missing file is not an error — it returns
// an empty Store so callers with no configured accounts behave identically
// to before this feature existed.
func Load() (*Store, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Store{Accounts: map[string]Account{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read accounts file %s: %w", path, err)
	}

	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to parse accounts file %s: %w", path, err)
	}
	if s.Accounts == nil {
		s.Accounts = map[string]Account{}
	}

	return &s, nil
}

// Save atomically writes the store: a temp file in the same directory
// followed by a rename, so a crash mid-write can't corrupt accounts.json.
func (s *Store) Save() error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode accounts file: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".accounts-*.json.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp accounts file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write accounts file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp accounts file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set accounts file permissions: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save accounts file: %w", err)
	}

	return nil
}

// Add registers a new account. It refuses to overwrite an existing name
// unless force is set.
func (s *Store) Add(a Account, force bool) error {
	if s.Accounts == nil {
		s.Accounts = map[string]Account{}
	}
	if _, exists := s.Accounts[a.Name]; exists && !force {
		return fmt.Errorf("account %q already exists (use --force to overwrite)", a.Name)
	}

	firstAccount := len(s.Accounts) == 0
	s.Accounts[a.Name] = a

	if firstAccount {
		s.Active = a.Name
	}

	return nil
}

// Remove deletes a stored account. Removing the active account requires
// force and clears Active rather than silently picking a replacement.
func (s *Store) Remove(name string, force bool) error {
	if _, exists := s.Accounts[name]; !exists {
		return fmt.Errorf("account %q not found", name)
	}
	if s.Active == name && !force {
		return fmt.Errorf("account %q is active; pass --yes to remove it (this clears the active account)", name)
	}

	delete(s.Accounts, name)
	if s.Active == name {
		s.Active = ""
	}

	return nil
}

// Get looks up a stored account by name.
func (s *Store) Get(name string) (Account, bool) {
	a, ok := s.Accounts[name]
	return a, ok
}

// List returns all stored accounts sorted by name for stable output.
func (s *Store) List() []Account {
	names := make([]string, 0, len(s.Accounts))
	for name := range s.Accounts {
		names = append(names, name)
	}
	sort.Strings(names)

	accounts := make([]Account, 0, len(names))
	for _, name := range names {
		accounts = append(accounts, s.Accounts[name])
	}

	return accounts
}

// Touch records that name's credentials were just used to make an actual API
// call (as opposed to admin/monitoring operations like listing or checking
// quota across every account, which should not count as "using" each one).
func (s *Store) Touch(name string) error {
	acct, found := s.Accounts[name]
	if !found {
		return fmt.Errorf("account %q not found", name)
	}
	acct.LastUsedAt = time.Now()
	s.Accounts[name] = acct
	return nil
}

// SetActive marks name as the active account.
func (s *Store) SetActive(name string) error {
	if _, exists := s.Accounts[name]; !exists {
		return fmt.Errorf("account %q not found", name)
	}
	s.Active = name
	return nil
}

// ActiveAccount returns the currently active account, if any is set and it
// still exists.
func (s *Store) ActiveAccount() (Account, bool) {
	if s.Active == "" {
		return Account{}, false
	}
	return s.Get(s.Active)
}
