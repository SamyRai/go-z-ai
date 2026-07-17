package client

// Tool (function) parameter schemas are passed to GLM as JSON Schema, but the
// endpoint's schema parser is strict: a node containing `anyOf`, `oneOf`,
// `allOf`, or a `$ref`/`$defs` reference makes chat/completions return HTTP 500
// rather than a usable error (observed in the wild — see e.g.
// github.com/musistudio/claude-code-router#1474). Tools generated from typed
// languages emit exactly these constructs routinely: a nullable field becomes
// `anyOf: [{...}, {"type": "null"}]`, a reused struct becomes a `$ref` into
// `$defs`, and a composed type becomes `allOf`.
//
// SanitizeToolSchemas rewrites such schemas into the flat, permissive subset
// GLM accepts, keeping as much type/description information as it can:
//
//   - `$ref` into `$defs`/`definitions` (or any local JSON pointer) is inlined;
//     the `$defs`/`definitions` container is then dropped. A ref that can't be
//     resolved, or a cyclic one, collapses to an unconstrained schema.
//   - `allOf` members are merged into the enclosing schema (properties unioned,
//     required unioned).
//   - `anyOf`/`oneOf` collapses to a single branch: a `{"type": "null"}` branch
//     is dropped (the field just becomes non-nullable), a lone remaining branch
//     is inlined, and a genuine union keeps a shared `type` when every branch
//     agrees or is left unconstrained otherwise.
//   - Draft keywords GLM doesn't parse (`$schema`, `$id`, `$anchor`, `not`,
//     `if`/`then`/`else`) are stripped.
//
// The client applies this to req.Tools before every chat request unless
// Config.DisableToolSchemaCompat is set. It is exported so callers assembling
// requests elsewhere can apply the same normalization explicitly. It is a
// no-op for schemas already within the supported subset, and never mutates its
// input — the returned tools are a fresh slice with freshly built parameter
// maps.
func SanitizeToolSchemas(tools []Tool) []Tool {
	if len(tools) == 0 {
		return tools
	}
	out := make([]Tool, len(tools))
	for i, t := range tools {
		out[i] = t
		if t.Function != nil && len(t.Function.Parameters) > 0 {
			fn := *t.Function
			fn.Parameters = sanitizeParameters(t.Function.Parameters)
			out[i].Function = &fn
		}
	}
	return out
}

// sanitizeParameters normalizes a single function's top-level parameter schema.
// The top level is expected to be an object schema; if a combinator/ref
// collapses it to something else, it falls back to an unconstrained object so
// the request still carries a valid `parameters` value.
func sanitizeParameters(params map[string]any) map[string]any {
	res := sanitizeNode(params, params, map[string]bool{})
	if m, ok := res.(map[string]any); ok {
		return m
	}
	return map[string]any{"type": "object"}
}

// sanitizeNode dispatches on the JSON shape. root is the enclosing schema used
// to resolve local `$ref` pointers; active tracks refs currently being resolved
// to break cycles.
func sanitizeNode(node any, root map[string]any, active map[string]bool) any {
	switch v := node.(type) {
	case map[string]any:
		return sanitizeSchemaObject(v, root, active)
	case []any:
		arr := make([]any, len(v))
		for i, e := range v {
			arr[i] = sanitizeNode(e, root, active)
		}
		return arr
	default:
		return v
	}
}

// droppedKeywords are JSON Schema keywords GLM's parser rejects or ignores;
// they are removed outright during sanitization.
var droppedKeywords = map[string]bool{
	"$schema": true, "$id": true, "$anchor": true, "$comment": true,
	"$defs": true, "definitions": true,
	"not": true, "if": true, "then": true, "else": true,
}

// combinatorKeywords are handled explicitly (merged/collapsed), not copied.
var combinatorKeywords = map[string]bool{
	"allOf": true, "anyOf": true, "oneOf": true, "$ref": true,
}

// singleSchemaKeywords hold one nested schema to recurse into.
var singleSchemaKeywords = map[string]bool{
	"items": true, "additionalProperties": true,
	"contains": true, "propertyNames": true,
}

// schemaMapKeywords hold a map of name->schema to recurse into.
var schemaMapKeywords = map[string]bool{
	"properties": true, "patternProperties": true,
}

func sanitizeSchemaObject(m map[string]any, root map[string]any, active map[string]bool) any {
	// A `$ref` replaces the whole node with the (sanitized) referent, with any
	// sibling keywords layered on top.
	if ref, ok := stringField(m, "$ref"); ok {
		return resolveRef(ref, m, root, active)
	}

	// Combinators are merged into a base schema first, then the node's own
	// keywords are layered on top (own scalars win; properties/required union).
	base := map[string]any{}
	if members, ok := m["allOf"].([]any); ok {
		for _, member := range members {
			if sm, ok := sanitizeNode(member, root, active).(map[string]any); ok {
				mergeSchema(base, sm, false)
			}
		}
	}
	for _, key := range []string{"anyOf", "oneOf"} {
		if members, ok := m[key].([]any); ok {
			mergeSchema(base, collapseUnion(members, root, active), false)
		}
	}

	own := map[string]any{}
	for k, v := range m {
		switch {
		case droppedKeywords[k] || combinatorKeywords[k]:
			continue
		case schemaMapKeywords[k]:
			own[k] = sanitizeSchemaMap(v, root, active)
		case singleSchemaKeywords[k]:
			own[k] = sanitizeNode(v, root, active)
		case k == "prefixItems":
			own[k] = sanitizeNode(v, root, active)
		default:
			own[k] = v
		}
	}
	mergeSchema(base, own, true)
	return base
}

// resolveRef inlines a local `$ref`. src is the node carrying the ref (its
// sibling keywords are layered over the referent); active guards against
// cycles.
func resolveRef(ref string, src, root map[string]any, active map[string]bool) any {
	if active[ref] {
		return map[string]any{} // cycle — unconstrained
	}
	target := lookupPointer(ref, root)
	if target == nil {
		// Unresolvable ref: keep whatever sibling keywords the node had.
		out := map[string]any{}
		for k, v := range src {
			if k == "$ref" || droppedKeywords[k] {
				continue
			}
			out[k] = sanitizeNode(v, root, active)
		}
		return out
	}
	active[ref] = true
	resolved := sanitizeNode(target, root, active)
	delete(active, ref)

	rm, ok := resolved.(map[string]any)
	if !ok {
		return resolved
	}
	out := cloneSchema(rm)
	for k, v := range src {
		if k == "$ref" || droppedKeywords[k] {
			continue
		}
		out[k] = sanitizeNode(v, root, active)
	}
	return out
}

// lookupPointer resolves a local JSON pointer ("#/$defs/Foo") against root.
// Only same-document pointers are supported; anything else returns nil.
func lookupPointer(ref string, root map[string]any) map[string]any {
	if len(ref) < 2 || ref[0] != '#' {
		return nil
	}
	path := ref[1:]
	if path == "" || path == "/" {
		return root
	}
	if path[0] != '/' {
		return nil
	}
	cur := any(root)
	for _, raw := range splitPointer(path[1:]) {
		token := unescapePointer(raw)
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		next, ok := m[token]
		if !ok {
			return nil
		}
		cur = next
	}
	if m, ok := cur.(map[string]any); ok {
		return m
	}
	return nil
}

func splitPointer(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return append(parts, s[start:])
}

// unescapePointer decodes JSON Pointer escaping (~1 -> /, ~0 -> ~).
func unescapePointer(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '~' && i+1 < len(s) {
			switch s[i+1] {
			case '1':
				out = append(out, '/')
				i++
				continue
			case '0':
				out = append(out, '~')
				i++
				continue
			}
		}
		out = append(out, s[i])
	}
	return string(out)
}

// collapseUnion reduces an anyOf/oneOf branch list to a single schema.
func collapseUnion(members []any, root map[string]any, active map[string]bool) map[string]any {
	var nonNull []map[string]any
	for _, member := range members {
		sm, ok := sanitizeNode(member, root, active).(map[string]any)
		if !ok || isNullSchema(sm) {
			continue
		}
		nonNull = append(nonNull, sm)
	}
	switch len(nonNull) {
	case 0:
		return map[string]any{}
	case 1:
		return nonNull[0]
	default:
		if t := commonType(nonNull); t != "" {
			return map[string]any{"type": t}
		}
		return map[string]any{}
	}
}

// isNullSchema reports whether a branch only permits JSON null, so it can be
// dropped from a union.
func isNullSchema(m map[string]any) bool {
	t, ok := m["type"]
	if !ok {
		return false
	}
	if s, ok := t.(string); ok {
		return s == "null"
	}
	return false
}

// commonType returns the type shared by every branch, or "" if they differ or
// any branch is untyped or multi-typed.
func commonType(schemas []map[string]any) string {
	common := ""
	for _, s := range schemas {
		t, ok := s["type"].(string)
		if !ok {
			return ""
		}
		if common == "" {
			common = t
		} else if common != t {
			return ""
		}
	}
	return common
}

// mergeSchema layers src onto dst. `properties` maps and `required` lists are
// unioned; for every other keyword src overwrites dst only when srcWins is set
// (otherwise the existing dst value is kept).
func mergeSchema(dst, src map[string]any, srcWins bool) {
	for k, v := range src {
		switch k {
		case "properties":
			sp, _ := v.(map[string]any)
			dp, _ := dst["properties"].(map[string]any)
			if dp == nil {
				dp = map[string]any{}
			}
			for pk, pv := range sp {
				if _, exists := dp[pk]; !exists || srcWins {
					dp[pk] = pv
				}
			}
			dst["properties"] = dp
		case "required":
			dst["required"] = unionRequired(dst["required"], v)
		default:
			if _, exists := dst[k]; !exists || srcWins {
				dst[k] = v
			}
		}
	}
}

// unionRequired merges two `required` arrays, preserving order and dropping
// duplicates.
func unionRequired(a, b any) []any {
	seen := map[string]bool{}
	var out []any
	for _, list := range []any{a, b} {
		arr, ok := list.([]any)
		if !ok {
			continue
		}
		for _, item := range arr {
			if name, ok := item.(string); ok {
				if seen[name] {
					continue
				}
				seen[name] = true
			}
			out = append(out, item)
		}
	}
	return out
}

// sanitizeSchemaMap sanitizes each value of a name->schema map (properties,
// patternProperties).
func sanitizeSchemaMap(v any, root map[string]any, active map[string]bool) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	out := make(map[string]any, len(m))
	for name, sub := range m {
		out[name] = sanitizeNode(sub, root, active)
	}
	return out
}

// cloneSchema shallow-copies a map so sibling keywords can be layered onto an
// inlined referent without mutating the shared referent.
func cloneSchema(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func stringField(m map[string]any, key string) (string, bool) {
	v, ok := m[key].(string)
	return v, ok
}
