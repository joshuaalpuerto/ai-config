package hooks

import (
	"encoding/json"
	"testing"
)

func TestFormatOutput_ClaudeAllowWithContext(t *testing.T) {
	event := bashEvent("npm test")
	resp := Response{Continue: true, Context: "follow coding standards"}

	stdout, stderr, exitCode := FormatOutput(PlatformClaude, event, resp)

	if exitCode != 0 {
		t.Fatalf("want exit 0, got %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("want empty stderr, got %q", stderr)
	}

	var out claudePreToolUseOutput
	if err := json.Unmarshal(stdout, &out); err != nil {
		t.Fatalf("stdout is not valid JSON: %v — raw: %s", err, stdout)
	}
	if out.HookSpecificOutput.HookEventName != string(EventPreToolUse) {
		t.Errorf("hookEventName: want %q, got %q", EventPreToolUse, out.HookSpecificOutput.HookEventName)
	}
	if out.HookSpecificOutput.AdditionalContext != "follow coding standards" {
		t.Errorf("additionalContext: want %q, got %q", "follow coding standards", out.HookSpecificOutput.AdditionalContext)
	}
}

func TestFormatOutput_ClaudeAllowNoContext(t *testing.T) {
	event := bashEvent("npm test")
	resp := Response{Continue: true, Context: ""}

	stdout, stderr, exitCode := FormatOutput(PlatformClaude, event, resp)

	if exitCode != 0 {
		t.Fatalf("want exit 0, got %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("want empty stderr, got %q", stderr)
	}
	if len(stdout) != 0 {
		t.Fatalf("want empty stdout, got %q", stdout)
	}
}

func TestFormatOutput_ClaudeBlock(t *testing.T) {
	event := bashEvent("rm -rf /")
	resp := Response{Continue: false, Context: "Blocked: destructive command"}

	stdout, stderr, exitCode := FormatOutput(PlatformClaude, event, resp)

	if exitCode != 2 {
		t.Fatalf("want exit 2, got %d", exitCode)
	}
	if stderr != "Blocked: destructive command" {
		t.Errorf("stderr: want %q, got %q", "Blocked: destructive command", stderr)
	}
	if len(stdout) != 0 {
		t.Fatalf("want empty stdout on block, got %q", stdout)
	}
}

func TestFormatOutput_GitHubNotYetSupported(t *testing.T) {
	event := bashEvent("npm test")
	resp := Response{Continue: true, Context: "some context"}

	_, stderr, exitCode := FormatOutput(PlatformGitHub, event, resp)

	if exitCode != 2 {
		t.Fatalf("want exit 2 for unsupported platform, got %d", exitCode)
	}
	if stderr == "" {
		t.Fatal("want non-empty stderr for unsupported platform")
	}
}

func TestFormatOutput_UnknownPlatform(t *testing.T) {
	event := bashEvent("npm test")
	resp := Response{Continue: true, Context: "some context"}

	_, stderr, exitCode := FormatOutput(Platform("vscode"), event, resp)

	if exitCode != 2 {
		t.Fatalf("want exit 2 for unknown platform, got %d", exitCode)
	}
	if stderr == "" {
		t.Fatal("want non-empty stderr for unknown platform")
	}
}
