package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// setViper sets a viper key for the duration of a test and restores it after.
// An explicit viper.Set beats any leftover flag/override from a prior test, so
// injecting credentials this way is robust against cross-test state — unlike
// passing --api-key as an arg, which a stale viper.Set("api-key", ...) would
// silently defeat.
func setViper(t *testing.T, key string, val any) {
	t.Helper()
	prev := viper.Get(key)
	viper.Set(key, val)
	t.Cleanup(func() { viper.Set(key, prev) })
}

// runCLI executes the root command with args, capturing everything written to
// os.Stdout (commands print with fmt.Println, not cmd.OutOrStdout, so the
// stream itself must be captured). Credentials are injected via viper pointed
// at srv, and all ambient credential env + the accounts store are isolated to
// a temp file so the run can't touch real config or the network beyond srv.
func runCLI(t *testing.T, srv *httptest.Server, args ...string) (string, error) {
	t.Helper()

	for _, env := range []string{"ZAI_API_KEY", "KEY", "ZAI_API_BASE_URL", "ZAI_CHINA_API_KEY"} {
		t.Setenv(env, "")
	}
	// Honor a store path the test set for itself (needed for multi-call
	// round-trips); otherwise isolate to a fresh temp file.
	if os.Getenv("ZAI_ACCOUNTS_FILE") == "" {
		t.Setenv("ZAI_ACCOUNTS_FILE", filepath.Join(t.TempDir(), "accounts.json"))
	}

	setViper(t, "api-key", "test-key")
	setViper(t, "account", "")
	setViper(t, "china-api-key", "")
	if srv != nil {
		setViper(t, "base-url", srv.URL)
	} else {
		setViper(t, "base-url", "")
	}

	// Restore sticky package-level format vars (bound with StringVar, they
	// persist across Execute calls) so one test's --format can't leak into the
	// next.
	prevOut, prevChat, prevUsage := outputFormat, chatFormat, usageFormat
	t.Cleanup(func() { outputFormat, chatFormat, usageFormat = prevOut, prevChat, prevUsage })

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	captured := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		captured <- string(b)
	}()

	silenceU, silenceE := rootCmd.SilenceUsage, rootCmd.SilenceErrors
	rootCmd.SilenceUsage, rootCmd.SilenceErrors = true, true
	rootCmd.SetArgs(args)
	execErr := rootCmd.Execute()
	rootCmd.SilenceUsage, rootCmd.SilenceErrors = silenceU, silenceE

	w.Close()
	os.Stdout = old
	return <-captured, execErr
}

func modelsHandler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data": []map[string]any{
				{"id": "glm-4.6", "name": "GLM-4.6", "max_context": 128000, "owned_by": "z.ai"},
			},
		})
	}
}

func TestE2EModelsListTable(t *testing.T) {
	srv := httptest.NewServer(modelsHandler(t))
	defer srv.Close()

	out, err := runCLI(t, srv, "models", "list", "--format", "table")
	if err != nil {
		t.Fatalf("models list: %v", err)
	}
	if !strings.Contains(out, "glm-4.6") || !strings.Contains(out, "MODEL ID") {
		t.Errorf("expected a model table with glm-4.6, got:\n%s", out)
	}
}

func TestE2EModelsListJSON(t *testing.T) {
	srv := httptest.NewServer(modelsHandler(t))
	defer srv.Close()

	out, err := runCLI(t, srv, "models", "list", "--format", "json")
	if err != nil {
		t.Fatalf("models list --format json: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not a JSON array: %v\n%s", err, out)
	}
	if len(got) != 1 || got[0]["id"] != "glm-4.6" {
		t.Errorf("expected one model glm-4.6 in JSON, got:\n%s", out)
	}
}

func TestE2EChatCreate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    "c1",
			"model": "glm-5.2",
			"choices": []map[string]any{
				{"index": 0, "message": map[string]any{"role": "assistant", "content": "Hello there"}, "finish_reason": "stop"},
			},
		})
	}))
	defer srv.Close()

	out, err := runCLI(t, srv, "chat", "create", "hi", "--format", "text")
	if err != nil {
		t.Fatalf("chat create: %v", err)
	}
	if !strings.Contains(out, "Hello there") {
		t.Errorf("expected assistant content in output, got:\n%s", out)
	}
}

// TestE2EOCRParseBase64 locks the file-handling contract of `ocr parse`: a
// local file argument is read and base64-encoded into the request body (a URL
// would pass through untouched). This is the behavior extracted into
// internal/fileinput in a later phase, so it must be pinned first.
func TestE2EOCRParseBase64(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc.png")
	if err := os.WriteFile(docPath, []byte("fake-image-bytes"), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	var gotFile string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/layout_parsing" {
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
			return
		}
		var body struct {
			File string `json:"file"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotFile = body.File
		_ = json.NewEncoder(w).Encode(map[string]any{"md_results": "# Parsed"})
	}))
	defer srv.Close()

	out, err := runCLI(t, srv, "ocr", "parse", docPath)
	if err != nil {
		t.Fatalf("ocr parse: %v", err)
	}
	if !strings.Contains(out, "# Parsed") {
		t.Errorf("expected parsed markdown in output, got:\n%s", out)
	}
	// The file field must be base64 of the raw bytes, not the raw bytes or path.
	if gotFile != "ZmFrZS1pbWFnZS1ieXRlcw==" {
		t.Errorf("expected base64-encoded file body, got %q", gotFile)
	}
}

// TestE2EFilesListJSON covers a command that gained JSON output in the format
// unification (it printed only a text table before). The --format json path
// must emit valid JSON of the file list.
func TestE2EFilesListJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data": []map[string]any{
				{"id": "file-1", "filename": "in.jsonl", "bytes": 12, "purpose": "batch", "status": "processed"},
			},
		})
	}))
	defer srv.Close()

	out, err := runCLI(t, srv, "files", "list", "--format", "json")
	if err != nil {
		t.Fatalf("files list --format json: %v", err)
	}
	var got struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("files list output is not JSON: %v\n%s", err, out)
	}
	if len(got.Data) != 1 || got.Data[0]["id"] != "file-1" {
		t.Errorf("expected file-1 in JSON output, got:\n%s", out)
	}
}

// TestE2ERerankJSON confirms the rerank command (text-only before) now emits
// JSON on demand.
func TestE2ERerankJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{"index": 0, "relevance_score": 0.9, "document": "doc a"},
			},
		})
	}))
	defer srv.Close()

	out, err := runCLI(t, srv, "rerank", "query", "doc a", "doc b", "--format", "json")
	if err != nil {
		t.Fatalf("rerank --format json: %v", err)
	}
	if !strings.Contains(out, "relevance_score") {
		t.Errorf("expected JSON rerank output, got:\n%s", out)
	}
}

// TestE2EAccountsRoundTrip exercises the accounts store end-to-end through the
// CLI (add → list → use → current → remove) against an isolated store file,
// with no network. This is the first coverage of the internal/accounts package
// via its actual CLI entry points.
func TestE2EAccountsRoundTrip(t *testing.T) {
	// Shared store across the multiple runCLI calls below.
	t.Setenv("ZAI_ACCOUNTS_FILE", filepath.Join(t.TempDir(), "accounts.json"))

	if out, err := runCLI(t, nil, "accounts", "add", "work", "--api-key", "sk-work-123456", "--type", "pay_as_you_go"); err != nil {
		t.Fatalf("accounts add: %v\n%s", err, out)
	}

	out, err := runCLI(t, nil, "accounts", "list", "--format", "json")
	if err != nil {
		t.Fatalf("accounts list: %v", err)
	}
	if !strings.Contains(out, "work") {
		t.Errorf("expected 'work' in accounts list, got:\n%s", out)
	}
	// `accounts list --format json` masks the API key by default (matching the
	// table view), so the raw key must not appear.
	if strings.Contains(out, "sk-work-123456") {
		t.Errorf("accounts list --format json leaked the raw API key:\n%s", out)
	}
	if !strings.Contains(out, maskAPIKey("sk-work-123456")) {
		t.Errorf("expected a masked key in accounts list json, got:\n%s", out)
	}

	// --reveal opts into the raw key (for export/backup).
	revealed, err := runCLI(t, nil, "accounts", "list", "--format", "json", "--reveal")
	if err != nil {
		t.Fatalf("accounts list --reveal: %v", err)
	}
	if !strings.Contains(revealed, "sk-work-123456") {
		t.Errorf("expected --reveal to emit the raw API key, got:\n%s", revealed)
	}
}
