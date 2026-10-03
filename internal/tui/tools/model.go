// Package tools implements the TUI's Tools tab: web search, web reader, and
// tokenizer forms over the same ToolsService the "go-z-ai tools" commands
// use.
package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/SamyRai/go-z-ai/internal/tui/formtab"
	"github.com/SamyRai/go-z-ai/pkg/client"
)

// New builds the Tools screen. c returns the current API client; selfTab is
// the screen's tab index in the root model.
func New(c func() *client.Client, selfTab int) formtab.Model {
	return formtab.New(c, selfTab,
		formtab.Form{Name: "Web Search", Placeholder: "search query", Run: webSearch},
		formtab.Form{Name: "Web Reader", Placeholder: "https://...", Run: readPage},
		formtab.Form{Name: "Tokenizer", Placeholder: "text to tokenize", Run: tokenize},
	)
}

func webSearch(ctx context.Context, c *client.Client, query string) (string, error) {
	resp, err := c.Tools().WebSearch(ctx, client.WebSearchRequest{SearchQuery: query, SearchEngine: client.SearchEnginePrime})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, r := range resp.SearchResult {
		fmt.Fprintf(&b, "%s\n%s\n\n", r.Title, r.Link)
	}
	return b.String(), nil
}

func readPage(ctx context.Context, c *client.Client, url string) (string, error) {
	resp, err := c.Tools().WebReader(ctx, client.WebReaderRequest{URL: url, WithImagesSummary: true})
	if err != nil {
		return "", err
	}
	if resp.ReaderResult == nil {
		return "", fmt.Errorf("empty response (id %s)", resp.ID)
	}
	return resp.ReaderResult.Content, nil
}

func tokenize(ctx context.Context, c *client.Client, text string) (string, error) {
	resp, err := c.Tools().Tokenize(ctx, client.TokenizerRequest{
		Model:    client.DefaultModel,
		Messages: []client.Message{{Role: "user", Content: text}},
	})
	if err != nil {
		return "", err
	}
	if resp.Usage == nil {
		return "", errors.New("empty response")
	}
	return fmt.Sprintf("%s: %d tokens", client.DefaultModel, resp.Usage.TotalTokens), nil
}
