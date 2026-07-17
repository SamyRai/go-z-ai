package cli

import (
	"encoding/json"
	"fmt"
	"os"

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
		return client.Config{}, fmt.Errorf("API key is required. Set it via --api-key flag, ZAI_API_KEY environment variable, KEY environment variable, 'zai-client accounts use <name>', or --account <name>")
	}

	chinaAPIKey := viper.GetString("china-api-key")
	if chinaAPIKey == "" {
		chinaAPIKey = os.Getenv("ZAI_CHINA_API_KEY")
	}

	return client.Config{
		APIKey:      apiKey,
		BaseURL:     baseURL,
		ChinaAPIKey: chinaAPIKey,
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
		return accounts.Account{}, fmt.Errorf("account %q not found (run 'zai-client accounts list')", name)
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

func validateAPIKey(cmd *cobra.Command, args []string) error {
	apiClient, err := getClient()
	if err != nil {
		return err
	}

	// Test API key by making a simple request
	_, err = apiClient.Models().List(cmd.Context())
	if err != nil {
		return fmt.Errorf("invalid API key: %w", err)
	}

	fmt.Println("✓ API key is valid")
	return nil
}

func outputJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

// maskAPIKey renders an API key safely for display (e.g. account listings),
// keeping only the first/last 4 characters visible.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "********"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
