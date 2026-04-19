package analyzer

import (
	"fmt"
	"time"

	"github.com/joshuaalpuerto/ai-config/internal/analyzer/parser"
)

// Analyzer runs all five analysis phases and produces an AnalysisResult.
type Analyzer struct {
	// Since controls the git history window for churn analysis (default "6 months ago").
	Since string
}

// New returns an Analyzer with default settings.
func New() *Analyzer {
	return &Analyzer{Since: "6 months ago"}
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
	clusters := buildClusters(nodes)

	result := &AnalysisResult{
		Root:              root,
		AnalyzedAt:        time.Now().UTC(),
		GitChurnAvailable: churn.available,
		TechStack:         scan.TechStack,
		Domains:           scan.Domains,
		Hubs:              topHubs(nodes, 10),
		Hotspots:          topHotspots(nodes, 10),
		Clusters:          clusters,
		Files:             nodesToFileMap(nodes),
	}

	return result, nil
}
