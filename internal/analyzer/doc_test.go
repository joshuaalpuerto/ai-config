package analyzer

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestCollectDocFiles_GlobRecursiveWildcard(t *testing.T) {
	root := t.TempDir()
	// Create nested structure with .md files at various depths.
	mkFile(t, root, "docs/guide.md")
	mkFile(t, root, "docs/nested/deep.md")
	mkFile(t, root, "src/AGENTS.md")
	mkFile(t, root, "AGENTS.md")
	mkFile(t, root, "README.md")
	mkFile(t, root, "src/main.go") // not .md

	got, err := collectDocFiles(root, []string{"**/*.md"})
	if err != nil {
		t.Fatal(err)
	}

	// Should find all .md files recursively.
	want := []string{
		filepath.Join(root, "AGENTS.md"),
		filepath.Join(root, "README.md"),
		filepath.Join(root, "docs/guide.md"),
		filepath.Join(root, "docs/nested/deep.md"),
		filepath.Join(root, "src/AGENTS.md"),
	}
	assertPaths(t, got, want)
}

func TestCollectDocFiles_GlobSpecificFilename(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, "AGENTS.md")
	mkFile(t, root, ".claude/agents/AGENTS.md")
	mkFile(t, root, "docs/guide.md")

	got, err := collectDocFiles(root, []string{"**/AGENTS.md"})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		filepath.Join(root, ".claude/agents/AGENTS.md"),
		filepath.Join(root, "AGENTS.md"),
	}
	assertPaths(t, got, want)
}

func TestCollectDocFiles_GlobSingleLevel(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, "docs/one.md")
	mkFile(t, root, "docs/two.md")
	mkFile(t, root, "docs/sub/three.md")

	got, err := collectDocFiles(root, []string{"docs/*.md"})
	if err != nil {
		t.Fatal(err)
	}

	// Single * should not match across path separators.
	want := []string{
		filepath.Join(root, "docs/one.md"),
		filepath.Join(root, "docs/two.md"),
	}
	assertPaths(t, got, want)
}

func TestCollectDocFiles_GlobNoMatch(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, "src/main.go")

	got, err := collectDocFiles(root, []string{"**/AGENTS.md"})
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 0 {
		t.Fatalf("expected no results, got %v", got)
	}
}

func TestCollectDocFiles_MixedGlobAndLiteral_Dedup(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, "docs/guide.md")
	mkFile(t, root, "docs/api.md")
	mkFile(t, root, "README.md")

	// docs/ directory + glob that overlaps with it.
	got, err := collectDocFiles(root, []string{"docs/", "docs/*.md", "README.md"})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		filepath.Join(root, "README.md"),
		filepath.Join(root, "docs/api.md"),
		filepath.Join(root, "docs/guide.md"),
	}
	assertPaths(t, got, want)
}

func TestCollectDocFiles_LiteralPath(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, "README.md")
	mkFile(t, root, "docs/intro.md")

	got, err := collectDocFiles(root, []string{"README.md", "docs/"})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		filepath.Join(root, "README.md"),
		filepath.Join(root, "docs/intro.md"),
	}
	assertPaths(t, got, want)
}

// --- helpers ---

func mkFile(t *testing.T, root, rel string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("# "+rel+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertPaths(t *testing.T, got, want []string) {
	t.Helper()
	sort.Strings(got)
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("length mismatch:\n  got  %v\n  want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("index %d:\n  got  %s\n  want %s", i, got[i], want[i])
		}
	}
}
