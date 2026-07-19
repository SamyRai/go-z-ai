package sitegen

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Commit is a recent commit summary for the activity feed.
type Commit struct {
	Hash        string    // short hash
	Subject     string    // first line of message (full, for tooltips)
	Description string    // description without the conventional-commit prefix
	Author      string    // author name
	Date        time.Time // commit date (for relative-time display)
	URL         string    // full GitHub commit URL
	Type        string    // conventional-commit type: feat, fix, docs, chore, etc.
	Scope       string    // conventional-commit scope (e.g. "site", "ci")
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
	s.Commits30d = gitCount(`rev-list`, `--count`, `--since=30 days ago`, `HEAD`)
	s.Contributors = gitLineCount(`shortlog`, `-sne`, `--all`)
	s.RecentCommits = safeGitLog(`-10`)
	s.LastRelease = safeGitFirst(`describe`, `--tags`, `--abbrev=0`)
	return s
}

// gitCount parses the integer output of `git rev-list --count …`.
func gitCount(args ...string) int {
	out, err := gitOutput(args...)
	if err != nil {
		return 0
	}
	out = strings.TrimSpace(out)
	var n int
	if _, err := fmt.Sscanf(out, "%d", &n); err != nil {
		return 0
	}
	return n
}

// gitLineCount counts non-empty output lines. Used for `git shortlog -sne`
// where each line is one contributor. Trims a possible missing trailing
// newline (git shortlog emits one line per group, no trailing newline) so a
// single-contributor repo reports 1, not 0.
func gitLineCount(args ...string) int {
	out, err := gitOutput(args...)
	if err != nil {
		return 0
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return 0
	}
	return len(strings.Split(out, "\n"))
}

// conventionalCommitRE parses Conventional Commits format:
//
//	type(scope): description    → type=scope, scope=scope
//	type: description           → type=type, scope=""
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
			Hash:        parts[0],
			Subject:     parts[1],
			Author:      parts[2],
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

// gitCommitDate returns the committer date of HEAD as a time.Time and true,
// or zero/false if git is unavailable or HEAD has no commits (e.g. a fresh
// checkout in CI before any commit lands). Used as the deterministic build
// clock so two builds of the same commit produce identical site output.
func gitCommitDate() (time.Time, bool) {
	out, err := gitOutput("log", "-1", "--format=%cI")
	if err != nil {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(out))
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func gitOutput(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), nil
}
