package accounts

import (
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/accounts"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// storeWith returns an in-memory store seeded with the given account names and
// an isolated ZAI_ACCOUNTS_FILE so any Save() the tab performs can't touch the
// developer's real config.
func storeWith(t *testing.T, names ...string) *accounts.Store {
	t.Helper()
	t.Setenv("ZAI_ACCOUNTS_FILE", filepath.Join(t.TempDir(), "accounts.json"))
	s := &accounts.Store{Accounts: map[string]accounts.Account{}}
	for _, n := range names {
		_ = s.Add(accounts.Account{Name: n, APIKey: "sk-" + n, Type: client.AccountTypePayAsYouGo, CreatedAt: time.Now()}, false)
	}
	return s
}

// 'a' opens the add form.
func TestAccountsAddKeyOpensForm(t *testing.T) {
	m := New(storeWith(t))
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	got := next.(Model)
	if got.mode != modeAdd {
		t.Error("expected mode switched to add")
	}
	if got.form == nil || cmd == nil {
		t.Error("expected an add form and its init command")
	}
}

// 'd' on a selected account arms the delete confirmation.
func TestAccountsDeleteKeyArmsConfirm(t *testing.T) {
	m := New(storeWith(t, "work"))
	next, _ := m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	got := next.(Model)
	if got.mode != modeConfirmDelete {
		t.Fatalf("expected confirm-delete mode, got %d", got.mode)
	}
	if got.confirmName != "work" {
		t.Errorf("expected confirmName 'work', got %q", got.confirmName)
	}
}

// Confirming a delete removes the account from the store and returns to list
// mode with a status toast.
func TestAccountsConfirmDeleteRemoves(t *testing.T) {
	store := storeWith(t, "work")
	m := New(store)
	armed, _ := m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})

	next, cmd := armed.(Model).Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if next.(Model).mode != modeList {
		t.Error("expected return to list mode after confirming delete")
	}
	if _, ok := store.Get("work"); ok {
		t.Error("expected 'work' removed from the store")
	}
	if cmd == nil {
		t.Error("expected a status toast command after delete")
	}
}
