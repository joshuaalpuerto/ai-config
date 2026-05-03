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
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
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

	exportCount, exportNames := countGoExports(f)

	var fileDoc string
	if f.Doc != nil {
		fileDoc = strings.TrimSpace(f.Doc.Text())
		// Take only the first sentence/line for brevity.
		if idx := strings.IndexByte(fileDoc, '\n'); idx > 0 {
			fileDoc = fileDoc[:idx]
		}
	}

	return Result{
		Imports:     dedupeStrings(imports),
		ExportCount: exportCount,
		ExportNames: exportNames,
		Lines:       countLines(string(src)),
		FileDoc:     fileDoc,
	}, nil
}

// countGoExports counts top-level exported declarations (uppercase first char)
// and returns both the count and the list of exported symbol names.
func countGoExports(f *ast.File) (int, []string) {
	var names []string
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name != nil && isExported(d.Name.Name) {
				names = append(names, d.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if isExported(s.Name.Name) {
						names = append(names, s.Name.Name)
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if isExported(name.Name) {
							names = append(names, name.Name)
						}
					}
				}
			}
		}
	}
	return len(names), names
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	r := []rune(name)
	return unicode.IsUpper(r[0])
}
