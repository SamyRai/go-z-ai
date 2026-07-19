package sitegen

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Commit is a recent commit summary for the activity feed.
type Commit struct {
	Hash         string    // short hash
	Subject      string    // first line of message (full, for tooltips)
	Description  string    // description without the conventional-commit prefix
	Author       string    // author name
	Date         time.Time // commit date (for relative-time display)
	URL          string    // full GitHub commit URL
	Type         string    // conventional-commit type: feat, fix, docs, chore, etc.
	Scope        string    // conventional-commit scope (e.g. "site", "ci")
}

// GitStats summarises repository activity from `git`.
type GitStats struct {
	Commits30d    int      // commits in last 30 days
	Contributors  int      // unique authors across all history
	RecentCommits []Commit // last 10
	LastRelease   string   // most recent tag, or empty
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

// conventionalCommitRE parses Conventional Commits format:
//   type(scope): description    → type=scope, scope=scope
//   type: description           → type=type, scope=""
var conventionalCommitRE = regexp.MustCompile(`^([a-z]+)(?:\(([^)]+)\))?!?:\s*(.+)$`)

func safeGitLog(arg string) []Commit {
	out, err := gitOutput(`log`, `--pretty=format:%h|%s|%an|%aI`, `--no-merges`, `-n`, strings.TrimPrefix(arg, "-"))
	if err != nil {
		return nil
	}
	var commits []Commit
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) != 4 {
			continue
		}
		c := Commit{
			Hash:       parts[0],
			Subject:    parts[1],
			Author:     parts[2],
			Description: parts[1], // default: full subject
		}
		if t, err := time.Parse(time.RFC3339, parts[3]); err == nil {
			c.Date = t
		}
		// Parse conventional-commit type/scope and strip the prefix from the
		// description so the type badge + scope badge don't duplicate it.
		if m := conventionalCommitRE.FindStringSubmatch(parts[1]); m != nil {
			c.Type = m[1]
			c.Scope = m[2]
			c.Description = m[3]
		}
		commits = append(commits, c)
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
