package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Parse defaults Model to glm-ocr when the caller doesn't set one, and
// returns the recognized Markdown.
func TestLayoutParseDefaultsModel(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/layout_parsing" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"id":"1","md_results":"# Hello"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	resp, err := c.Layout().Parse(context.Background(), LayoutParsingRequest{File: "aGVsbG8="})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if resp.MDResults != "# Hello" {
		t.Fatalf("unexpected md_results: %q", resp.MDResults)
	}
	if !strings.Contains(gotBody, `"model":"glm-ocr"`) {
		t.Fatalf("expected default model glm-ocr in request body, got: %s", gotBody)
	}
}

// File is required.
func TestLayoutParseValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Layout().Parse(context.Background(), LayoutParsingRequest{}); err == nil {
		t.Fatal("expected error for missing file")
	}
}

// HandwritingOCR uploads the image as multipart with tool_type=hand_write,
// and parses the per-word results including bounding box and probability.
func TestHandwritingOCR(t *testing.T) {
	var gotToolType, gotLanguage, gotProbability, gotFileName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files/ocr" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		if fh := r.MultipartForm.File["file"]; len(fh) == 1 {
			gotFileName = fh[0].Filename
		}
		gotToolType = r.FormValue("tool_type")
		gotLanguage = r.FormValue("language_type")
		gotProbability = r.FormValue("probability")
		writeJSON(w, http.StatusOK, `{
			"task_id": "task-1",
			"message": "success",
			"status": "done",
			"words_result_num": 1,
			"words_result": [
				{
					"location": {"left": 1, "top": 2, "width": 3, "height": 4},
					"words": "hello",
					"probability": {"average": 0.9, "variance": 0.01, "min": 0.8}
				}
			]
		}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 3}) // retries must not apply to uploads
	resp, err := c.Layout().HandwritingOCR(context.Background(), HandwritingOCRRequest{
		FileName:     "note.jpg",
		FileData:     []byte("fake-image-bytes"),
		LanguageType: "en",
		Probability:  true,
	})
	if err != nil {
		t.Fatalf("HandwritingOCR: %v", err)
	}
	if resp.WordsResultNum != 1 || len(resp.WordsResult) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	wr := resp.WordsResult[0]
	if wr.Words != "hello" {
		t.Errorf("expected words 'hello', got %q", wr.Words)
	}
	if wr.Location != (Location{Left: 1, Top: 2, Width: 3, Height: 4}) {
		t.Errorf("unexpected location: %+v", wr.Location)
	}
	if wr.Probability == nil || wr.Probability.Average != 0.9 {
		t.Errorf("unexpected probability: %+v", wr.Probability)
	}
	if gotFileName != "note.jpg" {
		t.Errorf("expected filename note.jpg, got %q", gotFileName)
	}
	if gotToolType != "hand_write" {
		t.Errorf("expected tool_type=hand_write, got %q", gotToolType)
	}
	if gotLanguage != "en" {
		t.Errorf("expected language_type=en, got %q", gotLanguage)
	}
	if gotProbability != "true" {
		t.Errorf("expected probability=true, got %q", gotProbability)
	}
}

// FileData and FileName are both required.
func TestHandwritingOCRValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Layout().HandwritingOCR(context.Background(), HandwritingOCRRequest{FileName: "x.jpg"}); err == nil {
		t.Error("expected error for missing file data")
	}
	if _, err := c.Layout().HandwritingOCR(context.Background(), HandwritingOCRRequest{FileData: []byte("x")}); err == nil {
		t.Error("expected error for missing file name")
	}
}

// A non-200 response must surface as a structured APIError.
func TestHandwritingOCRAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusBadRequest, `{"error":{"code":"-1","message":"unsupported image format"}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{})
	_, err := c.Layout().HandwritingOCR(context.Background(), HandwritingOCRRequest{FileName: "x.jpg", FileData: []byte("x")})
	if err == nil {
		t.Fatal("expected error")
	}
}
