package client

import "encoding/json"

// ContentPart is one part of a multimodal message's content array — the
// OpenAI-compatible shape Z.AI's vision models (GLM-4.6V/4.5V) expect once a
// message carries more than plain text.
type ContentPart struct {
	Type     string        `json:"type"` // "text" or "image_url"
	Text     string        `json:"text,omitempty"`
	ImageURL *ImageURLPart `json:"image_url,omitempty"`
}

// ImageURLPart is the image reference inside a ContentPart of type
// "image_url". URL may be an https:// link or a data: URI (base64).
type ImageURLPart struct {
	URL string `json:"url"`
}

// messageWire mirrors Message's JSON shape but types Content as any, so
// MarshalJSON/UnmarshalJSON can switch between a plain string and a
// content-parts array without touching every other field.
type messageWire struct {
	Role       string     `json:"role"`
	Content    any        `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// MarshalJSON emits the plain-string wire shape Message has always used,
// unless Images is set, in which case Content becomes a content-parts array
// (text part first, then one image_url part per entry in Images) — the
// shape GLM-4.6V/4.5V expect for multimodal input.
func (m Message) MarshalJSON() ([]byte, error) {
	wire := messageWire{
		Role:       m.Role,
		ToolCalls:  m.ToolCalls,
		ToolCallID: m.ToolCallID,
		Name:       m.Name,
	}
	if len(m.Images) == 0 {
		wire.Content = m.Content
		return json.Marshal(wire)
	}

	parts := make([]ContentPart, 0, len(m.Images)+1)
	if m.Content != "" {
		parts = append(parts, ContentPart{Type: "text", Text: m.Content})
	}
	for _, url := range m.Images {
		parts = append(parts, ContentPart{Type: "image_url", ImageURL: &ImageURLPart{URL: url}})
	}
	wire.Content = parts
	return json.Marshal(wire)
}

// UnmarshalJSON accepts either wire shape Content can take: a plain string,
// or a content-parts array (text + image_url parts), reconstituting Images
// from any image_url parts found.
func (m *Message) UnmarshalJSON(data []byte) error {
	var raw struct {
		Role       string          `json:"role"`
		Content    json.RawMessage `json:"content"`
		ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`
		ToolCallID string          `json:"tool_call_id,omitempty"`
		Name       string          `json:"name,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.Role = raw.Role
	m.ToolCalls = raw.ToolCalls
	m.ToolCallID = raw.ToolCallID
	m.Name = raw.Name
	m.Content = ""
	m.Images = nil

	if len(raw.Content) == 0 {
		return nil
	}

	var asString string
	if err := json.Unmarshal(raw.Content, &asString); err == nil {
		m.Content = asString
		return nil
	}

	var parts []ContentPart
	if err := json.Unmarshal(raw.Content, &parts); err != nil {
		return err
	}
	for _, p := range parts {
		switch p.Type {
		case "text":
			m.Content += p.Text
		case "image_url":
			if p.ImageURL != nil {
				m.Images = append(m.Images, p.ImageURL.URL)
			}
		}
	}
	return nil
}
