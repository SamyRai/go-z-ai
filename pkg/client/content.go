package client

import "encoding/json"

// Content-part types of a multimodal message.
const (
	PartText     = "text"
	PartImageURL = "image_url"
	PartVideoURL = "video_url"
	PartFileURL  = "file_url"
)

// ContentPart is one element of a multimodal message's content array.
type ContentPart struct {
	Type     string   `json:"type"` // one of the Part* constants
	Text     string   `json:"text,omitempty"`
	ImageURL *URLPart `json:"image_url,omitempty"`
	VideoURL *URLPart `json:"video_url,omitempty"`
	FileURL  *URLPart `json:"file_url,omitempty"`
}

// URLPart references media inside a ContentPart. URL may be an https:// link
// or a data: URI (base64).
type URLPart struct {
	URL string `json:"url"`
}

// mediaKinds maps each media part type to the Message field holding its
// URLs and the ContentPart field carrying it on the wire — the one place the
// two shapes are related, used by both MarshalJSON and UnmarshalJSON.
var mediaKinds = []struct {
	partType string
	urls     func(*Message) *[]string
	slot     func(*ContentPart) **URLPart
}{
	{PartImageURL, func(m *Message) *[]string { return &m.Images }, func(p *ContentPart) **URLPart { return &p.ImageURL }},
	{PartVideoURL, func(m *Message) *[]string { return &m.Videos }, func(p *ContentPart) **URLPart { return &p.VideoURL }},
	{PartFileURL, func(m *Message) *[]string { return &m.Files }, func(p *ContentPart) **URLPart { return &p.FileURL }},
}

// messageWire is Message's JSON shape with Content typed as any, so it can
// be either a plain string or a content-parts array.
type messageWire struct {
	Role             string     `json:"role"`
	Content          any        `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	Name             string     `json:"name,omitempty"`
}

// MarshalJSON emits Content as a plain string, or — when the message carries
// media — as a content-parts array: the text part first, then one part per
// image, video, and file URL.
func (m Message) MarshalJSON() ([]byte, error) {
	wire := messageWire{
		Role:             m.Role,
		Content:          m.Content,
		ReasoningContent: m.ReasoningContent,
		ToolCalls:        m.ToolCalls,
		ToolCallID:       m.ToolCallID,
		Name:             m.Name,
	}
	var parts []ContentPart
	for _, k := range mediaKinds {
		for _, url := range *k.urls(&m) {
			p := ContentPart{Type: k.partType}
			*k.slot(&p) = &URLPart{URL: url}
			parts = append(parts, p)
		}
	}
	if len(parts) > 0 {
		if m.Content != "" {
			parts = append([]ContentPart{{Type: PartText, Text: m.Content}}, parts...)
		}
		wire.Content = parts
	}
	return json.Marshal(wire)
}

// UnmarshalJSON accepts either wire shape of Content — a plain string or a
// content-parts array — reconstituting Images/Videos/Files from media parts.
func (m *Message) UnmarshalJSON(data []byte) error {
	var raw struct {
		messageWire
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*m = Message{
		Role:             raw.Role,
		ReasoningContent: raw.ReasoningContent,
		ToolCalls:        raw.ToolCalls,
		ToolCallID:       raw.ToolCallID,
		Name:             raw.Name,
	}
	if len(raw.Content) == 0 || string(raw.Content) == "null" {
		return nil
	}
	if err := json.Unmarshal(raw.Content, &m.Content); err == nil {
		return nil
	}
	var parts []ContentPart
	if err := json.Unmarshal(raw.Content, &parts); err != nil {
		return err
	}
	for _, p := range parts {
		if p.Type == PartText {
			m.Content += p.Text
			continue
		}
		for _, k := range mediaKinds {
			if u := *k.slot(&p); p.Type == k.partType && u != nil {
				*k.urls(m) = append(*k.urls(m), u.URL)
			}
		}
	}
	return nil
}
