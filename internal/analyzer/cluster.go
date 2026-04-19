package analyzer

import (
	"sort"
	"strings"
)

// buildClusters groups files into connected components using undirected BFS
// over the import graph, then labels each cluster by its dominant top-level directory.
// sourceRootName is the bare name of the source root (e.g. "src"), or "" if root itself.
func buildClusters(nodes map[string]*graphNode, sourceRootName string) []Cluster {
	visited := make(map[string]bool, len(nodes))
	var clusters []Cluster

	for startPath := range nodes {
		if visited[startPath] {
			continue
		}

		// BFS from startPath treating edges as undirected.
		queue := []string{startPath}
		var members []string

		for len(queue) > 0 {
			path := queue[0]
			queue = queue[1:]

			if visited[path] {
				continue
			}
			visited[path] = true
			members = append(members, path)

			node, ok := nodes[path]
			if !ok {
				continue
			}
			for _, imp := range node.imports {
				if !visited[imp] {
					queue = append(queue, imp)
				}
			}
			for _, dep := range node.importedBy {
				if !visited[dep] {
					queue = append(queue, dep)
				}
			}
		}

		label := labelCluster(members, sourceRootName)
		sort.Strings(members)
		clusters = append(clusters, Cluster{
			Label:     label,
			Size:      len(members),
			Singleton: len(members) == 1,
			Files:     members,
		})
	}

	// Sort clusters by size descending for stable output.
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

// labelCluster determines the label for a cluster by finding the most common
// top-level directory among its members. When the dominant directory is a known
// source root (e.g. "src"), it falls back to the second-level directory.
// Ties are broken lexicographically.
func labelCluster(members []string, sourceRootName string) string {
	counts := make(map[string]int)
	for _, path := range members {
		parts := strings.SplitN(path, "/", 3)
		dir := parts[0]
		// If the top-level dir is a known source root and a second level exists, use it.
		if knownSourceRoots[dir] && len(parts) >= 2 {
			dir = parts[1]
		} else if sourceRootName != "" && dir == sourceRootName && len(parts) >= 2 {
			dir = parts[1]
		}
		counts[dir]++
	}

	best := ""
	bestCount := 0
	for dir, count := range counts {
		if count > bestCount || (count == bestCount && dir < best) {
			best = dir
			bestCount = count
		}
	}
	return best
}
