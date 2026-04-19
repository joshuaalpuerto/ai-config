package analyzer

import (
	"sort"
	"strings"
)

// buildClusters groups files into connected components using undirected BFS
// over the import graph, then labels each cluster by its dominant top-level directory.
func buildClusters(nodes map[string]*graphNode) []Cluster {
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

		label := labelCluster(members)
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

	return clusters
}

// labelCluster determines the label for a cluster by finding the most common
// top-level directory among its members. Ties are broken lexicographically.
func labelCluster(members []string) string {
	counts := make(map[string]int)
	for _, path := range members {
		parts := strings.SplitN(path, "/", 2)
		counts[parts[0]]++
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
