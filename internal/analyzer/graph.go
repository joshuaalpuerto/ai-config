package analyzer

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/joshuaalpuerto/ai-config/internal/analyzer/parser"
)

// graphNode is the internal mutable equivalent of FileNode used during construction.
type graphNode struct {
	path         string
	imports      []string
	importedBy   []string
	lines        int
	exportCount  int
	churn        int
	isEntryPoint bool
	folderDepth  int
	priority     float64
}

// buildGraph parses all source files and constructs the import graph.
func buildGraph(files []string, cfg parser.Config) (map[string]*graphNode, error) {
	nodes := make(map[string]*graphNode, len(files))

	for _, absPath := range files {
		rel, err := filepath.Rel(cfg.RepoRoot, absPath)
		if err != nil {
			rel = absPath
		}
		rel = filepath.ToSlash(rel)

		p := parser.For(absPath, cfg)
		if p == nil {
			continue
		}
		res, err := p.Parse(absPath)
		if err != nil {
			// Log and continue — one bad file should not abort the whole analysis.
			continue
		}

		nodes[rel] = &graphNode{
			path:        rel,
			imports:     res.Imports,
			lines:       res.Lines,
			exportCount: res.ExportCount,
			folderDepth: strings.Count(rel, "/"),
		}
	}

	// Resolve directory-level imports to actual file paths.
	// Go imports resolve to package directories; expand them to the files inside.
	for _, node := range nodes {
		var resolved []string
		for _, imp := range node.imports {
			if _, ok := nodes[imp]; ok {
				resolved = append(resolved, imp)
			} else {
				// Package-directory import: collect all files under imp/.
				prefix := imp + "/"
				for targetPath := range nodes {
					if strings.HasPrefix(targetPath, prefix) {
						resolved = append(resolved, targetPath)
					}
				}
			}
		}
		node.imports = resolved
	}

	// Reverse-edge pass: populate importedBy.
	for path, node := range nodes {
		for _, imp := range node.imports {
			if target, ok := nodes[imp]; ok {
				target.importedBy = append(target.importedBy, path)
			}
		}
	}

	// Mark entry points.
	for _, node := range nodes {
		node.isEntryPoint = isEntryPoint(node.path)
	}

	// Compute initial priority (churn is 0 at this stage).
	for _, node := range nodes {
		node.churn = 0
	}
	computePriorities(nodes)

	return nodes, nil
}

// computePriorities recomputes priority for all nodes (called after churn is filled).
func computePriorities(nodes map[string]*graphNode) {
	for _, node := range nodes {
		fanIn := float64(len(node.importedBy))
		fanOut := float64(len(node.imports))
		exports := float64(node.exportCount)
		churn := float64(node.churn)
		depth := float64(node.folderDepth)

		var entryBonus float64
		if node.isEntryPoint {
			entryBonus = 10.0
		}

		_ = fanOut // fanOut is stored but not used in priority formula
		node.priority = (fanIn * 3.0) + (exports * 1.5) + entryBonus + (churn * 2.0) + (1.0 / (depth + 1))
	}
}

// topHubs returns the top N nodes by priority.
func topHubs(nodes map[string]*graphNode, n int) []Hub {
	type ranked struct {
		node *graphNode
	}
	all := make([]ranked, 0, len(nodes))
	for _, node := range nodes {
		all = append(all, ranked{node})
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].node.priority > all[j].node.priority
	})
	if len(all) > n {
		all = all[:n]
	}
	hubs := make([]Hub, 0, len(all))
	for _, r := range all {
		hubs = append(hubs, Hub{
			Path:     r.node.path,
			FanIn:    len(r.node.importedBy),
			FanOut:   len(r.node.imports),
			Priority: r.node.priority,
		})
	}
	return hubs
}

// isEntryPoint returns true for files that are conventional entry points.
func isEntryPoint(repoRelPath string) bool {
	parts := strings.Split(repoRelPath, "/")
	name := parts[len(parts)-1]
	nameNoExt := strings.TrimSuffix(name, filepath.Ext(name))

	// Go: main package files or files inside cmd/.
	if strings.HasSuffix(name, ".go") {
		if parts[0] == "cmd" || nameNoExt == "main" {
			return true
		}
	}

	// Python: __main__.py or main.py.
	if name == "__main__.py" || name == "main.py" {
		return true
	}

	// JS/TS: index.*, app.*, server.* at source root (depth 0 or 1 inside src/).
	if strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".tsx") ||
		strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".jsx") {
		switch nameNoExt {
		case "index", "app", "server":
			if len(parts) <= 2 { // root or one level deep (e.g. src/index.ts)
				return true
			}
		}
	}
	return false
}

// nodesToFileMap converts the internal graph to the exported FileNode map.
func nodesToFileMap(nodes map[string]*graphNode) map[string]FileNode {
	out := make(map[string]FileNode, len(nodes))
	for path, n := range nodes {
		imports := n.imports
		if imports == nil {
			imports = []string{}
		}
		importedBy := n.importedBy
		if importedBy == nil {
			importedBy = []string{}
		}
		out[path] = FileNode{
			Imports:      imports,
			ImportedBy:   importedBy,
			Lines:        n.lines,
			ExportCount:  n.exportCount,
			Churn:        n.churn,
			FanIn:        len(n.importedBy),
			FanOut:       len(n.imports),
			IsEntryPoint: n.isEntryPoint,
			Priority:     n.priority,
			FolderDepth:  n.folderDepth,
		}
	}
	return out
}
