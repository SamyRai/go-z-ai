package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// ModerationsService screens text/image/video/audio content for policy
// violations, against BigModelBaseURL (open.bigmodel.cn) — the only
// platform that documents this endpoint (docs.z.ai's doc index has no
// Moderations API at all). Request/response types confirmed against the
// live OpenAPI spec at https://docs.bigmodel.cn/openapi/openapi.json
// (POST /paas/v4/moderations) and cross-checked against the official Java
// SDK's typed models (github.com/zai-org/z-ai-sdk-java,
// service/moderations/*.java) — the official Python SDK never modeled the
// real response fields at all (its Completion type only declares
// model/input), but Java's ModerationResult/ModerationUsage match the spec
// field-for-field except ModerationText.CallCount (see
// ModerationCallCount). No public repo has recorded HTTP fixtures/VCR
// cassettes for this endpoint to validate further — searched the Python
// and Java SDKs' test suites (live-integration-only, no stubs) and GitHub
// code search broadly; this client's own bigmodel_same_key.yaml cassette
// is the only real recorded traffic that exists anywhere for it.
// "moderation" (the request's only valid model value) currently returns
// 400 "Unknown Model" (code 1211) on ProdBaseURL AND on BigModelBaseURL for
// at least one GLM-Coding-Plan account, live-verified 2026-07-11 to be an
// account/plan-entitlement gate rather than a platform-routing issue — see
// EmbeddingsService's doc comment and docs/en/accounts-and-quota.md.
type ModerationsService struct {
	client *Client
}

// ModerationModel is the only value ModerationRequest.Model accepts.
const ModerationModel = "moderation"

// Moderation risk levels (ModerationResult.RiskLevel).
const (
	ModerationRiskPass   = "PASS"   // normal content
	ModerationRiskReview = "REVIEW" // suspicious content
	ModerationRiskReject = "REJECT" // violating content
)

// ModerationContent is one multimodal item to moderate, used when
// ModerationRequest.Input is a single object or an array of them instead of
// a plain string.
type ModerationContent struct {
	Type     string         `json:"type"` // text, image_url, video_url, audio_url
	Text     string         `json:"text,omitempty"`
	ImageURL *ModerationURL `json:"image_url,omitempty"`
	VideoURL *ModerationURL `json:"video_url,omitempty"`
	AudioURL *ModerationURL `json:"audio_url,omitempty"`
}

// ModerationURL wraps a URL reference for a ModerationContent media field.
type ModerationURL struct {
	URL string `json:"url"`
}

// ModerationRequest submits content for moderation. Input accepts a plain
// string (max 2000 chars), a single ModerationContent, or a []ModerationContent
// for mixed-media review.
type ModerationRequest struct {
	Model string `json:"model"` // defaults to ModerationModel when empty
	Input any    `json:"input"`
}

// ModerationResult is one flagged/cleared item in ModerationResponse.ResultList.
type ModerationResult struct {
	ContentType string   `json:"content_type"`
	RiskLevel   string   `json:"risk_level"` // ModerationRiskPass/Review/Reject
	RiskType    []string `json:"risk_type"`
}

// ModerationTextUsage counts moderation calls spent on text content.
type ModerationTextUsage struct {
	CallCount ModerationCallCount `json:"call_count"`
}

// ModerationCallCount is an int that unmarshals from either a JSON number or
// a JSON string of digits. The two official Z.AI SDKs disagree on this
// field's wire type — docs.bigmodel.cn's live OpenAPI spec declares
// call_count a number, but the official Java SDK's ModerationText class
// (github.com/zai-org/z-ai-sdk-java) declares it a String — and there's no
// live success response to settle it (see ModerationsService's doc
// comment), so this accepts either instead of risking a hard unmarshal
// failure on a real response.
type ModerationCallCount int

func (c *ModerationCallCount) UnmarshalJSON(b []byte) error {
	var n int
	if err := json.Unmarshal(b, &n); err == nil {
		*c = ModerationCallCount(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("call_count: expected a number or a numeric string, got %s", b)
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("call_count: expected a numeric string, got %q: %w", s, err)
	}
	*c = ModerationCallCount(v)
	return nil
}

// ModerationUsage is the usage accounting on a ModerationResponse.
type ModerationUsage struct {
	ModerationText *ModerationTextUsage `json:"moderation_text,omitempty"`
}

// ModerationResponse is the result of ModerationsService.Create.
type ModerationResponse struct {
	ID         string             `json:"id"`
	Created    int64              `json:"created"`
	RequestID  string             `json:"request_id"`
	ResultList []ModerationResult `json:"result_list"`
	Usage      ModerationUsage    `json:"usage"`
}

// Create submits content for moderation. req.Model defaults to
// ModerationModel ("moderation", the only value the API accepts) when
// empty. Authenticates with Config.ChinaAPIKey (falling back to
// Config.APIKey) against BigModelBaseURL, independent of Config.BaseURL.
func (s *ModerationsService) Create(ctx context.Context, req ModerationRequest) (*ModerationResponse, error) {
	if req.Model == "" {
		req.Model = ModerationModel
	}
	if req.Input == nil {
		return nil, fmt.Errorf("input is required")
	}

	var resp ModerationResponse
	apiKey := s.client.chinaAPIKey()
	if err := s.client.doRequestBaseKey(ctx, BigModelBaseURL, apiKey, "POST", "/moderations", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
