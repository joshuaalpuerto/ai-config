package analyzer_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/joshuaalpuerto/ai-config/internal/analyzer"
)

// makeGoRepo creates a minimal Go repo in a temp directory.
func makeGoRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// go.mod
	goMod := "module github.com/example/testrepo\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	// cmd/main.go (entry point)
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0o755); err != nil {
		t.Fatal(err)
	}
	mainGo := `package main

import "github.com/example/testrepo/internal/utils"

func main() { utils.Hello() }
`
	if err := os.WriteFile(filepath.Join(root, "cmd", "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatal(err)
	}

	// internal/utils/utils.go (hub)
	if err := os.MkdirAll(filepath.Join(root, "internal", "utils"), 0o755); err != nil {
		t.Fatal(err)
	}
	utilsGo := `package utils

func Hello() {}
func helper() {}
`
	if err := os.WriteFile(filepath.Join(root, "internal", "utils", "utils.go"), []byte(utilsGo), 0o644); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestAnalyze_DetectsGoLanguage(t *testing.T) {
	root := makeGoRepo(t)
	a := analyzer.New()
	result, err := a.Analyze(root)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	hasGo := false
	for _, lang := range result.TechStack.Languages {
		if lang == "go" {
			hasGo = true
		}
	}
	if !hasGo {
		t.Errorf("expected 'go' in languages, got %v", result.TechStack.Languages)
	}
}

func TestAnalyze_GraphEdges(t *testing.T) {
	root := makeGoRepo(t)
	a := analyzer.New()
	a.Verbose = true // Files map is only populated in verbose mode
	result, err := a.Analyze(root)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	// cmd/main.go should import internal/utils/utils.go
	mainNode, ok := result.Files["cmd/main.go"]
	if !ok {
		t.Fatal("expected cmd/main.go in files map")
	}
	if len(mainNode.Imports) == 0 {
		t.Error("expected cmd/main.go to have imports")
	}

	// internal/utils/utils.go should be imported by cmd/main.go
	utilsNode, ok := result.Files["internal/utils/utils.go"]
	if !ok {
		t.Fatal("expected internal/utils/utils.go in files map")
	}
	if len(utilsNode.ImportedBy) == 0 {
		t.Error("expected internal/utils/utils.go to have importedBy entries")
	}
}

func TestAnalyze_NoGitChurn(t *testing.T) {
	root := makeGoRepo(t)
	// No .git directory → gitChurnAvailable must be false.
	a := analyzer.New()
	result, err := a.Analyze(root)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if result.GitChurnAvailable {
		t.Error("expected gitChurnAvailable=false when no .git directory")
	}
}

func TestAnalyze_JSONRoundtrip(t *testing.T) {
	root := makeGoRepo(t)
	a := analyzer.New()
	result, err := a.Analyze(root)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var decoded analyzer.AnalysisResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if decoded.Root != result.Root {
		t.Errorf("root mismatch after JSON roundtrip")
	}
}

func TestAnalyze_ClustersConnected(t *testing.T) {
	root := makeGoRepo(t)
	a := analyzer.New()
	result, err := a.Analyze(root)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(result.Clusters) == 0 {
		t.Error("expected at least one cluster")
	}
	// cmd/main.go and internal/utils/utils.go are connected → should be in the same cluster.
	found := false
	for _, cl := range result.Clusters {
		hasMain, hasUtils := false, false
		for _, f := range cl.Files {
			if f == "cmd/main.go" {
				hasMain = true
			}
			if f == "internal/utils/utils.go" {
				hasUtils = true
			}
		}
		if hasMain && hasUtils {
			found = true
		}
	}
	if !found {
		t.Error("expected cmd/main.go and internal/utils/utils.go in the same cluster")
	}
}
