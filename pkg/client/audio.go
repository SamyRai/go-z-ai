package client

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
)

// AudioService handles audio transcription.
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
