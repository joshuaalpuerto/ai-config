package analyzer

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// churnResult holds per-file churn counts.
type churnResult struct {
	available bool
	counts    map[string]int // repo-relative path → churn count
}

// analyzeGitChurn runs git log to count per-file changes in the last `since` window.
// If .git is absent, it returns available=false without error.
func analyzeGitChurn(root, since string) (churnResult, error) {
	gitDir := filepath.Join(root, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return churnResult{available: false, counts: map[string]int{}}, nil
	}

	args := []string{
		"-C", root,
		"log", "--format=format:", "--name-only",
		"--since=" + since,
	}
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		// git exists but command failed (e.g. empty repo); treat as unavailable.
		return churnResult{available: false, counts: map[string]int{}}, nil
	}

	counts := make(map[string]int)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		counts[filepath.ToSlash(line)]++
	}

	return churnResult{available: true, counts: counts}, nil
}

// applyChurn backfills churn counts into the graph and recomputes priorities.
func applyChurn(nodes map[string]*graphNode, churn churnResult) {
	for path, node := range nodes {
		node.churn = churn.counts[path]
	}
	computePriorities(nodes)
}

// topHotspots returns the top N hotspot files (churn > 3 AND lines > 50),
// sorted descending by churn × lines.
func topHotspots(nodes map[string]*graphNode, n int) []Hotspot {
	var candidates []Hotspot
	for _, node := range nodes {
		if node.churn > 3 && node.lines > 50 {
			candidates = append(candidates, Hotspot{
				Path:  node.path,
				Churn: node.churn,
				Lines: node.lines,
				Score: node.churn * node.lines,
			})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > n {
		candidates = candidates[:n]
	}
	return candidates
}

// lastUpdatedResult holds per-file last-commit timestamps.
type lastUpdatedResult struct {
	available bool
	dates     map[string]time.Time // repo-relative path → most recent commit time
}

// analyzeLastUpdated returns the most recent commit timestamp for each given file.
// It runs a single git log invocation and uses the first (most recent) timestamp
// seen per path. If .git is absent or the command fails, it returns available=false
// without error (consistent with analyzeGitChurn).
func analyzeLastUpdated(root string, files []string) (lastUpdatedResult, error) {
	gitDir := filepath.Join(root, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return lastUpdatedResult{available: false, dates: map[string]time.Time{}}, nil
	}

	args := []string{
		"-C", root,
		"log", "--format=%ai", "--name-only",
		"--diff-filter=ACM",
	}
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return lastUpdatedResult{available: false, dates: map[string]time.Time{}}, nil
	}

	// Build a set of the requested files for fast lookup.
	wanted := make(map[string]struct{}, len(files))
	for _, f := range files {
		wanted[filepath.ToSlash(f)] = struct{}{}
	}

	dates := make(map[string]time.Time)
	var currentDate time.Time

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Lines starting with a digit are timestamps from --format=%ai (e.g. "2026-01-15 10:23:00 +0000").
		if len(line) > 0 && line[0] >= '0' && line[0] <= '9' {
			if t, err := time.Parse("2006-01-02 15:04:05 -0700", line); err == nil {
				currentDate = t
			}
			continue
		}
		// Otherwise it's a filename from --name-only.
		path := filepath.ToSlash(line)
		if _, ok := wanted[path]; !ok {
			continue
		}
		// First occurrence = most recent commit (git log is reverse-chrono).
		if _, seen := dates[path]; !seen {
			dates[path] = currentDate
		}
	}

	return lastUpdatedResult{available: true, dates: dates}, nil
}
