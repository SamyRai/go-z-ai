package client

import (
	"context"
	"errors"
	"fmt"
)

// AudioService handles audio transcription and text-to-speech.
type AudioService struct {
	client *Client
}

// AudioTranscriptionRequest is the request for audio transcription. Exactly
// one of FileData or FileBase64 must be set; if both are set, FileData wins.
type AudioTranscriptionRequest struct {
	FileName   string // required when FileData is set, e.g. "clip.wav"
	FileData   []byte // raw audio bytes (.wav or .mp3, <=25MB, <=30s)
	FileBase64 string // alternative to FileData

	Model     string   // defaults to DefaultASRModel
	Prompt    string   // previous transcription context, <8000 chars recommended
	Hotwords  []string // domain vocabulary, max 100 items
	RequestID string
	UserID    string
}

// AudioTranscriptionResponse is the non-streaming transcription result.
type AudioTranscriptionResponse struct {
	ID        string `json:"id"`
	Created   int64  `json:"created"`
	RequestID string `json:"request_id"`
	Model     string `json:"model"`
	Text      string `json:"text"`
}

// Transcribe uploads an audio clip (FileData) or passes it inline
// (FileBase64) and returns its transcription. Non-streaming.
func (s *AudioService) Transcribe(ctx context.Context, req AudioTranscriptionRequest) (*AudioTranscriptionResponse, error) {
	if len(req.FileData) == 0 && req.FileBase64 == "" {
		return nil, errors.New("file data or file_base64 is required")
	}
	if req.Model == "" {
		req.Model = DefaultASRModel
	}

	var file *formFile
	if len(req.FileData) > 0 {
		file = &formFile{field: "file", name: req.FileName, data: req.FileData}
	}
	fields := []string{
		"file_base64", req.FileBase64,
		"model", req.Model,
		"prompt", req.Prompt,
		"request_id", req.RequestID,
		"user_id", req.UserID,
	}
	for _, h := range req.Hotwords {
		fields = append(fields, "hotwords", h)
	}
	if file != nil {
		fields[1] = "" // the file part wins; never send both
	}
	form, err := newMultipartBody(file, fields...)
	if err != nil {
		return nil, err
	}

	var result AudioTranscriptionResponse
	r := apiRequest{method: "POST", path: "/audio/transcriptions", form: form, service: "audio", model: req.Model}
	if err := s.client.do(ctx, r, &result); err != nil {
		return nil, fmt.Errorf("failed to transcribe audio: %w", err)
	}
	return &result, nil
}

// GLM-TTS system voice choices for AudioSpeechRequest.Voice. Cloned voices
// (VoiceService.Clone) are also valid — pass the clone's Voice ID instead
// of one of these constants.
const (
	VoiceTongtong = "tongtong" // API default
	VoiceChuichui = "chuichui"
	VoiceXiaochen = "xiaochen"
	VoiceJam      = "jam"
	VoiceKazi     = "kazi"
	VoiceDouji    = "douji"
	VoiceLuodo    = "luodo"
)

// SystemVoices lists the GLM-TTS system voices.
var SystemVoices = []string{VoiceTongtong, VoiceChuichui, VoiceXiaochen, VoiceJam, VoiceKazi, VoiceDouji, VoiceLuodo}

// AudioSpeechRequest requests text-to-speech synthesis (GLM-TTS). Model,
// Input, and Voice are required by the API; Model/Voice default when empty.
type AudioSpeechRequest struct {
	Model          string  `json:"model"`                     // defaults to DefaultTTSModel
	Input          string  `json:"input"`                     // text to synthesize, max 1024 chars
	Voice          string  `json:"voice"`                     // defaults to VoiceTongtong; or a VoiceService.Clone result
	Speed          float64 `json:"speed,omitempty"`           // 0.5-2, API default 1.0
	Volume         float64 `json:"volume,omitempty"`          // (0,10], API default 1.0
	ResponseFormat string  `json:"response_format,omitempty"` // "wav" or "pcm" (API default)
	// WatermarkEnabled controls the AI-generated audio watermark, API
	// default true. A pointer for the same reason as
	// ImageGenerationRequest.WatermarkEnabled — omitempty on a plain bool
	// would silently drop an explicit false.
	WatermarkEnabled *bool `json:"watermark_enabled,omitempty"`
}

// Speech synthesizes req.Input as audio and returns the raw bytes in
// req.ResponseFormat (the API's own default is "pcm"). This is the
// non-streaming variant only, matching Transcribe's precedent — streaming
// TTS (SSE-chunked audio) can be added later if needed. req.Model and
// req.Voice default when empty.
func (s *AudioService) Speech(ctx context.Context, req AudioSpeechRequest) ([]byte, error) {
	if req.Model == "" {
		req.Model = DefaultTTSModel
	}
	if req.Input == "" {
		return nil, errors.New("input is required")
	}
	if req.Voice == "" {
		req.Voice = VoiceTongtong
	}

	data, err := s.client.doRaw(ctx, apiRequest{method: "POST", path: "/audio/speech", body: req, service: "audio", model: req.Model})
	if err != nil {
		return nil, fmt.Errorf("failed to synthesize speech: %w", err)
	}
	return data, nil
}
