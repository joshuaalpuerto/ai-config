package hooks

// Integration tests exercise the full LoadConfig → Evaluate pipeline using
// realistic Claude Code PreToolUse event payloads for every supported tool type.
// Each test writes a temporary hooks.yaml, loads it, and evaluates a real event.

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Bash tool ---

func TestIntegration_Bash_BlockForcePush(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - match:
      tools: [Bash]
      command_match: "git push.*--force"
    action:
      block: true
      message: "Force-pushing is not allowed."
`)
	resp, err := Evaluate(newBashEvent("git push --force origin main", "Force push"), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Continue {
		t.Fatal("expected block for force push")
	}
	if resp.Context != "Force-pushing is not allowed." {
		t.Fatalf("unexpected block message: %q", resp.Context)
	}
}

func TestIntegration_Bash_AllowSafeCommand(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - match:
      tools: [Bash]
      command_match: "git push.*--force"
    action:
      block: true
      message: "Force-pushing is not allowed."
`)
	resp, err := Evaluate(newBashEvent("npm test", "Run test suite"), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Continue {
		t.Fatalf("expected safe Bash command to be allowed, got: %q", resp.Context)
	}
}

func TestIntegration_Bash_BlockDestructiveCommand(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - match:
      tools: [Bash]
      command_match: "rm -rf /"
    action:
      block: true
      message: "Deleting the root filesystem is not allowed."
`)
	resp, _ := Evaluate(newBashEvent("rm -rf /", "Delete everything"), cfg, nil)
	if resp.Continue {
		t.Fatal("expected block for rm -rf /")
	}
}

func TestIntegration_Bash_WarnMode_ExitsOpen(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - mode: warn
    match:
      tools: [Bash]
      command_match: "git push"
    action:
      block: true
      message: "Pushing without review."
`)
	resp, _ := Evaluate(newBashEvent("git push origin main", "Push branch"), cfg, nil)
	if !resp.Continue {
		t.Fatal("warn mode must not block")
	}
	if resp.Context == "" {
		t.Fatal("expected [WARNING] context in warn mode")
	}
}

func TestIntegration_Bash_AuditMode_NoEffect(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - mode: audit
    match:
      tools: [Bash]
    action:
      block: true
`)
	resp, _ := Evaluate(newBashEvent("anything", ""), cfg, nil)
	if !resp.Continue {
		t.Fatal("audit mode must never block")
	}
	if resp.Context != "" {
		t.Fatal("audit mode must not inject context")
	}
}

// --- Write tool ---
// Claude Code sends: { "file_path": "/abs/path/file.ext", "content": "..." }

func TestIntegration_Write_BlockEnvFile(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - match:
      tools: [Write]
      paths: [.env]
    action:
      block: true
      message: "Writing .env files is not allowed."
`)
	resp, _ := Evaluate(newWriteEvent("/home/user/my-project/.env", "SECRET=abc"), cfg, nil)
	if resp.Continue {
		t.Fatal("expected .env write to be blocked")
	}
}

func TestIntegration_Write_InjectPythonStandards(t *testing.T) {
	tmp := t.TempDir()
	contextFile := filepath.Join(tmp, "python-standards.md")
	if err := os.WriteFile(contextFile, []byte("# Python Standards\nFollow PEP8."), 0o644); err != nil {
		t.Fatal(err)
	}

	resetCache()
	hooksPath := filepath.Join(tmp, "hooks.yaml")
	yaml := `version: "1"
PreToolUse:
  - match:
      tools: [Write]
      paths: ["*.py"]
    action:
      inject: ` + contextFile + `
`
	if err := os.WriteFile(hooksPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := LoadConfig(hooksPath)
	if err != nil {
		t.Fatal(err)
	}

	resp, _ := Evaluate(newWriteEvent("/home/user/my-project/src/api/routes.py", "def hello(): pass"), cfg, nil)
	if !resp.Continue {
		t.Fatal("expected allow with inject")
	}
	if resp.Context != "# Python Standards\nFollow PEP8." {
		t.Fatalf("unexpected context: %q", resp.Context)
	}
}

func TestIntegration_Write_AllowNonRestrictedFile(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - match:
      tools: [Write]
      paths: [.env]
    action:
      block: true
`)
	resp, _ := Evaluate(newWriteEvent("/home/user/my-project/src/main.go", "package main"), cfg, nil)
	if !resp.Continue {
		t.Fatal("expected non-.env Write to be allowed")
	}
}

// --- Edit tool ---
// Claude Code sends: { "file_path": "...", "old_string": "...", "new_string": "...", "replace_all": false }

func TestIntegration_Edit_InjectInlineForPython(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - match:
      tools: [Edit]
      paths: ["*.py"]
    action:
      inject_inline: "Reminder: follow PEP8 and add type hints."
`)
	resp, _ := Evaluate(
		newEditEvent("/home/user/my-project/src/service.py", "def foo():", "def foo() -> None:"),
		cfg,
		nil,
	)
	if !resp.Continue {
		t.Fatal("expected Continue=true for inject")
	}
	if resp.Context != "Reminder: follow PEP8 and add type hints." {
		t.Fatalf("unexpected context: %q", resp.Context)
	}
}

func TestIntegration_Edit_BlockRestrictedDirectory(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - match:
      tools: [Edit]
      paths: [/home/user/my-project/infra/]
    action:
      block: true
      message: "Direct edits to infra/ are not allowed."
`)
	resp, _ := Evaluate(
		newEditEvent("/home/user/my-project/infra/main.tf", "old", "new"),
		cfg,
		nil,
	)
	if resp.Continue {
		t.Fatal("expected edit in infra/ to be blocked")
	}
}

func TestIntegration_Edit_AllowUnrestrictedFile(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - match:
      tools: [Edit]
      paths: [/home/user/my-project/infra/]
    action:
      block: true
`)
	resp, _ := Evaluate(
		newEditEvent("/home/user/my-project/src/handler.go", "old", "new"),
		cfg,
		nil,
	)
	if !resp.Continue {
		t.Fatal("expected Edit outside infra/ to be allowed")
	}
}

// --- Read tool ---
// Claude Code sends: { "file_path": "...", "offset": N, "limit": N }

func TestIntegration_Read_BlockEnvFile(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - match:
      tools: [Read]
      paths: [.env]
    action:
      block: true
      message: "Reading .env files is not allowed."
`)
	resp, _ := Evaluate(newReadEvent("/home/user/my-project/.env"), cfg, nil)
	if resp.Continue {
		t.Fatal("expected .env Read to be blocked")
	}
}

func TestIntegration_Read_AllowNormalFile(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - match:
      tools: [Read]
      paths: [.env]
    action:
      block: true
`)
	resp, _ := Evaluate(newReadEvent("/home/user/my-project/README.md"), cfg, nil)
	if !resp.Continue {
		t.Fatal("expected normal Read to be allowed")
	}
}

func TestIntegration_Read_InjectContextForKeyFiles(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - match:
      tools: [Read]
      paths: ["*.go"]
    action:
      inject_inline: "Note: follow the project Go style guide."
`)
	resp, _ := Evaluate(newReadEvent("/home/user/my-project/internal/handler.go"), cfg, nil)
	if !resp.Continue {
		t.Fatal("expected Continue=true")
	}
	if resp.Context != "Note: follow the project Go style guide." {
		t.Fatalf("unexpected context: %q", resp.Context)
	}
}

// --- WebFetch tool ---
// Claude Code sends: { "url": "https://...", "prompt": "..." }

func TestIntegration_WebFetch_BlockAllExternalFetches(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - match:
      tools: [WebFetch]
    action:
      block: true
      message: "External web fetches are not permitted."
`)
	resp, _ := Evaluate(newWebFetchEvent("https://external.example.com/api", "Extract API endpoints"), cfg, nil)
	if resp.Continue {
		t.Fatal("expected WebFetch to be blocked")
	}
	if resp.Context != "External web fetches are not permitted." {
		t.Fatalf("unexpected block message: %q", resp.Context)
	}
}

func TestIntegration_WebFetch_WarnMode_AllowsWithContext(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - mode: warn
    match:
      tools: [WebFetch]
    action:
      block: true
      message: "External fetch detected."
`)
	resp, _ := Evaluate(newWebFetchEvent("https://external.example.com/api", "Get data"), cfg, nil)
	if !resp.Continue {
		t.Fatal("warn mode must not block WebFetch")
	}
	if resp.Context == "" {
		t.Fatal("expected warning context")
	}
}

// --- Cross-tool: multiple rules, multiple tool types ---

func TestIntegration_MultipleRules_OnlyMatchingApply(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PreToolUse:
  - match:
      tools: [Bash]
      command_match: "git push.*--force"
    action:
      block: true
      message: "Force push blocked."
  - match:
      tools: [Write, Edit]
      paths: ["*.py"]
    action:
      inject_inline: "Python hint applied."
  - match:
      tools: [Read]
      paths: [.env]
    action:
      block: true
`)
	tests := []struct {
		name      string
		event     Event
		wantAllow bool
		wantCtx   string
	}{
		{
			name:      "Bash safe command passes all rules",
			event:     newBashEvent("npm test", "Run tests"),
			wantAllow: true,
		},
		{
			name:      "Bash force-push blocked",
			event:     newBashEvent("git push --force origin main", ""),
			wantAllow: false,
		},
		{
			name:      "Write .py triggers inject only",
			event:     newWriteEvent("/project/routes.py", "def f(): pass"),
			wantAllow: true,
			wantCtx:   "Python hint applied.",
		},
		{
			name:      "Edit .py triggers inject only",
			event:     newEditEvent("/project/routes.py", "old", "new"),
			wantAllow: true,
			wantCtx:   "Python hint applied.",
		},
		{
			name:      "Read .env blocked",
			event:     newReadEvent("/project/.env"),
			wantAllow: false,
		},
		{
			name:      "Read .go passes",
			event:     newReadEvent("/project/main.go"),
			wantAllow: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := Evaluate(tc.event, cfg, nil)
			if err != nil {
				t.Fatalf("Evaluate error: %v", err)
			}
			if resp.Continue != tc.wantAllow {
				t.Fatalf("Continue: got %v, want %v (context=%q)", resp.Continue, tc.wantAllow, resp.Context)
			}
			if tc.wantCtx != "" && resp.Context != tc.wantCtx {
				t.Fatalf("context: got %q, want %q", resp.Context, tc.wantCtx)
			}
		})
	}
}

// --- run_inline action ---

func TestIntegration_PostToolUse_RunInline(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PostToolUse:
  - match:
      tools: [Write]
    action:
      run_inline: "echo 'formatted'"
`)
	resp, err := Evaluate(postToolUseEvent("Write", "/project/main.go", map[string]any{"success": true}), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Continue {
		t.Fatal("expected Continue=true")
	}
	if resp.Context != "formatted" {
		t.Fatalf("expected 'formatted', got %q", resp.Context)
	}
}

func TestIntegration_PostToolUse_RunInline_NonZeroBlocks(t *testing.T) {
	cfg := intCfg(t, `
version: "1"
PostToolUse:
  - match:
      tools: [Write]
    action:
      run_inline: "exit 1"
`)
	resp, _ := Evaluate(postToolUseEvent("Write", "/project/main.go", map[string]any{"success": true}), cfg, nil)
	if resp.Continue {
		t.Fatal("expected block on non-zero exit")
	}
}

// --- Fail-open: missing hooks.yaml must allow all tool events ---

func TestIntegration_FailOpen_MissingConfig_AllowAllTools(t *testing.T) {
	tools := []struct {
		name  string
		event Event
	}{
		{"Bash", newBashEvent("echo hi", "")},
		{"Write", newWriteEvent("/project/file.txt", "content")},
		{"Edit", newEditEvent("/project/file.txt", "a", "b")},
		{"Read", newReadEvent("/project/file.txt")},
		{"WebFetch", newWebFetchEvent("https://example.com", "get data")},
	}

	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			resetCache()
			_, failOpen, err := LoadConfig("/nonexistent/hooks.yaml")
			if err == nil {
				t.Fatal("expected error for missing config")
			}
			if !failOpen {
				t.Fatal("expected fail_open=true")
			}
			// When config is unavailable the caller honours fail_open and allows.
			// Evaluate with an empty config simulates the fail-open path.
			resp, _ := Evaluate(tc.event, HooksConfig{}, nil)
			if !resp.Continue {
				t.Fatalf("fail-open path must allow %s events", tc.name)
			}
		})
	}
}
