package parser

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/js"
)

// JSParser parses JS/TS source files using tdewolff/parse.
type JSParser struct {
	aliases  map[string]string // TS path alias prefix → repo-relative dir
	repoRoot string
	filePath string
}

func (p *JSParser) Parse(path string) (Result, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return Result{}, wrapErr(path, err)
	}

	input := parse.NewInputBytes(src)
	ast, parseErr := js.Parse(input, js.Options{})
	if parseErr != nil {
		// Best-effort: return line count without imports rather than failing.
		return Result{Lines: countLines(string(src))}, nil
	}

	var rawImports []string
	exportCount := 0

	for _, stmt := range ast.BlockStmt.List {
		switch s := stmt.(type) {
		case *js.ImportStmt:
			if len(s.Module) > 0 {
				rawImports = append(rawImports, strings.Trim(string(s.Module), `"'`))
			}
		case *js.ExportStmt:
			exportCount++
			if len(s.Module) > 0 {
				rawImports = append(rawImports, strings.Trim(string(s.Module), `"'`))
			}
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
		ExportCount: exportCount,
		Lines:       countLines(string(src)),
	}, nil
}

// resolve converts a raw JS/TS import string to a repo-relative path.
// Returns empty string if the import is external or unresolvable.
func (p *JSParser) resolve(raw, importingFile string) string {
	// Relative import.
	if strings.HasPrefix(raw, "./") || strings.HasPrefix(raw, "../") {
		return resolveRelative(raw, importingFile, p.repoRoot)
	}

	// TS alias (e.g. "@/utils/foo").
	for prefix, dir := range p.aliases {
		if strings.HasPrefix(raw, prefix) {
			rest := strings.TrimPrefix(raw, prefix)
			candidate := filepath.Join(p.repoRoot, dir, rest)
			rel, err := filepath.Rel(p.repoRoot, candidate)
			if err != nil {
				continue
			}
			return filepath.ToSlash(rel)
		}
	}

	// Not a local import — external package, skip.
	return ""
}
