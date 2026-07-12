package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Create posts to /moderations, defaults Model to "moderation", and parses
// the real response shape (result_list/risk_level/usage.moderation_text) —
// the shape the official Python SDK never modeled.
func TestModerationsCreateDefaultsModel(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"id":"mod-1","created":123,"request_id":"req-1","result_list":[{"content_type":"text","risk_level":"PASS","risk_type":[]}],"usage":{"moderation_text":{"call_count":1}}}`)
	}))
	defer srv.Close()

	c := newBigModelTestClient(t, srv)
	resp, err := c.Moderations().Create(context.Background(), ModerationRequest{
		Input: "审核内容安全样例字符串。",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotPath != "/api/paas/v4/moderations" {
		t.Errorf("expected path /api/paas/v4/moderations, got %q", gotPath)
	}
	if !strings.Contains(gotBody, `"model":"moderation"`) {
		t.Errorf("expected default model=moderation in request body, got: %s", gotBody)
	}
	if len(resp.ResultList) != 1 || resp.ResultList[0].RiskLevel != ModerationRiskPass {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Usage.ModerationText == nil || resp.Usage.ModerationText.CallCount != 1 {
		t.Errorf("unexpected usage: %+v", resp.Usage)
	}
}

// Create accepts a single multimodal ModerationContent object as Input.
func TestModerationsCreateMultimodalInput(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = string(buf)
		writeJSON(w, http.StatusOK, `{"id":"mod-2","result_list":[{"content_type":"image","risk_level":"REJECT","risk_type":["porn"]}],"usage":{"moderation_text":{"call_count":1}}}`)
	}))
	defer srv.Close()

	c := newBigModelTestClient(t, srv)
	resp, err := c.Moderations().Create(context.Background(), ModerationRequest{
		Input: ModerationContent{Type: "image_url", ImageURL: &ModerationURL{URL: "https://example.com/x.png"}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !strings.Contains(gotBody, `"image_url":{"url":"https://example.com/x.png"}`) {
		t.Errorf("expected image_url in request body, got: %s", gotBody)
	}
	if resp.ResultList[0].RiskLevel != ModerationRiskReject {
		t.Errorf("expected REJECT, got %q", resp.ResultList[0].RiskLevel)
	}
}

// Missing input must fail before any request is sent.
func TestModerationsCreateValidation(t *testing.T) {
	c := newTestClient(t, "http://unused", Config{})
	if _, err := c.Moderations().Create(context.Background(), ModerationRequest{}); err == nil {
		t.Error("expected error for missing input")
	}
}

// ModerationCallCount must accept both wire shapes the two official Z.AI
// SDKs disagree on: a bare JSON number (docs.bigmodel.cn's OpenAPI spec)
// and a quoted numeric string (the official Java SDK's ModerationText
// class) — see ModerationCallCount's doc comment.
func TestModerationCallCountAcceptsNumberOrString(t *testing.T) {
	for _, body := range []string{`3`, `"3"`} {
		var got ModerationCallCount
		if err := got.UnmarshalJSON([]byte(body)); err != nil {
			t.Fatalf("UnmarshalJSON(%s): %v", body, err)
		}
		if got != 3 {
			t.Errorf("UnmarshalJSON(%s) = %d, want 3", body, got)
		}
	}
	var bad ModerationCallCount
	if err := bad.UnmarshalJSON([]byte(`"not-a-number"`)); err == nil {
		t.Error("expected an error for a non-numeric string")
	}
}
