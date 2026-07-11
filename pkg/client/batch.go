package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// BatchService manages async bulk-processing jobs (OpenAI-Batch-style):
// submit many requests as a JSONL file (upload it first via
// FilesService.Upload with FilePurposeBatch), then poll for completion and
// download the result via FilesService.Content(batch.OutputFileID).
type BatchService struct {
	client *Client
}

// BatchEndpoint is the API endpoint a batch's requests target.
type BatchEndpoint string

const (
	BatchEndpointChatCompletions BatchEndpoint = "/v1/chat/completions"
	BatchEndpointEmbeddings      BatchEndpoint = "/v1/embeddings"
)

// defaultCompletionWindow is the only value the API currently supports.
const defaultCompletionWindow = "24h"

// BatchCreateRequest submits a new batch job.
type BatchCreateRequest struct {
	InputFileID         string            `json:"input_file_id"`
	Endpoint            BatchEndpoint     `json:"endpoint"`
	CompletionWindow    string            `json:"completion_window,omitempty"` // defaults to "24h"
	Metadata            map[string]string `json:"metadata,omitempty"`
	AutoDeleteInputFile bool              `json:"auto_delete_input_file,omitempty"`
}

// BatchRequestCounts summarizes a batch's request completion progress.
type BatchRequestCounts struct {
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Total     int `json:"total"`
}

// BatchError is one error entry in a batch's Errors.Data.
type BatchError struct {
	Code    string `json:"code,omitempty"`
	Line    int    `json:"line,omitempty"`
	Message string `json:"message,omitempty"`
	Param   string `json:"param,omitempty"`
}

// BatchErrors wraps the error list a failed batch reports.
type BatchErrors struct {
	Object string       `json:"object,omitempty"`
	Data   []BatchError `json:"data,omitempty"`
}

// Batch is the state of a batch job. Status transitions through
// validating -> in_progress -> finalizing -> completed (or failed/expired/
// cancelled); see IsTerminal. On success, download the result via
// FilesService.Content(OutputFileID).
type Batch struct {
	ID               string              `json:"id"`
	Object           string              `json:"object"`
	Endpoint         string              `json:"endpoint"`
	InputFileID      string              `json:"input_file_id"`
	CompletionWindow string              `json:"completion_window"`
	Status           string              `json:"status"`
	OutputFileID     string              `json:"output_file_id,omitempty"`
	ErrorFileID      string              `json:"error_file_id,omitempty"`
	Errors           *BatchErrors        `json:"errors,omitempty"`
	CreatedAt        int64               `json:"created_at"`
	InProgressAt     int64               `json:"in_progress_at,omitempty"`
	ExpiresAt        int64               `json:"expires_at,omitempty"`
	FinalizingAt     int64               `json:"finalizing_at,omitempty"`
	CompletedAt      int64               `json:"completed_at,omitempty"`
	FailedAt         int64               `json:"failed_at,omitempty"`
	ExpiredAt        int64               `json:"expired_at,omitempty"`
	CancellingAt     int64               `json:"cancelling_at,omitempty"`
	CancelledAt      int64               `json:"cancelled_at,omitempty"`
	RequestCounts    *BatchRequestCounts `json:"request_counts,omitempty"`
	Metadata         map[string]string   `json:"metadata,omitempty"`
}

// Batch status constants.
const (
	BatchStatusValidating = "validating"
	BatchStatusInProgress = "in_progress"
	BatchStatusFinalizing = "finalizing"
	BatchStatusCompleted  = "completed"
	BatchStatusFailed     = "failed"
	BatchStatusExpired    = "expired"
	BatchStatusCancelling = "cancelling"
	BatchStatusCancelled  = "cancelled"
)

// IsTerminal reports whether the batch has reached a final state.
func (b *Batch) IsTerminal() bool {
	switch b.Status {
	case BatchStatusCompleted, BatchStatusFailed, BatchStatusExpired, BatchStatusCancelled:
		return true
	default:
		return false
	}
}

// BatchListResponse is the response from BatchService.List.
type BatchListResponse struct {
	Object  string  `json:"object"`
	Data    []Batch `json:"data"`
	HasMore bool    `json:"has_more,omitempty"`
}

// Create submits a new batch job. req.CompletionWindow defaults to "24h"
// when empty (the only value the API currently supports).
func (s *BatchService) Create(ctx context.Context, req BatchCreateRequest) (*Batch, error) {
	if req.InputFileID == "" {
		return nil, fmt.Errorf("input_file_id is required")
	}
	if req.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}
	if req.CompletionWindow == "" {
		req.CompletionWindow = defaultCompletionWindow
	}

	var result Batch
	if err := s.client.doRequest(ctx, "POST", "/batches", req, &result); err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}
	return &result, nil
}

// Retrieve fetches the current state of a batch by ID.
func (s *BatchService) Retrieve(ctx context.Context, batchID string) (*Batch, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch id is required")
	}
	var result Batch
	if err := s.client.doRequest(ctx, "GET", "/batches/"+batchID, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to retrieve batch: %w", err)
	}
	return &result, nil
}

// List returns the organization's batches, cursor-paginated (pass after=""
// and limit<=0 for the first page with the server's default page size).
func (s *BatchService) List(ctx context.Context, after string, limit int) (*BatchListResponse, error) {
	endpoint := "/batches"
	q := url.Values{}
	if after != "" {
		q.Set("after", after)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if enc := q.Encode(); enc != "" {
		endpoint += "?" + enc
	}

	var result BatchListResponse
	if err := s.client.doRequest(ctx, "GET", endpoint, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to list batches: %w", err)
	}
	return &result, nil
}

// Cancel requests cancellation of an in-progress batch.
func (s *BatchService) Cancel(ctx context.Context, batchID string) (*Batch, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch id is required")
	}
	var result Batch
	if err := s.client.doRequest(ctx, "POST", "/batches/"+batchID+"/cancel", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to cancel batch: %w", err)
	}
	return &result, nil
}
