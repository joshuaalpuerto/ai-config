package analyzer

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// FormatContext renders an AnalysisResult as a compact markdown summary suitable for
// use as LLM context. The output describes the codebase structure without reading any
// source files.
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

	// File tree structure — use AllFiles for a complete snapshot; fall back to SourceFiles.
	treeFiles := r.AllFiles
	if len(treeFiles) == 0 {
		treeFiles = r.SourceFiles
	}
	if len(treeFiles) > 0 {
		fmt.Fprintf(&b, "## Structure\n")
		fmt.Fprint(&b, renderFileTree(treeFiles, false))
		fmt.Fprintln(&b)
	}

	// Hub files
	if len(r.Hubs) > 0 {
		fmt.Fprintf(&b, "## Hub Files (most depended-on)\n")
		for i, hub := range r.Hubs {
			if i >= 10 {
				break
			}
			line := fmt.Sprintf("%d. `%s` — imported by %d files", i+1, hub.Path, hub.FanIn)
			if hub.FileDoc != "" {
				line += fmt.Sprintf(". %s", hub.FileDoc)
			} else if len(hub.ExportNames) > 0 {
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

	// Clusters
	nonSingleton := 0
	for _, c := range r.Clusters {
		if !c.Singleton {
			nonSingleton++
		}
	}
	if nonSingleton > 0 {
		fmt.Fprintf(&b, "## Clusters (package groups)\n")
		for _, c := range r.Clusters {
			if c.Singleton {
				continue
			}
			line := fmt.Sprintf("- **%s** — %d files", c.Label, c.Size)
			if len(c.DependsOn) > 0 {
				line += fmt.Sprintf(". Depends on: %s", strings.Join(c.DependsOn, ", "))
			}
			fmt.Fprintln(&b, line)
		}
		fmt.Fprintln(&b)
	}

	// Doc Coverage
	if len(r.Coverage) > 0 {
		fmt.Fprintf(&b, "## Doc Coverage\n")
		fmt.Fprintf(&b, "| Area | Status | Mentioned In |\n")
		fmt.Fprintf(&b, "|------|--------|-------------|\n")
		for _, c := range r.Coverage {
			status := "covered"
			docs := strings.Join(c.DocFiles, ", ")
			if !c.Covered {
				status = "**GAP**"
				docs = "—"
			}
			fmt.Fprintf(&b, "| %s | %s | %s |\n", c.Area, status, docs)
		}
		fmt.Fprintln(&b)
	}

	return strings.TrimRight(b.String(), "\n")
}

// treeNode represents a directory or file in the rendered tree.
type treeNode struct {
	name     string
	children map[string]*treeNode // nil for file nodes
}

// renderFileTree builds and renders a hierarchical file tree from repo-relative paths.
// When collapse is true, directories with more than 20 child files are summarised as
// "dir/ (N files)" rather than being fully expanded.
func renderFileTree(files []string, collapse bool) string {

	// Build tree structure.
	root := &treeNode{children: make(map[string]*treeNode)}
	for _, f := range files {
		parts := strings.Split(f, "/")
		node := root
		for i, part := range parts {
			if _, ok := node.children[part]; !ok {
				n := &treeNode{name: part}
				if i < len(parts)-1 {
					n.children = make(map[string]*treeNode)
				}
				node.children[part] = n
			}
			node = node.children[part]
		}
	}

	var b strings.Builder
	renderNode(&b, root, "", collapse)

	return b.String()
}

// renderNode recursively renders a treeNode with two-space indentation.
func renderNode(b *strings.Builder, node *treeNode, indent string, collapse bool) {
	if node.children == nil {
		return
	}

	// Separate dirs and files, sort each group.
	var dirs, fileNames []string
	for name, child := range node.children {
		if child.children != nil {
			dirs = append(dirs, name)
		} else {
			fileNames = append(fileNames, name)
		}
	}
	sort.Strings(dirs)
	sort.Strings(fileNames)

	// Render dirs first, then files.
	for _, name := range dirs {
		child := node.children[name]
		childFileCount := countFiles(child)
		if collapse && childFileCount > 20 {
			fmt.Fprintf(b, "%s%s/ (%d files)\n", indent, name, childFileCount)
		} else {
			fmt.Fprintf(b, "%s%s/\n", indent, name)
			renderNode(b, child, indent+"  ", collapse)
		}
	}
	for _, name := range fileNames {
		fmt.Fprintf(b, "%s%s\n", indent, name)
	}
}

// countFiles returns the total number of file (leaf) nodes under a treeNode.
func countFiles(node *treeNode) int {
	if node.children == nil {
		return 1
	}
	count := 0
	for _, child := range node.children {
		count += countFiles(child)
	}
	return count
}

// FormatDocContext renders a DocAnalysisResult as a compact markdown summary
// showing each documentation file's last-updated date and staleness.
func FormatDocContext(r *DocAnalysisResult) string {
	var b strings.Builder

	name := filepath.Base(r.Root)
	if name == "." || name == "" {
		name = r.Root
	}
	fmt.Fprintf(&b, "## Doc Freshness: %s\n", name)
	fmt.Fprintf(&b, "**Analyzed:** %s\n\n", r.AnalyzedAt.Format("2006-01-02"))

	if len(r.DocRoots) > 0 {
		fmt.Fprintf(&b, "**Doc roots:** %s\n\n", strings.Join(r.DocRoots, ", "))
	}

	if !r.GitChurnAvailable {
		fmt.Fprintf(&b, "_Git history unavailable — last-updated data not shown._\n")
		return strings.TrimRight(b.String(), "\n")
	}

	if len(r.DocFiles) == 0 {
		fmt.Fprintf(&b, "_No documentation files found._\n")
		return strings.TrimRight(b.String(), "\n")
	}

	fmt.Fprintf(&b, "| File | Last Updated | Days Since Update |\n")
	fmt.Fprintf(&b, "|------|-------------|-------------------|\n")
	for _, f := range r.DocFiles {
		if f.DaysSinceUpdate < 0 {
			fmt.Fprintf(&b, "| `%s` | unknown | — |\n", f.Path)
		} else {
			fmt.Fprintf(&b, "| `%s` | %s | %d |\n",
				f.Path,
				f.LastUpdated.Format("2006-01-02"),
				f.DaysSinceUpdate,
			)
		}
	}
	fmt.Fprintln(&b)

	return strings.TrimRight(b.String(), "\n")
}
