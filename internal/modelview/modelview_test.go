package modelview

import (
	"testing"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

func TestFormatTokens(t *testing.T) {
	for n, want := range map[int]string{0: Unknown, 512: "512", 131_072: "128K", 202_752: "198K", 1_048_576: "1M", 98_304: "96K"} {
		if got := FormatTokens(n); got != want {
			t.Errorf("FormatTokens(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestFormatRate(t *testing.T) {
	for v, want := range map[float64]string{0: "$0", 1.4: "$1.40", 4.40: "$4.40", 0.075: "$0.075", 0.004: "$0.004", 0.04: "$0.04", 0.15: "$0.15"} {
		if got := FormatRate(v); got != want {
			t.Errorf("FormatRate(%v) = %q, want %q", v, got, want)
		}
	}
	if in, out, cached := Rates(nil); in != Unknown || out != Unknown || cached != Unknown {
		t.Error("nil pricing must render as unknown")
	}
}

func TestCapabilities(t *testing.T) {
	caps := []string{client.CapTools, client.CapText, client.CapVision}
	if got := Capabilities(caps); got != "text,vision,tools" {
		t.Errorf("Capabilities = %q", got)
	}
	if got := CapabilityCodes(caps); got != "T V tl" {
		t.Errorf("CapabilityCodes = %q", got)
	}
	if got := Capabilities(nil); got != Unknown {
		t.Errorf("Capabilities(nil) = %q", got)
	}
}
