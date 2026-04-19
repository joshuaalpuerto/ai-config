package parser

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/js"
)

var (
	// jsExportDecl matches: export function Foo, export class Bar, export const Baz, etc.
	jsExportDecl = regexp.MustCompile(`(?m)^export\s+(?:async\s+)?(?:function\*?|class|const|let|var|enum|type|interface|abstract\s+class)\s+(\w+)`)
	// jsExportBraces matches: export { foo, bar as baz }
	jsExportBraces = regexp.MustCompile(`(?m)^export\s*\{([^}]+)\}`)
	// jsExportDefaultNamed matches: export default function Foo / export default class Foo
	jsExportDefaultNamed = regexp.MustCompile(`(?m)^export\s+default\s+(?:async\s+)?(?:function\*?|class)\s+(\w+)`)
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

	exportNames := extractJSExportNames(string(src))

	return Result{
		Imports:     dedupeStrings(resolved),
		ExportCount: exportCount,
		ExportNames: exportNames,
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

// extractJSExportNames extracts exported symbol names from JS/TS source text.
func extractJSExportNames(src string) []string {
	seen := make(map[string]bool)
	var names []string

	add := func(name string) {
		name = strings.TrimSpace(name)
		if name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}

	// export function Foo / export class Bar / export const Baz etc.
	for _, m := range jsExportDecl.FindAllStringSubmatch(src, -1) {
		add(m[1])
	}
	// export default function Foo / export default class Bar
	for _, m := range jsExportDefaultNamed.FindAllStringSubmatch(src, -1) {
		add(m[1])
	}
	// export { foo, bar as baz } — capture original names before "as"
	for _, m := range jsExportBraces.FindAllStringSubmatch(src, -1) {
		for _, item := range strings.Split(m[1], ",") {
			parts := strings.Fields(item)
			if len(parts) > 0 {
				add(parts[0])
			}
		}
	}

	return names
}
