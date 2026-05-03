package analyzer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// computeCoverage checks which hub/cluster areas are mentioned in any .md file and
// returns a CoverageEntry per area showing coverage status and the specific doc files
// that reference it. Results are sorted: uncovered first, then alphabetically.
func computeCoverage(root string, allFiles []string, hubs []Hub, clusters []Cluster) []CoverageEntry {
	docIndex := buildDocIndex(root, allFiles)
	if len(docIndex) == 0 {
		return nil // no docs to cross-reference
	}

	// Collect unique area keywords from clusters (non-singleton) and hub directories.
	terms := make(map[string]bool)
	for _, c := range clusters {
		if c.Singleton {
			continue
		}
		keyword := lastSegment(c.Label)
		if keyword != "" {
			terms[keyword] = true
		}
	}
	for _, h := range hubs {
		keyword := lastSegment(filepath.Dir(h.Path))
		if keyword != "" {
			terms[keyword] = true
		}
	}

	var entries []CoverageEntry
	for term := range terms {
		lowerTerm := strings.ToLower(term)
		var matchedDocs []string
		for docPath, content := range docIndex {
			if strings.Contains(content, lowerTerm) {
				matchedDocs = append(matchedDocs, docPath)
			}
		}
		sort.Strings(matchedDocs)
		entries = append(entries, CoverageEntry{
			Area:     term,
			Covered:  len(matchedDocs) > 0,
			DocFiles: matchedDocs,
		})
	}

	// Sort: uncovered first, then alphabetically by area.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Covered != entries[j].Covered {
			return !entries[i].Covered // uncovered first
		}
		return entries[i].Area < entries[j].Area
	})

	return entries
}

// buildDocIndex reads all .md files and returns a map of repo-relative path → lowercase content.
func buildDocIndex(root string, allFiles []string) map[string]string {
	index := make(map[string]string)
	for _, rel := range allFiles {
		if !strings.HasSuffix(strings.ToLower(rel), ".md") {
			continue
		}
		abs := filepath.Join(root, rel)
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		index[rel] = strings.ToLower(string(data))
	}
	return index
}

// lastSegment returns the last path segment, or "" for empty/root paths.
func lastSegment(p string) string {
	p = strings.TrimSuffix(filepath.ToSlash(p), "/")
	if p == "" || p == "." {
		return ""
	}
	parts := strings.Split(p, "/")
	return parts[len(parts)-1]
}
