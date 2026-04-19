package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joshuaalpuerto/ai-config/internal/analyzer/parser"
)

func TestGoParser_IntraModuleImports(t *testing.T) {
	root := t.TempDir()
	src := `package main

import (
	"fmt"
	"github.com/example/myapp/internal/utils"
	"github.com/example/myapp/internal/config"
	"github.com/some/external/lib"
)

func main() {}
func ExportedFunc() {}
func unexportedFunc() {}
`
	filePath := filepath.Join(root, "main.go")
	if err := os.WriteFile(filePath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &parser.GoParser{}
	// Access via the For dispatcher.
	cfg := parser.Config{ModulePath: "github.com/example/myapp", RepoRoot: root}
	lp := parser.For(filePath, cfg)
	if lp == nil {
		t.Fatal("expected parser for .go file")
	}
	res, err := lp.Parse(filePath)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(res.Imports) != 2 {
		t.Errorf("expected 2 intra-module imports, got %d: %v", len(res.Imports), res.Imports)
	}
	if res.ExportCount != 1 {
		// ExportedFunc is the only exported declaration; main is lowercase.
		t.Errorf("expected 1 exported declaration, got %d", res.ExportCount)
	}
	if res.Lines == 0 {
		t.Error("expected Lines > 0")
	}
	_ = p
}

func TestGoParser_ExportCount(t *testing.T) {
	root := t.TempDir()
	src := `package mypackage

type MyType struct{}
type unexported struct{}

func MyFunc() {}
func helper() {}

var MyVar = 1
var myVar = 2

const MyConst = "hello"
`
	filePath := filepath.Join(root, "types.go")
	if err := os.WriteFile(filePath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := parser.Config{ModulePath: "github.com/example/myapp", RepoRoot: root}
	lp := parser.For(filePath, cfg)
	res, err := lp.Parse(filePath)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// MyType, MyFunc, MyVar, MyConst = 4 exported
	if res.ExportCount != 4 {
		t.Errorf("expected 4 exported declarations, got %d", res.ExportCount)
	}
}

func TestGoParser_InvalidFile(t *testing.T) {
	root := t.TempDir()
	// Syntactically broken file: should return lines but no error.
	src := `package main
this is not valid go`
	filePath := filepath.Join(root, "broken.go")
	if err := os.WriteFile(filePath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := parser.Config{RepoRoot: root}
	lp := parser.For(filePath, cfg)
	res, err := lp.Parse(filePath)
	if err != nil {
		t.Fatalf("expected no error on broken Go, got: %v", err)
	}
	if res.Lines == 0 {
		t.Error("expected Lines > 0 even for broken file")
	}
}
