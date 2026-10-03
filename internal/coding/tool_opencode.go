package coding

import (
	"path/filepath"
	"strings"
)

// OpenCode: a provider entry named after the plan in
// ~/.config/opencode/opencode.json, plus default model/small_model.
var openCode = Tool{
	ID:          "opencode",
	Command:     "opencode",
	DisplayName: "OpenCode",
	InstallHint: "npm install -g opencode-ai",
	configPath:  openCodeConfigPath,
	load:        loadOpenCode,
	unload:      unloadOpenCode,
	detect:      detectOpenCode,
	mcp: mcpTarget{
		path: openCodeConfigPath,
		key:  "mcp",
		local: func(command string, args []string, env map[string]any) map[string]any {
			return map[string]any{"type": "local", "command": anySlice(append([]string{command}, args...)), "environment": env}
		},
		remote: func(url string, headers map[string]any) map[string]any {
			return map[string]any{"type": "remote", "url": url, "headers": headers}
		},
	},
}

func openCodeConfigPath(home string) string {
	return filepath.Join(home, ".config", "opencode", "opencode.json")
}

// openCodeProviders maps OpenCode's built-in provider IDs to plans.
var openCodeProviders = map[string]string{
	"zai-coding-plan":     PlanGlobal,
	"zhipuai-coding-plan": PlanChina,
}

func openCodeProvider(plan string) string {
	for name, p := range openCodeProviders {
		if p == plan {
			return name
		}
	}
	return "zai-coding-plan"
}

func isOpenCodePlanModel(model string) bool {
	for name := range openCodeProviders {
		if strings.HasPrefix(model, name+"/") {
			return true
		}
	}
	return false
}

func loadOpenCode(home string, cfg LoadConfig) error {
	return editJSONMap(openCodeConfigPath(home), func(m map[string]any) {
		name := openCodeProvider(cfg.Plan)
		providers := objectField(m, "provider")
		for other := range openCodeProviders {
			delete(providers, other)
		}
		providers[name] = map[string]any{"options": map[string]any{"apiKey": cfg.APIKey}}
		m["$schema"] = "https://opencode.ai/config.json"
		// Set the default models only when unset or already a plan model, so
		// a user's own choice survives.
		for key, model := range map[string]string{"model": MainModel, "small_model": FastModel} {
			if cur, _ := m[key].(string); cur == "" || isOpenCodePlanModel(cur) {
				m[key] = name + "/" + model
			}
		}
	})
}

func unloadOpenCode(home string) error {
	d, err := detectOpenCode(home)
	if err != nil {
		return err
	}
	if !d.Configured {
		return ErrNotConfigured
	}
	return editJSONMap(openCodeConfigPath(home), func(m map[string]any) {
		for name := range openCodeProviders {
			deleteFromObject(m, "provider", name)
		}
		for _, key := range []string{"model", "small_model"} {
			if cur, _ := m[key].(string); isOpenCodePlanModel(cur) {
				delete(m, key)
			}
		}
	})
}

func detectOpenCode(home string) (Detection, error) {
	m, err := readJSONMap(openCodeConfigPath(home))
	if err != nil {
		return Detection{}, err
	}
	providers, _ := m["provider"].(map[string]any)
	for name, plan := range openCodeProviders {
		if entry, ok := providers[name].(map[string]any); ok {
			opts, _ := entry["options"].(map[string]any)
			key, _ := opts["apiKey"].(string)
			return Detection{Configured: true, Plan: plan, APIKey: key}, nil
		}
	}
	return Detection{}, nil
}
