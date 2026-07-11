package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"zai-api-client/pkg/accounts"
	"zai-api-client/pkg/coding"
	"zai-api-client/pkg/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive terminal UI",
	Long:  `Launch a full-screen terminal UI with chat, models, usage, accounts, coding, media, and tools tabs.`,
	Args:  cobra.NoArgs,
	RunE:  runTUI,
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Fprintln(os.Stderr, "zai-client tui: stdout is not a terminal (piped or non-interactive); the TUI requires an interactive terminal. Use the individual subcommands instead (chat, models, usage, accounts, coding, tools).")
		return fmt.Errorf("not a tty")
	}

	apiClient, err := getClient()
	if err != nil {
		return err
	}

	store, err := accounts.Load()
	if err != nil {
		return err
	}

	codingStore, err := coding.NewStore()
	if err != nil {
		return err
	}

	return tui.Run(tui.Config{
		Client:   apiClient,
		Accounts: store,
		Coding:   codingStore,
	})
}
