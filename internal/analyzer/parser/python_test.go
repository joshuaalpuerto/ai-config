package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joshuaalpuerto/ai-config/internal/analyzer/parser"
)

func TestPythonParser_Imports(t *testing.T) {
	root := t.TempDir()
	src := `from .utils import helper
from .models import User
from django.db import models
import os
import sys
`
	filePath := filepath.Join(root, "app", "views.py")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := parser.Config{RepoRoot: root}
	lp := parser.For(filePath, cfg)
	if lp == nil {
		t.Fatal("expected parser for .py file")
	}
	res, err := lp.Parse(filePath)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// .utils and .models are relative → resolved within the repo
	// django.db is external (no django dir at root) → "django" at root does not exist but
	// the resolver tries filepath.Join(root, "django") which is still inside root, so it returns
	// "django" (not escaped). That is acceptable — it won't match any node.
	// os and sys are stdlib → "os" and "sys" at root don't exist but won't escape.
	if len(res.Imports) == 0 {
		t.Error("expected at least some imports")
	}
	if res.Lines == 0 {
		t.Error("expected Lines > 0")
	}
}

func TestPythonParser_RelativeDots(t *testing.T) {
	root := t.TempDir()
	// Two-level relative import: from ..shared import util
	src := `from ..shared import util
`
	filePath := filepath.Join(root, "app", "api", "handler.py")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := parser.Config{RepoRoot: root}
	lp := parser.For(filePath, cfg)
	res, err := lp.Parse(filePath)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// ..shared from app/api → resolves to app/shared
	if len(res.Imports) != 1 {
		t.Errorf("expected 1 import, got %d: %v", len(res.Imports), res.Imports)
	}
	if res.Imports[0] != "app/shared" {
		t.Errorf("expected 'app/shared', got %q", res.Imports[0])
	}
}
