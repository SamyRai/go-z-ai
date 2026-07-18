package client

import (
	"context"
	"fmt"
	"net/url"
)

// VoiceService manages custom voice clones for GLM-TTS: create a clone from
// a sample audio file (uploaded via FilesService.Upload with
// FilePurposeVoiceCloneInput), then reference the resulting voice name in
// AudioService.Speech. Confirmed against docs.bigmodel.cn's live OpenAPI
// spec (https://docs.bigmodel.cn/openapi/openapi.json:
// POST /paas/v4/voice/clone, /voice/delete; GET /voice/list). Not
// mentioned in docs.z.ai's doc index at all — same "only documented on the
// China platform" situation as Embeddings/Moderations, and likewise
// unconfirmed against a live success response (see docs/en/roadmap.md).
type VoiceService struct {
	client *Client
}

// VoiceCloneModel is the only value VoiceCloneRequest.Model currently accepts.
const VoiceCloneModel = "glm-tts-clone"

// Voice types (VoiceInfo.VoiceType, VoiceListRequest.VoiceType).
const (
	VoiceTypeOfficial = "OFFICIAL" // system-provided voices
	VoiceTypePrivate  = "PRIVATE"  // caller's own clones
)

// VoiceCloneRequest clones a voice from a sample audio file. VoiceName,
// Input, FileID, and Model are all required.
type VoiceCloneRequest struct {
	Model     string `json:"model"`          // defaults to VoiceCloneModel when empty
	VoiceName string `json:"voice_name"`     // unique name to assign the cloned voice
	Text      string `json:"text,omitempty"` // transcript of the sample audio, optional
	Input     string `json:"input"`          // text to synthesize as a preview using the new voice
	FileID    string `json:"file_id"`        // sample audio's file ID (upload via Files, purpose voice-clone-input); <=10MB, 3-30s recommended
	RequestID string `json:"request_id,omitempty"`
}

// VoiceCloneResponse is the result of VoiceService.Clone.
type VoiceCloneResponse struct {
	Voice       string `json:"voice"`        // the new voice's ID, for use as AudioSpeechRequest.Voice
	FileID      string `json:"file_id"`      // preview audio's file ID
	FilePurpose string `json:"file_purpose"` // "voice-clone-output"
	RequestID   string `json:"request_id"`
}

// Clone creates a new voice clone. req.VoiceName, req.Input, and
// req.FileID are required; req.Model defaults to VoiceCloneModel when empty.
func (s *VoiceService) Clone(ctx context.Context, req VoiceCloneRequest) (*VoiceCloneResponse, error) {
	if req.Model == "" {
		req.Model = VoiceCloneModel
	}
	if req.VoiceName == "" {
		return nil, fmt.Errorf("voice_name is required")
	}
	if req.Input == "" {
		return nil, fmt.Errorf("input is required")
	}
	if req.FileID == "" {
		return nil, fmt.Errorf("file_id is required")
	}

	var resp VoiceCloneResponse
	if err := s.client.doRequest(ctx, "POST", "/voice/clone", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to clone voice: %w", err)
	}
	return &resp, nil
}

// VoiceDeleteResponse is the result of VoiceService.Delete.
type VoiceDeleteResponse struct {
	Voice      string `json:"voice"`
	UpdateTime string `json:"update_time"`
}

// Delete removes a cloned voice by its ID (VoiceCloneResponse.Voice).
func (s *VoiceService) Delete(ctx context.Context, voice string) (*VoiceDeleteResponse, error) {
	if voice == "" {
		return nil, fmt.Errorf("voice is required")
	}

	var resp VoiceDeleteResponse
	if err := s.client.doRequest(ctx, "POST", "/voice/delete", map[string]string{"voice": voice}, &resp); err != nil {
		return nil, fmt.Errorf("failed to delete voice: %w", err)
	}
	return &resp, nil
}

// VoiceInfo is one voice in VoiceService.List's result.
type VoiceInfo struct {
	Voice       string `json:"voice"`
	VoiceName   string `json:"voice_name"`
	VoiceType   string `json:"voice_type"` // VoiceTypeOfficial or VoiceTypePrivate
	DownloadURL string `json:"download_url"`
	CreateTime  string `json:"create_time"`
}

// List returns available voices, optionally filtered by name (fuzzy match)
// and/or type. Pass "" for either filter to skip it.
func (s *VoiceService) List(ctx context.Context, voiceName, voiceType string) ([]VoiceInfo, error) {
	endpoint := "/voice/list"
	q := url.Values{}
	if voiceName != "" {
		q.Set("voiceName", voiceName)
	}
	if voiceType != "" {
		q.Set("voiceType", voiceType)
	}
	if enc := q.Encode(); enc != "" {
		endpoint += "?" + enc
	}

	var resp struct {
		VoiceList []VoiceInfo `json:"voice_list"`
	}
	if err := s.client.doRequest(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, fmt.Errorf("failed to list voices: %w", err)
	}
	return resp.VoiceList, nil
}
