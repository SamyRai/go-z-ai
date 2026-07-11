package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetAsyncResultImageAndVideo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/async-result/task-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, http.StatusOK, `{"task_status":"SUCCESS","data":[{"url":"https://x/img.png"}],"video_result":[{"url":"https://x/v.mp4","cover_image_url":"https://x/c.png"}]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	result, err := c.GetAsyncResult(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetAsyncResult: %v", err)
	}
	if result.TaskStatus != TaskStatusSuccess {
		t.Fatalf("unexpected status: %s", result.TaskStatus)
	}
	if len(result.Data) != 1 || result.Data[0].URL != "https://x/img.png" {
		t.Fatalf("unexpected image data: %+v", result.Data)
	}
	if len(result.VideoResult) != 1 || result.VideoResult[0].URL != "https://x/v.mp4" {
		t.Fatalf("unexpected video result: %+v", result.VideoResult)
	}
}

// WaitForResult must poll until a terminal state and return the final result.
func TestWaitForResultPollsUntilTerminal(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			writeJSON(w, http.StatusOK, `{"task_status":"PROCESSING"}`)
			return
		}
		writeJSON(w, http.StatusOK, `{"task_status":"SUCCESS","data":[{"url":"https://x/img.png"}]}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	result, err := c.WaitForResult(context.Background(), "task-1", time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForResult: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 polls (2 processing + 1 terminal), got %d", calls)
	}
	if result.TaskStatus != TaskStatusSuccess {
		t.Fatalf("unexpected final status: %s", result.TaskStatus)
	}
}

// A FAIL status is terminal too — WaitForResult must return it, not treat it
// like PROCESSING and loop forever.
func TestWaitForResultStopsOnFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"task_status":"FAIL"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	result, err := c.WaitForResult(context.Background(), "task-1", time.Millisecond)
	if err != nil {
		t.Fatalf("WaitForResult: %v", err)
	}
	if result.TaskStatus != TaskStatusFail {
		t.Fatalf("expected FAIL status, got %s", result.TaskStatus)
	}
}

// Context cancellation must abort the poll loop instead of hanging forever.
func TestWaitForResultRespectsContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, `{"task_status":"PROCESSING"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, Config{MaxRetries: 0})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := c.WaitForResult(ctx, "task-1", time.Second)
	if err == nil {
		t.Fatal("expected context deadline error")
	}
	if time.Since(start) > time.Second {
		t.Fatalf("WaitForResult should abort promptly on context cancel; took %v", time.Since(start))
	}
}
