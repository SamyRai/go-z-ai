package client

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
)

// LayoutService handles layout parsing (OCR / document-to-markdown).
type LayoutService struct {
	client *Client
}

// LayoutParsingRequest is the request body for layout parsing. Model is
// always "glm-ocr". File is an image or PDF, as a URL or base64 string.
type LayoutParsingRequest struct {
	Model                   string `json:"model"`
	File                    string `json:"file"`
	ReturnCropImages        bool   `json:"return_crop_images,omitempty"`
	NeedLayoutVisualization bool   `json:"need_layout_visualization,omitempty"`
	StartPageID             int    `json:"start_page_id,omitempty"` // PDFs only
	EndPageID               int    `json:"end_page_id,omitempty"`   // PDFs only
	RequestID               string `json:"request_id,omitempty"`
	UserID                  string `json:"user_id,omitempty"`
}

// LayoutParsingResponse is the parsed document result.
type LayoutParsingResponse struct {
	ID                  string   `json:"id"`
	Created             int64    `json:"created"`
	Model               string   `json:"model"`
	MDResults           string   `json:"md_results"` // recognized content as Markdown
	LayoutDetails       []any    `json:"layout_details,omitempty"`
	LayoutVisualization []string `json:"layout_visualization,omitempty"`
	DataInfo            struct {
		NumPages int `json:"num_pages"`
	} `json:"data_info"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

const layoutParsingModel = "glm-ocr"

// Parse recognizes an image or PDF's layout, returning the content as
// Markdown. req.Model is set to "glm-ocr" automatically if empty.
func (s *LayoutService) Parse(ctx context.Context, req LayoutParsingRequest) (*LayoutParsingResponse, error) {
	if req.File == "" {
		return nil, fmt.Errorf("file is required")
	}
	if req.Model == "" {
		req.Model = layoutParsingModel
	}

	var resp LayoutParsingResponse
	if err := s.client.doRequest(ctx, "POST", "/layout_parsing", req, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse layout: %w", err)
	}
	return &resp, nil
}

// --- Handwriting OCR (POST /files/ocr) ---
//
// Distinct from Parse's /layout_parsing (glm-ocr) endpoint: this targets
// short handwritten snippets rather than full documents, and returns
// per-word bounding boxes/confidence instead of a Markdown transcript. Both
// live under the official SDK's single "ocr" resource, hence one service.
// Verified against the SDK's declared types (zai/types/ocr/
// handwriting_ocr_resp.py), not just the README.

// HandwritingOCRRequest is the request for handwriting recognition.
type HandwritingOCRRequest struct {
	FileName     string // required, e.g. "note.jpg"
	FileData     []byte // required, image bytes
	LanguageType string // optional
	Probability  bool   // optional: include per-word confidence statistics
}

// Location is a recognized word's bounding box within the source image.
type Location struct {
	Left   int `json:"left"`
	Top    int `json:"top"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// WordProbability is confidence statistics for one recognized word,
// present only when HandwritingOCRRequest.Probability was set.
type WordProbability struct {
	Average  float64 `json:"average"`
	Variance float64 `json:"variance"`
	Min      float64 `json:"min"`
}

// WordsResult is one recognized word/phrase with its position and
// (optionally) confidence.
type WordsResult struct {
	Location    Location         `json:"location"`
	Words       string           `json:"words"`
	Probability *WordProbability `json:"probability,omitempty"`
}

// HandwritingOCRResponse is the handwriting-recognition result.
type HandwritingOCRResponse struct {
	TaskID         string        `json:"task_id"`
	Message        string        `json:"message"`
	Status         string        `json:"status"`
	WordsResultNum int           `json:"words_result_num"`
	WordsResult    []WordsResult `json:"words_result,omitempty"`
}

// HandwritingOCR recognizes handwritten text in an image.
func (s *LayoutService) HandwritingOCR(ctx context.Context, req HandwritingOCRRequest) (*HandwritingOCRResponse, error) {
	if len(req.FileData) == 0 {
		return nil, fmt.Errorf("file data is required")
	}
	if req.FileName == "" {
		return nil, fmt.Errorf("file name is required")
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", req.FileName)
	if err != nil {
		return nil, fmt.Errorf("failed to build multipart file field: %w", err)
	}
	if _, err := fw.Write(req.FileData); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}
	if err := w.WriteField("tool_type", "hand_write"); err != nil {
		return nil, fmt.Errorf("failed to write tool_type field: %w", err)
	}
	if req.LanguageType != "" {
		if err := w.WriteField("language_type", req.LanguageType); err != nil {
			return nil, fmt.Errorf("failed to write language_type field: %w", err)
		}
	}
	if req.Probability {
		if err := w.WriteField("probability", "true"); err != nil {
			return nil, fmt.Errorf("failed to write probability field: %w", err)
		}
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize multipart body: %w", err)
	}

	resp, err := s.client.sendMultipart(ctx, "/files/ocr", w.FormDataContentType(), buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result HandwritingOCRResponse
	if err := s.client.decodeBody(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
