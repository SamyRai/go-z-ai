package tui

import (
	"errors"

	"zai-api-client/pkg/client"
	"zai-api-client/pkg/tui/uistyle"
)

// toastLevel controls how an errMsg is styled on the status line.
type toastLevel int

const (
	toastError toastLevel = iota
	toastWarn
	toastInfo
)

// describeErr turns a Go error into a human-readable status-line message and
// a severity level, using pkg/client's structured API error categories when
// available. It never panics on a nil or unrecognized error.
func describeErr(err error) (string, toastLevel) {
	if err == nil {
		return "", toastInfo
	}
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		msg := apiErr.UserMessage
		if msg == "" {
			msg = apiErr.Error()
		}
		switch {
		case apiErr.IsQuotaError(), apiErr.IsRateLimitError():
			return msg, toastWarn
		default:
			return msg, toastError
		}
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
