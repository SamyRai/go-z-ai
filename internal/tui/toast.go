package tui

import (
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/SamyRai/go-z-ai/internal/tui/uistyle"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// toastTTL is how long a toast stays visible before auto-dismissing.
const toastTTL = 4 * time.Second

// toastLevel controls how a toast is styled on the status line.
type toastLevel int

const (
	toastError toastLevel = iota
	toastWarn
	toastInfo
)

// toastExpiredMsg clears the toast with the given id; a stale tick from an
// older toast can't dismiss a newer one.
type toastExpiredMsg struct{ id int }

// setToast shows text and schedules its dismissal. Each toast gets a fresh
// monotonic id and its own timer.
func (m *rootModel) setToast(text string, level toastLevel) tea.Cmd {
	m.toastText = text
	m.toastLevel = level
	m.toastID++
	id := m.toastID
	return tea.Tick(toastTTL, func(time.Time) tea.Msg { return toastExpiredMsg{id: id} })
}

// describeErr turns an error into a status-line message and severity, using
// the client's structured API error categories when available.
func describeErr(err error) (string, toastLevel) {
	if err == nil {
		return "", toastInfo
	}
	if apiErr, ok := errors.AsType[*client.APIError](err); ok && (apiErr.IsQuotaError() || apiErr.IsRateLimitError()) {
		return err.Error(), toastWarn
	}
	return err.Error(), toastError
}

func toastStyleFor(level toastLevel) func(...string) string {
	switch level {
	case toastWarn:
		return uistyle.ToastWarn.Render
	case toastInfo:
		return uistyle.ToastInfo.Render
	default:
		return uistyle.ToastError.Render
	}
}
