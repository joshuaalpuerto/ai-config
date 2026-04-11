package transpiler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joshuaalpuerto/ai-config/internal/hooks"
)

// --- translateInjectPath ---

func TestTranslateInjectPath_Empty(t *testing.T) {
	got, err := translateInjectPath("", ".claude/context", "/tmp/root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestTranslateInjectPath_PlatformPrefix_Claude(t *testing.T) {
	got, err := translateInjectPath("$/context/python-standards.md", ".claude/context", "/tmp/root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(".claude", "context", "python-standards.md")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTranslateInjectPath_PlatformPrefix_GitHub(t *testing.T) {
	got, err := translateInjectPath("$/context/api-guidelines.md", ".github/context", "/tmp/root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(".github", "context", "api-guidelines.md")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTranslateInjectPath_PlatformPrefix_PreservesSubdirs(t *testing.T) {
	got, err := translateInjectPath("$/context/deep/nested/file.md", ".claude/context", "/tmp/root")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(".claude", "context", "deep", "nested", "file.md")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTranslateInjectPath_BarePath_PassesThrough(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "docs", "GUIDELINES.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("# guide"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := translateInjectPath("docs/GUIDELINES.md", ".claude/context", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "docs/GUIDELINES.md" {
		t.Fatalf("bare path should pass through unchanged, got %q", got)
	}
}

func TestTranslateInjectPath_BarePath_DotSlash_PassesThrough(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "docs", "GUIDELINES.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("# guide"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := translateInjectPath("./docs/GUIDELINES.md", ".claude/context", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "./docs/GUIDELINES.md" {
		t.Fatalf("bare path should pass through unchanged, got %q", got)
	}
}

func TestTranslateInjectPath_BarePath_MissingFile_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	_, err := translateInjectPath("docs/MISSING.md", ".claude/context", dir)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

// --- transformRules ---

func TestTransformRules_MixedPaths(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "docs", "GUIDELINES.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("# guide"), 0o644); err != nil {
		t.Fatal(err)
	}

	rules := []hooks.Rule{
		{
			Match:  hooks.Matchers{Tools: []string{"Write"}},
			Action: hooks.Actions{Inject: "$/context/standards.md"},
		},
		{
			Match:  hooks.Matchers{Tools: []string{"Write"}},
			Action: hooks.Actions{Inject: "docs/GUIDELINES.md"},
		},
	}

	err := transformRules(rules, nil, ".claude/context", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantPlatform := filepath.Join(".claude", "context", "standards.md")
	if rules[0].Action.Inject != wantPlatform {
		t.Fatalf("rule[0]: got %q, want %q", rules[0].Action.Inject, wantPlatform)
	}
	if rules[1].Action.Inject != "docs/GUIDELINES.md" {
		t.Fatalf("rule[1]: got %q, want unchanged %q", rules[1].Action.Inject, "docs/GUIDELINES.md")
	}
}

func TestTransformRules_ErrorOnMissingBareFile(t *testing.T) {
	dir := t.TempDir()
	rules := []hooks.Rule{
		{
			Match:  hooks.Matchers{Tools: []string{"Write"}},
			Action: hooks.Actions{Inject: "missing/file.md"},
		},
	}

	err := transformRules(rules, nil, ".claude/context", dir)
	if err == nil {
		t.Fatal("expected error for missing bare inject path, got nil")
	}
}

// --- translateTools (regression) ---

func TestTranslateTools_MapsToGitHub(t *testing.T) {
	pm := map[string]string{"Bash": "execute", "Write": "edit"}
	got := translateTools([]string{"Bash", "Write"}, pm)
	if got[0] != "execute" || got[1] != "edit" {
		t.Fatalf("unexpected mapping: %v", got)
	}
}

func TestTranslateTools_KeepsUnmapped(t *testing.T) {
	pm := map[string]string{}
	got := translateTools([]string{"Write"}, pm)
	if got[0] != "Write" {
		t.Fatalf("unmapped tool should be kept, got %q", got[0])
	}
}

func TestTranslateTools_EmptySlice(t *testing.T) {
	got := translateTools(nil, nil)
	if got != nil {
		t.Fatalf("expected nil for nil input, got %v", got)
	}
}
