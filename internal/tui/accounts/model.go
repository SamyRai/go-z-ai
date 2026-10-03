// Package accounts implements the TUI's Accounts tab: list, add, switch, and
// remove stored Z.AI account credentials via internal/accounts.Store, the
// same store the "go-z-ai accounts" commands use. The store is only touched
// on the UI goroutine; background work (key detection) returns messages.
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
	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/internal/usageview"
	"github.com/SamyRai/go-z-ai/pkg/client"
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

// detectedMsg carries a new account whose type and region were detected in
// the background, ready to be stored on the UI goroutine.
type detectedMsg struct {
	account accounts.Account
	err     error
}

// Model is the Accounts tab's screen model.
type Model struct {
	store   *accounts.Store
	selfTab int // routes background results back to this tab
	list    list.Model
	mode    mode

	form        *huh.Form
	formName    string
	formAPIKey  string
	confirmName string
}

// New builds the Accounts screen. store must be non-nil; selfTab is this
// screen's tab index in the root model.
func New(store *accounts.Store, selfTab int) Model {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.SetShowTitle(false)
	// Enable the list's built-in fuzzy filter (sahilm/fuzzy, already an
	// indirect dep): press '/' to start typing a query, 'esc' to clear.
	l.SetFilteringEnabled(true)

	m := Model{store: store, selfTab: selfTab, list: l}
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

	case detectedMsg:
		if msg.err != nil {
			return m, errCmd(msg.err)
		}
		wasEmpty := m.store.Active == ""
		if err := m.store.Add(msg.account, false); err != nil {
			return m, errCmd(err)
		}
		return m, m.saved(fmt.Sprintf("added %s (%s)", msg.account.Name, msg.account.Type), wasEmpty)

	case tea.KeyPressMsg:
		switch m.mode {
		case modeAdd:
			return m.updateAdd(msg)
		case modeConfirmDelete:
			return m.updateConfirmDelete(msg)
		}
		// While the filter is being typed into, every key belongs to it.
		if m.list.FilterState() == list.Filtering {
			break
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
			it, ok := m.selected()
			if !ok || it.Name == m.store.Active {
				return m, nil
			}
			if err := m.store.SetActive(it.Name); err != nil {
				return m, errCmd(err)
			}
			return m, m.saved("active account: "+it.Name, true)
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// saved persists the store, reloads the list, and reports status; when the
// active account may have changed it also tells the root to switch clients.
func (m *Model) saved(status string, activeChanged bool) tea.Cmd {
	if err := m.store.Save(); err != nil {
		return errCmd(err)
	}
	m.reload()
	cmds := []tea.Cmd{func() tea.Msg { return uimsg.Status{Text: status} }}
	if activeChanged {
		cmds = append(cmds, func() tea.Msg { return uimsg.AccountChanged{} })
	}
	return tea.Batch(cmds...)
}

func errCmd(err error) tea.Cmd { return func() tea.Msg { return uimsg.Err{Err: err} } }

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
		wasActive := m.store.Active == name
		if err := m.store.Remove(name, true); err != nil {
			return m, errCmd(err)
		}
		return m, m.saved(fmt.Sprintf("removed account %q", name), wasActive)
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
		return m, uimsg.Route(m.selfTab, m.submitAdd())
	}
	if m.form.State == huh.StateAborted {
		m.mode = modeList
		return m, nil
	}
	return m, cmd
}

// submitAdd detects the new key's type and region in the background (a free
// quota probe, no tokens spent) and hands the account back as a detectedMsg.
func (m Model) submitAdd() tea.Cmd {
	name, apiKey := m.formName, m.formAPIKey
	return func() tea.Msg {
		c, err := client.NewClient(client.Config{APIKey: apiKey})
		if err != nil {
			return detectedMsg{err: err}
		}
		det, err := c.Detection().DetectAccountType(context.Background())
		if err != nil {
			return detectedMsg{err: fmt.Errorf("detect account type: %w", err)}
		}
		return detectedMsg{account: accounts.Account{
			Name:      name,
			APIKey:    apiKey,
			Type:      det.Type,
			Region:    det.Region,
			CreatedAt: time.Now(),
		}}
	}
}

func (m Model) View() tea.View {
	switch m.mode {
	case modeAdd:
		return tea.NewView(m.form.View())
	case modeConfirmDelete:
		return tea.NewView(fmt.Sprintf("Remove account %q? (y/enter to confirm, any other key to cancel)", m.confirmName))
	default:
		// Friendly empty state with a CTA — the bare list empty rendering
		// gives no hint that 'a' adds an account, and the list component
		// itself only shows a muted "No items." line.
		if len(m.list.Items()) == 0 {
			body := uistyle.EmptyTitle.Render("No accounts yet") + "\n\n" +
				uistyle.EmptyHint.Render("press 'a' to add your first Z.AI key")
			return tea.NewView(body)
		}
		return tea.NewView(m.list.View())
	}
}

// CapturesInput reports whether a text field has focus (the add form or the
// list filter), so the root leaves printable keys to it.
func (m Model) CapturesInput() bool {
	return m.mode == modeAdd || m.list.FilterState() == list.Filtering
}

// ShortHelp implements the root model's helpProvider interface.
func (m Model) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		key.NewBinding(key.WithKeys("enter", "u"), key.WithHelp("enter/u", "set active")),
		key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "remove")),
		key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	}
}
