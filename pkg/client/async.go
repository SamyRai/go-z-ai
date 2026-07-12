package client

import (
	"context"
	"fmt"
	"time"
)

// Task status values returned by async image/video generation and by
// GetAsyncResult.
const (
	TaskStatusProcessing = "PROCESSING"
	TaskStatusSuccess    = "SUCCESS"
	TaskStatusFail       = "FAIL"
)

// AsyncTaskResponse is returned immediately by an async generation request
// (image or video) — the actual result must be retrieved via GetAsyncResult.
type AsyncTaskResponse struct {
	Model      string `json:"model"`
	ID         string `json:"id"`
	RequestID  string `json:"request_id"`
	TaskStatus string `json:"task_status"`
}

// AsyncResultResponse is the result of polling GetAsyncResult. Exactly one
// of Data (image tasks), VideoResult (video tasks), or Choices (chat
// completion tasks, via ChatService.CreateAsync) is populated, depending on
// what kind of task ID was polled — confirmed against docs.bigmodel.cn's
// live OpenAPI spec, whose GET /paas/v4/async-result/{id} response is a
// oneOf across ChatCompletionResponse/AsyncVideoGenerationResponse/
// AsyncImageGenerationResponse.
type AsyncResultResponse struct {
	ID         string `json:"id,omitempty"`
	Created    int64  `json:"created,omitempty"`
	Model      string `json:"model"`
	TaskStatus string `json:"task_status"`
	RequestID  string `json:"request_id"`
	Data       []struct {
		URL string `json:"url"`
	} `json:"data,omitempty"`
	VideoResult []struct {
		URL           string `json:"url"`
		CoverImageURL string `json:"cover_image_url"`
	} `json:"video_result,omitempty"`
	Choices []Choice `json:"choices,omitempty"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// GetAsyncResult polls the shared async-result endpoint used by both image
// and video generation. Callers should re-poll (e.g. on a timer) while
// TaskStatus == TaskStatusProcessing.
func (c *Client) GetAsyncResult(ctx context.Context, id string) (*AsyncResultResponse, error) {
	var result AsyncResultResponse
	err := c.doRequest(ctx, "GET", fmt.Sprintf("/async-result/%s", id), nil, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get async result: %w", err)
	}
	return &result, nil
}

// defaultPollInterval is used by WaitForResult when interval <= 0.
const defaultPollInterval = 3 * time.Second

// WaitForResult polls GetAsyncResult every interval until the task reaches a
// terminal state (TaskStatusSuccess or TaskStatusFail) or ctx is cancelled,
// so SDK callers don't have to hand-roll the poll loop themselves.
func (c *Client) WaitForResult(ctx context.Context, id string, interval time.Duration) (*AsyncResultResponse, error) {
	if interval <= 0 {
		interval = defaultPollInterval
	}
	for {
		result, err := c.GetAsyncResult(ctx, id)
		if err != nil {
			return nil, err
		}
		if result.TaskStatus != TaskStatusProcessing {
			return result, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}
