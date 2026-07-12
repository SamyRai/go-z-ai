package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Transcribe uploads audio bytes as a multipart file field alongside the
// model/prompt/hotwords form fields, and doesn't retry on failure (a
// mid-upload retry would resend the whole file).
func TestAudioTranscribeMultipartWireFormat(t *testing.T) {
	var gotContentType string
	var gotFileName, gotModel, gotPrompt string
	var gotHotwords []string
	calls := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/audio/transcriptions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotContentType = r.Header.Get("Content-Type")
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if fh := r.MultipartForm.File["file"]; len(fh) == 1 {
			gotFileName = fh[0].Filename
		}
		gotModel = r.FormValue("model")
		gotPrompt = r.FormValue("prompt")
		gotHotwords = r.MultipartForm.Value["hotwords"]
		writeJSON(w, http.StatusOK, `{"id":"1","model":"glm-asr-2512","text":"hello world"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 3}) // retries must not apply here
	resp, err := c.Audio().Transcribe(context.Background(), AudioTranscriptionRequest{
		FileName: "clip.wav",
		FileData: []byte("fake-audio-bytes"),
		Prompt:   "domain context",
		Hotwords: []string{"zai", "glm"},
	})
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if resp.Text != "hello world" {
		t.Fatalf("unexpected text: %q", resp.Text)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call (no retry on multipart upload), got %d", calls)
	}
	if gotFileName != "clip.wav" {
		t.Fatalf("expected filename clip.wav, got %q", gotFileName)
	}
	if gotModel != audioTranscriptionModel {
		t.Fatalf("expected default model %q, got %q", audioTranscriptionModel, gotModel)
	}
	if gotPrompt != "domain context" {
		t.Fatalf("expected prompt to be sent, got %q", gotPrompt)
	}
	if len(gotHotwords) != 2 || gotHotwords[0] != "zai" || gotHotwords[1] != "glm" {
		t.Fatalf("expected 2 hotwords [zai glm], got %v", gotHotwords)
	}
	if gotContentType == "" || gotContentType[:19] != "multipart/form-data" {
		t.Fatalf("expected multipart/form-data content type, got %q", gotContentType)
	}
}

// Exactly one of FileData/FileBase64 must be provided.
func TestAudioTranscribeValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Audio().Transcribe(context.Background(), AudioTranscriptionRequest{}); err == nil {
		t.Fatal("expected error when neither FileData nor FileBase64 is set")
	}
}

// A non-200 response must surface as a structured APIError.
func TestAudioTranscribeAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusRequestEntityTooLarge, `{"error":{"code":"-1","message":"file too large"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{})
	_, err := c.Audio().Transcribe(context.Background(), AudioTranscriptionRequest{FileName: "x.wav", FileData: []byte("x")})
	if err == nil {
		t.Fatal("expected error")
	}
}

// Speech posts JSON (not multipart) to /audio/speech and returns the raw
// audio bytes from the response body, defaulting Model/Voice when unset.
func TestAudioSpeech(t *testing.T) {
	var gotPath, gotBody, gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		buf, _ := io.ReadAll(r.Body)
		gotBody = string(buf)
		w.Header().Set("Content-Type", "audio/wav")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-audio-bytes"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	data, err := c.Audio().Speech(context.Background(), AudioSpeechRequest{Input: "hello"})
	if err != nil {
		t.Fatalf("Speech: %v", err)
	}
	if gotPath != "/audio/speech" {
		t.Errorf("expected path /audio/speech, got %q", gotPath)
	}
	if gotContentType != "application/json" {
		t.Errorf("expected JSON request (not multipart), got content-type %q", gotContentType)
	}
	if !strings.Contains(gotBody, `"model":"glm-tts"`) || !strings.Contains(gotBody, `"voice":"tongtong"`) {
		t.Errorf("expected default model/voice in request body, got: %s", gotBody)
	}
	if string(data) != "fake-audio-bytes" {
		t.Errorf("expected raw audio bytes, got %q", data)
	}
}

// Missing input must fail before any request is sent.
func TestAudioSpeechValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Audio().Speech(context.Background(), AudioSpeechRequest{}); err == nil {
		t.Error("expected error for missing input")
	}
}

// A non-200 response must surface as a structured APIError, not raw bytes.
func TestAudioSpeechAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusBadRequest, `{"error":{"code":"1210","message":"invalid parameter"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	if _, err := c.Audio().Speech(context.Background(), AudioSpeechRequest{Input: "hi"}); err == nil {
		t.Error("expected error")
	}
}
