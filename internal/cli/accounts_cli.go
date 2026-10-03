package cli

import (
	"fmt"
	"time"

	"github.com/SamyRai/go-z-ai/internal/accounts"
	"github.com/SamyRai/go-z-ai/internal/usageview"
	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Manage multiple Z.AI account credentials",
	Long:  `Add, list, switch between, and check quota for multiple named Z.AI accounts, instead of hand-editing .env.`,
}

var accountsAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a Z.AI account",
	Long:  `Registers a named account. Its type (coding_plan or pay_as_you_go) and region are detected with a free probe unless --type is given.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsAdd,
}

var accountsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stored accounts",
	RunE:  runAccountsList,
}

var accountsUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch the active account",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsUse,
}

var accountsRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a stored account",
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountsRemove,
}

var accountsShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show one account's details (defaults to the active account)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAccountsShow,
}

var accountsCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the active account (shorthand for 'accounts show')",
	Args:  cobra.NoArgs,
	RunE:  runAccountsShow,
}

func init() {
	rootCmd.AddCommand(accountsCmd)
	accountsCmd.AddCommand(accountsAddCmd, accountsListCmd, accountsUseCmd, accountsRemoveCmd, accountsShowCmd, accountsCurrentCmd)

	accountsAddCmd.Flags().String("api-key", "", "Z.AI API key for this account (required)")
	accountsAddCmd.Flags().String("type", "", "Account type: coding_plan or pay_as_you_go (detected if omitted)")
	accountsAddCmd.Flags().String("region", "", "Gateway the key was issued on: global or china (detected with the type if omitted)")
	accountsAddCmd.Flags().String("base-url-override", "", "Custom base URL, overriding the type-derived default")
	accountsAddCmd.Flags().Bool("force", false, "Overwrite an existing account with the same name")
	accountsAddCmd.MarkFlagRequired("api-key")

	accountsRemoveCmd.Flags().Bool("yes", false, "Confirm removal of the active account")

	addFormatFlag("text", accountsListCmd, accountsShowCmd, accountsCurrentCmd)
	for _, c := range []*cobra.Command{accountsListCmd, accountsShowCmd, accountsCurrentCmd} {
		c.Flags().Bool("reveal", false, "Show full API keys instead of masking them (json format; for export/backup)")
	}
}

func runAccountsAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	apiKey, _ := cmd.Flags().GetString("api-key")
	typeFlag, _ := cmd.Flags().GetString("type")
	regionFlag, _ := cmd.Flags().GetString("region")
	baseURLOverride, _ := cmd.Flags().GetString("base-url-override")
	force, _ := cmd.Flags().GetBool("force")

	acct := accounts.Account{Name: name, APIKey: apiKey, BaseURLOverride: baseURLOverride, CreatedAt: time.Now()}
	switch accountType := client.AccountType(typeFlag); accountType {
	case client.AccountTypeCodingPlan, client.AccountTypePayAsYouGo:
		acct.Type, acct.Region = accountType, client.ParseRegion(regionFlag)
	case "":
		progressf("🔍 Detecting account type and region (free probe, no tokens spent)...\n")
		c, err := client.NewClient(client.Config{APIKey: apiKey, Region: client.ParseRegion(regionFlag)})
		if err != nil {
			return err
		}
		det, err := c.Detection().DetectAccountType(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to detect account type (pass --type to skip detection): %w", err)
		}
		acct.Type, acct.Region = det.Type, det.Region
		if !det.Confirmed {
			progressf("⚠️  Inferred %s: the coding-plan quota endpoint did not recognize the key. Pass --type to override.\n", det.Type)
		}
	default:
		return fmt.Errorf("invalid --type %q (expected %q or %q)", typeFlag, client.AccountTypeCodingPlan, client.AccountTypePayAsYouGo)
	}

	store, err := accounts.Load()
	if err != nil {
		return err
	}
	if err := store.Add(acct, force); err != nil {
		return err
	}
	if err := store.Save(); err != nil {
		return err
	}
	fmt.Printf("✅ Account %q added (%s, %s)\n", name, acct.Type, acct.Region.Host())
	if store.Active == name {
		fmt.Println("   Set as the active account (first account added).")
	}
	return nil
}

// accountView is the JSON shape of a stored account for `accounts` command
// output. It mirrors accounts.Account (the on-disk store shape) but routes the
// API key through maskAPIKey by default, so `accounts list/show --format json`
// doesn't spill raw secrets into shell history, logs, or pipelines the way
// serializing accounts.Account directly did. Pass --reveal to emit the real
// key (for export/backup). The on-disk store format is untouched — this masking
// lives purely in the CLI presentation layer.
type accountView struct {
	Name            string             `json:"name"`
	APIKey          string             `json:"api_key"`
	Type            client.AccountType `json:"type"`
	Region          client.Region      `json:"region,omitempty"`
	BaseURLOverride string             `json:"base_url_override,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	LastUsedAt      time.Time          `json:"last_used_at"`
}

// newAccountView builds an accountView, masking the API key unless reveal.
func newAccountView(a accounts.Account, reveal bool) accountView {
	key := a.APIKey
	if !reveal {
		key = maskAPIKey(a.APIKey)
	}
	return accountView{
		Name:            a.Name,
		APIKey:          key,
		Type:            a.Type,
		Region:          a.Region,
		BaseURLOverride: a.BaseURLOverride,
		CreatedAt:       a.CreatedAt,
		LastUsedAt:      a.LastUsedAt,
	}
}

func runAccountsList(cmd *cobra.Command, args []string) error {
	store, err := accounts.Load()
	if err != nil {
		return err
	}
	list := store.List()
	reveal, _ := cmd.Flags().GetBool("reveal")
	views := make([]accountView, len(list))
	for i, acct := range list {
		views[i] = newAccountView(acct, reveal)
	}
	return emit(cmd, views, func() error {
		if len(list) == 0 {
			fmt.Println("No accounts configured. Add one with: go-z-ai accounts add <name> --api-key <key>")
			return nil
		}
		fmt.Printf("%-16s %-14s %-42s %-14s %-8s %s\n", "NAME", "TYPE", "BASE URL", "API KEY", "ACTIVE", "LAST USED")
		for _, acct := range list {
			baseURL, err := acct.ResolvedBaseURL()
			if err != nil {
				baseURL = "(unresolved)"
			}
			active := ""
			if acct.Name == store.Active {
				active = "✅"
			}
			fmt.Printf("%-16s %-14s %-42s %-14s %-8s %s\n", acct.Name, acct.Type, baseURL, maskAPIKey(acct.APIKey), active, usageview.FormatRelativeTime(acct.LastUsedAt))
		}
		return nil
	})
}

func runAccountsUse(cmd *cobra.Command, args []string) error {
	name := args[0]

	store, err := accounts.Load()
	if err != nil {
		return err
	}
	if err := store.SetActive(name); err != nil {
		return err
	}
	if err := store.Save(); err != nil {
		return err
	}

	fmt.Printf("✅ Active account switched to %q\n", name)
	return nil
}

func runAccountsRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	yes, _ := cmd.Flags().GetBool("yes")

	store, err := accounts.Load()
	if err != nil {
		return err
	}
	if err := store.Remove(name, yes); err != nil {
		return err
	}
	if err := store.Save(); err != nil {
		return err
	}

	fmt.Printf("🗑️  Account %q removed\n", name)
	return nil
}

func runAccountsShow(cmd *cobra.Command, args []string) error {
	store, err := accounts.Load()
	if err != nil {
		return err
	}

	var acct accounts.Account
	var found bool
	if len(args) == 1 {
		acct, found = store.Get(args[0])
		if !found {
			return fmt.Errorf("account %q not found", args[0])
		}
	} else {
		acct, found = store.ActiveAccount()
		if !found {
			return fmt.Errorf("no active account set (run 'go-z-ai accounts use <name>' or 'go-z-ai accounts list')")
		}
	}

	reveal, _ := cmd.Flags().GetBool("reveal")
	return emit(cmd, newAccountView(acct, reveal), func() error {
		baseURL, err := acct.ResolvedBaseURL()
		if err != nil {
			baseURL = fmt.Sprintf("(unresolved: %v)", err)
		}
		fmt.Printf("👤 Account: %s\n", acct.Name)
		fmt.Printf("   Type: %s\n", acct.Type)
		fmt.Printf("   Base URL: %s\n", baseURL)
		fmt.Printf("   API Key: %s\n", maskAPIKey(acct.APIKey))
		fmt.Printf("   Active: %t\n", acct.Name == store.Active)
		fmt.Printf("   Added: %s\n", acct.CreatedAt.Format(time.DateTime))
		fmt.Printf("   Last Used: %s\n", usageview.FormatRelativeTime(acct.LastUsedAt))
		return nil
	})
}
