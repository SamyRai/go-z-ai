package appconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// AppConfig represents application configuration
type AppConfig struct {
	APIKey         string `json:"apiKey"`
	AnthropicAPIKey string `json:"anthropicApiKey"`
	BaseURL        string `json:"baseURL"`
	TemplateName   string `json:"templateName,omitempty"`
}

// AppManager handles application configuration
type AppManager struct {
	backupDir string
}

// NewAppManager creates a new app manager
func NewAppManager() *AppManager {
	backupDir := filepath.Join(os.TempDir(), "zai-client-backups")
	os.MkdirAll(backupDir, 0755)
	return &AppManager{backupDir: backupDir}
}

// GetClaudeCodeConfigPath returns Claude Code settings path
func (am *AppManager) GetClaudeCodeConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".claude", "settings.json")
	return configPath, nil
}

// GetCursorConfigPath returns Cursor settings path
func (am *AppManager) GetCursorConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	var configPath string
	switch runtime.GOOS {
	case "darwin":
		// macOS: ~/Library/Application Support/Cursor/User/settings.json
		configPath = filepath.Join(homeDir, "Library", "Application Support", "Cursor", "User", "settings.json")
	case "linux", "windows":
		// Linux/Windows: ~/.cursor/settings.json or ~/.config/Cursor/User/settings.json
		configPath = filepath.Join(homeDir, ".cursor", "settings.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			configPath = filepath.Join(homeDir, ".config", "Cursor", "User", "settings.json")
		}
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return configPath, nil
}

// ReadConfig reads application configuration
func (am *AppManager) ReadConfig(configPath string) (*AppConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// WriteConfig writes application configuration
func (am *AppManager) WriteConfig(configPath string, config *AppConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// BackupConfig backs up the current configuration
func (am *AppManager) BackupConfig(configPath string) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(am.backupDir, filepath.Base(configPath)+"-"+timestamp+".backup")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

// RestoreConfig restores configuration from backup
func (am *AppManager) RestoreConfig(configPath, backupPath string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore config: %w", err)
	}

	return nil
}

// EnableClaudeCode enables Z.AI for Claude Code
func (am *AppManager) EnableClaudeCode(apiKey, baseURL string) error {
	configPath, err := am.GetClaudeCodeConfigPath()
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Backup existing config
	if _, err := os.Stat(configPath); err == nil {
		backupPath, err := am.BackupConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to backup config: %w", err)
		}
		fmt.Printf("✅ Backed up Claude Code config to: %s\n", backupPath)
	}

	// Read existing config to preserve other settings
	var existingConfig map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &existingConfig); err != nil {
			// If file exists but is invalid, start fresh
			existingConfig = make(map[string]interface{})
		}
	} else {
		existingConfig = make(map[string]interface{})
	}

	// Update config following official Z.AI documentation
	// Use env section for environment variables and model mapping
	envSection := make(map[string]string)

	// Set the base URL and API key through environment variables
	envSection["ANTHROPIC_BASE_URL"] = baseURL
	envSection["ANTHROPIC_API_KEY"] = apiKey

	// Map Claude models to GLM models (official Z.AI recommendation)
	envSection["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = "glm-4.5-air"
	envSection["ANTHROPIC_DEFAULT_SONNET_MODEL"] = "glm-4.7"
	envSection["ANTHROPIC_DEFAULT_OPUS_MODEL"] = "glm-5.2"

	existingConfig["env"] = envSection

	// Write the updated config
	data, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// DisableClaudeCode disables Z.AI for Claude Code
func (am *AppManager) DisableClaudeCode() error {
	configPath, err := am.GetClaudeCodeConfigPath()
	if err != nil {
		return err
	}

	// Find latest backup
	backups, err := filepath.Glob(filepath.Join(am.backupDir, "settings.json-*.backup"))
	if err != nil {
		return fmt.Errorf("failed to find backups: %w", err)
	}

	if len(backups) == 0 {
		// No backup found, just delete the config
		if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove config: %w", err)
		}
		fmt.Println("✅ Removed Claude Code Z.AI configuration")
		return nil
	}

	// Get latest backup
	latestBackup := backups[len(backups)-1]
	if err := am.RestoreConfig(configPath, latestBackup); err != nil {
		return fmt.Errorf("failed to restore config: %w", err)
	}

	fmt.Printf("✅ Restored Claude Code config from: %s\n", latestBackup)
	return nil
}

// EnableCursor enables Z.AI for Cursor
func (am *AppManager) EnableCursor(apiKey, baseURL string) error {
	configPath, err := am.GetCursorConfigPath()
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Backup existing config
	if _, err := os.Stat(configPath); err == nil {
		backupPath, err := am.BackupConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to backup config: %w", err)
		}
		fmt.Printf("✅ Backed up Cursor config to: %s\n", backupPath)
	}

	// Cursor uses different config structure
	config := map[string]interface{}{
		"apiKey": apiKey,
	}

	// Add baseURL if provided
	if baseURL != "" {
		config["baseURL"] = baseURL
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// DisableCursor disables Z.AI for Cursor
func (am *AppManager) DisableCursor() error {
	configPath, err := am.GetCursorConfigPath()
	if err != nil {
		return err
	}

	// Find latest backup
	backups, err := filepath.Glob(filepath.Join(am.backupDir, filepath.Base(configPath)+".*.backup"))
	if err != nil {
		return fmt.Errorf("failed to find backups: %w", err)
	}

	if len(backups) == 0 {
		// No backup found, just delete the config
		if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove config: %w", err)
		}
		fmt.Println("✅ Removed Cursor Z.AI configuration")
		return nil
	}

	// Get latest backup
	latestBackup := backups[len(backups)-1]
	if err := am.RestoreConfig(configPath, latestBackup); err != nil {
		return fmt.Errorf("failed to restore config: %w", err)
	}

	fmt.Printf("✅ Restored Cursor config from: %s\n", latestBackup)
	return nil
}

// GetClaudeCodeStatus checks Claude Code configuration status
func (am *AppManager) GetClaudeCodeStatus() (bool, error) {
	configPath, err := am.GetClaudeCodeConfigPath()
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return false, err
	}

	// Check if using Z.AI by looking for env section with Z.AI-specific settings
	envSection, ok := config["env"].(map[string]interface{})
	if !ok {
		return false, nil
	}

	// Check for Z.AI base URL or model mapping
	baseURL, hasBaseURL := envSection["ANTHROPIC_BASE_URL"].(string)
	_, hasGLMModel := envSection["ANTHROPIC_DEFAULT_SONNET_MODEL"].(string)

	return hasBaseURL && baseURL == "https://api.z.ai/api/anthropic" && hasGLMModel, nil
}

// GetCursorStatus checks Cursor configuration status
func (am *AppManager) GetCursorStatus() (bool, error) {
	configPath, err := am.GetCursorConfigPath()
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return false, err
	}

	// Check if using Z.AI configuration
	_, hasAPIKey := config["apiKey"]
	_, hasBaseURL := config["baseURL"]

	return hasAPIKey || hasBaseURL, nil
}

// RestartClaudeCode attempts to restart Claude Code
func (am *AppManager) RestartClaudeCode() error {
	// Claude Code is typically restarted via the UI
	// We can provide instructions
	fmt.Println("💡 Please restart Claude Code for the changes to take effect")
	fmt.Println("   You can restart by:")
	fmt.Println("   - Closing and reopening Claude Code")
	fmt.Println("   - Using Cmd+Q on macOS or Alt+F4 on Windows/Linux")
	return nil
}

// RestartCursor attempts to restart Cursor
func (am *AppManager) RestartCursor() error {
	// Attempt to find and restart Cursor process
	switch runtime.GOOS {
	case "darwin":
		// Try to restart Cursor on macOS
		cmd := exec.Command("killall", "Cursor")
		_ = cmd.Run() // Ignore errors if Cursor isn't running
		fmt.Println("💡 Please restart Cursor for the changes to take effect")
		return nil
	default:
		fmt.Println("💡 Please restart Cursor for the changes to take effect")
		return nil
	}
}
