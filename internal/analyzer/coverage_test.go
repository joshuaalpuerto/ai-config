package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeCoverage_ShowsGapAndCovered(t *testing.T) {
	root := t.TempDir()

	// Create a doc file that mentions "hooks" but not "transpiler".
	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte("# Hooks\nHow to use hooks."), 0o644); err != nil {
		t.Fatal(err)
	}

	allFiles := []string{"docs/guide.md"}
	hubs := []Hub{
		{Path: "internal/hooks/engine.go"},
		{Path: "internal/transpiler/transpiler.go"},
	}
	clusters := []Cluster{
		{Label: "internal/hooks", Size: 3},
		{Label: "internal/transpiler", Size: 2},
	}

	cov := computeCoverage(root, allFiles, hubs, clusters)
	if len(cov) != 2 {
		t.Fatalf("expected 2 coverage entries, got %d", len(cov))
	}
	// Uncovered entries sort first.
	if cov[0].Area != "transpiler" || cov[0].Covered {
		t.Errorf("expected first entry to be uncovered transpiler, got %+v", cov[0])
	}
	if cov[1].Area != "hooks" || !cov[1].Covered {
		t.Errorf("expected second entry to be covered hooks, got %+v", cov[1])
	}
	if len(cov[1].DocFiles) != 1 || cov[1].DocFiles[0] != "docs/guide.md" {
		t.Errorf("expected hooks covered by docs/guide.md, got %v", cov[1].DocFiles)
	}
}

func TestComputeCoverage_AllCovered(t *testing.T) {
	root := t.TempDir()

	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte("# Hooks and Transpiler\nCovers hooks and transpiler."), 0o644); err != nil {
		t.Fatal(err)
	}

	allFiles := []string{"docs/guide.md"}
	hubs := []Hub{
		{Path: "internal/hooks/engine.go"},
		{Path: "internal/transpiler/transpiler.go"},
	}
	clusters := []Cluster{
		{Label: "internal/hooks", Size: 3},
		{Label: "internal/transpiler", Size: 2},
	}

	cov := computeCoverage(root, allFiles, hubs, clusters)
	for _, entry := range cov {
		if !entry.Covered {
			t.Errorf("expected all covered, but %q is not", entry.Area)
		}
	}
}

func TestComputeCoverage_NoDocs(t *testing.T) {
	root := t.TempDir()

	hubs := []Hub{{Path: "internal/hooks/engine.go"}}
	clusters := []Cluster{{Label: "internal/hooks", Size: 3}}

	cov := computeCoverage(root, nil, hubs, clusters)
	if cov != nil {
		t.Errorf("expected nil with no docs, got %v", cov)
	}
}

func TestComputeCoverage_SkipsSingletons(t *testing.T) {
	root := t.TempDir()

	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte("Nothing mentioned."), 0o644); err != nil {
		t.Fatal(err)
	}

	allFiles := []string{"docs/guide.md"}
	clusters := []Cluster{
		{Label: "singleton", Size: 1, Singleton: true},
	}

	cov := computeCoverage(root, allFiles, nil, clusters)
	if len(cov) != 0 {
		t.Errorf("expected no coverage entries for singletons, got %v", cov)
	}
}

func TestComputeCoverage_MultipleDocsMatchSameArea(t *testing.T) {
	root := t.TempDir()

	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte("See hooks for details."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "reference.md"), []byte("# Hooks API reference"), 0o644); err != nil {
		t.Fatal(err)
	}

	allFiles := []string{"docs/guide.md", "docs/reference.md"}
	clusters := []Cluster{{Label: "internal/hooks", Size: 3}}

	cov := computeCoverage(root, allFiles, nil, clusters)
	if len(cov) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(cov))
	}
	if len(cov[0].DocFiles) != 2 {
		t.Errorf("expected hooks mentioned in 2 docs, got %v", cov[0].DocFiles)
	}
}
