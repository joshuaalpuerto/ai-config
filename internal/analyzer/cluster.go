package analyzer

import (
	"path/filepath"
	"sort"
	"strings"
)

// buildClusters groups files by their package directory (depth-2 path segments) to
// produce deterministic, meaningful clusters. It then uses the import graph to compute
// inter-cluster dependencies. sourceRootName is the bare name of the source root
// (e.g. "src"), or "" if root itself.
func buildClusters(nodes map[string]*graphNode, sourceRootName string) []Cluster {
	// Group files by cluster key (directory path up to 2 segments deep).
	groups := make(map[string][]string)
	for path := range nodes {
		key := clusterKey(path, sourceRootName)
		groups[key] = append(groups[key], path)
	}

	var clusters []Cluster
	for label, members := range groups {
		sort.Strings(members)
		clusters = append(clusters, Cluster{
			Label:     label,
			Size:      len(members),
			Singleton: len(members) == 1,
			Files:     members,
		})
	}

	// Sort clusters by size descending, then label for stable output.
	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].Size != clusters[j].Size {
			return clusters[i].Size > clusters[j].Size
		}
		return clusters[i].Label < clusters[j].Label
	})

	// Build file → cluster-index map to compute inter-cluster dependencies.
	fileToCluster := make(map[string]int, len(nodes))
	for i, c := range clusters {
		for _, f := range c.Files {
			fileToCluster[f] = i
		}
	}

	for i := range clusters {
		depSet := make(map[string]bool)
		for _, filePath := range clusters[i].Files {
			node, ok := nodes[filePath]
			if !ok {
				continue
			}
			for _, imp := range node.imports {
				j, ok := fileToCluster[imp]
				if ok && j != i {
					depSet[clusters[j].Label] = true
				}
			}
		}
		if len(depSet) > 0 {
			deps := make([]string, 0, len(depSet))
			for dep := range depSet {
				deps = append(deps, dep)
			}
			sort.Strings(deps)
			clusters[i].DependsOn = deps
		}
	}

	return clusters
}

// clusterKey returns a deterministic grouping key for a file path. It uses up to
// two path segments (e.g. "internal/hooks") to produce meaningful package-level
// clusters. When the first segment is a known source root (e.g. "src"), it is
// stripped so that "src/components/Button.tsx" clusters as "components".
func clusterKey(filePath, sourceRootName string) string {
	dir := filepath.ToSlash(filepath.Dir(filePath))
	if dir == "." {
		// Root-level file — use filename without extension as a last resort.
		return strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}

	parts := strings.Split(dir, "/")

	// Strip known source root prefix so "src/hooks/..." → "hooks/...".
	if len(parts) > 1 && (knownSourceRoots[parts[0]] || (sourceRootName != "" && parts[0] == sourceRootName)) {
		parts = parts[1:]
	}

	// Use up to 2 segments: "internal/hooks", "cmd/aicfg", or "schemas".
	if len(parts) > 2 {
		parts = parts[:2]
	}
	return strings.Join(parts, "/")
}
