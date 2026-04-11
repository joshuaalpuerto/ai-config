package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

// --- matchesRule ---

func TestMatchesRule_ToolFilter_Match(t *testing.T) {
	rule := Rule{
		Match:  Matchers{Tools: []string{"Bash"}},
		Action: Actions{Block: boolPtr(true)},
	}
	if !matchesRule(bashEvent("echo hi"), rule) {
		t.Fatal("expected rule to match Bash event")
	}
}

func TestMatchesRule_ToolFilter_NoMatch(t *testing.T) {
	rule := Rule{
		Match:  Matchers{Tools: []string{"Write"}},
		Action: Actions{Block: boolPtr(true)},
	}
	if matchesRule(bashEvent("echo hi"), rule) {
		t.Fatal("expected rule NOT to match Bash when tools=[Write]")
	}
}

func TestMatchesRule_ToolFilter_CaseInsensitive(t *testing.T) {
	rule := Rule{
		Match:  Matchers{Tools: []string{"bash"}},
		Action: Actions{Block: boolPtr(true)},
	}
	if !matchesRule(bashEvent("echo hi"), rule) {
		t.Fatal("expected case-insensitive tool match")
	}
}

func TestMatchesRule_CommandMatch_Hit(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:        []string{"Bash"},
			CommandMatch: `git push.*--force`,
		},
		Action: Actions{Block: boolPtr(true)},
	}
	if !matchesRule(bashEvent("git push --force origin main"), rule) {
		t.Fatal("expected command regex to match")
	}
}

func TestMatchesRule_CommandMatch_Miss(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:        []string{"Bash"},
			CommandMatch: `git push.*--force`,
		},
		Action: Actions{Block: boolPtr(true)},
	}
	if matchesRule(bashEvent("git status"), rule) {
		t.Fatal("expected command regex NOT to match git status")
	}
}

func TestMatchesRule_Paths_ExtensionGlob_Match(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:   []string{"Write"},
			Paths: []string{"*.py"},
		},
		Action: Actions{InjectInline: "python standards"},
	}
	if !matchesRule(fileEvent("Write", "/home/user/project/routes.py"), rule) {
		t.Fatal("expected *.py target to match any .py file")
	}
}

func TestMatchesRule_Paths_ExtensionGlob_Miss(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:   []string{"Write"},
			Paths: []string{"*.py"},
		},
		Action: Actions{InjectInline: "python standards"},
	}
	if matchesRule(fileEvent("Write", "/home/user/project/routes.ts"), rule) {
		t.Fatal("expected *.py target NOT to match a .ts file")
	}
}

func TestMatchesRule_Paths_SuffixPattern_Match(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:   []string{"Edit"},
			Paths: []string{"*-view.tsx"},
		},
		Action: Actions{InjectInline: "view standards"},
	}
	if !matchesRule(fileEvent("Edit", "/home/user/project/src/user-view.tsx"), rule) {
		t.Fatal("expected *-view.tsx target to match a file with that suffix")
	}
	if matchesRule(fileEvent("Edit", "/home/user/project/src/user-form.tsx"), rule) {
		t.Fatal("expected *-view.tsx target NOT to match a file without that suffix")
	}
}

func TestMatchesRule_Paths_DirectoryTrailingSlash_Match(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:   []string{"Edit"},
			Paths: []string{"src/api/"},
		},
		Action: Actions{InjectInline: "src standards"},
	}
	if !matchesRule(fileEvent("Edit", "src/api/routes.py"), rule) {
		t.Fatal("expected trailing-slash target to match a file inside that directory")
	}
}

func TestMatchesRule_Paths_DirectoryTrailingSlash_Miss(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:   []string{"Edit"},
			Paths: []string{"src/api/"},
		},
		Action: Actions{InjectInline: "src standards"},
	}
	if matchesRule(fileEvent("Edit", "src/tests/test_routes.py"), rule) {
		t.Fatal("expected trailing-slash target NOT to match a file outside that directory")
	}
}

func TestMatchesRule_NoMatchers_AlwaysMatches(t *testing.T) {
	rule := Rule{
		Match:  Matchers{},
		Action: Actions{InjectInline: "global context"},
	}
	if !matchesRule(bashEvent("anything"), rule) {
		t.Fatal("expected rule with no matchers to always match")
	}
}

func TestMatchesRule_WebFetch_ToolFilter(t *testing.T) {
	rule := Rule{
		Match:  Matchers{Tools: []string{"WebFetch"}},
		Action: Actions{Block: boolPtr(true), Message: "no external fetch"},
	}
	if !matchesRule(webFetchEvent("https://example.com"), rule) {
		t.Fatal("expected WebFetch tool to match")
	}
}

func TestMatchesRule_Paths_DoubleStarPath_Match(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:   []string{"Edit"},
			Paths: []string{"src/documentation/**/*.tsx"},
		},
		Action: Actions{InjectInline: "docs component standards"},
	}
	if !matchesRule(fileEvent("Edit", "src/documentation/ui/buttons/Button.tsx"), rule) {
		t.Fatal("expected src/documentation/**/*.tsx to match a nested .tsx file")
	}
	if matchesRule(fileEvent("Edit", "src/other/ui/Button.tsx"), rule) {
		t.Fatal("expected src/documentation/**/*.tsx NOT to match a file outside src/documentation/")
	}
}

func TestMatchesRule_Paths_NestedDirectory_Match(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:   []string{"Read"},
			Paths: []string{"src/**/nested/"},
		},
		Action: Actions{Block: boolPtr(true)},
	}
	if !matchesRule(fileEvent("Read", "src/pkg/nested/config.go"), rule) {
		t.Fatal("expected src/**/nested/ to match a file inside a nested/ directory")
	}
	if matchesRule(fileEvent("Read", "src/pkg/other/config.go"), rule) {
		t.Fatal("expected src/**/nested/ NOT to match a file outside nested/")
	}
}

func TestMatchesRule_Paths_Read_ToolAndGlob(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:   []string{"Read"},
			Paths: []string{".env"},
		},
		Action: Actions{Block: boolPtr(true)},
	}
	if !matchesRule(fileEvent("Read", "/home/user/project/.env"), rule) {
		t.Fatal("expected Read + .env target to match")
	}
}

// --- executeActions ---

func TestExecuteActions_Block_Enforce(t *testing.T) {
	rule := Rule{
		Action: Actions{Block: boolPtr(true), Message: "not allowed"},
	}
	resp, err := executeActions(bashEvent("cmd"), rule, ModeEnforce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Continue {
		t.Fatal("expected Continue=false in enforce mode")
	}
	if resp.Context != "not allowed" {
		t.Fatalf("expected block message in context, got %q", resp.Context)
	}
}

func TestExecuteActions_Block_DefaultMessage(t *testing.T) {
	rule := Rule{
		Action: Actions{Block: boolPtr(true)},
	}
	resp, _ := executeActions(bashEvent("cmd"), rule, ModeEnforce)
	if resp.Continue {
		t.Fatal("expected Continue=false")
	}
	if resp.Context != "Blocked by hook rule." {
		t.Fatalf("unexpected default message: %q", resp.Context)
	}
}

func TestExecuteActions_Block_WarnMode(t *testing.T) {
	rule := Rule{
		Action: Actions{Block: boolPtr(true), Message: "risky"},
	}
	resp, _ := executeActions(bashEvent("cmd"), rule, ModeWarn)
	if !resp.Continue {
		t.Fatal("expected Continue=true in warn mode")
	}
	if resp.Context == "" {
		t.Fatal("expected warning message in context")
	}
	if resp.Context[:9] != "[WARNING]" {
		t.Fatalf("expected [WARNING] prefix, got %q", resp.Context)
	}
}

func TestExecuteActions_InjectInline(t *testing.T) {
	rule := Rule{
		Action: Actions{InjectInline: "inline context text"},
	}
	resp, _ := executeActions(bashEvent("cmd"), rule, ModeEnforce)
	if !resp.Continue {
		t.Fatal("expected Continue=true for inject_inline")
	}
	if resp.Context != "inline context text" {
		t.Fatalf("expected inline context, got %q", resp.Context)
	}
}

func TestExecuteActions_InjectFile_Readable(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "standards.md")
	if err := os.WriteFile(f, []byte("# Standards\nFollow these."), 0o644); err != nil {
		t.Fatal(err)
	}
	rule := Rule{Action: Actions{Inject: f}}
	resp, _ := executeActions(bashEvent("cmd"), rule, ModeEnforce)
	if !resp.Continue {
		t.Fatal("expected Continue=true")
	}
	if resp.Context != "# Standards\nFollow these." {
		t.Fatalf("unexpected context: %q", resp.Context)
	}
}

func TestExecuteActions_InjectFile_MissingIsFailOpen(t *testing.T) {
	rule := Rule{Action: Actions{Inject: "/nonexistent/path/standards.md"}}
	resp, err := executeActions(bashEvent("cmd"), rule, ModeEnforce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Continue {
		t.Fatal("expected fail-open (Continue=true) when inject file is missing")
	}
}

func TestExecuteActions_NoAction_PassThrough(t *testing.T) {
	rule := Rule{Action: Actions{}}
	resp, _ := executeActions(bashEvent("cmd"), rule, ModeEnforce)
	if !resp.Continue {
		t.Fatal("expected Continue=true with no actions")
	}
	if resp.Context != "" {
		t.Fatalf("expected empty context, got %q", resp.Context)
	}
}

// --- Evaluate (multi-rule integration) ---

func TestEvaluate_BlockShortCircuits(t *testing.T) {
	cfg := HooksConfig{
		PreToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"Bash"}},
				Action: Actions{Block: boolPtr(true), Message: "blocked"},
			},
			{
				Match:  Matchers{Tools: []string{"Bash"}},
				Action: Actions{InjectInline: "should not appear"},
			},
		},
	}
	resp, err := Evaluate(bashEvent("cmd"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Continue {
		t.Fatal("expected block to short-circuit")
	}
	if resp.Context == "should not appear" {
		t.Fatal("second rule must not run after block")
	}
}

func TestEvaluate_MultiRuleInjectAccumulates(t *testing.T) {
	cfg := HooksConfig{
		PreToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"Bash"}},
				Action: Actions{InjectInline: "context A"},
			},
			{
				Match:  Matchers{Tools: []string{"Bash"}},
				Action: Actions{InjectInline: "context B"},
			},
		},
	}
	resp, _ := Evaluate(bashEvent("cmd"), cfg)
	if !resp.Continue {
		t.Fatal("expected Continue=true")
	}
	expected := "context A\n\ncontext B"
	if resp.Context != expected {
		t.Fatalf("expected %q, got %q", expected, resp.Context)
	}
}

func TestEvaluate_AuditMode_Skipped(t *testing.T) {
	cfg := HooksConfig{
		PreToolUse: []Rule{
			{
				Mode:   string(ModeAudit),
				Match:  Matchers{Tools: []string{"Bash"}},
				Action: Actions{Block: boolPtr(true)},
			},
		},
	}
	resp, _ := Evaluate(bashEvent("cmd"), cfg)
	if !resp.Continue {
		t.Fatal("audit mode must not block")
	}
}

func TestEvaluate_NoMatchingRule_Allow(t *testing.T) {
	cfg := HooksConfig{
		PreToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"Write"}},
				Action: Actions{Block: boolPtr(true)},
			},
		},
	}
	resp, _ := Evaluate(bashEvent("echo hi"), cfg)
	if !resp.Continue {
		t.Fatal("expected allow when no rules match")
	}
}

func TestEvaluate_EmptyRules_Allow(t *testing.T) {
	resp, _ := Evaluate(bashEvent("anything"), HooksConfig{})
	if !resp.Continue {
		t.Fatal("expected allow with no rules")
	}
	if resp.Context != "" {
		t.Fatalf("expected empty context, got %q", resp.Context)
	}
}

func TestEvaluate_WriteEvent_TargetBlock(t *testing.T) {
	cfg := HooksConfig{
		PreToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"Write"}, Paths: []string{".env"}},
				Action: Actions{Block: boolPtr(true), Message: "cannot write .env files"},
			},
		},
	}
	resp, _ := Evaluate(fileEvent("Write", "/project/.env"), cfg)
	if resp.Continue {
		t.Fatal("expected .env Write to be blocked")
	}
}

func TestEvaluate_EditEvent_InjectInline(t *testing.T) {
	cfg := HooksConfig{
		PreToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"Edit"}, Paths: []string{"*.py"}},
				Action: Actions{InjectInline: "follow PEP8"},
			},
		},
	}
	resp, _ := Evaluate(fileEvent("Edit", "/project/src/api.py"), cfg)
	if !resp.Continue {
		t.Fatal("expected Continue=true for inject")
	}
	if resp.Context != "follow PEP8" {
		t.Fatalf("unexpected context: %q", resp.Context)
	}
}

func TestEvaluate_ReadEvent_NoMatch(t *testing.T) {
	cfg := HooksConfig{
		PreToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"Write"}},
				Action: Actions{Block: boolPtr(true)},
			},
		},
	}
	resp, _ := Evaluate(fileEvent("Read", "/project/README.md"), cfg)
	if !resp.Continue {
		t.Fatal("Read event must not match a Write-only rule")
	}
}

func TestEvaluate_WebFetchEvent_Block(t *testing.T) {
	cfg := HooksConfig{
		PreToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"WebFetch"}},
				Action: Actions{Block: boolPtr(true), Message: "external fetch blocked"},
			},
		},
	}
	resp, _ := Evaluate(webFetchEvent("https://external.example.com"), cfg)
	if resp.Continue {
		t.Fatal("expected WebFetch to be blocked")
	}
	if resp.Context != "external fetch blocked" {
		t.Fatalf("unexpected message: %q", resp.Context)
	}
}

// --- PostToolUse tests ---

func TestEvaluate_PostToolUse_EventRouting(t *testing.T) {
	cfg := HooksConfig{
		PostToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"Write"}},
				Action: Actions{InjectInline: "post-tool-use context"},
			},
		},
	}
	resp, _ := Evaluate(postToolUseEvent("Write", "/tmp/file.txt", map[string]any{"success": true}), cfg)
	if !resp.Continue {
		t.Fatal("expected Continue=true")
	}
	if resp.Context != "post-tool-use context" {
		t.Fatalf("expected post-tool-use context, got %q", resp.Context)
	}
}

func TestEvaluate_PostToolUse_BlockAfterExecution(t *testing.T) {
	cfg := HooksConfig{
		PostToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"Write"}, Paths: []string{"*.py"}},
				Action: Actions{Block: boolPtr(true), Message: "Python file write detected"},
			},
		},
	}
	resp, _ := Evaluate(postToolUseEvent("Write", "/tmp/test.py", map[string]any{"success": true}), cfg)
	if resp.Continue {
		t.Fatal("expected block after Write of .py file")
	}
	if resp.Context != "Python file write detected" {
		t.Fatalf("unexpected message: %q", resp.Context)
	}
}

func TestEvaluate_PostToolUse_IgnoresUnrelatedPreToolUse(t *testing.T) {
	cfg := HooksConfig{
		PreToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"Write"}},
				Action: Actions{InjectInline: "should not appear"},
			},
		},
		PostToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"Write"}},
				Action: Actions{InjectInline: "post context"},
			},
		},
	}
	resp, _ := Evaluate(postToolUseEvent("Write", "/tmp/file.txt", map[string]any{"success": true}), cfg)
	if resp.Context != "post context" {
		t.Fatalf("PostToolUse events must not evaluate PreToolUse rules, got %q", resp.Context)
	}
}

func TestEvaluate_PostToolUse_Bash_Run(t *testing.T) {
	tmp := t.TempDir()
	script := filepath.Join(tmp, "check.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 'validation passed'"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := HooksConfig{
		PostToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"Bash"}},
				Action: Actions{Run: script},
			},
		},
	}
	resp, err := Evaluate(postToolUseEvent("Bash", "npm test", map[string]any{"exit_code": 0}), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Continue {
		t.Fatal("expected Continue=true")
	}
	if resp.Context != "validation passed" {
		t.Fatalf("expected validation output, got %q", resp.Context)
	}
}

func TestExecuteActions_RunInline_ExitZero_InjectsContext(t *testing.T) {
	rule := Rule{
		Match:  Matchers{},
		Action: Actions{RunInline: "echo 'inline ok'"},
	}
	resp, err := executeActions(bashEvent("anything"), rule, ModeEnforce)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Continue {
		t.Fatal("expected Continue=true")
	}
	if resp.Context != "inline ok" {
		t.Fatalf("expected 'inline ok', got %q", resp.Context)
	}
}

func TestExecuteActions_RunInline_NonZeroExit_Blocks(t *testing.T) {
	rule := Rule{
		Match:  Matchers{},
		Action: Actions{RunInline: "echo 'bad input' && exit 1"},
	}
	resp, err := executeActions(bashEvent("anything"), rule, ModeEnforce)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Continue {
		t.Fatal("expected block on non-zero exit")
	}
}

func TestExecuteActions_RunInline_WarnMode_Continues(t *testing.T) {
	rule := Rule{
		Match:  Matchers{},
		Action: Actions{RunInline: "exit 1"},
	}
	resp, err := executeActions(bashEvent("anything"), rule, ModeWarn)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Continue {
		t.Fatal("warn mode must not block")
	}
	if resp.Context == "" {
		t.Fatal("expected [WARNING] context in warn mode")
	}
}
