package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// AudioService handles audio transcription and text-to-speech.
type AudioService struct {
	client *Client
}

const audioTranscriptionModel = "glm-asr-2512"

// AudioTranscriptionRequest is the request for audio transcription. Exactly
// one of FileData or FileBase64 must be set; if both are set, FileData wins.
type AudioTranscriptionRequest struct {
	FileName   string // required when FileData is set, e.g. "clip.wav"
	FileData   []byte // raw audio bytes (.wav or .mp3, <=25MB, <=30s)
	FileBase64 string // alternative to FileData

	Model     string   // defaults to glm-asr-2512
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

// Transcribe uploads an audio clip and returns its transcription. This is
// the non-streaming variant only — matching how ChatService splits
// Create/CreateStream, streaming transcription can be added later if needed.
func (s *AudioService) Transcribe(ctx context.Context, req AudioTranscriptionRequest) (*AudioTranscriptionResponse, error) {
	if len(req.FileData) == 0 && req.FileBase64 == "" {
		return nil, fmt.Errorf("file data or file_base64 is required")
	}
	if req.Model == "" {
		req.Model = audioTranscriptionModel
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if len(req.FileData) > 0 {
		fw, err := w.CreateFormFile("file", req.FileName)
		if err != nil {
			return nil, fmt.Errorf("failed to build multipart file field: %w", err)
		}
		if _, err := fw.Write(req.FileData); err != nil {
			return nil, fmt.Errorf("failed to write audio data: %w", err)
		}
	} else {
		_ = w.WriteField("file_base64", req.FileBase64)
	}

	_ = w.WriteField("model", req.Model)
	if req.Prompt != "" {
		_ = w.WriteField("prompt", req.Prompt)
	}
	for _, h := range req.Hotwords {
		_ = w.WriteField("hotwords", h)
	}
	if req.RequestID != "" {
		_ = w.WriteField("request_id", req.RequestID)
	}
	if req.UserID != "" {
		_ = w.WriteField("user_id", req.UserID)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize multipart body: %w", err)
	}

	resp, err := s.client.sendMultipart(ctx, "/audio/transcriptions", w.FormDataContentType(), buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result AudioTranscriptionResponse
	if err := s.client.decodeBody(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

const audioSpeechModel = "glm-tts"

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

// AudioSpeechRequest requests text-to-speech synthesis (GLM-TTS). Model,
// Input, and Voice are required by the API; Model/Voice default when empty.
type AudioSpeechRequest struct {
	Model          string  `json:"model"`                     // defaults to glm-tts
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
		req.Model = audioSpeechModel
	}
	if req.Input == "" {
		return nil, fmt.Errorf("input is required")
	}
	if req.Voice == "" {
		req.Voice = VoiceTongtong
	}

	resp, err := s.client.send(ctx, s.client.config.BaseURL, s.client.config.APIKey, "POST", "/audio/speech", req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}
	return data, nil
}
