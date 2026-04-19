package parser

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	pyFromImport  = regexp.MustCompile(`(?m)^from\s+(\.+[\w.]*|[\w.]+)\s+import\s+`)
	pyPlainImport = regexp.MustCompile(`(?m)^import\s+([\w.]+)`)
)

// PythonParser parses Python source files using regex.
type PythonParser struct {
	repoRoot string
	filePath string
}

func (p *PythonParser) Parse(path string) (Result, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return Result{}, wrapErr(path, err)
	}
	content := string(src)

	var rawImports []string

	for _, m := range pyFromImport.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			rawImports = append(rawImports, m[1])
		}
	}
	for _, m := range pyPlainImport.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			rawImports = append(rawImports, m[1])
		}
	}

	resolved := make([]string, 0, len(rawImports))
	for _, raw := range rawImports {
		r := p.resolve(raw, path)
		if r != "" {
			resolved = append(resolved, r)
		}
	}

	return Result{
		Imports:     dedupeStrings(resolved),
		ExportCount: 0, // Python has no explicit export mechanism
		Lines:       countLines(content),
	}, nil
}

// resolve converts a raw Python import string to a repo-relative path.
// Relative imports (leading dots) are resolved; absolute imports are dropped
// unless they map to a directory inside the repo.
func (p *PythonParser) resolve(raw, importingFile string) string {
	// Relative import: leading dots (e.g. ".utils" means "./utils").
	if strings.HasPrefix(raw, ".") {
		dots := 0
		for _, c := range raw {
			if c == '.' {
				dots++
			} else {
				break
			}
		}
		module := raw[dots:]
		dir := filepath.Dir(importingFile)
		// Each extra dot means one directory up.
		for i := 1; i < dots; i++ {
			dir = filepath.Dir(dir)
		}
		modPath := strings.ReplaceAll(module, ".", string(filepath.Separator))
		candidate := filepath.Join(dir, modPath)
		rel, err := filepath.Rel(p.repoRoot, candidate)
		if err != nil {
			return ""
		}
		return filepath.ToSlash(rel)
	}

	// Absolute import: check if the top-level module name matches a directory in the repo.
	parts := strings.SplitN(raw, ".", 2)
	topLevel := parts[0]
	candidate := filepath.Join(p.repoRoot, topLevel)
	rel, err := filepath.Rel(p.repoRoot, candidate)
	if err != nil {
		return ""
	}
	// Only include if the path stays within the repo (no ../ escaping).
	if strings.HasPrefix(rel, "..") {
		return ""
	}
	return filepath.ToSlash(rel)
}
