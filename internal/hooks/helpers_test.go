package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// --- pointer helpers ---

func boolPtr(b bool) *bool { return &b }

// --- minimal config fixture ---

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

// --- event builders (unit-test style, no description/extra fields) ---

func bashEvent(command string) Event {
	input, _ := json.Marshal(map[string]string{"command": command})
	return Event{
		HookEventName: EventPreToolUse,
		SessionID:     "test",
		ToolName:      "Bash",
		ToolInput:     input,
	}
}

func fileEvent(tool, filePath string) Event {
	input, _ := json.Marshal(map[string]string{"file_path": filePath})
	return Event{
		HookEventName: EventPreToolUse,
		SessionID:     "test",
		ToolName:      tool,
		ToolInput:     input,
	}
}

func webFetchEvent(url string) Event {
	input, _ := json.Marshal(map[string]string{"url": url, "prompt": "extract endpoints"})
	return Event{
		HookEventName: EventPreToolUse,
		SessionID:     "test",
		ToolName:      "WebFetch",
		ToolInput:     input,
	}
}

// --- event builders (integration style, full Claude Code payloads) ---

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

// --- config loader helpers ---

// writeTempHooksYAML writes content to hooks.yaml inside dir and returns the path.
func writeTempHooksYAML(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "hooks.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp hooks.yaml: %v", err)
	}
	return path
}

// intCfg writes a hooks.yaml to a temp dir, loads it, and returns the config.
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
