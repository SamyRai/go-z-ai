package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

// FilesService manages files uploaded for use in other API calls (batch
// input, fine-tuning, retrieval, voice-clone input).
type FilesService struct {
	client *Client
}

// FilePurpose is the intended use of an uploaded file.
type FilePurpose string

const (
	FilePurposeFineTune        FilePurpose = "fine-tune"
	FilePurposeRetrieval       FilePurpose = "retrieval"
	FilePurposeBatch           FilePurpose = "batch"
	FilePurposeVoiceCloneInput FilePurpose = "voice-clone-input"
)

// FileObject describes an uploaded file.
type FileObject struct {
	ID            string `json:"id"`
	Bytes         int64  `json:"bytes"`
	CreatedAt     int64  `json:"created_at"`
	Filename      string `json:"filename"`
	Object        string `json:"object"`
	Purpose       string `json:"purpose"`
	Status        string `json:"status"`
	StatusDetails string `json:"status_details,omitempty"`
}

// FileListResponse is the response from FilesService.List.
type FileListResponse struct {
	Object  string       `json:"object"`
	Data    []FileObject `json:"data"`
	HasMore bool         `json:"has_more,omitempty"`
}

// FileDeletedResponse confirms a file deletion.
type FileDeletedResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
	Object  string `json:"object"`
}

// Upload uploads a file for later use (e.g. as batch input, or the
// input_file_id in BatchCreateRequest).
func (s *FilesService) Upload(ctx context.Context, filename string, data []byte, purpose FilePurpose) (*FileObject, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("file data is required")
	}
	if filename == "" {
		return nil, fmt.Errorf("filename is required")
	}
	if purpose == "" {
		return nil, fmt.Errorf("purpose is required")
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to build multipart file field: %w", err)
	}
	if _, err := fw.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}
	if err := w.WriteField("purpose", string(purpose)); err != nil {
		return nil, fmt.Errorf("failed to write purpose field: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize multipart body: %w", err)
	}

	resp, err := s.client.sendMultipart(ctx, "/files", w.FormDataContentType(), buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result FileObject
	if err := s.client.decodeBody(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// List returns uploaded files, optionally filtered by purpose (pass "" for
// all purposes).
func (s *FilesService) List(ctx context.Context, purpose FilePurpose) (*FileListResponse, error) {
	endpoint := "/files"
	if purpose != "" {
		q := url.Values{}
		q.Set("purpose", string(purpose))
		endpoint += "?" + q.Encode()
	}
	var result FileListResponse
	if err := s.client.doRequest(ctx, "GET", endpoint, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	return &result, nil
}

// Delete removes an uploaded file by ID.
func (s *FilesService) Delete(ctx context.Context, fileID string) (*FileDeletedResponse, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file id is required")
	}
	var result FileDeletedResponse
	if err := s.client.doRequest(ctx, "DELETE", "/files/"+fileID, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to delete file: %w", err)
	}
	return &result, nil
}

// Content downloads the raw bytes of an uploaded file (e.g. a completed
// batch's OutputFileID or ErrorFileID).
func (s *FilesService) Content(ctx context.Context, fileID string) ([]byte, error) {
	if fileID == "" {
		return nil, fmt.Errorf("file id is required")
	}
	resp, err := s.client.send(ctx, s.client.config.BaseURL, "GET", "/files/"+fileID+"/content", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download file content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}
	return data, nil
}
