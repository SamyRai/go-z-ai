package client

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
)

// FileParserService parses documents (PDF, Office formats, images, ...)
// into text or a downloadable result, for RAG/retrieval preprocessing.
// Distinct from LayoutService.Parse (/layout_parsing, OCR-focused layout
// extraction) and LayoutService.HandwritingOCR — this is a separate
// product ("文件解析服务") confirmed against docs.bigmodel.cn's live
// OpenAPI spec (https://docs.bigmodel.cn/openapi/openapi.json:
// POST /paas/v4/files/parser/create, /files/parser/sync;
// GET /files/parser/result/{taskId}/{format_type}).
type FileParserService struct {
	client *Client
}

// File parser tool types (FileParserRequest.ToolType). Lite/Expert/Prime are
// for the async Create path; PrimeSync is for the synchronous Sync path
// only.
const (
	FileParserToolLite      = "lite"
	FileParserToolExpert    = "expert"
	FileParserToolPrime     = "prime"
	FileParserToolPrimeSync = "prime-sync"
)

// File parser result format types (FileParserService.Result's formatType
// parameter).
const (
	FileParserFormatText         = "text"
	FileParserFormatDownloadLink = "download_link"
)

// File parser task status values (FileParseResultResponse.Status).
const (
	FileParserStatusProcessing = "processing"
	FileParserStatusSucceeded  = "succeeded"
	FileParserStatusFailed     = "failed"
)

// FileParserRequest submits a document for parsing. FileData and ToolType
// are required; FileType is optional (the API can usually infer it from
// the filename/content).
type FileParserRequest struct {
	FileName string // e.g. "report.pdf"
	FileData []byte
	ToolType string // one of the FileParserTool* constants
	FileType string // optional, e.g. "PDF" — see the API docs for the full enum per tool type
}

// FileParserCreateResponse is the result of FileParserService.Create — an
// async task to poll via FileParserService.Result.
type FileParserCreateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	TaskID  string `json:"task_id"`
}

// FileParseResultResponse is the result of a file parse task, returned
// directly by FileParserService.Sync or polled via FileParserService.Result.
// Named "Parse" rather than "Parser" (unlike FileParserRequest/
// FileParserCreateResponse) deliberately — it matches the live OpenAPI
// spec's actual schema name (docs.bigmodel.cn/openapi/openapi.json,
// FileParseResultResponse) verbatim, not a naming inconsistency to "fix".
type FileParseResultResponse struct {
	Status           string `json:"status"` // FileParserStatusProcessing/Succeeded/Failed
	Message          string `json:"message"`
	TaskID           string `json:"task_id"`
	Content          string `json:"content,omitempty"`            // populated when queried with FileParserFormatText
	ParsingResultURL string `json:"parsing_result_url,omitempty"` // populated when queried with FileParserFormatDownloadLink
}

// buildParseMultipart is shared by Create and Sync — both take the same
// multipart shape (file + tool_type + optional file_type), differing only
// in endpoint and response shape.
func buildParseMultipart(req FileParserRequest) (*bytes.Buffer, string, error) {
	if len(req.FileData) == 0 {
		return nil, "", fmt.Errorf("file data is required")
	}
	if req.ToolType == "" {
		return nil, "", fmt.Errorf("tool_type is required")
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	fw, err := w.CreateFormFile("file", req.FileName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to build multipart file field: %w", err)
	}
	if _, err := fw.Write(req.FileData); err != nil {
		return nil, "", fmt.Errorf("failed to write file data: %w", err)
	}

	_ = w.WriteField("tool_type", req.ToolType)
	if req.FileType != "" {
		_ = w.WriteField("file_type", req.FileType)
	}
	if err := w.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to finalize multipart body: %w", err)
	}
	return &buf, w.FormDataContentType(), nil
}

// Create submits an async file parse task (req.ToolType one of
// FileParserToolLite/Expert/Prime); poll the result with Result.
func (s *FileParserService) Create(ctx context.Context, req FileParserRequest) (*FileParserCreateResponse, error) {
	buf, contentType, err := buildParseMultipart(req)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.sendMultipart(ctx, "/files/parser/create", contentType, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result FileParserCreateResponse
	if err := s.client.decodeBody(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Sync submits a file parse task and blocks until the result is ready.
// req.ToolType must be FileParserToolPrimeSync — the only tool type the
// synchronous endpoint accepts. req.FileType is required here even though
// the documented spec (docs.bigmodel.cn's OpenAPI schema) marks it
// optional: a real call without it, live-verified 2026-07-11, returned
// HTTP 200 with a body in a completely different, non-standard error shape
// ({"msg":"Required request parameter 'file_type' ... is not present",
// "code":500} — not the {"error":{"code","message"}} envelope every other
// endpoint uses, and not FileParseResultResponse's shape either). Because
// the status was 200, this doesn't reach parseAPIError; because the body
// doesn't match FileParseResultResponse, decodeBody succeeds but silently
// produces an empty, useless result with no error — this validation exists
// specifically to never let that scenario happen. Create's schema shares
// the same "FileType optional" claim and is unverified; pass it there too.
func (s *FileParserService) Sync(ctx context.Context, req FileParserRequest) (*FileParseResultResponse, error) {
	if req.FileType == "" {
		return nil, fmt.Errorf("file_type is required (the documented API optional-ness does not hold in practice — see this method's doc comment)")
	}

	buf, contentType, err := buildParseMultipart(req)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.sendMultipart(ctx, "/files/parser/sync", contentType, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp)
	}

	var result FileParseResultResponse
	if err := s.client.decodeBody(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Result fetches the result of an async file parse task submitted via
// Create. formatType is FileParserFormatText (populates Content) or
// FileParserFormatDownloadLink (populates ParsingResultURL).
func (s *FileParserService) Result(ctx context.Context, taskID, formatType string) (*FileParseResultResponse, error) {
	if taskID == "" {
		return nil, fmt.Errorf("task id is required")
	}
	if formatType == "" {
		return nil, fmt.Errorf("format type is required")
	}

	endpoint := "/files/parser/result/" + url.PathEscape(taskID) + "/" + url.PathEscape(formatType)
	var result FileParseResultResponse
	if err := s.client.doRequest(ctx, "GET", endpoint, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get file parse result: %w", err)
	}
	return &result, nil
}
