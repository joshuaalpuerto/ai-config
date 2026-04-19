package analyzer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/joshuaalpuerto/ai-config/internal/analyzer/parser"
)

const cacheFileName = ".aicfg-cache.json"

// Analyzer runs all five analysis phases and produces an AnalysisResult.
type Analyzer struct {
	// Since controls the git history window for churn analysis (default "6 months ago").
	Since string
	// Verbose includes the full per-file FileNode map in the result (default: omitted).
	Verbose bool
	// Cache enables reading/writing a cache file to avoid re-analysis on unchanged codebases.
	Cache bool
	// HubsN controls how many hub files are included in the report (default 10).
	HubsN int
	// HotspotsN controls how many hotspot files are included in the report (default 20).
	HotspotsN int
}

// New returns an Analyzer with default settings.
func New() *Analyzer {
	return &Analyzer{Since: "6 months ago", HubsN: 10, HotspotsN: 20}
}

// cacheEnvelope wraps an AnalysisResult with a fingerprint for cache validation.
type cacheEnvelope struct {
	Fingerprint string          `json:"fingerprint"`
	Result      *AnalysisResult `json:"result"`
}

// Analyze runs the full analysis pipeline on root and returns the result.
func (a *Analyzer) Analyze(root string) (*AnalysisResult, error) {
	since := a.Since
	if since == "" {
		since = "6 months ago"
	}

	// Phase 1: filesystem scan.
	scan, err := scan(root)
	if err != nil {
		return nil, fmt.Errorf("scanning %s: %w", root, err)
	}

	// Cache: compute fingerprint after we know the source files.
	var fingerprint string
	if a.Cache {
		fingerprint = computeFingerprint(scan.SourceFiles)
		if cached := a.readCache(root, fingerprint); cached != nil {
			return cached, nil
		}
	}

	// Phase 2 + 3: parse imports and build graph.
	cfg := parser.Config{
		ModulePath: scan.ModulePath,
		ModuleDir:  scan.GoModDir,
		TSAliases:  scan.TSAliases,
		RepoRoot:   root,
	}
	nodes, err := buildGraph(scan.SourceFiles, cfg)
	if err != nil {
		return nil, fmt.Errorf("building graph: %w", err)
	}

	// Phase 4: git churn.
	churn, err := analyzeGitChurn(root, since)
	if err != nil {
		return nil, fmt.Errorf("git churn analysis: %w", err)
	}
	applyChurn(nodes, churn)

	// Phase 5: domain clustering.
	clusters := buildClusters(nodes, scan.SourceRootName)

	result := &AnalysisResult{
		Root:              root,
		AnalyzedAt:        time.Now().UTC(),
		GitChurnAvailable: churn.available,
		TechStack:         scan.TechStack,
		TopLevelDirs:      scan.TopLevelDirs,
		Hubs:              topHubs(nodes, a.HubsN),
		Hotspots:          topHotspots(nodes, a.HotspotsN),
		Clusters:          clusters,
	}

	if a.Verbose {
		result.Files = nodesToFileMap(nodes)
	}

	if a.Cache && fingerprint != "" {
		a.writeCache(root, fingerprint, result)
	}

	return result, nil
}

// computeFingerprint produces a SHA-256 hash over sorted source file paths and their sizes.
// This is fast, deterministic, and does not require git.
func computeFingerprint(files []string) string {
	sorted := make([]string, len(files))
	copy(sorted, files)
	sort.Strings(sorted)

	h := sha256.New()
	for _, f := range sorted {
		fi, err := os.Stat(f)
		if err != nil {
			fmt.Fprintf(h, "%s\n", f)
			continue
		}
		fmt.Fprintf(h, "%s:%d\n", f, fi.Size())
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// readCache attempts to read a cached result from root/.aicfg-cache.json.
// Returns nil if the cache is absent, unreadable, or has a stale fingerprint.
func (a *Analyzer) readCache(root, fingerprint string) *AnalysisResult {
	path := filepath.Join(root, cacheFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var env cacheEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil
	}
	if env.Fingerprint != fingerprint {
		return nil
	}
	return env.Result
}

// writeCache persists the result to root/.aicfg-cache.json. Errors are silently ignored
// so a cache write failure never blocks analysis output.
func (a *Analyzer) writeCache(root, fingerprint string, result *AnalysisResult) {
	env := cacheEnvelope{Fingerprint: fingerprint, Result: result}
	data, err := json.Marshal(env)
	if err != nil {
		return
	}
	path := filepath.Join(root, cacheFileName)
	_ = os.WriteFile(path, data, 0o644)
}
