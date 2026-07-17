package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// schema is a small helper to write JSON-Schema fixtures inline.
func schema(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("bad fixture JSON: %v", err)
	}
	return m
}

func sanitized(t *testing.T, raw string) map[string]any {
	t.Helper()
	return sanitizeParameters(schema(t, raw))
}

// A schema already within GLM's supported subset must pass through unchanged.
func TestSanitizeNoopForFlatSchema(t *testing.T) {
	in := schema(t, `{
		"type": "object",
		"properties": {"city": {"type": "string", "description": "City name"}},
		"required": ["city"]
	}`)
	got := sanitizeParameters(in)
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("flat schema changed:\n in: %v\nout: %v", in, got)
	}
}

// A nullable field expressed as anyOf:[T, null] collapses to plain T, keeping
// the sibling description.
func TestSanitizeNullableAnyOf(t *testing.T) {
	got := sanitized(t, `{
		"type": "object",
		"properties": {
			"nickname": {
				"description": "optional nickname",
				"anyOf": [{"type": "string"}, {"type": "null"}]
			}
		}
	}`)
	props := got["properties"].(map[string]any)
	nick := props["nickname"].(map[string]any)
	if nick["type"] != "string" {
		t.Errorf("expected type string, got %v", nick["type"])
	}
	if nick["description"] != "optional nickname" {
		t.Errorf("description lost: %v", nick["description"])
	}
	if _, ok := nick["anyOf"]; ok {
		t.Error("anyOf should have been removed")
	}
}

// oneOf with distinct types drops the union and leaves the field unconstrained
// (no type), but retains sibling keywords.
func TestSanitizeOneOfDistinctTypes(t *testing.T) {
	got := sanitized(t, `{
		"type": "object",
		"properties": {
			"value": {
				"description": "string or number",
				"oneOf": [{"type": "string"}, {"type": "number"}]
			}
		}
	}`)
	val := got["properties"].(map[string]any)["value"].(map[string]any)
	if _, ok := val["oneOf"]; ok {
		t.Error("oneOf should have been removed")
	}
	if _, ok := val["type"]; ok {
		t.Errorf("expected no type for divergent union, got %v", val["type"])
	}
	if val["description"] != "string or number" {
		t.Errorf("description lost: %v", val["description"])
	}
}

// A union where every non-null branch shares a type keeps that type.
func TestSanitizeUnionCommonType(t *testing.T) {
	got := sanitized(t, `{
		"type": "object",
		"properties": {
			"id": {"anyOf": [{"type": "string", "minLength": 1}, {"type": "string", "format": "uuid"}, {"type": "null"}]}
		}
	}`)
	id := got["properties"].(map[string]any)["id"].(map[string]any)
	if id["type"] != "string" {
		t.Errorf("expected shared type string, got %v", id["type"])
	}
}

// allOf members merge into the enclosing schema: properties unioned, required
// unioned.
func TestSanitizeAllOfMerge(t *testing.T) {
	got := sanitized(t, `{
		"allOf": [
			{"type": "object", "properties": {"a": {"type": "string"}}, "required": ["a"]},
			{"properties": {"b": {"type": "number"}}, "required": ["b"]}
		]
	}`)
	if got["type"] != "object" {
		t.Errorf("expected type object, got %v", got["type"])
	}
	props := got["properties"].(map[string]any)
	if _, ok := props["a"]; !ok {
		t.Error("property a missing after allOf merge")
	}
	if _, ok := props["b"]; !ok {
		t.Error("property b missing after allOf merge")
	}
	req := got["required"].([]any)
	if len(req) != 2 {
		t.Errorf("expected 2 required fields, got %v", req)
	}
}

// $ref into $defs is inlined and the $defs container is dropped.
func TestSanitizeRefInlining(t *testing.T) {
	got := sanitized(t, `{
		"type": "object",
		"properties": {"user": {"$ref": "#/$defs/User"}},
		"$defs": {"User": {"type": "object", "properties": {"name": {"type": "string"}}}}
	}`)
	if _, ok := got["$defs"]; ok {
		t.Error("$defs should have been dropped")
	}
	user := got["properties"].(map[string]any)["user"].(map[string]any)
	if user["type"] != "object" {
		t.Errorf("ref not inlined, got %v", user)
	}
	if _, ok := user["properties"].(map[string]any)["name"]; !ok {
		t.Error("inlined ref lost its properties")
	}
	if _, ok := user["$ref"]; ok {
		t.Error("$ref keyword should be gone after inlining")
	}
}

// A $ref carrying sibling keywords (e.g. description) layers them over the
// inlined referent.
func TestSanitizeRefWithSiblingKeywords(t *testing.T) {
	got := sanitized(t, `{
		"type": "object",
		"properties": {"u": {"$ref": "#/$defs/U", "description": "the user"}},
		"$defs": {"U": {"type": "object", "description": "base"}}
	}`)
	u := got["properties"].(map[string]any)["u"].(map[string]any)
	if u["description"] != "the user" {
		t.Errorf("sibling description should win, got %v", u["description"])
	}
}

// A cyclic $ref must terminate and collapse to an unconstrained schema rather
// than recursing forever.
func TestSanitizeCyclicRef(t *testing.T) {
	got := sanitized(t, `{
		"type": "object",
		"properties": {"node": {"$ref": "#/$defs/Node"}},
		"$defs": {"Node": {"type": "object", "properties": {"next": {"$ref": "#/$defs/Node"}}}}
	}`)
	node := got["properties"].(map[string]any)["node"].(map[string]any)
	next := node["properties"].(map[string]any)["next"]
	// The cyclic inner ref collapses to {} (unconstrained), which is valid.
	if next == nil {
		t.Error("expected cyclic ref to collapse to a schema, got nil")
	}
	// The whole thing must be JSON-serializable (no cycles).
	if _, err := json.Marshal(got); err != nil {
		t.Fatalf("result not serializable (cycle leaked?): %v", err)
	}
}

// An unresolvable $ref keeps the node's own sibling keywords instead of erroring.
func TestSanitizeUnresolvableRef(t *testing.T) {
	got := sanitized(t, `{
		"type": "object",
		"properties": {"x": {"$ref": "#/$defs/Missing", "description": "kept"}}
	}`)
	x := got["properties"].(map[string]any)["x"].(map[string]any)
	if x["description"] != "kept" {
		t.Errorf("expected sibling kept, got %v", x)
	}
	if _, ok := x["$ref"]; ok {
		t.Error("dangling $ref should have been removed")
	}
}

// Draft keywords GLM's parser ignores/rejects are stripped.
func TestSanitizeStripsDraftKeywords(t *testing.T) {
	got := sanitized(t, `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id": "urn:x",
		"type": "object",
		"not": {"type": "string"},
		"properties": {"a": {"type": "string"}}
	}`)
	for _, k := range []string{"$schema", "$id", "not"} {
		if _, ok := got[k]; ok {
			t.Errorf("keyword %q should have been stripped", k)
		}
	}
	if got["type"] != "object" {
		t.Errorf("real keywords should survive, got %v", got)
	}
}

// Nested arrays (items) are recursed into so unions deep in the tree are
// flattened too.
func TestSanitizeNestedArrayItems(t *testing.T) {
	got := sanitized(t, `{
		"type": "object",
		"properties": {
			"tags": {"type": "array", "items": {"anyOf": [{"type": "string"}, {"type": "null"}]}}
		}
	}`)
	items := got["properties"].(map[string]any)["tags"].(map[string]any)["items"].(map[string]any)
	if items["type"] != "string" {
		t.Errorf("nested anyOf under items not flattened, got %v", items)
	}
}

// Sanitizing is idempotent: a second pass changes nothing.
func TestSanitizeIdempotent(t *testing.T) {
	once := sanitized(t, `{
		"type": "object",
		"properties": {"u": {"$ref": "#/$defs/U"}, "n": {"anyOf": [{"type": "integer"}, {"type": "null"}]}},
		"$defs": {"U": {"type": "object", "properties": {"name": {"type": "string"}}}}
	}`)
	twice := sanitizeParameters(once)
	if !reflect.DeepEqual(once, twice) {
		t.Fatalf("not idempotent:\nonce:  %v\ntwice: %v", once, twice)
	}
}

// SanitizeToolSchemas must not mutate the caller's tools or their parameter maps.
func TestSanitizeToolSchemasDoesNotMutateInput(t *testing.T) {
	params := schema(t, `{"type": "object", "properties": {"x": {"anyOf": [{"type": "string"}, {"type": "null"}]}}}`)
	tools := []Tool{{Type: "function", Function: &FunctionDef{Name: "f", Parameters: params}}}

	before, _ := json.Marshal(tools)
	out := SanitizeToolSchemas(tools)
	after, _ := json.Marshal(tools)

	if string(before) != string(after) {
		t.Fatalf("input mutated:\nbefore: %s\nafter:  %s", before, after)
	}
	// And the output actually changed (anyOf removed).
	outProps := out[0].Function.Parameters["properties"].(map[string]any)
	if _, ok := outProps["x"].(map[string]any)["anyOf"]; ok {
		t.Error("output should have flattened anyOf")
	}
}

// Empty/nil tool sets and tools without parameters are handled gracefully.
func TestSanitizeToolSchemasEdgeCases(t *testing.T) {
	if got := SanitizeToolSchemas(nil); got != nil {
		t.Errorf("nil in should return nil, got %v", got)
	}
	tools := []Tool{
		{Type: "function", Function: &FunctionDef{Name: "noparams"}},
		{Type: "function", Function: nil},
	}
	got := SanitizeToolSchemas(tools)
	if len(got) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(got))
	}
	// The no-parameter function's FunctionDef is passed through untouched.
	if got[0].Function != tools[0].Function {
		t.Error("expected untouched FunctionDef pointer for a tool without parameters")
	}
}

// End-to-end: Create sends flattened tool schemas by default, and honors the
// DisableToolSchemaCompat opt-out.
func TestChatCreateFlattensToolSchemasOnWire(t *testing.T) {
	toolWithUnion := func() []Tool {
		return []Tool{{Type: "function", Function: &FunctionDef{
			Name: "lookup",
			Parameters: schema(t, `{
				"type": "object",
				"properties": {"q": {"anyOf": [{"type": "string"}, {"type": "null"}]}}
			}`),
		}}}
	}

	capture := func(cfg Config) map[string]any {
		var body []byte
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ = io.ReadAll(r.Body)
			writeJSON(w, http.StatusOK, `{"id":"1","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`)
		}))
		defer srv.Close()

		c := newTestClient(t, srv.URL, cfg)
		req := ChatRequest{Model: "m", TopP: 0.95, Messages: []Message{{Role: "user", Content: "hi"}}, Tools: toolWithUnion()}
		if _, err := c.Chat().Create(context.Background(), req); err != nil {
			t.Fatalf("Create: %v", err)
		}
		var sent struct {
			Tools []struct {
				Function struct {
					Parameters map[string]any `json:"parameters"`
				} `json:"function"`
			} `json:"tools"`
		}
		if err := json.Unmarshal(body, &sent); err != nil {
			t.Fatalf("unmarshal request: %v", err)
		}
		return sent.Tools[0].Function.Parameters["properties"].(map[string]any)["q"].(map[string]any)
	}

	// Default: anyOf flattened away on the wire.
	def := capture(Config{MaxRetries: 0})
	if _, ok := def["anyOf"]; ok {
		t.Error("default Create should send flattened schema, anyOf still present")
	}
	if def["type"] != "string" {
		t.Errorf("expected flattened type string, got %v", def["type"])
	}

	// Opt-out: raw schema sent untouched.
	raw := capture(Config{MaxRetries: 0, DisableToolSchemaCompat: true})
	if _, ok := raw["anyOf"]; !ok {
		t.Error("DisableToolSchemaCompat should send the raw anyOf schema")
	}
}
