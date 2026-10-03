package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/SamyRai/go-z-ai/internal/coding"
	"github.com/spf13/cobra"
)

var codingStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show stored credentials and per-tool configuration status",
	Args:  cobra.NoArgs,
	RunE:  runCodingStatus,
}

var codingToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List supported coding tools, install status, and config paths",
	Args:  cobra.NoArgs,
	RunE:  runCodingTools,
}

var codingDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Health check: credentials, installed tools, and Node.js for the Vision MCP server",
	Args:  cobra.NoArgs,
	RunE:  runCodingDoctor,
}

func init() {
	codingCmd.AddCommand(codingStatusCmd, codingToolsCmd, codingDoctorCmd)
	addFormatFlag("text", codingStatusCmd, codingToolsCmd)
}

// codingToolView is one tool's install and configuration state, shared by
// status, tools, doctor, and mcp status.
type codingToolView struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Command    string           `json:"command"`
	Installed  bool             `json:"installed"`
	ConfigPath string           `json:"config_path"`
	Configured bool             `json:"configured"`
	Plan       string           `json:"plan,omitempty"`
	ModelMap   *coding.ModelMap `json:"model_map,omitempty"`
	MCPServers []string         `json:"mcp_servers,omitempty"`
	// Error reports a config that couldn't be read (e.g. malformed JSON).
	Error string `json:"error,omitempty"`
}

// inspectCodingTools reads every supported tool's state under home. A tool
// whose config can't be read is reported, not fatal.
func inspectCodingTools(home string) []codingToolView {
	views := make([]codingToolView, len(coding.Tools))
	for i, t := range coding.Tools {
		v := codingToolView{
			ID: t.ID, Name: t.DisplayName, Command: t.Command,
			Installed: t.IsInstalled(), ConfigPath: t.ConfigPath(home),
		}
		var errs []string
		if d, err := t.Detect(home); err != nil {
			errs = append(errs, err.Error())
		} else {
			v.Configured, v.Plan, v.ModelMap = d.Configured, d.Plan, d.ModelMap
		}
		if servers, err := t.MCPConfigured(home); err != nil {
			errs = append(errs, err.Error())
		} else {
			for _, s := range servers {
				v.MCPServers = append(v.MCPServers, s.ID)
			}
		}
		v.Error = strings.Join(errs, "; ")
		views[i] = v
	}
	return views
}

// configSummary is the one-line configuration state shown by status/doctor.
func (v codingToolView) configSummary() string {
	if v.Error != "" {
		return "unreadable config: " + v.Error
	}
	s := "native config"
	if v.Configured {
		s = "Z.AI"
		if v.Plan != "" {
			s += " · " + coding.DisplayName(v.Plan)
		}
	}
	if len(v.MCPServers) > 0 {
		s += " · MCP: " + strings.Join(v.MCPServers, ", ")
	}
	return s
}

// codingCredentialView describes the stored credential without any key
// material: whether a key is stored is all status and doctor need.
type codingCredentialView struct {
	Plan      string `json:"plan,omitempty"`
	KeyStored bool   `json:"key_stored"`
}

func storedCodingCredentials() (codingCredentialView, error) {
	store, err := coding.NewStore()
	if err != nil {
		return codingCredentialView{}, err
	}
	c, err := store.Load()
	if err != nil {
		return codingCredentialView{}, err
	}
	return codingCredentialView{Plan: c.Plan, KeyStored: c.APIKey != ""}, nil
}

func runCodingStatus(cmd *cobra.Command, _ []string) error {
	creds, err := storedCodingCredentials()
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	tools := inspectCodingTools(home)
	view := struct {
		Credentials codingCredentialView `json:"credentials"`
		Tools       []codingToolView     `json:"tools"`
	}{creds, tools}

	return emit(cmd, view, func() error {
		fmt.Println("Stored credentials")
		fmt.Println("==================")
		fmt.Printf("  Plan: %s\n", orNone(creds.Plan, coding.DisplayName))
		fmt.Printf("  Key:  %s\n", keyState(creds.KeyStored))
		fmt.Println("\nCoding tools")
		fmt.Println("============")
		for _, t := range tools {
			fmt.Printf("  %-14s %-15s %s\n", t.Name, installedLabel(t.Installed), t.configSummary())
			if m := t.ModelMap; m != nil {
				fmt.Printf("  %-14s models: haiku=%s sonnet=%s opus=%s\n", "", m.Haiku, m.Sonnet, m.Opus)
			}
		}
		return nil
	})
}

func runCodingTools(cmd *cobra.Command, _ []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	tools := inspectCodingTools(home)
	return emit(cmd, tools, func() error {
		fmt.Printf("%-14s %-16s %-8s %-10s %s\n", "ID", "TOOL", "CMD", "INSTALLED", "CONFIG PATH")
		for _, t := range tools {
			fmt.Printf("%-14s %-16s %-8s %-10s %s\n", t.ID, t.Name, t.Command, yesNo(t.Installed), t.ConfigPath)
		}
		return nil
	})
}

// runCodingDoctor checks credentials, installed tools, and npx, printing a
// line per finding; it fails when something needs fixing.
func runCodingDoctor(cmd *cobra.Command, _ []string) error {
	creds, err := storedCodingCredentials()
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	var problems int
	warn := func(format string, a ...any) {
		problems++
		fmt.Printf("⚠  "+format+"\n", a...)
	}

	if creds.Plan == "" || !creds.KeyStored {
		warn("No credentials stored (run 'go-z-ai coding auth <plan> <key>')")
	} else {
		fmt.Printf("✓ Credentials: %s, key stored\n", coding.DisplayName(creds.Plan))
	}

	installed := 0
	for _, t := range inspectCodingTools(home) {
		switch {
		case t.Error != "":
			warn("%s: %s", t.Name, t.configSummary())
		case t.Installed:
			installed++
			fmt.Printf("✓ %s installed (%s)\n", t.Name, t.configSummary())
		}
	}
	if installed == 0 {
		warn("No supported coding tools found on PATH")
	}
	if !coding.HasNPX() {
		fmt.Println("ℹ  npx not found on PATH — the Vision MCP server (zai-mcp-server) needs Node.js to run")
	}

	if problems > 0 {
		return fmt.Errorf("%d problem(s) found", problems)
	}
	fmt.Println("\nAll good.")
	return nil
}

// keyState renders whether a key is stored.
func keyState(stored bool) string {
	if stored {
		return "stored"
	}
	return "(none)"
}

// orNone formats v with format, or "(none)" when v is empty.
func orNone(v string, format func(string) string) string {
	if v == "" {
		return "(none)"
	}
	return format(v)
}

func installedLabel(installed bool) string {
	if installed {
		return "installed"
	}
	return "not installed"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
