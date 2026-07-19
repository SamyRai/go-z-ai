// Package palette implements the TUI's Ctrl+P command palette: a fuzzy-filter
// overlay of app-wide actions (go to tab, refresh, toggle help, switch chat
// model, quit). It is a self-contained tea.Model rendered as an overlay card
// by the root model; the chosen action is returned via a Result message the
// root interprets, so this package imports neither the root nor any screen.
package palette

import (
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/sahilm/fuzzy"

	"github.com/SamyRai/go-z-ai/internal/tui/uimsg"
	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
)

// Action identifies what the root should do when a palette command is chosen.
// Kept as a plain enum + optional arg so this package needn't import the root
// or any screen — the root switches on the Action.
type Action int

const (
	ActionNone Action = iota
	ActionSwitchTab
	ActionRefresh
	ActionToggleHelp
	ActionOpenModelPicker
	ActionQuit
)

// Result is the message the palette emits when the user picks a command. The
// root model handles it: performs the action, then closes the overlay.
type Result struct {
	Action Action
	Arg    int // tab index for ActionSwitchTab; unused otherwise
}

// Command is one selectable row in the palette.
type Command struct {
	Name  string // shown as the primary text
	Desc  string // shown muted, right of the name
	Hint  string // optional grouping label ("Navigation", "Actions")
	Do    Action
	DoArg int
}

// Model is the palette overlay. Construct via New; it owns a textinput for the
// query and re-ranks the command list on every keystroke.
type Model struct {
	commands []Command
	input    textinput.Model
	matched  []Command // current filtered+sorted view; len>=1 while typing
	cursor   int       // index into matched
}

// New builds the palette with the given commands. The root supplies the tab
// names (so this package doesn't import the tab enum) plus the static action
// set.
func New(commands []Command) Model {
	in := textinput.New()
	in.Placeholder = "search commands…"
	in.Prompt = "> "
	in.Focus()
	m := Model{commands: commands, input: in}
	m.applyFilter("")
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			// Cancel without action.
			return m, func() tea.Msg { return uimsg.CloseOverlay{} }
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "tab":
			if m.cursor < len(m.matched)-1 {
				m.cursor++
			}
			return m, nil
		case "enter":
			if m.cursor >= 0 && m.cursor < len(m.matched) {
				chosen := m.matched[m.cursor]
				action := chosen.Do
				arg := chosen.DoArg
				return m, func() tea.Msg {
					return Result{Action: action, Arg: arg}
				}
			}
			return m, func() tea.Msg { return uimsg.CloseOverlay{} }
		}
		// Any other key edits the query, which re-ranks the list.
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.applyFilter(m.input.Value())
		return m, cmd
	}
	return m, nil
}

// applyFilter ranks commands against the query via sahilm/fuzzy (the same
// matcher bubbles/list uses). An empty query shows everything in declared
// order; a non-empty query sorts by match score (best first) and drops
// non-matches.
func (m *Model) applyFilter(query string) {
	q := strings.TrimSpace(query)
	if q == "" {
		m.matched = append([]Command(nil), m.commands...)
		m.cursor = 0
		return
	}
	names := make([]string, len(m.commands))
	for i, c := range m.commands {
		names[i] = c.Name + " " + c.Desc
	}
	matches := fuzzy.Find(q, names)
	out := make([]Command, 0, len(matches))
	// fuzzy.Find returns matches in increasing score; we want best first.
	scoredMatches := make([]struct {
		c    Command
		rank int
	}, 0, len(matches))
	for _, mt := range matches {
		scoredMatches = append(scoredMatches, struct {
			c    Command
			rank int
		}{c: m.commands[mt.Index], rank: mt.Score})
	}
	sort.SliceStable(scoredMatches, func(i, j int) bool {
		return scoredMatches[i].rank > scoredMatches[j].rank
	})
	for _, s := range scoredMatches {
		out = append(out, s.c)
	}
	m.matched = out
	if m.cursor >= len(m.matched) {
		m.cursor = max(0, len(m.matched)-1)
	}
}

func (m Model) View() tea.View {
	var b strings.Builder
	b.WriteString(m.input.View())
	b.WriteByte('\n')
	if len(m.matched) == 0 {
		b.WriteString("no matching commands")
	} else {
		for i, c := range m.matched {
			line := c.Name
			if c.Desc != "" {
				line += "  " + c.Desc
			}
			if i == m.cursor {
				line = "▸ " + line
			} else {
				line = "  " + line
			}
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	body := strings.TrimRight(b.String(), "\n")
	return tea.NewView(uistyle.RenderOverlayCard("Command Palette", body))
}
