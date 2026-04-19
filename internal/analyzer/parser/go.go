package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// GoParser parses Go source files using go/ast.
type GoParser struct {
	modulePath string // e.g. "github.com/foo/bar"
	moduleDir  string // repo-relative dir containing go.mod, e.g. "backend" (empty = repo root)
	repoRoot   string
}

func (p *GoParser) Parse(path string) (Result, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return Result{}, wrapErr(path, err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		// Partial parse is fine for import extraction; only fail on read errors.
		return Result{Lines: countLines(string(src))}, nil
	}

	var imports []string
	for _, imp := range f.Imports {
		raw := strings.Trim(imp.Path.Value, `"`)
		if p.modulePath != "" && strings.HasPrefix(raw, p.modulePath) {
			// Strip module prefix and convert to repo-relative path.
			// Prepend moduleDir so the result is relative to the repo root, not
			// the directory that contains go.mod.
			rel := strings.TrimPrefix(raw, p.modulePath)
			rel = strings.TrimPrefix(rel, "/")
			if p.moduleDir != "" && p.moduleDir != "." {
				rel = p.moduleDir + "/" + rel
			}
			rel = filepath.ToSlash(filepath.Clean(rel))
			imports = append(imports, rel)
		}
		// External imports are silently dropped.
	}

	exportCount := countGoExports(f)

	return Result{
		Imports:     dedupeStrings(imports),
		ExportCount: exportCount,
		Lines:       countLines(string(src)),
	}, nil
}

// countGoExports counts top-level exported declarations (uppercase first char).
func countGoExports(f *ast.File) int {
	count := 0
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name != nil && isExported(d.Name.Name) {
				count++
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if isExported(s.Name.Name) {
						count++
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if isExported(name.Name) {
							count++
						}
					}
				}
			}
		}
	}
	return count
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	r := []rune(name)
	return unicode.IsUpper(r[0])
}
