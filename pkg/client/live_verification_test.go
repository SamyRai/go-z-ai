package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

// These two tests replay real recorded interactions from a live account
// (2026-07-10, coding_plan type) against the general PAYG base
// (ProdBaseURL) to lock in a confirmed finding: both endpoints are reachable
// and correctly routed, but reject the model names documented in the
// official SDK's own test fixtures ("moderation", "embedding-2") with
// error 1211 "Unknown Model". This rules out a base-URL/routing problem —
// the remaining unknown was the correct live model code vs. an account/plan
// gate, resolved by TestBigModelSameKeyAuthenticates below (it's a plan
// gate — see docs/accounts-and-quota.md).
//
// No service type exists yet for either endpoint (that's the point — we
// don't have a confirmed success response to model), so these replay
// directly through doRequestBase and assert on the parsed *APIError.

func TestModerationsLiveErrorIsUnknownModelNotRouting(t *testing.T) {
	c := newReplayClient(t, "moderations", ProdBaseURL)

	var result any
	err := c.doRequestBase(context.Background(), ProdBaseURL, "POST", "/moderations", map[string]any{
		"model": "moderation",
		"input": "hello world",
	}, &result)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError (confirms the endpoint is real and routes correctly), got %T: %v", err, err)
	}
	if apiErr.Code != 1211 {
		t.Errorf("expected code 1211 (Unknown Model), got %d: %s", apiErr.Code, apiErr.Message)
	}
}

// TestBigModelSameKeyAuthenticates replays four real recorded interactions
// against open.bigmodel.cn (2026-07-11, same z.ai API key as the ProdBaseURL
// cassettes above, Authorization redacted) — the finding that overturned an
// earlier online-research-only conclusion that Embeddings/Moderations were
// "China platform only" and needed a separate bigmodel.cn account/key. Kept
// as one cassette/test rather than split per-endpoint (unlike the tests
// below): the four calls aren't independent facts, they're one claim (a
// single z.ai key authenticates identically on both platforms) that only
// holds together as a set — splitting it would scatter one finding across
// four files with none of them individually meaningful.
// GET /models returns the identical 8-model chat-only catalog as
// api.z.ai — proving the same account, not a platform-specific one.
// POST /chat/completions clears auth and reaches a billing-level error
// (1113 insufficient balance), proving the key genuinely authenticates on
// this platform, not just gets rejected before reaching model logic.
// POST /embeddings and /moderations both still return 400 "Unknown Model"
// (1211) here too — identical to ProdBaseURL — which is why this is an
// account/plan-entitlement gate (this account's catalog is chat-only on
// both platforms), not a China-vs-international routing issue. See
// BigModelBaseURL's doc comment and docs/accounts-and-quota.md.
func TestBigModelSameKeyAuthenticates(t *testing.T) {
	c := newReplayClient(t, "bigmodel_same_key", BigModelBaseURL)
	apiKey := "replayed-from-cassette"

	var models struct {
		Data []struct{ ID string } `json:"data"`
	}
	if err := c.doRequestBaseKey(context.Background(), BigModelBaseURL, apiKey, "GET", "/models", nil, &models); err != nil {
		t.Fatalf("GET /models: %v", err)
	}
	if len(models.Data) != 8 || models.Data[0].ID != "glm-4.5" {
		t.Errorf("expected the same 8-model chat-only catalog as ProdBaseURL, got %+v", models.Data)
	}

	var chatResult any
	chatErr := c.doRequestBaseKey(context.Background(), BigModelBaseURL, apiKey, "POST", "/chat/completions", map[string]any{
		"model":    "glm-4.6",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	}, &chatResult)
	if apiErr, ok := chatErr.(*APIError); !ok || apiErr.Code != ErrCodeInsufficientBalance {
		t.Fatalf("expected code %d (insufficient balance — proves auth cleared and reached billing logic), got %v", ErrCodeInsufficientBalance, chatErr)
	}

	var embResult any
	embErr := c.doRequestBaseKey(context.Background(), BigModelBaseURL, apiKey, "POST", "/embeddings", map[string]any{
		"model": "embedding-3",
		"input": "hello world",
	}, &embResult)
	if apiErr, ok := embErr.(*APIError); !ok || apiErr.Code != 1211 {
		t.Errorf("expected code 1211 (Unknown Model) on BigModelBaseURL too, got %v", embErr)
	}

	var modResult any
	modErr := c.doRequestBaseKey(context.Background(), BigModelBaseURL, apiKey, "POST", "/moderations", map[string]any{
		"model": "moderation",
		"input": "hello world",
	}, &modResult)
	if apiErr, ok := modErr.(*APIError); !ok || apiErr.Code != 1211 {
		t.Errorf("expected code 1211 (Unknown Model) on BigModelBaseURL too, got %v", modErr)
	}
}

func TestEmbeddingsLiveErrorIsUnknownModelNotRouting(t *testing.T) {
	c := newReplayClient(t, "embeddings", ProdBaseURL)

	var result any
	err := c.doRequestBase(context.Background(), ProdBaseURL, "POST", "/embeddings", map[string]any{
		"model": "embedding-2",
		"input": "hello",
	}, &result)

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError (confirms the endpoint is real and routes correctly), got %T: %v", err, err)
	}
	if apiErr.Code != 1211 {
		t.Errorf("expected code 1211 (Unknown Model), got %d: %s", apiErr.Code, apiErr.Message)
	}
}

// assertInsufficientBalance is shared by every live-verification test below
// whose cassette records the account's universal pay-per-use gate (1113,
// wrapped in a service-method error) — the signal that a request reached
// real billing logic rather than 404ing on a wrong path.
func assertInsufficientBalance(t *testing.T, err error) {
	t.Helper()
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected a wrapped *APIError (confirms the endpoint is real and routes correctly), got %T: %v", err, err)
	}
	if apiErr.Code != ErrCodeInsufficientBalance {
		t.Errorf("expected code %d (insufficient balance), got %d: %s", ErrCodeInsufficientBalance, apiErr.Code, apiErr.Message)
	}
}

// TestToolsWebSearchLive replays a real recorded interaction (2026-07-11)
// proving tools.go's rewritten POST /web_search is real and correctly
// routed — direct evidence for the tools.go rewrite (see CHANGELOG.md):
// the previous version of this file hit an invented "ToolsBaseURL" +
// "/web/search" that was never live-verified. Replays as 1113 insufficient
// balance (this account has no PAYG balance for pay-per-use tools — the
// same entitlement gate as Embeddings/Moderations), which is itself the
// routing proof: a business-logic billing error only happens after the
// request reaches real endpoint logic.
func TestToolsWebSearchLive(t *testing.T) {
	c := newReplayClient(t, "tools_web_search", ProdBaseURL)
	_, err := c.Tools().WebSearch(context.Background(), WebSearchRequest{
		SearchQuery:  "golang generics",
		SearchEngine: SearchEnginePro,
		Count:        3,
	})
	assertInsufficientBalance(t, err)
}

// TestToolsWebReaderLive replays a real recorded interaction (2026-07-11)
// proving tools.go's rewritten POST /reader is real and correctly routed —
// see TestToolsWebSearchLive's doc comment for the full context. This
// endpoint was separately observed to fully succeed with real parsed
// content in an interactive session — not always rate/quota-gated, just
// intermittently (that transcript wasn't captured as a cassette, since it
// happened outside a recording session).
func TestToolsWebReaderLive(t *testing.T) {
	c := newReplayClient(t, "tools_web_reader", ProdBaseURL)
	_, err := c.Tools().WebReader(context.Background(), WebReaderRequest{URL: "https://go.dev"})
	assertInsufficientBalance(t, err)
}

// TestToolsTokenizerLive replays a real recorded interaction (2026-07-11)
// proving tools.go's rewritten POST /tokenizer is real and correctly
// routed — see TestToolsWebSearchLive's doc comment for the full context.
func TestToolsTokenizerLive(t *testing.T) {
	c := newReplayClient(t, "tools_tokenizer", ProdBaseURL)
	_, err := c.Tools().Tokenize(context.Background(), TokenizerRequest{
		Model:    "glm-4.6",
		Messages: []Message{{Role: "user", Content: "hello world, how are you today?"}},
	})
	assertInsufficientBalance(t, err)
}

// TestRerankCreateLive replays a real recorded interaction (2026-07-11)
// proving RerankService.Create's POST /rerank is real and correctly
// routed: 1211 Unknown Model, the same account-entitlement gate as
// Embeddings/Moderations (this account's catalog is chat-only).
func TestRerankCreateLive(t *testing.T) {
	c := newReplayClient(t, "rerank_create", ProdBaseURL)
	_, err := c.Rerank().Create(context.Background(), RerankRequest{
		Query:     "capital of France",
		Documents: []string{"Paris is the capital of France.", "Berlin is the capital of Germany."},
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 1211 {
		t.Errorf("expected code 1211 (Unknown Model — same entitlement gate as embeddings/moderations), got %v", err)
	}
}

// TestChatCreateAsyncLive replays a real recorded interaction (2026-07-11)
// proving ChatService.CreateAsync's POST /async/chat/completions is real
// and correctly routed: 1113 insufficient balance.
func TestChatCreateAsyncLive(t *testing.T) {
	c := newReplayClient(t, "chat_create_async", ProdBaseURL)
	_, err := c.Chat().CreateAsync(context.Background(), ChatRequest{
		Model:    "glm-4.6",
		Messages: []Message{{Role: "user", Content: "hi"}},
		TopP:     0.95,
	})
	assertInsufficientBalance(t, err)
}

// TestAgentsAsyncResultLive replays a real recorded interaction
// (2026-07-11: a real agent_id with a bogus async_id) proving
// AgentsService.AsyncResult's POST /v1/agents/async-result is real and
// correctly routed, and locking in the finding that shaped
// AgentAsyncResultResponse: the API returns HTTP 200 with a business-level
// failure embedded in the body ({"status":"failed","error":{...}}) — the
// documented schema doesn't mention this at all, matching the same
// 200-with-embedded-failure pattern as the synchronous Invoke path.
func TestAgentsAsyncResultLive(t *testing.T) {
	c := newReplayClient(t, "agents_async_result", ProdBaseURL)
	resp, err := c.Agents().AsyncResult(context.Background(), AgentAsyncResultRequest{
		AgentID: "intelligent_education_correction_polling",
		AsyncID: "nonexistent-async-id",
	})
	if err != nil {
		t.Fatalf("AsyncResult: %v (a business failure must not surface as a Go error — the HTTP status was 200)", err)
	}
	if !resp.Failed() || resp.Error == nil {
		t.Errorf("expected a business-level failure with a non-nil Error, got %+v", resp)
	}
}

// TestAudioSpeechLive replays a real recorded interaction (2026-07-11)
// proving AudioService.Speech's POST /audio/speech is real and correctly
// routed: 1211 Unknown Model — glm-tts isn't in this account's catalog
// either, the same entitlement gate as everything else pay-per-use.
func TestAudioSpeechLive(t *testing.T) {
	c := newReplayClient(t, "audio_speech", ProdBaseURL)
	_, err := c.Audio().Speech(context.Background(), AudioSpeechRequest{Input: "hello world"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 1211 {
		t.Errorf("expected code 1211 (Unknown Model — glm-tts not in this account's catalog either), got %v", err)
	}
}

// TestVoiceListLive replays a real recorded interaction (2026-07-11) —
// the one live-verification cassette in this whole file that captures an
// actual success, not just a business-logic error proving correct routing.
// GET /voice/list returned a real, populated, correctly-parsed 18-voice
// list despite this account having no PAYG balance: voice listing isn't
// pay-per-use, unlike every other endpoint verified in this file.
func TestVoiceListLive(t *testing.T) {
	c := newReplayClient(t, "voice_list", ProdBaseURL)
	voices, err := c.Voice().List(context.Background(), "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(voices) == 0 {
		t.Error("expected a non-empty voice list")
	}
	found := false
	for _, v := range voices {
		if v.VoiceName == "tongtong" && v.VoiceType == VoiceTypeOfficial {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find the default system voice \"tongtong\" (OFFICIAL) in the list, got %+v", voices)
	}
}

// TestFileParserSyncMissingFileTypeLive replays a real recorded interaction
// (2026-07-11: a sync parse call deliberately omitting file_type) proving
// the finding behind FileParserService.Sync's FileType validation: the
// documented spec marks file_type optional, but a real call without it
// returns HTTP 200 with a *third*, non-standard error envelope
// ({"msg":...,"code":500} — neither the usual {"error":{"code","message"}}
// shape nor FileParseResultResponse's shape). Because FileParserService.Sync
// now refuses a missing FileType before ever sending a request, this test
// bypasses it and calls sendMultipart directly — the same pattern
// TestModerationsLiveErrorIsUnknownModelNotRouting/
// TestEmbeddingsLiveErrorIsUnknownModelNotRouting use to replay a real
// interaction that predates (or in this case deliberately bypasses) a
// typed service method.
func TestFileParserSyncMissingFileTypeLive(t *testing.T) {
	c := newReplayClient(t, "files_parser_sync", ProdBaseURL)

	buf, contentType, err := buildParseMultipart(FileParserRequest{
		FileName: "test.txt",
		FileData: []byte("hello world"),
		ToolType: FileParserToolPrimeSync,
	})
	if err != nil {
		t.Fatalf("buildParseMultipart: %v", err)
	}

	resp, err := c.sendMultipart(context.Background(), "/files/parser/sync", contentType, buf.Bytes())
	if err != nil {
		t.Fatalf("sendMultipart: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 (the real server's non-standard-error-in-a-200 behavior), got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var result FileParseResultResponse
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Status != "" || result.TaskID != "" {
		t.Errorf("expected FileParseResultResponse to decode as empty (the response body doesn't match its shape at all), got %+v", result)
	}
}
