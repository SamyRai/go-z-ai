package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Clone posts to /voice/clone and defaults Model to glm-tts-clone when unset.
func TestVoiceCloneDefaultsModel(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"voice":"voice_clone_001","file_id":"file-1","file_purpose":"voice-clone-output","request_id":"req-1"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.Voice().Clone(context.Background(), VoiceCloneRequest{
		VoiceName: "my_voice",
		Input:     "preview text",
		FileID:    "file-sample",
	})
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if gotPath != "/voice/clone" {
		t.Errorf("expected path /voice/clone, got %q", gotPath)
	}
	if !strings.Contains(gotBody, `"model":"glm-tts-clone"`) {
		t.Errorf("expected default model in request body, got: %s", gotBody)
	}
	if resp.Voice != "voice_clone_001" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// Missing voice_name, input, or file_id must fail before any request is sent.
func TestVoiceCloneValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Voice().Clone(context.Background(), VoiceCloneRequest{Input: "x", FileID: "f"}); err == nil {
		t.Error("expected error for missing voice_name")
	}
	if _, err := c.Voice().Clone(context.Background(), VoiceCloneRequest{VoiceName: "n", FileID: "f"}); err == nil {
		t.Error("expected error for missing input")
	}
	if _, err := c.Voice().Clone(context.Background(), VoiceCloneRequest{VoiceName: "n", Input: "x"}); err == nil {
		t.Error("expected error for missing file_id")
	}
}

// Delete posts to /voice/delete with the voice ID in the body.
func TestVoiceDelete(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"voice":"voice_clone_001","update_time":"2026-07-11 12:00:00"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.Voice().Delete(context.Background(), "voice_clone_001")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !strings.Contains(gotBody, `"voice":"voice_clone_001"`) {
		t.Errorf("expected voice id in request body, got: %s", gotBody)
	}
	if resp.Voice != "voice_clone_001" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// Missing voice id must fail before any request is sent.
func TestVoiceDeleteValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Voice().Delete(context.Background(), ""); err == nil {
		t.Error("expected error for missing voice id")
	}
}

// List encodes voiceName/voiceType as query parameters.
func TestVoiceList(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		writeJSON(w, http.StatusOK, `{"voice_list":[{"voice":"v1","voice_name":"n1","voice_type":"PRIVATE"}]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	list, err := c.Voice().List(context.Background(), "n1", VoiceTypePrivate)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if gotQuery != "voiceName=n1&voiceType=PRIVATE" {
		t.Errorf("unexpected query: %q", gotQuery)
	}
	if len(list) != 1 || list[0].Voice != "v1" {
		t.Fatalf("unexpected response: %+v", list)
	}
}
