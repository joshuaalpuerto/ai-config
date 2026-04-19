package analyzer

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
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
