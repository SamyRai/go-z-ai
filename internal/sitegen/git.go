package sitegen

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Commit is a recent commit summary for the activity feed.
type Commit struct {
	Hash    string // short hash
	Subject string // first line of message
	Author  string
}

// GitStats summarises repository activity from `git`.
type GitStats struct {
	Commits30d    int       // commits in last 30 days
	Contributors  int       // unique authors across all history
	RecentCommits []Commit  // last 10
	LastRelease   string    // most recent tag, or empty
}

// CollectGitStats shells out to git. Any failure degrades to a zero-value
// GitStats — the generator never fails on git problems.
func CollectGitStats() GitStats {
	var s GitStats
	s.Commits30d = safeGitInt(`rev-list`, `--count`, `--since=30 days ago`, `HEAD`)
	s.Contributors = safeGitInt(`shortlog`, `-sne`, `--all`)
	s.RecentCommits = safeGitLog(`-10`)
	s.LastRelease = safeGitFirst(`describe`, `--tags`, `--abbrev=0`)
	return s
}

func safeGitInt(args ...string) int {
	out, err := gitOutput(args...)
	if err != nil || len(out) == 0 {
		return 0
	}
	// shortlog output has lines like "    42\tName <email>"; rev-list is just a number.
	// For shortlog, count non-empty lines. For rev-list, parse the int.
	if n := strings.Count(out, "\n"); n > 0 && !isPureNumber(strings.TrimSpace(out)) {
		return n
	}
	var n int
	for _, r := range out {
		if r < '0' || r > '9' {
			// Not a pure number — fall back to line count.
		}
		_ = r
	}
	if isPureNumber(strings.TrimSpace(out)) {
		_, _ = fmt.Sscanf(strings.TrimSpace(out), "%d", &n)
		return n
	}
	return strings.Count(out, "\n")
}

func safeGitLog(arg string) []Commit {
	out, err := gitOutput(`log`, `--pretty=format:%h|%s|%an`, `--no-merges`, `-n`, strings.TrimPrefix(arg, "-"))
	if err != nil {
		return nil
	}
	var commits []Commit
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		commits = append(commits, Commit{Hash: parts[0], Subject: parts[1], Author: parts[2]})
	}
	return commits
}

func safeGitFirst(args ...string) string {
	out, err := gitOutput(args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), nil
}

func isPureNumber(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
