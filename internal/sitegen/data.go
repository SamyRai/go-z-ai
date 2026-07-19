package sitegen

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GitHubRelease maps the fields we consume from the GitHub releases API.
type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	PublishedAt time.Time     `json:"published_at"`
	HTMLURL     string        `json:"html_url"`
	Body        string        `json:"body"`
	Prerelease  bool          `json:"prerelease"`
	Assets      []GitHubAsset `json:"assets"`
}

// GitHubAsset is a single downloadable artifact attached to a release.
type GitHubAsset struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	Download   int    `json:"download_count"`
	BrowserURL string `json:"browser_download_url"`
}

// GitHubRepo maps repo-summary fields we display on the landing page.
type GitHubRepo struct {
	Stars         int    `json:"stargazers_count"`
	Forks         int    `json:"forks_count"`
	Watchers      int    `json:"subscribers_count"`
	OpenIssues    int    `json:"open_issues_count"`
	Description   string `json:"description"`
	Homepage      string `json:"homepage"`
	DefaultBranch string `json:"default_branch"`
}

// FetchGitHubRepo returns repo summary stats, or a zero value on error.
// Unauthenticated calls are rate-limited to 60/h per IP, which is plenty
// for build-time generation.
func FetchGitHubRepo(ctx context.Context, client *http.Client, owner, repo string) GitHubRepo {
	var r GitHubRepo
	u := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	if err := getJSON(ctx, client, u, &r); err != nil {
		return GitHubRepo{}
	}
	return r
}

// FetchReleases returns up to limit releases. Newest first (API default).
func FetchReleases(ctx context.Context, client *http.Client, owner, repo string, limit int) []GitHubRelease {
	var rs []GitHubRelease
	u := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=%d", owner, repo, limit)
	if err := getJSON(ctx, client, u, &rs); err != nil {
		return nil
	}
	return rs
}

func getJSON(ctx context.Context, client *http.Client, url string, v any) error {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "go-z-ai-sitegen")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	// Bound the response: GitHub's API responses are small (<<4 MiB) but a
	// misrouted request or a non-JSON error page could otherwise stream
	// unbounded bytes into the decoder.
	return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(v)
}
