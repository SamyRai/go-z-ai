package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/SamyRai/go-z-ai/internal/accounts"
	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func getClient() (*client.Client, error) {
	config, err := resolveConfig()
	if err != nil {
		return nil, err
	}
	return client.NewClient(config)
}

// runWithClient adapts a command handler that needs an API client into a
// cobra RunE. It resolves the client once (via getClient) and passes it in, so
// the ~50 command handlers that all opened with the same four-line getClient
// preamble no longer repeat it. Handlers that don't need a client, or that must
// branch before constructing one (e.g. `usage check --watch`), keep a plain
// RunE and are not wrapped.
func runWithClient(fn func(cmd *cobra.Command, args []string, apiClient *client.Client) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		apiClient, err := getClient()
		if err != nil {
			return err
		}
		return fn(cmd, args, apiClient)
	}
}

// resolveConfig resolves the effective client.Config from, in precedence
// order: the --api-key flag; an explicitly-named --account; the ZAI_API_KEY
// (then KEY) env var; and finally the accounts store's active account. It is
// split out from getClient so this precedence — the load-bearing, easy-to-get
// -wrong part — is testable without constructing a live *client.Client.
func resolveConfig() (client.Config, error) {
	apiKey := viper.GetString("api-key")
	baseURL := viper.GetString("base-url")
	accountName := viper.GetString("account")

	switch {
	case apiKey != "":
		// --api-key is the most explicit override; nothing else applies.
	case accountName != "":
		// --account is an explicit choice on this invocation — it must win
		// over an ambient ZAI_API_KEY env var, not lose to it, and an
		// unknown name must fail loud rather than silently falling through.
		acct, err := lookupAccount(accountName)
		if err != nil {
			return client.Config{}, err
		}
		if err := applyAccount(acct, &apiKey, &baseURL); err != nil {
			return client.Config{}, err
		}
	default:
		apiKey = os.Getenv("ZAI_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("KEY") // Support KEY variable name
		}
	}

	if baseURL == "" {
		baseURL = os.Getenv("ZAI_API_BASE_URL")
	}

	// Last resort: the accounts store's active account, when nothing above
	// (flags, --account, env vars) resolved an API key.
	if apiKey == "" {
		if acct, ok, err := activeAccount(); err != nil {
			return client.Config{}, err
		} else if ok {
			if err := applyAccount(acct, &apiKey, &baseURL); err != nil {
				return client.Config{}, err
			}
		}
	}

	if apiKey == "" {
		return client.Config{}, fmt.Errorf("API key is required. Set it via --api-key flag, ZAI_API_KEY environment variable, KEY environment variable, 'go-z-ai accounts use <name>', or --account <name>")
	}

	chinaAPIKey := viper.GetString("china-api-key")
	if chinaAPIKey == "" {
		chinaAPIKey = os.Getenv("ZAI_CHINA_API_KEY")
	}

	// --region selects the regional gateway for monitor/biz/agents. It does
	// not override --base-url (chat surface) or the China key (embeddings/
	// moderations). The ZAI_REGION env var mirrors ZAI_API_BASE_URL /
	// ZAI_CHINA_API_KEY so a .env file can set it persistently. Unknown values
	// fall back to global (the historical default) rather than erroring, so a
	// typo doesn't break a command that never touches the region-scoped
	// services.
	regionValue := viper.GetString("region")
	if regionValue == "" {
		// viper.AutomaticEnv() makes the bare key "REGION" work too, but the
		// documented convention is ZAI_*; honor the obvious name explicitly.
		regionValue = os.Getenv("ZAI_REGION")
	}
	region := client.RegionGlobal
	switch strings.ToLower(regionValue) {
	case "china", "cn", "bigmodel":
		region = client.RegionChina
	case "", "global", "west":
		region = client.RegionGlobal
	}

	return client.Config{
		APIKey:      apiKey,
		BaseURL:     baseURL,
		ChinaAPIKey: chinaAPIKey,
		Region:      region,
	}, nil
}

// lookupAccount resolves a stored account by name, erroring if it doesn't
// exist (fail loud — an explicitly-named account that isn't found must never
// silently fall through to another credential source).
func lookupAccount(name string) (accounts.Account, error) {
	store, err := accounts.Load()
	if err != nil {
		return accounts.Account{}, err
	}
	acct, found := store.Get(name)
	if !found {
		return accounts.Account{}, fmt.Errorf("account %q not found (run 'go-z-ai accounts list')", name)
	}
	return acct, nil
}

// applyAccount fills *apiKey with acct's credential and, when *baseURL is
// still unset, resolves acct's base URL into it — the "use this account's
// credentials" step shared by the --account flag path and the accounts-store
// active-account fallback in getClient. Also marks acct as used.
func applyAccount(acct accounts.Account, apiKey, baseURL *string) error {
	*apiKey = acct.APIKey
	if *baseURL == "" {
		resolvedURL, err := acct.ResolvedBaseURL()
		if err != nil {
			return err
		}
		*baseURL = resolvedURL
	}
	markAccountUsed(acct.Name)
	return nil
}

// activeAccount returns the accounts store's active account, if any. ok is
// false when no account is configured/active, which is not itself an error.
func activeAccount() (accounts.Account, bool, error) {
	store, err := accounts.Load()
	if err != nil {
		return accounts.Account{}, false, err
	}
	acct, found := store.ActiveAccount()
	return acct, found, nil
}

// markAccountUsed records that name's credentials were just resolved for a
// real command, best-effort — a failure to persist "last used" bookkeeping
// must never block the command that's actually using the account.
func markAccountUsed(name string) {
	store, err := accounts.Load()
	if err != nil {
		return
	}
	if err := store.Touch(name); err != nil {
		return
	}
	_ = store.Save()
}

func validateAPIKey(cmd *cobra.Command, args []string, apiClient *client.Client) error {
	// Test API key by making a simple request
	_, err := apiClient.Models().List(cmd.Context())
	if err != nil {
		return fmt.Errorf("invalid API key: %w", err)
	}

	fmt.Println("✓ API key is valid")
	return nil
}

func outputJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

// addFormatFlag registers the shared --format flag on each command with the
// given default ("table"/"text" for human output, "json" for machine output),
// so every command selects its output mode the same way instead of the three
// ad-hoc conventions this replaced (a bound package var, a bare outputJSON with
// no flag, or no JSON at all).
func addFormatFlag(def string, cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().String("format", def, "Output format (text, json)")
	}
}

// emit renders v as pretty JSON when --format json is selected, otherwise runs
// textFn for the human-readable output. It reads the flag registered by
// addFormatFlag, so commands no longer each hand-roll the `switch format` block.
func emit(cmd *cobra.Command, v any, textFn func() error) error {
	if format, _ := cmd.Flags().GetString("format"); format == "json" {
		return outputJSON(v)
	}
	return textFn()
}

// maskAPIKey renders an API key safely for display (e.g. account listings),
// keeping only the first/last 4 characters visible.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "********"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
