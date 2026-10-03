package coding

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/SamyRai/go-z-ai/internal/atomicfile"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// readConfig reads path into a map for assertions, in the format its
// extension implies.
func readConfig(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	m := map[string]any{}
	if err := formatOf(path).unmarshal(data, &m); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return m
}

// writeConfig seeds path with m, in the format its extension implies.
func writeConfig(t *testing.T, path string, m map[string]any) {
	t.Helper()
	data, err := formatOf(path).marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// Every tool: load writes a detectable plan config for both plans, unload
// removes it, and a second unload reports ErrNotConfigured.
func TestToolsLoadDetectUnload(t *testing.T) {
	for _, tool := range Tools {
		for _, plan := range Plans {
			t.Run(tool.ID+"/"+plan, func(t *testing.T) {
				home := t.TempDir()
				if err := tool.Load(home, NewLoadConfig(plan, "key-1")); err != nil {
					t.Fatalf("Load: %v", err)
				}
				d, err := tool.Detect(home)
				if err != nil {
					t.Fatalf("Detect: %v", err)
				}
				if !d.Configured || d.Plan != plan || d.APIKey != "key-1" {
					t.Fatalf("Detect = %+v, want plan %s key-1", d, plan)
				}
				if err := tool.Unload(home); err != nil {
					t.Fatalf("Unload: %v", err)
				}
				if d, _ := tool.Detect(home); d.Configured {
					t.Errorf("still configured after Unload: %+v", d)
				}
				if err := tool.Unload(home); !errors.Is(err, ErrNotConfigured) {
					t.Errorf("second Unload = %v, want ErrNotConfigured", err)
				}
			})
		}
	}
}

// Load rejects an invalid plan or empty key before touching any file.
func TestToolLoadValidates(t *testing.T) {
	home := t.TempDir()
	if err := claudeCode.Load(home, LoadConfig{Plan: "global", APIKey: "k"}); err == nil {
		t.Error("expected an error for the invalid plan id \"global\"")
	}
	if err := claudeCode.Load(home, LoadConfig{Plan: PlanGlobal}); err == nil {
		t.Error("expected an error for an empty key")
	}
	if _, err := os.Stat(claudeSettingsPath(home)); !os.IsNotExist(err) {
		t.Error("a rejected Load must not write config")
	}
}

// Claude Code: region endpoint, documented env, default tuning with the
// catalog-derived [1m] model IDs, and the user's own env preserved.
func TestClaudeCodeConfig(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, claudeSettingsPath(home), map[string]any{"env": map[string]any{"MY_VAR": "keep", "ANTHROPIC_API_KEY": "old"}})
	if err := claudeCode.Load(home, NewLoadConfig(PlanChina, "k")); err != nil {
		t.Fatalf("Load: %v", err)
	}
	env := readConfig(t, claudeSettingsPath(home))["env"].(map[string]any)
	want := map[string]any{
		"ANTHROPIC_AUTH_TOKEN":           "k",
		"ANTHROPIC_BASE_URL":             client.ChinaAnthropicBaseURL,
		"API_TIMEOUT_MS":                 "3000000",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL":  client.DefaultFastModel + "[1m]",
		"ANTHROPIC_DEFAULT_SONNET_MODEL": client.DefaultModel + "[1m]",
		"MY_VAR":                         "keep",
	}
	for k, v := range want {
		if env[k] != v {
			t.Errorf("env[%s] = %v, want %v", k, env[k], v)
		}
	}
	if _, ok := env["ANTHROPIC_API_KEY"]; ok {
		t.Error("ANTHROPIC_API_KEY should be replaced by ANTHROPIC_AUTH_TOKEN")
	}
	if readConfig(t, claudeStatePath(home))["hasCompletedOnboarding"] != true {
		t.Error("onboarding flag not set")
	}

	// Reloading without tuning drops previously written tuning vars.
	if err := claudeCode.Load(home, LoadConfig{Plan: PlanChina, APIKey: "k"}); err != nil {
		t.Fatalf("reload: %v", err)
	}
	env = readConfig(t, claudeSettingsPath(home))["env"].(map[string]any)
	if _, ok := env["ANTHROPIC_DEFAULT_OPUS_MODEL"]; ok {
		t.Error("stale tuning survived a reload")
	}

	if err := claudeCode.Unload(home); err != nil {
		t.Fatalf("Unload: %v", err)
	}
	env = readConfig(t, claudeSettingsPath(home))["env"].(map[string]any)
	if env["MY_VAR"] != "keep" || len(env) != 1 {
		t.Errorf("Unload must leave only the user's env, got %v", env)
	}
}

// Claude Code settings pointing at another provider are not Z.AI's: they
// are not detected, and Unload leaves them alone.
func TestClaudeCodeForeignConfigUntouched(t *testing.T) {
	home := t.TempDir()
	foreign := map[string]any{"env": map[string]any{"ANTHROPIC_AUTH_TOKEN": "x", "ANTHROPIC_BASE_URL": "https://other.example"}}
	writeConfig(t, claudeSettingsPath(home), foreign)
	if d, _ := claudeCode.Detect(home); d.Configured {
		t.Errorf("foreign provider detected as Z.AI: %+v", d)
	}
	if err := claudeCode.Unload(home); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("Unload = %v, want ErrNotConfigured", err)
	}
	if env := readConfig(t, claudeSettingsPath(home))["env"].(map[string]any); env["ANTHROPIC_AUTH_TOKEN"] != "x" {
		t.Error("foreign token was removed")
	}
}

// OpenCode: China uses the zhipuai provider, and a user's own default model
// survives while plan defaults are written when unset.
func TestOpenCodeConfig(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, openCodeConfigPath(home), map[string]any{"model": "anthropic/claude", "provider": map[string]any{"mine": map[string]any{}}})
	if err := openCode.Load(home, NewLoadConfig(PlanChina, "k")); err != nil {
		t.Fatalf("Load: %v", err)
	}
	m := readConfig(t, openCodeConfigPath(home))
	providers := m["provider"].(map[string]any)
	if _, ok := providers["zhipuai-coding-plan"]; !ok {
		t.Errorf("missing zhipuai-coding-plan provider: %v", providers)
	}
	if _, ok := providers["mine"]; !ok {
		t.Error("user's provider was dropped")
	}
	if m["model"] != "anthropic/claude" {
		t.Errorf("user's model overwritten: %v", m["model"])
	}
	if m["small_model"] != "zhipuai-coding-plan/"+client.DefaultFastModel {
		t.Errorf("small_model = %v", m["small_model"])
	}
}

// Factory Droid: two plan entries using the catalog's flagship and output
// cap, alongside the user's own models.
func TestFactoryDroidConfig(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, droidSettingsPath(home), map[string]any{"customModels": []any{map[string]any{"displayName": "My Model"}}})
	if err := factoryDroid.Load(home, NewLoadConfig(PlanGlobal, "k")); err != nil {
		t.Fatalf("Load: %v", err)
	}
	models := readConfig(t, droidSettingsPath(home))["customModels"].([]any)
	if len(models) != 3 {
		t.Fatalf("want user model + 2 plan entries, got %d", len(models))
	}
	entry, _ := client.CatalogEntry(client.DefaultModel)
	for _, e := range models[1:] {
		em := e.(map[string]any)
		if em["model"] != client.DefaultModel || em["maxOutputTokens"] != float64(entry.MaxOutput) {
			t.Errorf("plan entry %v", em)
		}
		if !strings.Contains(em["displayName"].(string), "GLM Coding Plan Global") {
			t.Errorf("display name %v", em["displayName"])
		}
	}
}

// Writes keep a backup of the file as it was before the first write.
func TestToolWritesKeepOriginalBackup(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, crushConfigPath(home), map[string]any{"theme": "dark"})
	for range 2 {
		if err := crush.Load(home, NewLoadConfig(PlanGlobal, "k")); err != nil {
			t.Fatalf("Load: %v", err)
		}
	}
	backup := readConfig(t, crushConfigPath(home)+atomicfile.BackupSuffix)
	if backup["theme"] != "dark" || backup["providers"] != nil {
		t.Errorf("backup should hold the pre-first-write file, got %v", backup)
	}
}

func TestFindToolAliases(t *testing.T) {
	for alias, want := range map[string]string{"claude": "claude-code", "droid": "factory-droid", "factory": "factory-droid", "opencode": "opencode"} {
		if got, err := FindTool(alias); err != nil || got.ID != want {
			t.Errorf("FindTool(%q) = %v, %v; want %s", alias, got.ID, err, want)
		}
	}
	if _, err := FindTool("cursor"); err == nil {
		t.Error("cursor is configured in its GUI and must not be a supported tool")
	}
}

func TestPlanRegionAndEndpoints(t *testing.T) {
	if Region(PlanChina) != client.RegionChina || Region(PlanGlobal) != client.RegionGlobal {
		t.Error("plan → region mapping broken")
	}
	for _, plan := range Plans {
		r := Region(plan)
		for _, u := range []string{r.CodingBaseURL(), r.AnthropicBaseURL()} {
			if got, ok := planFromBaseURL(u); !ok || got != plan {
				t.Errorf("planFromBaseURL(%s) = %q, %v; want %s", u, got, ok, plan)
			}
		}
	}
	if _, ok := planFromBaseURL(client.DefaultBaseURL); ok {
		t.Error("the pay-as-you-go endpoint is not a plan endpoint")
	}
}

// Codex: config.toml gets the ZAI provider on the plan's Responses root and
// the catalog-derived model settings; models.json gets the model's entry,
// replacing a stale one and keeping others; unload keeps the user's settings.
func TestCodexConfig(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, codexConfigPath(home), map[string]any{
		"approval_policy": "never",
		"model_providers": map[string]any{"mine": map[string]any{"name": "mine"}},
	})
	writeConfig(t, codexModelsPath(home), map[string]any{"models": []any{
		map[string]any{"slug": MainModel, "description": "stale"},
		map[string]any{"slug": "my-model"},
	}})
	if err := codex.Load(home, NewLoadConfig(PlanChina, "k")); err != nil {
		t.Fatalf("Load: %v", err)
	}

	cfg := readConfig(t, codexConfigPath(home))
	if cfg["model_provider"] != codexProvider || cfg["model"] != MainModel || cfg["model_catalog_json"] != codexCatalogRef {
		t.Errorf("top-level settings: %v", cfg)
	}
	if cfg["model_reasoning_effort"] != client.EffortMax {
		t.Errorf("model_reasoning_effort = %v, want the strongest level", cfg["model_reasoning_effort"])
	}
	zai := cfg["model_providers"].(map[string]any)[codexProvider].(map[string]any)
	if zai["base_url"] != client.RegionChina.ResponsesBaseURL() || zai["wire_api"] != "responses" || zai["experimental_bearer_token"] != "k" {
		t.Errorf("ZAI provider: %v", zai)
	}

	models := readConfig(t, codexModelsPath(home))["models"].([]any)
	if len(models) != 2 {
		t.Fatalf("models.json = %v, want the user's model plus one fresh entry", models)
	}
	entry := models[1].(map[string]any)
	e, _ := client.CatalogEntry(MainModel)
	if entry["slug"] != MainModel || entry["description"] == "stale" || entry["context_window"] != float64(e.ContextSize) {
		t.Errorf("model entry: %v", entry)
	}
	var levels []string
	for _, l := range entry["supported_reasoning_levels"].([]any) {
		levels = append(levels, l.(map[string]any)["effort"].(string))
	}
	if strings.Join(levels, ",") != "low,high,max" {
		t.Errorf("reasoning levels = %v, want low,high,max", levels)
	}

	if err := codex.Unload(home); err != nil {
		t.Fatalf("Unload: %v", err)
	}
	cfg = readConfig(t, codexConfigPath(home))
	if cfg["approval_policy"] != "never" || cfg["model"] != nil {
		t.Errorf("after unload: %v", cfg)
	}
	if providers := cfg["model_providers"].(map[string]any); providers["mine"] == nil || providers[codexProvider] != nil {
		t.Errorf("providers after unload: %v", providers)
	}
}

// A Codex config whose active provider is the user's own keeps its model
// settings when the ZAI provider is removed.
func TestCodexUnloadKeepsForeignModel(t *testing.T) {
	home := t.TempDir()
	if err := codex.Load(home, NewLoadConfig(PlanGlobal, "k")); err != nil {
		t.Fatal(err)
	}
	cfg := readConfig(t, codexConfigPath(home))
	cfg["model_provider"], cfg["model"] = "mine", "my-model"
	writeConfig(t, codexConfigPath(home), cfg)
	if err := codex.Unload(home); err != nil {
		t.Fatal(err)
	}
	if cfg := readConfig(t, codexConfigPath(home)); cfg["model"] != "my-model" || cfg["model_provider"] != "mine" {
		t.Errorf("user's model settings removed: %v", cfg)
	}
}

// The repo's zai-claude-config.json example must match what Load writes for
// the Global plan, so the documented config can't drift from the code.
func TestClaudeCodeExampleConfigInSync(t *testing.T) {
	example := readConfig(t, filepath.Join("..", "..", "zai-claude-config.json"))
	home := t.TempDir()
	key := example["env"].(map[string]any)["ANTHROPIC_AUTH_TOKEN"].(string)
	if err := claudeCode.Load(home, NewLoadConfig(PlanGlobal, key)); err != nil {
		t.Fatal(err)
	}
	got := readConfig(t, claudeSettingsPath(home))["env"]
	if !reflect.DeepEqual(got, example["env"]) {
		t.Errorf("zai-claude-config.json is stale; Load writes env %v", got)
	}
}
