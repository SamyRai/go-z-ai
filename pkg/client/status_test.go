package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// chatProbe records the chat request CheckAccountStatus sends (if any) and
// answers the quota and chat endpoints with canned bodies.
type chatProbe struct {
	quota      stubReply
	chat       stubReply
	chatBodies []ChatRequest
}

func (p *chatProbe) RoundTrip(r *http.Request) (*http.Response, error) {
	reply := p.quota
	if strings.HasSuffix(r.URL.Path, "/chat/completions") {
		var req ChatRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		p.chatBodies = append(p.chatBodies, req)
		reply = p.chat
	}
	return &http.Response{StatusCode: reply.status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(reply.body)), Request: r}, nil
}

func statusWith(t *testing.T, p *chatProbe) *AccountStatus {
	t.Helper()
	c, err := NewClient(Config{APIKey: "k", MaxRetries: -1, HTTPClient: &http.Client{Transport: p}})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	st, err := c.Detection().CheckAccountStatus(context.Background())
	if err != nil {
		t.Fatalf("CheckAccountStatus: %v", err)
	}
	return st
}

// A coding-plan key is checked through the free quota endpoint — never with
// a billed chat completion, which would consume plan quota.
func TestAccountStatusCodingPlanIsFree(t *testing.T) {
	p := &chatProbe{quota: stubReply{status: 200, body: `{"success":true,"data":{"level":"pro","limits":[{"type":"CREDIT_LIMIT","unit":3,"number":5,"usage":100,"currentValue":10}]}}`}}
	st := statusWith(t, p)
	if !st.APIAccessible || !st.HasBalance || st.Account.Type != AccountTypeCodingPlan {
		t.Errorf("unexpected status %+v", st)
	}
	if len(p.chatBodies) != 0 {
		t.Errorf("coding-plan status check sent %d billed chat requests", len(p.chatBodies))
	}
}

// A pay-as-you-go key gets one minimal probe on the cheapest paid model with
// thinking off; an out-of-balance answer means accessible without balance.
func TestAccountStatusPayAsYouGoProbe(t *testing.T) {
	p := &chatProbe{
		quota: stubReply{status: 401, body: `{"error":{"code":"1000","message":"no plan"}}`},
		chat:  stubReply{status: 429, body: `{"error":{"code":"1113","message":"insufficient balance"}}`},
	}
	st := statusWith(t, p)
	if !st.APIAccessible || st.HasBalance {
		t.Errorf("unexpected status %+v", st)
	}
	if len(p.chatBodies) != 1 {
		t.Fatalf("sent %d probes, want 1", len(p.chatBodies))
	}
	probe := p.chatBodies[0]
	if probe.Model != BalanceProbeModel || probe.MaxTokens != 1 || probe.Thinking == nil || probe.Thinking.Type != ThinkingDisabled {
		t.Errorf("unexpected probe %+v", probe)
	}
}
