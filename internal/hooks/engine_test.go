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

func TestMatchesRule_ExtensionFilter_Match(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:      []string{"Write"},
			Extensions: []string{".py"},
		},
		Action: Actions{InjectInline: "python standards"},
	}
	if !matchesRule(fileEvent("Write", "/home/user/project/routes.py"), rule) {
		t.Fatal("expected .py extension to match")
	}
}

func TestMatchesRule_ExtensionFilter_Miss(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:      []string{"Write"},
			Extensions: []string{".py"},
		},
		Action: Actions{InjectInline: "python standards"},
	}
	if matchesRule(fileEvent("Write", "/home/user/project/routes.ts"), rule) {
		t.Fatal("expected .ts extension NOT to match .py filter")
	}
}

func TestMatchesRule_ExtensionFilter_CaseInsensitive(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:      []string{"Edit"},
			Extensions: []string{".PY"},
		},
		Action: Actions{InjectInline: "python standards"},
	}
	if !matchesRule(fileEvent("Edit", "/home/user/project/routes.py"), rule) {
		t.Fatal("expected case-insensitive extension match")
	}
}

func TestMatchesRule_DirectoryFilter_PrefixMatch(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:       []string{"Edit"},
			Directories: []string{"/home/user/project/src"},
		},
		Action: Actions{InjectInline: "src standards"},
	}
	if !matchesRule(fileEvent("Edit", "/home/user/project/src/api/routes.py"), rule) {
		t.Fatal("expected directory prefix to match")
	}
}

func TestMatchesRule_DirectoryFilter_Miss(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:       []string{"Edit"},
			Directories: []string{"/home/user/project/src"},
		},
		Action: Actions{InjectInline: "src standards"},
	}
	if matchesRule(fileEvent("Edit", "/home/user/project/tests/test_routes.py"), rule) {
		t.Fatal("expected directory filter NOT to match /tests/")
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

func TestMatchesRule_Read_ToolAndExtension(t *testing.T) {
	rule := Rule{
		Match: Matchers{
			Tools:      []string{"Read"},
			Extensions: []string{".env"},
		},
		Action: Actions{Block: boolPtr(true)},
	}
	if !matchesRule(fileEvent("Read", "/home/user/project/.env"), rule) {
		t.Fatal("expected Read + .env to match")
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

func TestEvaluate_WriteEvent_ExtensionBlock(t *testing.T) {
	cfg := HooksConfig{
		PreToolUse: []Rule{
			{
				Match:  Matchers{Tools: []string{"Write"}, Extensions: []string{".env"}},
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
				Match:  Matchers{Tools: []string{"Edit"}, Extensions: []string{".py"}},
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
