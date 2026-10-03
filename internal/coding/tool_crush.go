package coding

import "path/filepath"

// Crush: a providers.zai entry in ~/.config/crush/crush.json pointing at the
// plan's OpenAI-compatible endpoint; models are picked in Crush's UI.
var crush = Tool{
	ID:          "crush",
	Command:     "crush",
	DisplayName: "Crush",
	InstallHint: "npm install -g @charmland/crush",
	configPath:  crushConfigPath,
	load:        loadCrush,
	unload:      unloadCrush,
	detect:      detectCrush,
	mcp: mcpTarget{
		path:   crushConfigPath,
		key:    "mcp",
		local:  stdioEntry,
		remote: httpEntry,
	},
}

func crushConfigPath(home string) string {
	return filepath.Join(home, ".config", "crush", "crush.json")
}

func loadCrush(home string, cfg LoadConfig) error {
	return editJSONMap(crushConfigPath(home), func(m map[string]any) {
		objectField(m, "providers")["zai"] = map[string]any{
			"id":       "zai",
			"name":     "ZAI Provider",
			"base_url": Region(cfg.Plan).CodingBaseURL(),
			"api_key":  cfg.APIKey,
		}
	})
}

func unloadCrush(home string) error {
	d, err := detectCrush(home)
	if err != nil {
		return err
	}
	if !d.Configured {
		return ErrNotConfigured
	}
	return editJSONMap(crushConfigPath(home), func(m map[string]any) {
		deleteFromObject(m, "providers", "zai")
	})
}

// detectCrush reports a plan only when providers.zai targets a plan endpoint.
func detectCrush(home string) (Detection, error) {
	m, err := readJSONMap(crushConfigPath(home))
	if err != nil {
		return Detection{}, err
	}
	providers, _ := m["providers"].(map[string]any)
	zai, _ := providers["zai"].(map[string]any)
	baseURL, _ := zai["base_url"].(string)
	plan, ok := planFromBaseURL(baseURL)
	if !ok {
		return Detection{}, nil
	}
	key, _ := zai["api_key"].(string)
	return Detection{Configured: true, Plan: plan, APIKey: key}, nil
}
