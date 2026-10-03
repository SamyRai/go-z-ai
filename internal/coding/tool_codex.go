package coding

import (
	"path/filepath"
	"slices"

	"github.com/SamyRai/go-z-ai/pkg/client"
)

// CodexID is Codex's tool ID.
const CodexID = "codex"

// Codex: a ZAI provider in ~/.codex/config.toml speaking the OpenAI Responses
// protocol (wire_api = "responses") against the plan's /api/v1 root, plus the
// model's metadata in ~/.codex/models.json, which Codex reads through
// model_catalog_json — as docs.z.ai/devpack/tool/codex and the official
// helper's codex-manager write them.
var codex = Tool{
	ID:          CodexID,
	Command:     "codex",
	DisplayName: "Codex",
	InstallHint: "npm install -g @openai/codex",
	configPath:  codexConfigPath,
	load:        loadCodex,
	unload:      unloadCodex,
	detect:      detectCodex,
	mcp: mcpTarget{
		path:     codexConfigPath,
		key:      "mcp_servers",
		local:    codexLocalEntry,
		noRemote: "Codex's streamable-HTTP MCP client rejects Z.AI's hosted servers (openai/codex#14793)",
	},
}

// codexProvider is the model_providers key and display name the official
// config uses.
const codexProvider = "ZAI"

// codexCatalogRef is model_catalog_json as the official config writes it.
const codexCatalogRef = "~/.codex/models.json"

func codexConfigPath(home string) string { return filepath.Join(home, ".codex", "config.toml") }
func codexModelsPath(home string) string { return filepath.Join(home, ".codex", "models.json") }

// codexManagedKeys are the top-level config.toml keys a plan config sets.
var codexManagedKeys = []string{"model_provider", "model", "model_reasoning_effort", "model_catalog_json"}

func loadCodex(home string, cfg LoadConfig) error {
	model := MainModel
	entry := codexModelEntry(model)
	err := editConfigMap(codexModelsPath(home), func(m map[string]any) {
		models, _ := m["models"].([]any)
		models = slices.DeleteFunc(models, func(v any) bool {
			e, _ := v.(map[string]any)
			return e["slug"] == model
		})
		m["models"] = append(models, entry)
	})
	if err != nil {
		return err
	}
	return editConfigMap(codexConfigPath(home), func(m map[string]any) {
		m["model_provider"] = codexProvider
		m["model"] = model
		if effort, ok := entry["default_reasoning_level"].(string); ok {
			m["model_reasoning_effort"] = effort
		} else {
			delete(m, "model_reasoning_effort")
		}
		m["model_catalog_json"] = codexCatalogRef
		objectField(m, "model_providers")[codexProvider] = map[string]any{
			"name":                      codexProvider,
			"base_url":                  Region(cfg.Plan).ResponsesBaseURL(),
			"experimental_bearer_token": cfg.APIKey,
			"wire_api":                  "responses",
		}
	})
}

// unloadCodex removes the ZAI provider, and the top-level model settings when
// they still select it. models.json is declarative metadata and stays, as
// the official helper leaves it.
func unloadCodex(home string) error {
	d, err := detectCodex(home)
	if err != nil {
		return err
	}
	if !d.Configured {
		return ErrNotConfigured
	}
	return editConfigMap(codexConfigPath(home), func(m map[string]any) {
		if m["model_provider"] == codexProvider {
			deleteKeys(m, codexManagedKeys...)
		}
		deleteFromObject(m, "model_providers", codexProvider)
	})
}

// detectCodex reports a plan only when the ZAI provider targets a plan's
// Responses root.
func detectCodex(home string) (Detection, error) {
	m, err := readConfigMap(codexConfigPath(home))
	if err != nil {
		return Detection{}, err
	}
	providers, _ := m["model_providers"].(map[string]any)
	zai, _ := providers[codexProvider].(map[string]any)
	baseURL, _ := zai["base_url"].(string)
	plan, ok := planFromBaseURL(baseURL)
	if !ok {
		return Detection{}, nil
	}
	key, _ := zai["experimental_bearer_token"].(string)
	return Detection{Configured: true, Plan: plan, APIKey: key}, nil
}

// codexLocalEntry is Codex's stdio MCP entry; "type": "local" mirrors the
// official helper.
func codexLocalEntry(command string, args []string, env map[string]any) map[string]any {
	return map[string]any{"type": "local", "command": command, "args": anySlice(args), "env": env}
}

// effortDescriptions label the reasoning levels in Codex's model picker.
var effortDescriptions = map[string]string{
	client.EffortNone:    "No reasoning",
	client.EffortMinimal: "Minimal reasoning",
	client.EffortLow:     "Light reasoning",
	client.EffortMedium:  "Moderate reasoning",
	client.EffortHigh:    "Enhanced reasoning",
	client.EffortXhigh:   "Extended reasoning",
	client.EffortMax:     "Deep reasoning",
}

// codexModelEntry builds the models.json entry for model from the catalog:
// reasoning levels, context window, and input modalities. The remaining
// fields are the fixed values the official helper writes.
func codexModelEntry(model string) map[string]any {
	e, _ := client.CatalogEntry(model)
	// Weakest first, as Codex lists them (AllEfforts runs strongest first).
	levels := make([]any, 0, len(e.ReasoningEfforts))
	for _, effort := range slices.Backward(client.AllEfforts) {
		if slices.Contains(e.ReasoningEfforts, effort) {
			levels = append(levels, map[string]any{"effort": effort, "description": effortDescriptions[effort]})
		}
	}
	modalities := []any{"text"}
	if slices.Contains(e.Capabilities, client.CapVision) {
		modalities = append(modalities, "image")
	}
	entry := map[string]any{
		"slug":                             model,
		"display_name":                     model,
		"description":                      e.Description,
		"supported_reasoning_levels":       levels,
		"shell_type":                       "shell_command",
		"visibility":                       "list",
		"supported_in_api":                 true,
		"priority":                         0,
		"base_instructions":                "",
		"supports_reasoning_summaries":     true,
		"default_reasoning_summary":        "none",
		"support_verbosity":                false,
		"apply_patch_tool_type":            "freeform",
		"truncation_policy":                map[string]any{"mode": "bytes", "limit": 10000},
		"context_window":                   e.ContextSize,
		"max_context_window":               e.ContextSize,
		"effective_context_window_percent": 95,
		"supports_parallel_tool_calls":     true,
		"experimental_supported_tools":     []any{},
		"input_modalities":                 modalities,
	}
	if len(e.ReasoningEfforts) > 0 {
		// The strongest level the model takes, as the official config's "max".
		entry["default_reasoning_level"] = levels[len(levels)-1].(map[string]any)["effort"]
	}
	return entry
}
