package coding

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// Factory Droid: two customModels entries (Anthropic and OpenAI protocols)
// in ~/.factory/settings.json; MCP servers live in ~/.factory/mcp.json.
var factoryDroid = Tool{
	ID:          "factory-droid",
	Command:     "droid",
	DisplayName: "Factory Droid",
	InstallHint: droidInstallHint(),
	aliases:     []string{"droid", "factory"},
	configPath:  droidSettingsPath,
	load:        loadFactoryDroid,
	unload:      unloadFactoryDroid,
	detect:      detectFactoryDroid,
	mcp: mcpTarget{
		path: func(h string) string { return filepath.Join(h, ".factory", "mcp.json") },
		key:  "mcpServers",
		local: func(command string, args []string, env map[string]any) map[string]any {
			e := stdioEntry(command, args, env)
			e["disabled"] = false
			return e
		},
		remote: func(url string, headers map[string]any) map[string]any {
			e := httpEntry(url, headers)
			e["disabled"] = false
			return e
		},
	},
}

func droidSettingsPath(home string) string { return filepath.Join(home, ".factory", "settings.json") }

func droidInstallHint() string {
	if runtime.GOOS == "windows" {
		return "irm https://app.factory.ai/cli/windows | iex"
	}
	return "curl -fsSL https://app.factory.ai/cli | sh"
}

// droidPlanMarker identifies plan entries in customModels; the official
// helper's display names carry it too, so entries it wrote are recognized.
const droidPlanMarker = "GLM Coding Plan"

func isDroidPlanModel(e any) bool {
	em, _ := e.(map[string]any)
	name, _ := em["displayName"].(string)
	return strings.Contains(name, droidPlanMarker)
}

func loadFactoryDroid(home string, cfg LoadConfig) error {
	region := Region(cfg.Plan)
	planName := fmt.Sprintf("%s %s", droidPlanMarker, RegionLabel(cfg.Plan))
	entry := func(protocol, provider, baseURL string) map[string]any {
		return map[string]any{
			"displayName":     fmt.Sprintf("%s [%s] - %s", displayModelName(MainModel), planName, protocol),
			"model":           MainModel,
			"baseUrl":         baseURL,
			"apiKey":          cfg.APIKey,
			"provider":        provider,
			"maxOutputTokens": maxOutput(MainModel, 131_072),
		}
	}
	return editJSONMap(droidSettingsPath(home), func(m map[string]any) {
		existing, _ := m["customModels"].([]any)
		kept := removeDroidPlanModels(existing)
		m["customModels"] = append(kept,
			entry("Anthropic", "anthropic", region.AnthropicBaseURL()),
			entry("OpenAI", "generic-chat-completion-api", region.CodingBaseURL()),
		)
	})
}

func removeDroidPlanModels(models []any) []any {
	var kept []any
	for _, e := range models {
		if !isDroidPlanModel(e) {
			kept = append(kept, e)
		}
	}
	return kept
}

func unloadFactoryDroid(home string) error {
	d, err := detectFactoryDroid(home)
	if err != nil {
		return err
	}
	if !d.Configured {
		return ErrNotConfigured
	}
	return editJSONMap(droidSettingsPath(home), func(m map[string]any) {
		existing, _ := m["customModels"].([]any)
		if kept := removeDroidPlanModels(existing); len(kept) > 0 {
			m["customModels"] = kept
		} else {
			delete(m, "customModels")
		}
	})
}

func detectFactoryDroid(home string) (Detection, error) {
	m, err := readJSONMap(droidSettingsPath(home))
	if err != nil {
		return Detection{}, err
	}
	models, _ := m["customModels"].([]any)
	for _, e := range models {
		if !isDroidPlanModel(e) {
			continue
		}
		em := e.(map[string]any)
		key, _ := em["apiKey"].(string)
		baseURL, _ := em["baseUrl"].(string)
		plan, _ := planFromBaseURL(baseURL)
		return Detection{Configured: true, Plan: plan, APIKey: key}, nil
	}
	return Detection{}, nil
}
