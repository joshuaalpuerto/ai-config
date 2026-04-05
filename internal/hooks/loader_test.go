package hooks

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTempHooksYAML(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "hooks.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp hooks.yaml: %v", err)
	}
	return path
}

const minimalHooksYAML = `
version: "1.0"
rules:
  - name: block-force-push
    matchers:
      tools: [Bash]
      command_match: "git push.*--force"
    actions:
      block: true
      block_message: "Force-pushing is not allowed."
`

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
	if len(cfg.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(cfg.Rules))
	}
	if cfg.Rules[0].Name != "block-force-push" {
		t.Fatalf("unexpected rule name: %q", cfg.Rules[0].Name)
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
  - name: should-not-appear
    matchers:
      tools: [Write]
    actions:
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
	if len(cfg2.Rules) != len(cfg1.Rules) {
		t.Fatalf("cache miss: expected %d rules (cached), got %d", len(cfg1.Rules), len(cfg2.Rules))
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
  - name: new-rule
    matchers:
      tools: [Write]
    actions:
      block: true
      block_message: "new"
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
	if len(cfg.Rules) != 2 {
		t.Fatalf("expected 2 rules after cache invalidation, got %d", len(cfg.Rules))
	}
}

func TestLoadConfig_FailOpen_ExplicitFalse(t *testing.T) {
	resetCache()
	tmp := t.TempDir()
	content := `
version: "1.0"
rules: []
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
version: "1.0"
rules: []
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
	content := "version: \"1.0\"\nrules:\n\t- name: bad-tab-indent"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}
