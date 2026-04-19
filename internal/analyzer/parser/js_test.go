package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joshuaalpuerto/ai-config/internal/analyzer/parser"
)

func TestJSParser_RelativeImports(t *testing.T) {
	root := t.TempDir()
	src := `import { foo } from './utils/helpers';
import { bar } from '../shared/config';
import React from 'react';
import path from 'path';

export function myFunc() {}
export const myConst = 1;
`
	filePath := filepath.Join(root, "src", "components", "Button.tsx")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := parser.Config{RepoRoot: root}
	lp := parser.For(filePath, cfg)
	if lp == nil {
		t.Fatal("expected parser for .tsx file")
	}
	res, err := lp.Parse(filePath)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// './utils/helpers' and '../shared/config' are relative → resolved
	// 'react' and 'path' are external → dropped
	if len(res.Imports) != 2 {
		t.Errorf("expected 2 resolved imports, got %d: %v", len(res.Imports), res.Imports)
	}
	if res.ExportCount != 2 {
		t.Errorf("expected 2 exports, got %d", res.ExportCount)
	}
	if res.Lines == 0 {
		t.Error("expected Lines > 0")
	}
}

func TestJSParser_TSAliases(t *testing.T) {
	root := t.TempDir()
	src := `import { db } from '@/lib/database';
import { logger } from '@/utils/logger';
`
	filePath := filepath.Join(root, "src", "api", "handler.ts")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := parser.Config{
		RepoRoot:  root,
		TSAliases: map[string]string{"@": "src"},
	}
	lp := parser.For(filePath, cfg)
	res, err := lp.Parse(filePath)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(res.Imports) != 2 {
		t.Errorf("expected 2 resolved alias imports, got %d: %v", len(res.Imports), res.Imports)
	}
}

func TestJSParser_BrokenFile(t *testing.T) {
	root := t.TempDir()
	src := `this is {{{ not valid javascript`
	filePath := filepath.Join(root, "broken.js")
	if err := os.WriteFile(filePath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := parser.Config{RepoRoot: root}
	lp := parser.For(filePath, cfg)
	res, err := lp.Parse(filePath)
	if err != nil {
		t.Fatalf("expected no error on broken JS, got: %v", err)
	}
	if res.Lines == 0 {
		t.Error("expected Lines > 0 even for broken file")
	}
}
