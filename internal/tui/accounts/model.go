// Package accounts implements the TUI's Accounts tab: list, add, switch, and
// remove stored Z.AI account credentials via pkg/accounts.Store, the same
// store the "go-z-ai accounts" commands use.
package accounts

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/SamyRai/go-z-ai/internal/accounts"
	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/internal/usageview"
)

type item struct{ accounts.Account }

func (i item) FilterValue() string { return i.Name }
func (i item) Title() string       { return i.Name }
func (i item) Description() string {
	return fmt.Sprintf("%s · last used %s", i.Type, usageview.FormatRelativeTime(i.LastUsedAt))
}

type mode int

const (
	modeList mode = iota
	modeAdd
	modeConfirmDelete
)

type reloadedMsg struct{ err error }

// Model is the Accounts tab's screen model.
type Model struct {
	store *accounts.Store
	list  list.Model
	mode  mode

	form        *huh.Form
	formName    string
	formAPIKey  string
	confirmName string
}

// New builds the Accounts screen. store must be non-nil.
func New(store *accounts.Store) Model {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.SetShowTitle(false)

	m := Model{store: store, list: l}
	m.reload()
	return m
}

func (m *Model) reload() {
	accts := m.store.List()
	out := make([]list.Item, 0, len(accts))
	for _, a := range accts {
		out = append(out, item{a})
	}
	m.list.SetItems(out)
}

func (m *Model) newAddForm() *huh.Form {
	m.formName, m.formAPIKey = "", ""
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Account name").
				Value(&m.formName).
				Validate(huh.ValidateNotEmpty()),
			huh.NewInput().
				Title("Z.AI API key").
				EchoMode(huh.EchoModePassword).
				Value(&m.formAPIKey).
				Validate(huh.ValidateNotEmpty()),
		),
	)
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, max(msg.Height-2, 3))
		return m, nil

	case reloadedMsg:
		if msg.err != nil {
			return m, func() tea.Msg { return uimsg.Err{Err: msg.err} }
		}
		m.reload()
		return m, nil

	case tea.KeyPressMsg:
		switch m.mode {
		case modeAdd:
			return m.updateAdd(msg)
		case modeConfirmDelete:
			return m.updateConfirmDelete(msg)
		}

		switch msg.String() {
		case "a":
			m.mode = modeAdd
			m.form = m.newAddForm()
			return m, m.form.Init()
		case "d", "x":
			if it, ok := m.selected(); ok {
				m.mode = modeConfirmDelete
				m.confirmName = it.Name
			}
			return m, nil
		case "enter", "u":
			if it, ok := m.selected(); ok {
				if err := m.store.SetActive(it.Name); err == nil {
					_ = m.store.Save()
				}
				m.reload()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) selected() (accounts.Account, bool) {
	it, ok := m.list.SelectedItem().(item)
	if !ok {
		return accounts.Account{}, false
	}
	return it.Account, true
}

func (m Model) updateConfirmDelete(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		name := m.confirmName
		m.mode = modeList
		if err := m.store.Remove(name, true); err != nil {
			return m, func() tea.Msg { return uimsg.Err{Err: err} }
		}
		if err := m.store.Save(); err != nil {
			return m, func() tea.Msg { return uimsg.Err{Err: err} }
		}
		m.reload()
		return m, func() tea.Msg { return uimsg.Status{Text: fmt.Sprintf("removed account %q", name)} }
	default:
		m.mode = modeList
		return m, nil
	}
}

func (m Model) updateAdd(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = modeList
		return m, nil
	}

	updated, cmd := m.form.Update(msg)
	m.form = updated.(*huh.Form)

	if m.form.State == huh.StateCompleted {
		m.mode = modeList
		return m, m.submitAdd()
	}
	if m.form.State == huh.StateAborted {
		m.mode = modeList
		return m, nil
	}
	return m, cmd
}

func (m Model) submitAdd() tea.Cmd {
	name := m.formName
	apiKey := m.formAPIKey
	store := m.store
	return func() tea.Msg {
		accType, _, err := accounts.ProbeType(context.Background(), apiKey)
		if err != nil {
			return reloadedMsg{err: err}
		}
		if err := store.Add(accounts.Account{
			Name:      name,
			APIKey:    apiKey,
			Type:      accType,
			CreatedAt: time.Now(),
		}, false); err != nil {
			return reloadedMsg{err: err}
		}
		if err := store.Save(); err != nil {
			return reloadedMsg{err: err}
		}
		return reloadedMsg{}
	}
}

func (m Model) View() tea.View {
	switch m.mode {
	case modeAdd:
		return tea.NewView(m.form.View())
	case modeConfirmDelete:
		return tea.NewView(fmt.Sprintf("Remove account %q? (y/enter to confirm, any other key to cancel)", m.confirmName))
	default:
		return tea.NewView(m.list.View())
	}
}

// ShortHelp implements the root model's helpProvider interface.
func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		key.NewBinding(key.WithKeys("enter", "u"), key.WithHelp("enter/u", "set active")),
		key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "remove")),
	}
}
