package hooks

// Integration tests exercise the full LoadConfig → Evaluate pipeline using
// realistic Claude Code PreToolUse event payloads for every supported tool type.
// Each test writes a temporary hooks.yaml, loads it, and evaluates a real event.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// intCfg writes hooks.yaml to a temp dir, loads it, and returns the config.
// It resets the loader cache so that tests are fully isolated.
func intCfg(t *testing.T, yaml string) HooksConfig {
	t.Helper()
	resetCache()
	tmp := t.TempDir()
	path := filepath.Join(tmp, "hooks.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("writing hooks.yaml: %v", err)
	}
	cfg, _, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	return cfg
}

// --- Bash tool ---
// Claude Code sends: { "command": "...", "description": "...", "timeout": N, "run_in_background": false }

func newBashEvent(command, description string) Event {
	input, _ := json.Marshal(map[string]any{
		"command":           command,
		"description":       description,
		"timeout":           120000,
		"run_in_background": false,
	})
	return Event{
		HookEventName: EventPreToolUse,
		SessionID:     "int-test",
		ToolName:      "Bash",
		CWD:           "/home/user/my-project",
		ToolInput:     input,
	}
}

func TestIntegration_Bash_BlockForcePush(t *testing.T) {
	cfg := intCfg(t, `
version: "1.0"
rules:
  - name: block-force-push
    matchers:
      tools: [Bash]
      command_match: "git push.*--force"
    actions:
      block: true
      block_message: "Force-pushing is not allowed."
`)
	resp, err := Evaluate(newBashEvent("git push --force origin main", "Force push"), cfg)
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
version: "1.0"
rules:
  - name: block-force-push
    matchers:
      tools: [Bash]
      command_match: "git push.*--force"
    actions:
      block: true
      block_message: "Force-pushing is not allowed."
`)
	resp, err := Evaluate(newBashEvent("npm test", "Run test suite"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Continue {
		t.Fatalf("expected safe Bash command to be allowed, got: %q", resp.Context)
	}
}

func TestIntegration_Bash_BlockDestructiveCommand(t *testing.T) {
	cfg := intCfg(t, `
version: "1.0"
rules:
  - name: block-rm-rf
    matchers:
      tools: [Bash]
      command_match: "rm -rf /"
    actions:
      block: true
      block_message: "Deleting the root filesystem is not allowed."
`)
	resp, _ := Evaluate(newBashEvent("rm -rf /", "Delete everything"), cfg)
	if resp.Continue {
		t.Fatal("expected block for rm -rf /")
	}
}

func TestIntegration_Bash_WarnMode_ExitsOpen(t *testing.T) {
	cfg := intCfg(t, `
version: "1.0"
rules:
  - name: warn-git-push
    mode: warn
    matchers:
      tools: [Bash]
      command_match: "git push"
    actions:
      block: true
      block_message: "Pushing without review."
`)
	resp, _ := Evaluate(newBashEvent("git push origin main", "Push branch"), cfg)
	if !resp.Continue {
		t.Fatal("warn mode must not block")
	}
	if resp.Context == "" {
		t.Fatal("expected [WARNING] context in warn mode")
	}
}

func TestIntegration_Bash_AuditMode_NoEffect(t *testing.T) {
	cfg := intCfg(t, `
version: "1.0"
rules:
  - name: audit-all-bash
    mode: audit
    matchers:
      tools: [Bash]
    actions:
      block: true
`)
	resp, _ := Evaluate(newBashEvent("anything", ""), cfg)
	if !resp.Continue {
		t.Fatal("audit mode must never block")
	}
	if resp.Context != "" {
		t.Fatal("audit mode must not inject context")
	}
}

// --- Write tool ---
// Claude Code sends: { "file_path": "/abs/path/file.ext", "content": "..." }

func newWriteEvent(filePath, content string) Event {
	input, _ := json.Marshal(map[string]string{
		"file_path": filePath,
		"content":   content,
	})
	return Event{
		HookEventName: EventPreToolUse,
		SessionID:     "int-test",
		ToolName:      "Write",
		CWD:           "/home/user/my-project",
		ToolInput:     input,
	}
}

func TestIntegration_Write_BlockEnvFile(t *testing.T) {
	cfg := intCfg(t, `
version: "1.0"
rules:
  - name: block-env-writes
    matchers:
      tools: [Write]
      extensions: [.env]
    actions:
      block: true
      block_message: "Writing .env files is not allowed."
`)
	resp, _ := Evaluate(newWriteEvent("/home/user/my-project/.env", "SECRET=abc"), cfg)
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
	yaml := `version: "1.0"
rules:
  - name: inject-python-standards
    matchers:
      tools: [Write]
      extensions: [.py]
    actions:
      inject: ` + contextFile + `
`
	if err := os.WriteFile(hooksPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := LoadConfig(hooksPath)
	if err != nil {
		t.Fatal(err)
	}

	resp, _ := Evaluate(newWriteEvent("/home/user/my-project/src/api/routes.py", "def hello(): pass"), cfg)
	if !resp.Continue {
		t.Fatal("expected allow with inject")
	}
	if resp.Context != "# Python Standards\nFollow PEP8." {
		t.Fatalf("unexpected context: %q", resp.Context)
	}
}

func TestIntegration_Write_AllowNonRestrictedFile(t *testing.T) {
	cfg := intCfg(t, `
version: "1.0"
rules:
  - name: block-env-writes
    matchers:
      tools: [Write]
      extensions: [.env]
    actions:
      block: true
`)
	resp, _ := Evaluate(newWriteEvent("/home/user/my-project/src/main.go", "package main"), cfg)
	if !resp.Continue {
		t.Fatal("expected non-.env Write to be allowed")
	}
}

// --- Edit tool ---
// Claude Code sends: { "file_path": "...", "old_string": "...", "new_string": "...", "replace_all": false }

func newEditEvent(filePath, oldString, newString string) Event {
	input, _ := json.Marshal(map[string]any{
		"file_path":   filePath,
		"old_string":  oldString,
		"new_string":  newString,
		"replace_all": false,
	})
	return Event{
		HookEventName: EventPreToolUse,
		SessionID:     "int-test",
		ToolName:      "Edit",
		CWD:           "/home/user/my-project",
		ToolInput:     input,
	}
}

func TestIntegration_Edit_InjectInlineForPython(t *testing.T) {
	cfg := intCfg(t, `
version: "1.0"
rules:
  - name: py-edit-standards
    matchers:
      tools: [Edit]
      extensions: [.py]
    actions:
      inject_inline: "Reminder: follow PEP8 and add type hints."
`)
	resp, _ := Evaluate(
		newEditEvent("/home/user/my-project/src/service.py", "def foo():", "def foo() -> None:"),
		cfg,
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
version: "1.0"
rules:
  - name: block-infra-edits
    matchers:
      tools: [Edit]
      directories: [/home/user/my-project/infra]
    actions:
      block: true
      block_message: "Direct edits to infra/ are not allowed."
`)
	resp, _ := Evaluate(
		newEditEvent("/home/user/my-project/infra/main.tf", "old", "new"),
		cfg,
	)
	if resp.Continue {
		t.Fatal("expected edit in infra/ to be blocked")
	}
}

func TestIntegration_Edit_AllowUnrestrictedFile(t *testing.T) {
	cfg := intCfg(t, `
version: "1.0"
rules:
  - name: block-infra-edits
    matchers:
      tools: [Edit]
      directories: [/home/user/my-project/infra]
    actions:
      block: true
`)
	resp, _ := Evaluate(
		newEditEvent("/home/user/my-project/src/handler.go", "old", "new"),
		cfg,
	)
	if !resp.Continue {
		t.Fatal("expected Edit outside infra/ to be allowed")
	}
}

// --- Read tool ---
// Claude Code sends: { "file_path": "...", "offset": N, "limit": N }

func newReadEvent(filePath string) Event {
	input, _ := json.Marshal(map[string]any{
		"file_path": filePath,
		"offset":    0,
		"limit":     50,
	})
	return Event{
		HookEventName: EventPreToolUse,
		SessionID:     "int-test",
		ToolName:      "Read",
		CWD:           "/home/user/my-project",
		ToolInput:     input,
	}
}

func TestIntegration_Read_BlockEnvFile(t *testing.T) {
	cfg := intCfg(t, `
version: "1.0"
rules:
  - name: block-env-reads
    matchers:
      tools: [Read]
      extensions: [.env]
    actions:
      block: true
      block_message: "Reading .env files is not allowed."
`)
	resp, _ := Evaluate(newReadEvent("/home/user/my-project/.env"), cfg)
	if resp.Continue {
		t.Fatal("expected .env Read to be blocked")
	}
}

func TestIntegration_Read_AllowNormalFile(t *testing.T) {
	cfg := intCfg(t, `
version: "1.0"
rules:
  - name: block-env-reads
    matchers:
      tools: [Read]
      extensions: [.env]
    actions:
      block: true
`)
	resp, _ := Evaluate(newReadEvent("/home/user/my-project/README.md"), cfg)
	if !resp.Continue {
		t.Fatal("expected normal Read to be allowed")
	}
}

func TestIntegration_Read_InjectContextForKeyFiles(t *testing.T) {
	cfg := intCfg(t, `
version: "1.0"
rules:
  - name: annotate-go-reads
    matchers:
      tools: [Read]
      extensions: [.go]
    actions:
      inject_inline: "Note: follow the project Go style guide."
`)
	resp, _ := Evaluate(newReadEvent("/home/user/my-project/internal/handler.go"), cfg)
	if !resp.Continue {
		t.Fatal("expected Continue=true")
	}
	if resp.Context != "Note: follow the project Go style guide." {
		t.Fatalf("unexpected context: %q", resp.Context)
	}
}

// --- WebFetch tool ---
// Claude Code sends: { "url": "https://...", "prompt": "..." }

func newWebFetchEvent(url, prompt string) Event {
	input, _ := json.Marshal(map[string]string{
		"url":    url,
		"prompt": prompt,
	})
	return Event{
		HookEventName: EventPreToolUse,
		SessionID:     "int-test",
		ToolName:      "WebFetch",
		CWD:           "/home/user/my-project",
		ToolInput:     input,
	}
}

func TestIntegration_WebFetch_BlockAllExternalFetches(t *testing.T) {
	cfg := intCfg(t, `
version: "1.0"
rules:
  - name: no-external-fetch
    matchers:
      tools: [WebFetch]
    actions:
      block: true
      block_message: "External web fetches are not permitted."
`)
	resp, _ := Evaluate(newWebFetchEvent("https://external.example.com/api", "Extract API endpoints"), cfg)
	if resp.Continue {
		t.Fatal("expected WebFetch to be blocked")
	}
	if resp.Context != "External web fetches are not permitted." {
		t.Fatalf("unexpected block message: %q", resp.Context)
	}
}

func TestIntegration_WebFetch_WarnMode_AllowsWithContext(t *testing.T) {
	cfg := intCfg(t, `
version: "1.0"
rules:
  - name: warn-external-fetch
    mode: warn
    matchers:
      tools: [WebFetch]
    actions:
      block: true
      block_message: "External fetch detected."
`)
	resp, _ := Evaluate(newWebFetchEvent("https://external.example.com/api", "Get data"), cfg)
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
version: "1.0"
rules:
  - name: block-bash-force-push
    matchers:
      tools: [Bash]
      command_match: "git push.*--force"
    actions:
      block: true
      block_message: "Force push blocked."
  - name: inject-python-hint
    matchers:
      tools: [Write, Edit]
      extensions: [.py]
    actions:
      inject_inline: "Python hint applied."
  - name: block-env-reads
    matchers:
      tools: [Read]
      extensions: [.env]
    actions:
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
			resp, err := Evaluate(tc.event, cfg)
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
			resp, _ := Evaluate(tc.event, HooksConfig{})
			if !resp.Continue {
				t.Fatalf("fail-open path must allow %s events", tc.name)
			}
		})
	}
}
