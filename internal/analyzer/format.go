package analyzer

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// FormatContext renders an AnalysisResult as a compact markdown summary suitable for
// use as LLM context. The output describes the codebase structure without reading any
// source files — typically under 300 tokens for most codebases.
func FormatContext(r *AnalysisResult) string {
	var b strings.Builder

	// Header
	name := filepath.Base(r.Root)
	if name == "." || name == "" {
		name = r.Root
	}
	fmt.Fprintf(&b, "## Codebase: %s\n", name)
	fmt.Fprintf(&b, "**Analyzed:** %s\n\n", r.AnalyzedAt.Format("2006-01-02"))

	// Tech stack
	if len(r.TechStack.Languages) > 0 || len(r.TechStack.Frameworks) > 0 {
		parts := append([]string{}, r.TechStack.Languages...)
		parts = append(parts, r.TechStack.Frameworks...)
		fmt.Fprintf(&b, "**Stack:** %s\n\n", strings.Join(parts, ", "))
	}

	// Top-level directories
	if len(r.TopLevelDirs) > 0 {
		sorted := append([]string{}, r.TopLevelDirs...)
		sort.Strings(sorted)
		fmt.Fprintf(&b, "**Source folders:** %s\n\n", strings.Join(sorted, ", "))
	}

	// Hub files
	if len(r.Hubs) > 0 {
		fmt.Fprintf(&b, "## Hub Files (most depended-on)\n")
		for i, hub := range r.Hubs {
			if i >= 10 {
				break
			}
			line := fmt.Sprintf("%d. `%s` — imported by %d files", i+1, hub.Path, hub.FanIn)
			if len(hub.ExportNames) > 0 {
				shown := hub.ExportNames
				if len(shown) > 6 {
					shown = shown[:6]
				}
				line += fmt.Sprintf(". Exports: %s", strings.Join(shown, ", "))
			}
			fmt.Fprintln(&b, line)
		}
		fmt.Fprintln(&b)
	}

	// Feature clusters (non-singleton only; singletons collapsed into misc count)
	nonSingletons := make([]Cluster, 0, len(r.Clusters))
	singletonCount := 0
	for _, c := range r.Clusters {
		if c.Singleton {
			singletonCount++
		} else {
			nonSingletons = append(nonSingletons, c)
		}
	}
	if len(nonSingletons) > 0 {
		fmt.Fprintf(&b, "## Feature Clusters\n")
		for _, c := range nonSingletons {
			line := fmt.Sprintf("- **%s** (%d files)", c.Label, c.Size)
			if len(c.DependsOn) > 0 {
				line += fmt.Sprintf(" → depends on: %s", strings.Join(c.DependsOn, ", "))
			}
			fmt.Fprintln(&b, line)
		}
		if singletonCount > 0 {
			fmt.Fprintf(&b, "- **misc** (%d isolated files)\n", singletonCount)
		}
		fmt.Fprintln(&b)
	}

	// Hotspots
	if len(r.Hotspots) > 0 {
		fmt.Fprintf(&b, "## Hotspots (high churn + size)\n")
		for i, h := range r.Hotspots {
			if i >= 10 {
				break
			}
			fmt.Fprintf(&b, "- `%s` — changed %dx, %d lines\n", h.Path, h.Churn, h.Lines)
		}
		fmt.Fprintln(&b)
	} else if !r.GitChurnAvailable {
		fmt.Fprintf(&b, "_Git history unavailable — hotspot analysis skipped._\n\n")
	}

	return strings.TrimRight(b.String(), "\n")
}
