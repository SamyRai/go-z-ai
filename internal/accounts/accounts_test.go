package accounts

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// isolateStore points ZAI_ACCOUNTS_FILE at a fresh temp path so each test gets
// an empty store and never touches the developer's real ~/.config file.
func isolateStore(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "accounts.json")
	t.Setenv("ZAI_ACCOUNTS_FILE", path)
	return path
}

func acct(name string, typ client.AccountType) Account {
	return Account{Name: name, APIKey: "sk-" + name, Type: typ, CreatedAt: time.Now()}
}

// Load on a missing file returns an empty, usable store rather than erroring.
func TestLoadMissingFileIsEmpty(t *testing.T) {
	isolateStore(t)
	s, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(s.Accounts) != 0 || s.Active != "" {
		t.Errorf("expected empty store, got %+v", s)
	}
}

// The first account added becomes active automatically; a second does not.
func TestAddFirstAccountBecomesActive(t *testing.T) {
	isolateStore(t)
	s, _ := Load()
	if err := s.Add(acct("one", client.AccountTypePayAsYouGo), false); err != nil {
		t.Fatalf("Add one: %v", err)
	}
	if s.Active != "one" {
		t.Errorf("expected first account to become active, got %q", s.Active)
	}
	if err := s.Add(acct("two", client.AccountTypePayAsYouGo), false); err != nil {
		t.Fatalf("Add two: %v", err)
	}
	if s.Active != "one" {
		t.Errorf("adding a second account must not change the active one, got %q", s.Active)
	}
}

// Add refuses to overwrite an existing name unless force is set.
func TestAddOverwriteGuard(t *testing.T) {
	isolateStore(t)
	s, _ := Load()
	_ = s.Add(acct("dup", client.AccountTypePayAsYouGo), false)
	if err := s.Add(acct("dup", client.AccountTypeCodingPlan), false); err == nil {
		t.Error("expected an error overwriting an existing account without force")
	}
	if err := s.Add(acct("dup", client.AccountTypeCodingPlan), true); err != nil {
		t.Errorf("force overwrite should succeed: %v", err)
	}
	if got, _ := s.Get("dup"); got.Type != client.AccountTypeCodingPlan {
		t.Errorf("expected forced overwrite to update the account, got type %q", got.Type)
	}
}

// Removing the active account requires force and clears Active rather than
// silently promoting another account.
func TestRemoveActiveRequiresForce(t *testing.T) {
	isolateStore(t)
	s, _ := Load()
	_ = s.Add(acct("a", client.AccountTypePayAsYouGo), false) // becomes active
	_ = s.Add(acct("b", client.AccountTypePayAsYouGo), false)

	if err := s.Remove("a", false); err == nil {
		t.Error("expected removing the active account without force to error")
	}
	if err := s.Remove("a", true); err != nil {
		t.Fatalf("forced remove: %v", err)
	}
	if s.Active != "" {
		t.Errorf("expected Active cleared after removing it, got %q", s.Active)
	}
	if _, ok := s.Get("a"); ok {
		t.Error("expected account 'a' gone after remove")
	}
}

func TestRemoveUnknownErrors(t *testing.T) {
	isolateStore(t)
	s, _ := Load()
	if err := s.Remove("ghost", false); err == nil {
		t.Error("expected an error removing a non-existent account")
	}
}

// List returns accounts sorted by name for stable output.
func TestListSorted(t *testing.T) {
	isolateStore(t)
	s, _ := Load()
	for _, n := range []string{"charlie", "alice", "bob"} {
		_ = s.Add(acct(n, client.AccountTypePayAsYouGo), false)
	}
	got := s.List()
	want := []string{"alice", "bob", "charlie"}
	for i, a := range got {
		if a.Name != want[i] {
			t.Errorf("List[%d] = %q, want %q", i, a.Name, want[i])
		}
	}
}

func TestSetActiveAndActiveAccount(t *testing.T) {
	isolateStore(t)
	s, _ := Load()
	_ = s.Add(acct("a", client.AccountTypePayAsYouGo), false)
	_ = s.Add(acct("b", client.AccountTypeCodingPlan), false)

	if err := s.SetActive("b"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	active, ok := s.ActiveAccount()
	if !ok || active.Name != "b" {
		t.Errorf("expected active account 'b', got %q (ok=%v)", active.Name, ok)
	}
	if err := s.SetActive("nope"); err == nil {
		t.Error("expected SetActive on unknown name to error")
	}
}

// A dangling Active (name no longer present) reports no active account rather
// than returning a zero-value one as if it existed.
func TestActiveAccountDangling(t *testing.T) {
	isolateStore(t)
	s := &Store{Active: "gone", Accounts: map[string]Account{}}
	if _, ok := s.ActiveAccount(); ok {
		t.Error("expected ok=false when Active names a missing account")
	}
}

func TestTouchSetsLastUsed(t *testing.T) {
	isolateStore(t)
	s, _ := Load()
	_ = s.Add(acct("a", client.AccountTypePayAsYouGo), false)
	before, _ := s.Get("a")
	if !before.LastUsedAt.IsZero() {
		t.Fatal("expected zero LastUsedAt before Touch")
	}
	if err := s.Touch("a"); err != nil {
		t.Fatalf("Touch: %v", err)
	}
	after, _ := s.Get("a")
	if after.LastUsedAt.IsZero() {
		t.Error("expected LastUsedAt set after Touch")
	}
	if err := s.Touch("ghost"); err == nil {
		t.Error("expected Touch on unknown name to error")
	}
}

func TestResolvedBaseURL(t *testing.T) {
	override := Account{Name: "o", BaseURLOverride: "https://custom/v1", Type: client.AccountTypePayAsYouGo}
	if u, err := override.ResolvedBaseURL(); err != nil || u != "https://custom/v1" {
		t.Errorf("override should win: got %q err=%v", u, err)
	}

	coding := Account{Name: "c", Type: client.AccountTypeCodingPlan}
	if u, err := coding.ResolvedBaseURL(); err != nil || u != client.CodingBaseURL {
		t.Errorf("coding_plan URL: got %q err=%v", u, err)
	}

	payg := Account{Name: "p", Type: client.AccountTypePayAsYouGo}
	if u, err := payg.ResolvedBaseURL(); err != nil || u != client.ProdBaseURL {
		t.Errorf("pay_as_you_go URL: got %q err=%v", u, err)
	}

	unknown := Account{Name: "u", Type: "mystery"}
	if _, err := unknown.ResolvedBaseURL(); err == nil {
		t.Error("expected an error for an unrecognized account type")
	}
}

// Save creates the config dir 0700 and the key-bearing file 0600, and the
// round-trip reloads identically.
func TestSaveCreatesRestrictivePermsAndRoundTrips(t *testing.T) {
	// Nest under a not-yet-existing dir so Save's MkdirAll actually creates it.
	path := filepath.Join(t.TempDir(), "cfg", "accounts.json")
	t.Setenv("ZAI_ACCOUNTS_FILE", path)

	s, _ := Load()
	_ = s.Add(acct("a", client.AccountTypeCodingPlan), false)
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("expected accounts file mode 0600, got %o", perm)
	}
	di, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := di.Mode().Perm(); perm != 0o700 {
		t.Errorf("expected config dir mode 0700, got %o", perm)
	}

	reloaded, err := Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got, ok := reloaded.Get("a"); !ok || got.APIKey != "sk-a" || reloaded.Active != "a" {
		t.Errorf("round-trip mismatch: %+v", reloaded)
	}
}

func TestSupportsMonitorEndpoints(t *testing.T) {
	if !(Account{Type: client.AccountTypeCodingPlan}).SupportsMonitorEndpoints() {
		t.Error("coding_plan should support monitor endpoints")
	}
	if (Account{Type: client.AccountTypePayAsYouGo}).SupportsMonitorEndpoints() {
		t.Error("pay_as_you_go should not support monitor endpoints")
	}
}
