package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"zai-api-client/pkg/appconfig"
	"zai-api-client/pkg/coding"
	"zai-api-client/pkg/provider"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Provider management operations",
	Long:  `Manage and switch between different AI providers (Z.AI, Anthropic, OpenAI, Custom).`,
}

var providerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured providers",
	Long:  `List all available providers and their configurations.`,
	RunE:  runProviderList,
}

var providerActivateCmd = &cobra.Command{
	Use:   "activate [name]",
	Short: "Activate a provider",
	Long:  `Switch to a specific provider and set environment variables.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProviderActivate,
}

var providerAddCmd = &cobra.Command{
	Use:   "add [name] [type] [api-key]",
	Short: "Add a new provider",
	Long:  `Add a new provider configuration. Supported types: zai, anthropic, openai, custom.`,
	Args:  cobra.ExactArgs(3),
	RunE:  runProviderAdd,
}

var providerRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a provider",
	Long:  `Remove a provider configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProviderRemove,
}

var providerDeactivateCmd = &cobra.Command{
	Use:   "deactivate",
	Short: "Deactivate current provider",
	Long:  `Deactivate the currently active provider and clear environment variables.`,
	RunE:  runProviderDeactivate,
}

var providerCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current provider",
	Long:  `Display the currently active provider configuration.`,
	RunE:  runProviderCurrent,
}

var providerConfigCmd = &cobra.Command{
	Use:   "config [name]",
	Short: "Show provider configuration",
	Long:  `Display detailed configuration for a specific provider.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runProviderConfig,
}

var providerEnableClaudeCmd = &cobra.Command{
	Use:   "enable-claude",
	Short: "Enable Z.AI for Claude Code",
	Long:  `Configure Claude Code app to use Z.AI instead of Anthropic.`,
	RunE:  runProviderEnableClaude,
}

var providerEnableCursorCmd = &cobra.Command{
	Use:   "enable-cursor",
	Short: "Enable Z.AI for Cursor",
	Long:  `Configure Cursor app to use Z.AI instead of Anthropic.`,
	RunE:  runProviderEnableCursor,
}

var providerDisableClaudeCmd = &cobra.Command{
	Use:   "disable-claude",
	Short: "Disable Z.AI for Claude Code",
	Long:  `Reset Claude Code app to use native Anthropic provider.`,
	RunE:  runProviderDisableClaude,
}

var providerDisableCursorCmd = &cobra.Command{
	Use:   "disable-cursor",
	Short: "Disable Z.AI for Cursor",
	Long:  `Reset Cursor app to use native Anthropic provider.`,
	RunE:  runProviderDisableCursor,
}

var providerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show provider status",
	Long:  `Show current Z.AI integration status with Claude Code and Cursor.`,
	RunE:  runProviderStatus,
}

func init() {
	rootCmd.AddCommand(providerCmd)
	providerCmd.AddCommand(providerListCmd)
	providerCmd.AddCommand(providerActivateCmd)
	providerCmd.AddCommand(providerAddCmd)
	providerCmd.AddCommand(providerRemoveCmd)
	providerCmd.AddCommand(providerDeactivateCmd)
	providerCmd.AddCommand(providerCurrentCmd)
	providerCmd.AddCommand(providerConfigCmd)
	providerCmd.AddCommand(providerEnableClaudeCmd)
	providerCmd.AddCommand(providerEnableCursorCmd)
	providerCmd.AddCommand(providerDisableClaudeCmd)
	providerCmd.AddCommand(providerDisableCursorCmd)
	providerCmd.AddCommand(providerStatusCmd)
}

func runProviderList(cmd *cobra.Command, args []string) error {
	pm := provider.NewProviderManager("")

	// Load default providers from environment
	zaiProvider := getZAIProvider()
	if zaiProvider != nil {
		if err := pm.AddProvider(zaiProvider); err != nil {
			fmt.Printf("Warning: Could not load Z.AI provider: %v\n", err)
		}
	}

	anthropicProvider := getAnthropicProvider()
	if anthropicProvider != nil {
		if err := pm.AddProvider(anthropicProvider); err != nil {
			fmt.Printf("Warning: Could not load Anthropic provider: %v\n", err)
		}
	}

	providers := pm.ListProviders()

	if len(providers) == 0 {
		fmt.Println("No providers configured.")
		fmt.Println("\n💡 Use 'zai-client provider add' to add a provider.")
		return nil
	}

	fmt.Println("🔧 Configured Providers:")
	fmt.Println("======================")

	for _, prov := range providers {
		status := "  "
		if prov.Name == getActiveProvider(pm) {
			status = "✅ "
		}

		fmt.Printf("%s%s (%s)\n", status, prov.Name, prov.Type)
		fmt.Printf("   Base URL: %s\n", prov.BaseURL)
		fmt.Printf("   Model: %s\n", prov.Model)
		if prov.APIKey != "" {
			maskedKey := maskAPIKey(prov.APIKey)
			fmt.Printf("   API Key: %s\n", maskedKey)
		}
		fmt.Println()
	}

	return nil
}

func runProviderActivate(cmd *cobra.Command, args []string) error {
	name := args[0]

	pm := provider.NewProviderManager("")

	// Load default providers from environment
	zaiProvider := getZAIProvider()
	if zaiProvider != nil {
		if err := pm.AddProvider(zaiProvider); err != nil {
			fmt.Printf("Warning: Could not load Z.AI provider: %v\n", err)
		}
	}

	anthropicProvider := getAnthropicProvider()
	if anthropicProvider != nil {
		if err := pm.AddProvider(anthropicProvider); err != nil {
			fmt.Printf("Warning: Could not load Anthropic provider: %v\n", err)
		}
	}

	if err := pm.ActivateProvider(name); err != nil {
		return fmt.Errorf("failed to activate provider: %w", err)
	}

	config, err := pm.GetActiveProvider()
	if err != nil {
		return fmt.Errorf("failed to get active provider: %w", err)
	}

	fmt.Printf("✅ Provider '%s' activated!\n", name)
	fmt.Printf("   Type: %s\n", config.Type)
	fmt.Printf("   Base URL: %s\n", config.BaseURL)
	fmt.Printf("   Model: %s\n", config.Model)

	fmt.Println("\n🔧 Environment variables set:")
	for key, value := range config.Environment {
		fmt.Printf("   %s=%s\n", key, value)
	}

	return nil
}

func getZAIProvider() *provider.ProviderConfig {
	apiKey := os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		return nil
	}

	return &provider.ProviderConfig{
		Type:    provider.ProviderZAI,
		Name:    "zai-coding",
		BaseURL: "https://api.z.ai/api/coding/paas/v4",
		APIKey:  apiKey,
		Model:   "glm-4.7",
		Enabled: true,
	}
}

func getAnthropicProvider() *provider.ProviderConfig {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil
	}

	return &provider.ProviderConfig{
		Type:    provider.ProviderAnthropic,
		Name:    "anthropic-default",
		BaseURL: "https://api.anthropic.com",
		APIKey:  apiKey,
		Model:   "claude-sonnet-4-20250514",
		Enabled: false,
	}
}

func getActiveProvider(pm *provider.ProviderManager) string {
	if config, err := pm.GetActiveProvider(); err == nil {
		return config.Name
	}
	return ""
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "********"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func runProviderAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	providerType := args[1]
	apiKey := args[2]

	pm := provider.NewProviderManager("")

	config := &provider.ProviderConfig{
		Name:   name,
		Type:   provider.ProviderType(providerType),
		APIKey: apiKey,
	}

	if err := pm.AddProvider(config); err != nil {
		return fmt.Errorf("failed to add provider: %w", err)
	}

	fmt.Printf("✅ Provider '%s' added successfully!\n", name)
	fmt.Printf("   Type: %s\n", config.Type)
	fmt.Printf("   Base URL: %s\n", config.BaseURL)
	fmt.Printf("   Model: %s\n", config.Model)

	return nil
}

func runProviderRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	pm := provider.NewProviderManager("")

	// Load default providers from environment
	zaiProvider := getZAIProvider()
	if zaiProvider != nil {
		if err := pm.AddProvider(zaiProvider); err != nil {
			fmt.Printf("Warning: Could not load Z.AI provider: %v\n", err)
		}
	}

	anthropicProvider := getAnthropicProvider()
	if anthropicProvider != nil {
		if err := pm.AddProvider(anthropicProvider); err != nil {
			fmt.Printf("Warning: Could not load Anthropic provider: %v\n", err)
		}
	}

	if err := pm.RemoveProvider(name); err != nil {
		return fmt.Errorf("failed to remove provider: %w", err)
	}

	fmt.Printf("✅ Provider '%s' removed successfully!\n", name)
	return nil
}

func runProviderDeactivate(cmd *cobra.Command, args []string) error {
	pm := provider.NewProviderManager("")

	// Load default providers from environment
	zaiProvider := getZAIProvider()
	if zaiProvider != nil {
		if err := pm.AddProvider(zaiProvider); err != nil {
			fmt.Printf("Warning: Could not load Z.AI provider: %v\n", err)
		}
	}

	anthropicProvider := getAnthropicProvider()
	if anthropicProvider != nil {
		if err := pm.AddProvider(anthropicProvider); err != nil {
			fmt.Printf("Warning: Could not load Anthropic provider: %v\n", err)
		}
	}

	if err := pm.ActivateProvider(zaiProvider.Name); err != nil {
		return fmt.Errorf("failed to set active provider: %w", err)
	}

	if err := pm.DeactivateProvider(); err != nil {
		return fmt.Errorf("failed to deactivate provider: %w", err)
	}

	fmt.Println("✅ Provider deactivated successfully!")
	fmt.Println("   Environment variables cleared")

	return nil
}

func runProviderCurrent(cmd *cobra.Command, args []string) error {
	pm := provider.NewProviderManager("")

	// Load default providers from environment
	zaiProvider := getZAIProvider()
	if zaiProvider != nil {
		if err := pm.AddProvider(zaiProvider); err != nil {
			fmt.Printf("Warning: Could not load Z.AI provider: %v\n", err)
		}
	}

	anthropicProvider := getAnthropicProvider()
	if anthropicProvider != nil {
		if err := pm.AddProvider(anthropicProvider); err != nil {
			fmt.Printf("Warning: Could not load Anthropic provider: %v\n", err)
		}
	}

	config, err := pm.GetActiveProvider()
	if err != nil {
		return fmt.Errorf("no active provider: %w", err)
	}

	fmt.Printf("✅ Current Provider: %s\n", config.Name)
	fmt.Printf("   Type: %s\n", config.Type)
	fmt.Printf("   Base URL: %s\n", config.BaseURL)
	fmt.Printf("   Model: %s\n", config.Model)

	if config.APIKey != "" {
		maskedKey := maskAPIKey(config.APIKey)
		fmt.Printf("   API Key: %s\n", maskedKey)
	}

	fmt.Println("\n🔧 Environment variables:")
	for key, value := range config.Environment {
		if key == "API_KEY" {
			fmt.Printf("   %s=%s\n", key, maskAPIKey(value))
		} else {
			fmt.Printf("   %s=%s\n", key, value)
		}
	}

	return nil
}

func runProviderConfig(cmd *cobra.Command, args []string) error {
	name := args[0]

	pm := provider.NewProviderManager("")

	// Load default providers from environment
	zaiProvider := getZAIProvider()
	if zaiProvider != nil {
		if err := pm.AddProvider(zaiProvider); err != nil {
			fmt.Printf("Warning: Could not load Z.AI provider: %v\n", err)
		}
	}

	anthropicProvider := getAnthropicProvider()
	if anthropicProvider != nil {
		if err := pm.AddProvider(anthropicProvider); err != nil {
			fmt.Printf("Warning: Could not load Anthropic provider: %v\n", err)
		}
	}

	config, err := pm.GetProviderConfig(name)
	if err != nil {
		return fmt.Errorf("failed to get provider config: %w", err)
	}

	fmt.Printf("🔧 Provider Configuration: %s\n", name)
	fmt.Printf("   Type: %s\n", config.Type)
	fmt.Printf("   Base URL: %s\n", config.BaseURL)
	fmt.Printf("   Model: %s\n", config.Model)
	fmt.Printf("   Enabled: %t\n", config.Enabled)

	if config.APIKey != "" {
		maskedKey := maskAPIKey(config.APIKey)
		fmt.Printf("   API Key: %s\n", maskedKey)
	}

	if len(config.Headers) > 0 {
		fmt.Println("\n📋 Headers:")
		for key, value := range config.Headers {
			if key == "Authorization" || key == "x-api-key" {
				fmt.Printf("   %s: %s\n", key, maskAPIKey(value))
			} else {
				fmt.Printf("   %s: %s\n", key, value)
			}
		}
	}

	if len(config.Environment) > 0 {
		fmt.Println("\n🔧 Environment Variables:")
		for key, value := range config.Environment {
			if key == "API_KEY" {
				fmt.Printf("   %s: %s\n", key, maskAPIKey(value))
			} else {
				fmt.Printf("   %s: %s\n", key, value)
			}
		}
	}

	return nil
}

func runProviderEnableClaude(cmd *cobra.Command, args []string) error {
	fmt.Println("🔧 Configuring Claude Code to use Z.AI...")

	// Get Z.AI configuration
	apiKey := os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ZAI_API_KEY environment variable not set")
	}

	// Derive plan from base URL (China mirror uses open.bigmodel.cn).
	plan := coding.PlanGlobal
	if strings.Contains(os.Getenv("ZAI_BASE_URL"), "bigmodel.cn") {
		plan = coding.PlanChina
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	// Writes the official @z_ai/coding-helper format (ANTHROPIC_AUTH_TOKEN, not
	// ANTHROPIC_API_KEY) into ~/.claude/settings.json + ~/.claude.json.
	if err := coding.LoadClaudeCode(home, plan, apiKey); err != nil {
		return fmt.Errorf("failed to enable Claude Code: %w", err)
	}

	fmt.Printf("✅ Claude Code configured to use Z.AI (%s)!\n", coding.DisplayName(plan))
	fmt.Printf("   API Key: %s\n", maskAPIKey(apiKey))
	fmt.Printf("   Base URL: %s\n", coding.AnthropicBaseURL(plan))
	fmt.Println("   💡 Restart Claude Code for changes to take effect.")
	return nil
}

func runProviderEnableCursor(cmd *cobra.Command, args []string) error {
	fmt.Println("🔧 Configuring Cursor to use Z.AI...")

	// Get Z.AI configuration
	apiKey := os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ZAI_API_KEY environment variable not set")
	}

	baseURL := "https://api.z.ai/api/anthropic"
	if customURL := os.Getenv("ZAI_BASE_URL"); customURL != "" {
		baseURL = customURL
	}

	// Create app manager and enable
	appManager := appconfig.NewAppManager()
	if err := appManager.EnableCursor(apiKey, baseURL); err != nil {
		return fmt.Errorf("failed to enable Cursor: %w", err)
	}

	fmt.Println("✅ Cursor configured to use Z.AI!")
	fmt.Printf("   API Key: %s\n", maskAPIKey(apiKey))
	fmt.Printf("   Base URL: %s\n", baseURL)
	
	if err := appManager.RestartCursor(); err != nil {
		return err
	}

	return nil
}

func runProviderDisableClaude(cmd *cobra.Command, args []string) error {
	fmt.Println("🔄 Removing Z.AI configuration from Claude Code...")

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if err := coding.UnloadClaudeCode(home); err != nil {
		return fmt.Errorf("failed to disable Claude Code: %w", err)
	}

	fmt.Println("✅ Claude Code Z.AI configuration removed (reverted to native).")
	fmt.Println("   💡 Restart Claude Code for changes to take effect.")
	return nil
}

func runProviderDisableCursor(cmd *cobra.Command, args []string) error {
	fmt.Println("🔄 Resetting Cursor to native Anthropic...")

	appManager := appconfig.NewAppManager()
	if err := appManager.DisableCursor(); err != nil {
		return fmt.Errorf("failed to disable Cursor: %w", err)
	}

	fmt.Println("✅ Cursor reset to use native Anthropic!")
	
	if err := appManager.RestartCursor(); err != nil {
		return err
	}

	return nil
}

func runProviderStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("📊 Z.AI Integration Status")
	fmt.Print("==========================\n\n")

	appManager := appconfig.NewAppManager()

	// Check Claude Code status (coding-helper-compatible config).
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("❌ Claude Code: %v\n", err)
	} else {
		d, err := coding.DetectClaudeCode(home)
		if err != nil {
			fmt.Printf("❌ Claude Code: %v\n", err)
		} else if d.Configured {
			fmt.Println("✅ Claude Code: Using Z.AI")
			fmt.Printf("   Plan: %s\n", coding.DisplayName(d.Plan))
			fmt.Printf("   API Key: %s\n", maskAPIKey(d.APIKey))
		} else {
			fmt.Println("🅰️  Claude Code: Using native Anthropic")
		}
	}

	fmt.Println()

	// Check Cursor status
	cursorPath, err := appManager.GetCursorConfigPath()
	if err != nil {
		fmt.Printf("❌ Cursor: Error getting config path - %v\n", err)
	} else {
		cursorEnabled, err := appManager.GetCursorStatus()
		if err != nil {
			fmt.Printf("❌ Cursor: Error checking status - %v\n", err)
		} else if cursorEnabled {
			fmt.Println("✅ Cursor: Using Z.AI")
			data, _ := os.ReadFile(cursorPath)
			var config map[string]interface{}
			if json.Unmarshal(data, &config) == nil {
				if apiKey, ok := config["apiKey"].(string); ok {
					fmt.Printf("   API Key: %s\n", maskAPIKey(apiKey))
				}
				if baseURL, ok := config["baseURL"].(string); ok {
					fmt.Printf("   Base URL: %s\n", baseURL)
				}
			}
		} else {
			fmt.Println("🅰️  Cursor: Using native Anthropic")
		}
	}

	fmt.Println("\n💡 Use 'zai-client provider enable-claude' or 'zai-client provider enable-cursor' to switch to Z.AI")
	fmt.Println("💡 Use 'zai-client provider disable-claude' or 'zai-client provider disable-cursor' to switch back to Anthropic")

	return nil
}
