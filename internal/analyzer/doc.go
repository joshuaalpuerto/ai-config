package analyzer

import (
	"os"
	"path/filepath"
	"sort"
	"time"
)

// AnalyzeDocFreshness runs a git-based freshness analysis on documentation files
// found under the given docRoots (relative or absolute paths). It returns a
// DocAnalysisResult with each .md file's last-commit date and days-since-update,
// sorted stalest-first. When git is unavailable, DaysSinceUpdate is -1 for all files.
func AnalyzeDocFreshness(root string, docRoots []string) (*DocAnalysisResult, error) {
	files, err := collectDocFiles(root, docRoots)
	if err != nil {
		return nil, err
	}

	// Collect repo-relative paths for git lookup.
	relPaths := make([]string, 0, len(files))
	for _, abs := range files {
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			rel = abs
		}
		relPaths = append(relPaths, filepath.ToSlash(rel))
	}

	updated, err := analyzeLastUpdated(root, relPaths)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	docFiles := make([]DocFile, 0, len(relPaths))
	for _, rel := range relPaths {
		df := DocFile{Path: rel, DaysSinceUpdate: -1}
		if updated.available {
			if t, ok := updated.dates[rel]; ok {
				df.LastUpdated = t
				df.DaysSinceUpdate = int(now.Sub(t).Hours() / 24)
			}
		}
		docFiles = append(docFiles, df)
	}

	// Sort stalest-first; unknown (-1) sorts to the top.
	sort.Slice(docFiles, func(i, j int) bool {
		return docFiles[i].DaysSinceUpdate > docFiles[j].DaysSinceUpdate
	})

	// Normalise docRoots to slash-separated repo-relative strings for the report.
	roots := make([]string, 0, len(docRoots))
	for _, dr := range docRoots {
		if filepath.IsAbs(dr) {
			if rel, err := filepath.Rel(root, dr); err == nil {
				dr = filepath.ToSlash(rel)
			}
		}
		roots = append(roots, dr)
	}

	return &DocAnalysisResult{
		Root:              root,
		AnalyzedAt:        now,
		GitChurnAvailable: updated.available,
		DocRoots:          roots,
		DocFiles:          docFiles,
	}, nil
}

// collectDocFiles walks each docRoot and returns the absolute paths of all .md files
// found. docRoots may be files or directories; relative paths are resolved against root.
func collectDocFiles(root string, docRoots []string) ([]string, error) {
	var results []string

	for _, dr := range docRoots {
		if !filepath.IsAbs(dr) {
			dr = filepath.Join(root, dr)
		}

		info, err := os.Stat(dr)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			if isDocFile(dr) {
				results = append(results, dr)
			}
			continue
		}

		if err := filepath.WalkDir(dr, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && isDocFile(path) {
				results = append(results, path)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	return results, nil
}

func isDocFile(path string) bool {
	return filepath.Ext(path) == ".md"
}
