package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/SamyRai/go-z-ai/internal/coding"
	"github.com/spf13/cobra"
)

var codingMcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage Z.AI's official MCP servers in a coding tool",
	Long:  codingMcpLong(),
}

// codingMcpLong renders the help from the server registry.
func codingMcpLong() string {
	var b strings.Builder
	b.WriteString("Register or remove Z.AI's official GLM Coding Plan MCP servers in a supported\ncoding tool. Calls to them draw plan quota. Servers:\n")
	for _, s := range coding.MCPServers {
		fmt.Fprintf(&b, "  %-18s %s\n", s.ID, s.Description)
	}
	b.WriteString("\nHosted servers authenticate with the plan key; the Vision server runs locally\nand needs Node.js (npx). --server selects servers (default: all).")
	return b.String()
}

var codingMcpAddCmd = &cobra.Command{
	Use:   "add <tool>",
	Short: "Register MCP servers in a tool (default: all official servers)",
	Args:  cobra.ExactArgs(1),
	RunE:  runCodingMcpAdd,
}

var codingMcpRemoveCmd = &cobra.Command{
	Use:   "remove <tool>",
	Short: "Remove MCP server entries from a tool (default: all official servers)",
	Args:  cobra.ExactArgs(1),
	RunE:  runCodingMcpRemove,
}

var codingMcpStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show which official MCP servers each tool has registered",
	Args:  cobra.NoArgs,
	RunE:  runCodingMcpStatus,
}

func init() {
	codingCmd.AddCommand(codingMcpCmd)
	codingMcpCmd.AddCommand(codingMcpAddCmd, codingMcpRemoveCmd, codingMcpStatusCmd)
	addCredentialFlags(codingMcpAddCmd)
	for _, c := range []*cobra.Command{codingMcpAddCmd, codingMcpRemoveCmd} {
		c.Flags().StringSliceVar(&codingFlags.servers, "server", nil, "MCP server IDs (repeatable; default: all)")
	}
	addFormatFlag("text", codingMcpStatusCmd)
}

// mcpTarget resolves the tool argument and --server selection.
func mcpTarget(toolID string) (coding.Tool, []coding.MCPServer, string, error) {
	tool, err := coding.FindTool(toolID)
	if err != nil {
		return coding.Tool{}, nil, "", err
	}
	servers, err := coding.FindMCPServers(codingFlags.servers...)
	if err != nil {
		return coding.Tool{}, nil, "", err
	}
	home, err := os.UserHomeDir()
	return tool, servers, home, err
}

func runCodingMcpAdd(_ *cobra.Command, args []string) error {
	tool, servers, home, err := mcpTarget(args[0])
	if err != nil {
		return err
	}
	store, err := coding.NewStore()
	if err != nil {
		return err
	}
	// Hosted servers live on the plan's region, so a plan is required too.
	plan, key, err := codingCredentials(store)
	if err != nil {
		return err
	}
	if err := tool.LoadMCP(home, plan, key, servers); err != nil {
		return err
	}
	fmt.Printf("✓ Registered %s in %s (%s)\n", serverIDs(servers), tool.DisplayName, coding.DisplayName(plan))
	if needsNPX(servers) && !coding.HasNPX() {
		fmt.Println("⚠  npx not found on PATH — install Node.js before the Vision server can run")
	}
	return nil
}

func runCodingMcpRemove(_ *cobra.Command, args []string) error {
	tool, servers, home, err := mcpTarget(args[0])
	if err != nil {
		return err
	}
	if err := tool.UnloadMCP(home, servers); err != nil {
		return err
	}
	fmt.Printf("✓ Removed %s from %s\n", serverIDs(servers), tool.DisplayName)
	return nil
}

func runCodingMcpStatus(cmd *cobra.Command, _ []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	tools := inspectCodingTools(home)
	return emit(cmd, tools, func() error {
		fmt.Printf("%-16s %s\n", "TOOL", "MCP SERVERS")
		for _, t := range tools {
			status := strings.Join(t.MCPServers, ", ")
			switch {
			case t.Error != "":
				status = t.configSummary()
			case status == "":
				status = "(none)"
			}
			fmt.Printf("%-16s %s\n", t.Name, status)
		}
		if !coding.HasNPX() {
			fmt.Println("\nℹ  npx not found on PATH — the Vision server (zai-mcp-server) needs Node.js to run")
		}
		return nil
	})
}

func serverIDs(servers []coding.MCPServer) string {
	ids := make([]string, len(servers))
	for i, s := range servers {
		ids[i] = s.ID
	}
	return strings.Join(ids, ", ")
}

// needsNPX reports whether any selected server runs locally via npx.
func needsNPX(servers []coding.MCPServer) bool {
	for _, s := range servers {
		if !s.IsRemote() {
			return true
		}
	}
	return false
}
