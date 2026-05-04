package analyzer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

// DefaultDocExclude is the default set of directory names excluded from doc
// collection. These are dependency/build/cache folders that should never be
// treated as project documentation.
var DefaultDocExclude = []string{
	"node_modules",
	"vendor",
	".terraform",
	"__pycache__",
	".venv",
	"venv",
	".tox",
	".mypy_cache",
	".pytest_cache",
	"dist",
	"build",
	"target",
	".next",
	".nuxt",
	".output",
	".gradle",
	".m2",
	".cargo",
	"pkg/mod",
	"Pods",
	".bundle",
}

// AnalyzeDocFreshness runs a git-based freshness analysis on documentation files
// found under the given docRoots (relative or absolute paths). It returns a
// DocAnalysisResult with each .md file's last-commit date and days-since-update,
// sorted stalest-first. When git is unavailable, DaysSinceUpdate is -1 for all files.
// The exclude parameter specifies directory names to skip; if nil, DefaultDocExclude is used.
func AnalyzeDocFreshness(root string, docRoots []string, exclude []string) (*DocAnalysisResult, error) {
	if exclude == nil {
		exclude = DefaultDocExclude
	}
	files, err := collectDocFiles(root, docRoots, exclude)
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
// found. docRoots may be files, directories, or glob patterns (supporting ** via
// doublestar); relative paths are resolved against root. Paths containing any
// directory segment in exclude are skipped.
func collectDocFiles(root string, docRoots []string, exclude []string) ([]string, error) {
	excludeSet := make(map[string]struct{}, len(exclude))
	for _, e := range exclude {
		excludeSet[e] = struct{}{}
	}

	seen := make(map[string]struct{})
	var results []string

	add := func(abs string) {
		if _, ok := seen[abs]; !ok {
			rel, _ := filepath.Rel(root, abs)
			if isExcludedPath(filepath.ToSlash(rel), excludeSet) {
				return
			}
			seen[abs] = struct{}{}
			results = append(results, abs)
		}
	}

	for _, dr := range docRoots {
		if isGlobPattern(dr) {
			matches, err := expandGlob(root, dr)
			if err != nil {
				return nil, err
			}
			for _, m := range matches {
				if isDocFile(m) {
					add(m)
				}
			}
			continue
		}

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
				add(dr)
			}
			continue
		}

		if err := filepath.WalkDir(dr, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && isDocFile(path) {
				add(path)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	return results, nil
}

// isGlobPattern reports whether s contains glob metacharacters.
func isGlobPattern(s string) bool {
	return strings.ContainsAny(s, "*?[{")
}

// expandGlob expands a glob pattern relative to root and returns absolute paths
// of matched files. Uses doublestar for ** support.
func expandGlob(root, pattern string) ([]string, error) {
	fsys := os.DirFS(root)
	matches, err := doublestar.Glob(fsys, pattern)
	if err != nil {
		return nil, err
	}
	abs := make([]string, 0, len(matches))
	for _, m := range matches {
		abs = append(abs, filepath.Join(root, filepath.FromSlash(m)))
	}
	return abs, nil
}

func isDocFile(path string) bool {
	return filepath.Ext(path) == ".md"
}

// isExcludedPath reports whether any path segment is in the exclude set.
func isExcludedPath(slashRel string, excludeSet map[string]struct{}) bool {
	for _, seg := range strings.Split(slashRel, "/") {
		if _, ok := excludeSet[seg]; ok {
			return true
		}
	}
	return false
}
