package cli

import (
	"reflect"
	"testing"

	"github.com/SamyRai/go-z-ai/internal/coding"
	"github.com/spf13/cobra"
)

// claudeOptions parses args into a fresh claudeFlags and resolves them.
func claudeOptions(t *testing.T, args ...string) coding.ClaudeOptions {
	t.Helper()
	var f claudeFlags
	cmd := &cobra.Command{Use: "x"}
	f.register(cmd)
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatal(err)
	}
	return f.options(cmd)
}

func TestClaudeFlagsOptions(t *testing.T) {
	def := coding.DefaultClaudeOptions()
	if got := claudeOptions(t); !reflect.DeepEqual(got, def) {
		t.Errorf("no flags: got %+v, want the defaults %+v", got, def)
	}

	got := claudeOptions(t, "--sonnet", "glm-x", "--max-output-tokens", "4096")
	if got.ModelMap.Sonnet != "glm-x" || got.ModelMap.Opus != def.ModelMap.Opus || got.MaxOutputTokens != 4096 {
		t.Errorf("overrides: got %+v", got)
	}

	if got := claudeOptions(t, "--no-model-mapping", "--opus", "ignored"); got.ModelMap != nil {
		t.Errorf("--no-model-mapping must drop the mapping, got %+v", got.ModelMap)
	}

	// An explicit 0 omits the window; leaving the flag unset keeps the
	// catalog-derived default.
	if got := claudeOptions(t, "--auto-compact-window", "0"); got.AutoCompactWindow != 0 {
		t.Errorf("explicit 0: got %d", got.AutoCompactWindow)
	}
	if def.AutoCompactWindow == 0 {
		t.Error("the default auto-compact window should come from the catalog")
	}
}

func TestCodingCredentialsFlags(t *testing.T) {
	prev := codingFlags
	t.Cleanup(func() { codingFlags = prev })

	// Both flags set: the store is never consulted.
	codingFlags.plan, codingFlags.key = coding.PlanChina, "k"
	plan, key, err := codingCredentials(nil)
	if err != nil || plan != coding.PlanChina || key != "k" {
		t.Errorf("got %q %q %v", plan, key, err)
	}

	codingFlags.plan = "glm_coding_plan_mars"
	if _, _, err := codingCredentials(nil); err == nil {
		t.Error("an unknown plan must be rejected")
	}
}
