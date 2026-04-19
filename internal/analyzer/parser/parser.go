package parser

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Result holds the parsed data from a single source file.
type Result struct {
	Imports     []string // raw import paths as written in the file
	ExportCount int
	Lines       int
}

// LanguageParser parses a single source file.
type LanguageParser interface {
	Parse(path string) (Result, error)
}

// Config holds shared context needed by parsers for path resolution.
type Config struct {
	// ModulePath is the Go module path from go.mod (e.g. "github.com/foo/bar").
	ModulePath string
	// TSAliases maps an alias prefix (e.g. "@/") to a repo-relative directory (e.g. "src/").
	TSAliases map[string]string
	// RepoRoot is the absolute path to the repository root.
	RepoRoot string
}

// For returns the appropriate LanguageParser for the given file path, or nil if unsupported.
func For(path string, cfg Config) LanguageParser {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return &GoParser{modulePath: cfg.ModulePath, repoRoot: cfg.RepoRoot}
	case ".ts", ".tsx", ".js", ".jsx":
		return &JSParser{aliases: cfg.TSAliases, repoRoot: cfg.RepoRoot, filePath: path}
	case ".py":
		return &PythonParser{repoRoot: cfg.RepoRoot, filePath: path}
	default:
		return nil
	}
}

// resolveRelative resolves a relative import path (starting with ./ or ../) against
// the directory of the importing file. It tries several extensions if the path has none.
// Returns a repo-relative path on success, or empty string if unresolvable.
func resolveRelative(importPath, importingFile, repoRoot string) string {
	dir := filepath.Dir(importingFile)
	joined := filepath.Join(dir, importPath)

	// If already has an extension that is a source file, just normalise.
	ext := filepath.Ext(joined)
	if ext != "" && isSrcExt(ext) {
		rel, err := filepath.Rel(repoRoot, joined)
		if err != nil {
			return ""
		}
		return filepath.ToSlash(rel)
	}

	// Try extensions and index files.
	candidates := []string{
		joined + ".go",
		joined + ".ts",
		joined + ".tsx",
		joined + ".js",
		joined + ".jsx",
		joined + ".py",
		joined + "/index.ts",
		joined + "/index.tsx",
		joined + "/index.js",
	}
	for _, c := range candidates {
		rel, err := filepath.Rel(repoRoot, c)
		if err != nil {
			continue
		}
		return filepath.ToSlash(rel)
	}

	// Fall back to normalised path with no extension (won't match a node but still useful).
	rel, err := filepath.Rel(repoRoot, joined)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

func isSrcExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py":
		return true
	}
	return false
}

// countLines counts the number of newline-terminated lines in s.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

// dedupeStrings returns a deduplicated copy of ss preserving order.
func dedupeStrings(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

// wrapErr wraps an error with the file path for context.
func wrapErr(path string, err error) error {
	return fmt.Errorf("parsing %s: %w", path, err)
}
