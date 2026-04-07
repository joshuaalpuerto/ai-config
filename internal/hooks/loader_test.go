package hooks

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig_MissingFile_FailOpen(t *testing.T) {
	resetCache()
	_, failOpen, err := LoadConfig("/nonexistent/path/hooks.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !failOpen {
		t.Fatal("expected fail_open=true when file is missing")
	}
}

func TestLoadConfig_ValidFile_Parsed(t *testing.T) {
	resetCache()
	tmp := t.TempDir()
	path := writeTempHooksYAML(t, tmp, minimalHooksYAML)

	cfg, failOpen, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !failOpen {
		t.Fatal("expected fail_open=true (default)")
	}
	if len(cfg.PreToolUse) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(cfg.PreToolUse))
	}
	if cfg.PreToolUse[0].Match.Tools[0] != "Bash" {
		t.Fatalf("unexpected first tool: %q", cfg.PreToolUse[0].Match.Tools[0])
	}
}

func TestLoadConfig_CacheHit_NoDiskRead(t *testing.T) {
	resetCache()
	tmp := t.TempDir()
	path := writeTempHooksYAML(t, tmp, minimalHooksYAML)

	// First load — populates cache.
	cfg1, _, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	// Overwrite file content on disk — cache must return the original.
	overwritten := minimalHooksYAML + `
  - match:
      tools: [Write]
    action:
      block: true
`
	// Keep same mtime to simulate cache hit.
	info, _ := os.Stat(path)
	origMtime := info.ModTime()
	if err := os.WriteFile(path, []byte(overwritten), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, origMtime, origMtime); err != nil {
		t.Fatal(err)
	}

	cfg2, _, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg2.PreToolUse) != len(cfg1.PreToolUse) {
		t.Fatalf("cache miss: expected %d rules (cached), got %d", len(cfg1.PreToolUse), len(cfg2.PreToolUse))
	}
}

func TestLoadConfig_CacheMiss_MtimeChanged(t *testing.T) {
	resetCache()
	tmp := t.TempDir()
	path := writeTempHooksYAML(t, tmp, minimalHooksYAML)

	_, _, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	// Update file with a newer mtime — cache must be invalidated.
	updated := minimalHooksYAML + `
  - match:
      tools: [Write]
    action:
      block: true
      message: "new"
`
	// Ensure mtime advances.
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.PreToolUse) != 2 {
		t.Fatalf("expected 2 rules after cache invalidation, got %d", len(cfg.PreToolUse))
	}
}

func TestLoadConfig_FailOpen_ExplicitFalse(t *testing.T) {
	resetCache()
	tmp := t.TempDir()
	content := `
version: "1"
PreToolUse: []
settings:
  fail_open: false
`
	path := writeTempHooksYAML(t, tmp, content)

	_, failOpen, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if failOpen {
		t.Fatal("expected fail_open=false when explicitly set")
	}
}

func TestLoadConfig_FailOpen_ExplicitTrue(t *testing.T) {
	resetCache()
	tmp := t.TempDir()
	content := `
version: "1"
PreToolUse: []
settings:
  fail_open: true
`
	path := writeTempHooksYAML(t, tmp, content)

	_, failOpen, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !failOpen {
		t.Fatal("expected fail_open=true when explicitly set")
	}
}

func TestLoadConfig_MalformedYAML_Error(t *testing.T) {
	resetCache()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "hooks.yaml")
	// A tab used as indentation is rejected by gopkg.in/yaml.v3.
	content := "version: \"1\"\nPreToolUse:\n\t- bad-tab-indent"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}
