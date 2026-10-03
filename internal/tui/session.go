package tui

import (
	"sync/atomic"

	"github.com/SamyRai/go-z-ai/internal/accounts"
	"github.com/SamyRai/go-z-ai/internal/coding"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// Config is what the CLI resolves before starting the TUI: the client
// configuration (flags, env, active account) and the two credential stores.
type Config struct {
	Client   client.Config
	Accounts *accounts.Store
	Coding   *coding.Store
}

// session is the state the screens share and that can change while the TUI
// runs: the API client (rebuilt when the active account changes), the
// stores, and the stored coding plan shown in the header. Screens read the
// client through Client, which is safe to call from tea.Cmd goroutines.
type session struct {
	base     client.Config
	client   atomic.Pointer[client.Client]
	accounts *accounts.Store
	coding   *coding.Store
	plan     string
}

func newSession(cfg Config) (*session, error) {
	s := &session{base: cfg.Client, accounts: cfg.Accounts, coding: cfg.Coding}
	c, err := client.NewClient(cfg.Client)
	if err != nil {
		return nil, err
	}
	s.client.Store(c)
	if cfg.Coding != nil {
		if stored, err := cfg.Coding.Load(); err == nil {
			s.plan = stored.Plan
		}
	}
	return s, nil
}

// Client returns the current API client.
func (s *session) Client() *client.Client { return s.client.Load() }

// useActiveAccount points the client at the store's active account, keeping
// the rest of the base configuration (timeouts, monitor zone, …).
func (s *session) useActiveAccount() (accounts.Account, error) {
	acct, ok := s.activeAccount()
	if !ok {
		return accounts.Account{}, nil
	}
	ac, err := acct.ClientConfig()
	if err != nil {
		return acct, err
	}
	cfg := s.base
	cfg.APIKey, cfg.BaseURL, cfg.Region = ac.APIKey, ac.BaseURL, ac.Region
	c, err := client.NewClient(cfg)
	if err != nil {
		return acct, err
	}
	s.client.Store(c)
	return acct, nil
}

// activeAccount returns the active stored account, if any.
func (s *session) activeAccount() (accounts.Account, bool) {
	if s == nil || s.accounts == nil {
		return accounts.Account{}, false
	}
	return s.accounts.ActiveAccount()
}
