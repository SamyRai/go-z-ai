package coding

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Detection is the result of probing a tool's config for a Z.AI plan.
type Detection struct {
	Configured bool      // a Z.AI plan configuration is present
	Plan       string    // PlanGlobal / PlanChina, or "" if unknown/custom
	APIKey     string    // the key found in the config (may be empty)
	ModelMap   *ModelMap // Claude Code tier→model mapping, when present (Claude only)
}

// ModelMap maps Claude Code's haiku/sonnet/opus tiers to model ids via the
// ANTHROPIC_DEFAULT_*_MODEL env vars. The official @z_ai/coding-helper does not
// set these (it relies on Z.AI's endpoint auto-mapping), but Claude Code reads
// them, so offering explicit control is strictly better.
type ModelMap struct {
	Haiku  string
	Sonnet string
	Opus   string
}

// DefaultModelMap is Z.AI's documented recommended Claude Code mapping
// (https://docs.z.ai/scenario-example/develop-tools/claude → "How to Switch the
// Model"): haiku→glm-4.5-air (fast/cheap), sonnet/opus→glm-4.7.
func DefaultModelMap() *ModelMap {
	return &ModelMap{Haiku: "glm-4.5-air", Sonnet: "glm-4.7", Opus: "glm-4.7"}
}

// ClaudeOptions tunes LoadClaudeCode. Zero values omit the corresponding env
// var. These are enhancements over the official @z_ai/coding-helper (which only
// writes AUTH_TOKEN, BASE_URL, API_TIMEOUT_MS, DISABLE_NONESSENTIAL_TRAFFIC).
type ClaudeOptions struct {
	ModelMap          *ModelMap // ANTHROPIC_DEFAULT_*_MODEL tier mapping
	AutoCompactWindow int       // CLAUDE_CODE_AUTO_COMPACT_WINDOW (best for large-context GLM models)
	MaxThinkingTokens int       // MAX_THINKING_TOKENS extended-thinking budget
	MaxOutputTokens   int       // CLAUDE_CODE_MAX_OUTPUT_TOKENS
}

// DefaultClaudeOptions returns the recommended Claude Code tuning for Z.AI:
// Z.AI's documented model mapping plus a 1M-token auto-compact window (GLM-5.2
// ships a 1M context). Lower AutoCompactWindow (e.g. 128000) for 128K models.
func DefaultClaudeOptions() ClaudeOptions {
	return ClaudeOptions{
		ModelMap:          DefaultModelMap(),
		AutoCompactWindow: 1000000,
	}
}

// --- JSON helpers (preserve unknown fields, mirroring the helper's spreads) ---

func readJSONMap(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return m, nil
}

func writeJSONMap(path string, m map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// objectField returns the child object at key, creating an empty one if missing
// or the wrong type.
func objectField(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	child := map[string]interface{}{}
	m[key] = child
	return child
}

// --- Claude Code (~/.claude/settings.json + ~/.claude.json) ---

// LoadClaudeCode writes the GLM Coding Plan into Claude Code without model
// mapping (byte-compatible with @z_ai/coding-helper). For the enhanced path
// with ANTHROPIC_DEFAULT_*_MODEL tier mapping, use LoadClaudeCodeOpts.
func LoadClaudeCode(home, plan, key string) error {
	return LoadClaudeCodeOpts(home, plan, key, ClaudeOptions{})
}

// LoadClaudeCodeOpts writes the GLM Coding Plan into Claude Code. Following the
// official helper: it sets hasCompletedOnboarding in ~/.claude.json, then writes
// settings.json env with ANTHROPIC_AUTH_TOKEN (not ANTHROPIC_API_KEY — the
// Claude Code env-vars reference maps AUTH_TOKEN to "Authorization: Bearer",
// which is what Z.AI's anthropic endpoint expects), ANTHROPIC_BASE_URL,
// API_TIMEOUT_MS, and CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC. When opts.
// ModelMap is non-nil it additionally sets the three ANTHROPIC_DEFAULT_*_MODEL
// tier vars (an enhancement over the official helper).
func LoadClaudeCodeOpts(home, plan, key string, opts ClaudeOptions) error {
	claudeJSON := filepath.Join(home, ".claude.json")
	m, err := readJSONMap(claudeJSON)
	if err != nil {
		return err
	}
	if _, ok := m["hasCompletedOnboarding"].(bool); !ok {
		m["hasCompletedOnboarding"] = true
		if err := writeJSONMap(claudeJSON, m); err != nil {
			return err
		}
	}

	settings := Tools[0].ConfigPath(home)
	s, err := readJSONMap(settings)
	if err != nil {
		return err
	}
	env := objectField(s, "env")
	delete(env, "ANTHROPIC_API_KEY") // standardize on AUTH_TOKEN
	env["ANTHROPIC_AUTH_TOKEN"] = key
	env["ANTHROPIC_BASE_URL"] = AnthropicBaseURL(plan)
	env["API_TIMEOUT_MS"] = "3000000"
	env["CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC"] = 1
	if opts.ModelMap != nil {
		env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = opts.ModelMap.Haiku
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = opts.ModelMap.Sonnet
		env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = opts.ModelMap.Opus
	}
	if opts.AutoCompactWindow > 0 {
		env["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] = strconv.Itoa(opts.AutoCompactWindow)
	}
	if opts.MaxThinkingTokens > 0 {
		env["MAX_THINKING_TOKENS"] = strconv.Itoa(opts.MaxThinkingTokens)
	}
	if opts.MaxOutputTokens > 0 {
		env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = strconv.Itoa(opts.MaxOutputTokens)
	}
	return writeJSONMap(settings, s)
}

// UnloadClaudeCode removes the Z.AI env vars from Claude Code settings.
func UnloadClaudeCode(home string) error {
	settings := Tools[0].ConfigPath(home)
	s, err := readJSONMap(settings)
	if err != nil {
		return err
	}
	env, ok := s["env"].(map[string]interface{})
	if !ok {
		return nil
	}
	for _, k := range []string{
		"ANTHROPIC_AUTH_TOKEN",
		"ANTHROPIC_BASE_URL",
		"API_TIMEOUT_MS",
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
		"CLAUDE_CODE_AUTO_COMPACT_WINDOW",
		"MAX_THINKING_TOKENS",
		"CLAUDE_CODE_MAX_OUTPUT_TOKENS",
	} {
		delete(env, k)
	}
	if len(env) == 0 {
		delete(s, "env")
	} else {
		s["env"] = env
	}
	return writeJSONMap(settings, s)
}

// DetectClaudeCode reports whether Claude Code is configured for a Z.AI plan,
// including any ANTHROPIC_DEFAULT_*_MODEL tier mapping that is present.
func DetectClaudeCode(home string) (Detection, error) {
	s, err := readJSONMap(Tools[0].ConfigPath(home))
	if err != nil {
		return Detection{}, err
	}
	env, ok := s["env"].(map[string]interface{})
	if !ok {
		return Detection{}, nil
	}
	key, _ := env["ANTHROPIC_AUTH_TOKEN"].(string)
	if key == "" {
		return Detection{}, nil
	}
	baseURL, _ := env["ANTHROPIC_BASE_URL"].(string)
	plan, _ := planFromBaseURL(baseURL)
	d := Detection{Configured: true, Plan: plan, APIKey: key}
	haiku, _ := env["ANTHROPIC_DEFAULT_HAIKU_MODEL"].(string)
	sonnet, _ := env["ANTHROPIC_DEFAULT_SONNET_MODEL"].(string)
	opus, _ := env["ANTHROPIC_DEFAULT_OPUS_MODEL"].(string)
	if haiku != "" || sonnet != "" || opus != "" {
		d.ModelMap = &ModelMap{Haiku: haiku, Sonnet: sonnet, Opus: opus}
	}
	return d, nil
}

// --- OpenCode (~/.config/opencode/opencode.json) ---

func opencodeProviderName(plan string) string {
	if plan == PlanChina {
		return "zhipuai-coding-plan"
	}
	return "zai-coding-plan"
}

// LoadOpenCode writes the GLM Coding Plan into OpenCode's provider config.
func LoadOpenCode(home, plan, key string) error {
	path := Tools[1].ConfigPath(home)
	m, err := readJSONMap(path)
	if err != nil {
		return err
	}
	name := opencodeProviderName(plan)

	providers := map[string]interface{}{}
	if old, ok := m["provider"].(map[string]interface{}); ok {
		for k, v := range old {
			if k != "zai-coding-plan" && k != "zhipuai-coding-plan" {
				providers[k] = v
			}
		}
	}
	providers[name] = map[string]interface{}{
		"options": map[string]interface{}{"apiKey": key},
	}
	m["$schema"] = "https://opencode.ai/config.json"
	m["provider"] = providers
	m["model"] = name + "/glm-4.6"
	m["small_model"] = name + "/glm-4.5-air"
	return writeJSONMap(path, m)
}

// UnloadOpenCode removes the Z.AI coding-plan provider and model defaults.
func UnloadOpenCode(home string) error {
	path := Tools[1].ConfigPath(home)
	m, err := readJSONMap(path)
	if err != nil {
		return err
	}
	if prov, ok := m["provider"].(map[string]interface{}); ok {
		delete(prov, "zai-coding-plan")
		delete(prov, "zhipuai-coding-plan")
		if len(prov) == 0 {
			delete(m, "provider")
		} else {
			m["provider"] = prov
		}
	}
	if model, ok := m["model"].(string); ok && strings.Contains(model, "coding-plan") {
		delete(m, "model")
	}
	if sm, ok := m["small_model"].(string); ok && strings.Contains(sm, "coding-plan") {
		delete(m, "small_model")
	}
	return writeJSONMap(path, m)
}

// DetectOpenCode reports OpenCode's Z.AI plan configuration.
func DetectOpenCode(home string) (Detection, error) {
	m, err := readJSONMap(Tools[1].ConfigPath(home))
	if err != nil {
		return Detection{}, err
	}
	prov, ok := m["provider"].(map[string]interface{})
	if !ok {
		return Detection{}, nil
	}
	for name, plan := range map[string]string{
		"zai-coding-plan":     PlanGlobal,
		"zhipuai-coding-plan": PlanChina,
	} {
		if entry, ok := prov[name].(map[string]interface{}); ok {
			opts, _ := entry["options"].(map[string]interface{})
			key, _ := opts["apiKey"].(string)
			return Detection{Configured: true, Plan: plan, APIKey: key}, nil
		}
	}
	return Detection{}, nil
}

// --- Crush (~/.config/crush/crush.json) ---

// LoadCrush writes the GLM Coding Plan into Crush's providers.zai entry.
func LoadCrush(home, plan, key string) error {
	path := Tools[2].ConfigPath(home)
	m, err := readJSONMap(path)
	if err != nil {
		return err
	}
	prov := objectField(m, "providers")
	prov["zai"] = map[string]interface{}{
		"id":       "zai",
		"name":     "ZAI Provider",
		"base_url": CodingBaseURL(plan),
		"api_key":  key,
	}
	m["providers"] = prov
	return writeJSONMap(path, m)
}

// UnloadCrush removes providers.zai.
func UnloadCrush(home string) error {
	path := Tools[2].ConfigPath(home)
	m, err := readJSONMap(path)
	if err != nil {
		return err
	}
	if prov, ok := m["providers"].(map[string]interface{}); ok {
		delete(prov, "zai")
		if len(prov) == 0 {
			delete(m, "providers")
		} else {
			m["providers"] = prov
		}
	}
	return writeJSONMap(path, m)
}

// DetectCrush reports Crush's Z.AI plan configuration.
func DetectCrush(home string) (Detection, error) {
	m, err := readJSONMap(Tools[2].ConfigPath(home))
	if err != nil {
		return Detection{}, err
	}
	prov, ok := m["providers"].(map[string]interface{})
	if !ok {
		return Detection{}, nil
	}
	zai, ok := prov["zai"].(map[string]interface{})
	if !ok {
		return Detection{}, nil
	}
	key, _ := zai["api_key"].(string)
	baseURL, _ := zai["base_url"].(string)
	plan, _ := planFromBaseURL(baseURL)
	return Detection{Configured: true, Plan: plan, APIKey: key}, nil
}

// --- Factory Droid (~/.factory/settings.json) ---

func factoryDisplayName(plan, protocol string) string {
	planName := "GLM Coding Plan Global"
	if plan == PlanChina {
		planName = "GLM Coding Plan China"
	}
	protoName := "Anthropic"
	if protocol != "anthropic" {
		protoName = "Openai"
	}
	return fmt.Sprintf("GLM-4.7 [%s] - %s", planName, protoName)
}

// LoadFactoryDroid writes the GLM Coding Plan into Factory Droid as two
// customModels entries (Anthropic + OpenAI protocols).
func LoadFactoryDroid(home, plan, key string) error {
	path := Tools[3].ConfigPath(home)
	m, err := readJSONMap(path)
	if err != nil {
		return err
	}
	existing, _ := m["customModels"].([]interface{})
	var models []interface{}
	for _, e := range existing {
		if em, ok := e.(map[string]interface{}); ok {
			if dn, _ := em["displayName"].(string); strings.Contains(dn, "GLM Coding Plan") {
				continue
			}
		}
		models = append(models, e)
	}
	maxTokens := 131072
	models = append(models,
		map[string]interface{}{
			"displayName":     factoryDisplayName(plan, "anthropic"),
			"model":           "glm-4.7",
			"baseUrl":         AnthropicBaseURL(plan),
			"apiKey":          key,
			"provider":        "anthropic",
			"maxOutputTokens": maxTokens,
		},
		map[string]interface{}{
			"displayName":     factoryDisplayName(plan, "openai"),
			"model":           "glm-4.7",
			"baseUrl":         CodingBaseURL(plan),
			"apiKey":          key,
			"provider":        "generic-chat-completion-api",
			"maxOutputTokens": maxTokens,
		},
	)
	m["customModels"] = models
	return writeJSONMap(path, m)
}

// UnloadFactoryDroid removes GLM Coding Plan custom models.
func UnloadFactoryDroid(home string) error {
	path := Tools[3].ConfigPath(home)
	m, err := readJSONMap(path)
	if err != nil {
		return err
	}
	existing, ok := m["customModels"].([]interface{})
	if !ok {
		return nil
	}
	var kept []interface{}
	for _, e := range existing {
		if em, ok := e.(map[string]interface{}); ok {
			if dn, _ := em["displayName"].(string); strings.Contains(dn, "GLM Coding Plan") {
				continue
			}
		}
		kept = append(kept, e)
	}
	if len(kept) == 0 {
		delete(m, "customModels")
	} else {
		m["customModels"] = kept
	}
	return writeJSONMap(path, m)
}

// DetectFactoryDroid reports Factory Droid's Z.AI plan configuration.
func DetectFactoryDroid(home string) (Detection, error) {
	m, err := readJSONMap(Tools[3].ConfigPath(home))
	if err != nil {
		return Detection{}, err
	}
	models, ok := m["customModels"].([]interface{})
	if !ok {
		return Detection{}, nil
	}
	for _, e := range models {
		em, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		dn, _ := em["displayName"].(string)
		if !strings.Contains(dn, "GLM Coding Plan") {
			continue
		}
		key, _ := em["apiKey"].(string)
		baseURL, _ := em["baseUrl"].(string)
		plan, _ := planFromBaseURL(baseURL)
		return Detection{Configured: true, Plan: plan, APIKey: key}, nil
	}
	return Detection{}, nil
}

// --- Cursor (settings.json, path varies by OS — see cursorConfigPath) ---

// LoadCursor writes the GLM Coding Plan into Cursor's settings as a simple
// {apiKey, baseURL} pair — Cursor's config shape is far simpler than the
// other tools', it has no env/provider nesting.
func LoadCursor(home, plan, key string) error {
	path := Tools[4].ConfigPath(home)
	m, err := readJSONMap(path)
	if err != nil {
		return err
	}
	m["apiKey"] = key
	m["baseURL"] = AnthropicBaseURL(plan)
	return writeJSONMap(path, m)
}

// UnloadCursor removes the Z.AI apiKey/baseURL from Cursor's settings.
func UnloadCursor(home string) error {
	path := Tools[4].ConfigPath(home)
	m, err := readJSONMap(path)
	if err != nil {
		return err
	}
	delete(m, "apiKey")
	delete(m, "baseURL")
	return writeJSONMap(path, m)
}

// DetectCursor reports Cursor's Z.AI configuration state.
func DetectCursor(home string) (Detection, error) {
	m, err := readJSONMap(Tools[4].ConfigPath(home))
	if err != nil {
		return Detection{}, err
	}
	key, _ := m["apiKey"].(string)
	if key == "" {
		return Detection{}, nil
	}
	baseURL, _ := m["baseURL"].(string)
	plan, _ := planFromBaseURL(baseURL)
	return Detection{Configured: true, Plan: plan, APIKey: key}, nil
}

// --- Dispatch by tool ---

// Load writes the plan into the named tool's config.
func Load(home, toolID, plan, key string) error {
	t, err := FindTool(toolID)
	if err != nil {
		return err
	}
	switch t.ID {
	case "claude-code":
		return LoadClaudeCode(home, plan, key)
	case "opencode":
		return LoadOpenCode(home, plan, key)
	case "crush":
		return LoadCrush(home, plan, key)
	case "factory-droid":
		return LoadFactoryDroid(home, plan, key)
	case "cursor":
		return LoadCursor(home, plan, key)
	}
	return fmt.Errorf("unsupported tool %q", toolID)
}

// Unload removes the plan from the named tool's config.
func Unload(home, toolID string) error {
	t, err := FindTool(toolID)
	if err != nil {
		return err
	}
	switch t.ID {
	case "claude-code":
		return UnloadClaudeCode(home)
	case "opencode":
		return UnloadOpenCode(home)
	case "crush":
		return UnloadCrush(home)
	case "factory-droid":
		return UnloadFactoryDroid(home)
	case "cursor":
		return UnloadCursor(home)
	}
	return fmt.Errorf("unsupported tool %q", toolID)
}

// Detect reports the named tool's Z.AI configuration state.
func Detect(home, toolID string) (Detection, error) {
	t, err := FindTool(toolID)
	if err != nil {
		return Detection{}, err
	}
	switch t.ID {
	case "claude-code":
		return DetectClaudeCode(home)
	case "opencode":
		return DetectOpenCode(home)
	case "crush":
		return DetectCrush(home)
	case "factory-droid":
		return DetectFactoryDroid(home)
	case "cursor":
		return DetectCursor(home)
	}
	return Detection{}, fmt.Errorf("unsupported tool %q", toolID)
}
