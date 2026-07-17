package cli

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/SamyRai/go-z-ai/internal/accounts"
	"github.com/SamyRai/go-z-ai/pkg/client"
	"github.com/spf13/viper"
)

// isolateCreds neutralizes every ambient credential source so a precedence
// test starts from a clean slate: the four viper flag keys are cleared (and
// restored on cleanup — never viper.Reset(), which would permanently wipe the
// one-time BindPFlag bindings from init()), the credential env vars are
// emptied via t.Setenv (auto-restored), and ZAI_ACCOUNTS_FILE is pointed at a
// fresh temp file so the accounts store is empty unless the test seeds it.
func isolateCreds(t *testing.T) string {
	t.Helper()

	for _, key := range []string{"api-key", "base-url", "account", "china-api-key"} {
		prev := viper.GetString(key)
		viper.Set(key, "")
		t.Cleanup(func() { viper.Set(key, prev) })
	}

	for _, env := range []string{"ZAI_API_KEY", "KEY", "ZAI_API_BASE_URL", "ZAI_CHINA_API_KEY"} {
		t.Setenv(env, "")
	}

	accountsFile := filepath.Join(t.TempDir(), "accounts.json")
	t.Setenv("ZAI_ACCOUNTS_FILE", accountsFile)
	return accountsFile
}

// seedAccount writes a single account into the isolated store and optionally
// marks it active.
func seedAccount(t *testing.T, acct accounts.Account, active bool) {
	t.Helper()
	store, err := accounts.Load()
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	if err := store.Add(acct, false); err != nil {
		t.Fatalf("add account: %v", err)
	}
	if active {
		if err := store.SetActive(acct.Name); err != nil {
			t.Fatalf("set active: %v", err)
		}
	}
	if err := store.Save(); err != nil {
		t.Fatalf("save store: %v", err)
	}
}

func TestResolveConfigFlagBeatsEverything(t *testing.T) {
	isolateCreds(t)
	viper.Set("api-key", "flag-key")
	t.Setenv("ZAI_API_KEY", "env-key")
	seedAccount(t, accounts.Account{
		Name:      "acct",
		APIKey:    "acct-key",
		Type:      client.AccountTypePayAsYouGo,
		CreatedAt: time.Now(),
	}, true)

	cfg, err := resolveConfig()
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.APIKey != "flag-key" {
		t.Errorf("expected --api-key to win, got %q", cfg.APIKey)
	}
}

func TestResolveConfigAccountBeatsAmbientEnv(t *testing.T) {
	isolateCreds(t)
	// An explicit --account must win over an ambient ZAI_API_KEY, not lose to
	// it — the account is a deliberate per-invocation choice.
	t.Setenv("ZAI_API_KEY", "env-key")
	seedAccount(t, accounts.Account{
		Name:      "work",
		APIKey:    "work-key",
		Type:      client.AccountTypePayAsYouGo,
		CreatedAt: time.Now(),
	}, false)
	viper.Set("account", "work")

	cfg, err := resolveConfig()
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.APIKey != "work-key" {
		t.Errorf("expected --account key to win over env, got %q", cfg.APIKey)
	}
	// The account's type resolves its base URL when --base-url is unset.
	if cfg.BaseURL != client.ProdBaseURL {
		t.Errorf("expected account base URL %q, got %q", client.ProdBaseURL, cfg.BaseURL)
	}
}

func TestResolveConfigUnknownAccountFailsLoud(t *testing.T) {
	isolateCreds(t)
	// A named account that doesn't exist must error, never silently fall
	// through to an env var or the active account.
	t.Setenv("ZAI_API_KEY", "env-key")
	viper.Set("account", "does-not-exist")

	if _, err := resolveConfig(); err == nil {
		t.Fatal("expected an error for an unknown --account, got nil")
	}
}

func TestResolveConfigEnvKeyBeatsKEY(t *testing.T) {
	isolateCreds(t)
	t.Setenv("ZAI_API_KEY", "primary")
	t.Setenv("KEY", "fallback")

	cfg, err := resolveConfig()
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.APIKey != "primary" {
		t.Errorf("expected ZAI_API_KEY to win over KEY, got %q", cfg.APIKey)
	}
}

func TestResolveConfigKEYFallback(t *testing.T) {
	isolateCreds(t)
	t.Setenv("KEY", "fallback-key")

	cfg, err := resolveConfig()
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.APIKey != "fallback-key" {
		t.Errorf("expected KEY fallback, got %q", cfg.APIKey)
	}
}

func TestResolveConfigActiveAccountLastResort(t *testing.T) {
	isolateCreds(t)
	// No flag, no --account, no env — the active account is the last resort.
	seedAccount(t, accounts.Account{
		Name:      "default",
		APIKey:    "active-key",
		Type:      client.AccountTypeCodingPlan,
		CreatedAt: time.Now(),
	}, true)

	cfg, err := resolveConfig()
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.APIKey != "active-key" {
		t.Errorf("expected active-account fallback, got %q", cfg.APIKey)
	}
	if cfg.BaseURL != client.CodingBaseURL {
		t.Errorf("expected coding-plan base URL %q, got %q", client.CodingBaseURL, cfg.BaseURL)
	}
}

func TestResolveConfigExplicitBaseURLOverridesAccount(t *testing.T) {
	isolateCreds(t)
	viper.Set("base-url", "https://custom.example/v1")
	seedAccount(t, accounts.Account{
		Name:      "default",
		APIKey:    "active-key",
		Type:      client.AccountTypeCodingPlan,
		CreatedAt: time.Now(),
	}, true)

	cfg, err := resolveConfig()
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.BaseURL != "https://custom.example/v1" {
		t.Errorf("expected explicit --base-url to win over account type, got %q", cfg.BaseURL)
	}
}

func TestResolveConfigNoCredentialsFailsLoud(t *testing.T) {
	isolateCreds(t)
	if _, err := resolveConfig(); err == nil {
		t.Fatal("expected an error when no credential source is set, got nil")
	}
}

func TestResolveConfigChinaKeyFallsBackToEnv(t *testing.T) {
	isolateCreds(t)
	t.Setenv("ZAI_API_KEY", "primary")
	t.Setenv("ZAI_CHINA_API_KEY", "china-key")

	cfg, err := resolveConfig()
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.ChinaAPIKey != "china-key" {
		t.Errorf("expected ZAI_CHINA_API_KEY, got %q", cfg.ChinaAPIKey)
	}
}
