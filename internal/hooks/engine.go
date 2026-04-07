package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Evaluate runs all rules against event sequentially and returns the final Response.
// Block in Enforce mode short-circuits immediately. Injections from multiple rules
// accumulate separated by "\n\n". Warn mode converts blocks to injected warnings.
// Audit mode skips all blocks and injections.
func Evaluate(event Event, cfg HooksConfig) (Response, error) {
	var accumulated string

	for _, rule := range cfg.PreToolUse {
		if !matchesRule(event, rule) {
			continue
		}

		mode := PolicyMode(rule.Mode)
		if mode == "" {
			mode = ModeEnforce
		}

		if mode == ModeAudit {
			continue
		}

		resp, err := executeActions(event, rule, mode)
		if err != nil {
			return Response{Continue: true}, err
		}

		if !resp.Continue && mode == ModeEnforce {
			return resp, nil
		}

		if resp.Context != "" {
			if accumulated != "" {
				accumulated += "\n\n"
			}
			accumulated += resp.Context
		}
	}

	return Response{Continue: true, Context: accumulated}, nil
}

// matchesRule returns true only when ALL configured matchers pass.
// An unconfigured matcher always passes (opt-in narrowing).
func matchesRule(event Event, rule Rule) bool {
	m := rule.Match

	if len(m.Tools) > 0 && !toolMatches(event.ToolName, m.Tools) {
		return false
	}

	if m.CommandMatch != "" {
		cmd, err := extractCommand(event.ToolInput)
		if err != nil || cmd == "" {
			return false
		}
		matched, err := regexp.MatchString(m.CommandMatch, cmd)
		if err != nil || !matched {
			return false
		}
	}

	filePath, _ := extractFilePath(event.ToolInput)

	if len(m.Extensions) > 0 {
		if filePath == "" || !extensionMatches(filePath, m.Extensions) {
			return false
		}
	}

	if len(m.Directories) > 0 {
		if filePath == "" || !directoryMatches(filePath, m.Directories) {
			return false
		}
	}

	return true
}

func executeActions(event Event, rule Rule, mode PolicyMode) (Response, error) {
	a := rule.Action

	if a.Block != nil && *a.Block {
		reason := a.Message
		if reason == "" {
			reason = "Blocked by hook rule."
		}
		if mode == ModeWarn {
			return Response{Continue: true, Context: fmt.Sprintf("[WARNING] A hook rule would block this action: %s", reason)}, nil
		}
		return Response{Continue: false, Context: reason}, nil
	}

	if a.InjectInline != "" {
		return Response{Continue: true, Context: a.InjectInline}, nil
	}

	if a.Inject != "" {
		data, err := os.ReadFile(a.Inject)
		if err != nil {
			// Context file unreadable: fail-open. The inject path depends on the
			// deployed .claude/context/ directory which may not exist on first run.
			return Response{Continue: true}, nil
		}
		return Response{Continue: true, Context: string(data)}, nil
	}

	if a.Run != "" {
		return executeRun(event, a.Run, mode)
	}

	return Response{Continue: true}, nil
}

// executeRun invokes a validator script with the event JSON on stdin.
// Exit 0: stdout is injected as context. Non-zero exit: blocks (or warns).
func executeRun(event Event, scriptPath string, mode PolicyMode) (Response, error) {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return Response{Continue: true}, fmt.Errorf("marshalling event for run: %w", err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(scriptPath) // #nosec G204 — path comes from operator-controlled hooks.yaml
	cmd.Stdin = bytes.NewReader(eventJSON)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if runErr := cmd.Run(); runErr != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = fmt.Sprintf("Validator %q failed.", scriptPath)
		}
		if mode == ModeWarn {
			return Response{Continue: true, Context: fmt.Sprintf("[WARNING] %s", msg)}, nil
		}
		return Response{Continue: false, Context: msg}, nil
	}

	return Response{Continue: true, Context: strings.TrimSpace(stdout.String())}, nil
}

func toolMatches(toolName string, tools []string) bool {
	for _, t := range tools {
		if strings.EqualFold(t, toolName) {
			return true
		}
	}
	return false
}

func extensionMatches(filePath string, extensions []string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, e := range extensions {
		if strings.ToLower(e) == ext {
			return true
		}
	}
	return false
}

// directoryMatches returns true if filePath is under any of the directory glob patterns.
// Bare directory entries without wildcards are treated as prefix matches.
func directoryMatches(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		if !strings.ContainsAny(pattern, "*?[") {
			prefix := strings.TrimSuffix(pattern, string(os.PathSeparator))
			if strings.HasPrefix(filePath, prefix+string(os.PathSeparator)) || filePath == prefix {
				return true
			}
			continue
		}
		matched, err := filepath.Match(pattern, filePath)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// extractCommand reads the "command" field from Bash tool_input JSON.
// Returns an error so callers can distinguish missing input from empty command.
func extractCommand(raw json.RawMessage) (string, error) {
	if raw == nil {
		return "", fmt.Errorf("tool_input is nil")
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", fmt.Errorf("parsing tool_input: %w", err)
	}
	v, _ := m["command"].(string)
	return v, nil
}

// extractFilePath reads "file_path" or "path" from tool_input JSON.
func extractFilePath(raw json.RawMessage) (string, error) {
	if raw == nil {
		return "", fmt.Errorf("tool_input is nil")
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", fmt.Errorf("parsing tool_input: %w", err)
	}
	for _, key := range []string{"file_path", "path"} {
		if v, ok := m[key].(string); ok && v != "" {
			return v, nil
		}
	}
	return "", nil
}
