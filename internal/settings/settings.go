package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// MergeClaudeSettings reads .claude/settings.json (if present), injects the
// PreToolUse aicfg hooks entry, and writes back atomically.
// All existing top-level keys are preserved. Idempotent on repeated runs.
// platform is the value passed to --platform (e.g. "claude").
func MergeClaudeSettings(claudeDir, platform string) error {
	path := filepath.Join(claudeDir, "settings.json")

	existing := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		if jsonErr := json.Unmarshal(data, &existing); jsonErr != nil {
			return fmt.Errorf("parsing %s: %w", path, jsonErr)
		}
	}

	hooksMap := resolveHooksMap(existing)
	hooksMap["PreToolUse"] = buildPreToolUseEntry(platform)
	existing["hooks"] = hooksMap

	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("serialising settings.json: %w", err)
	}

	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", claudeDir, err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	return os.Rename(tmp, path)
}

func resolveHooksMap(root map[string]any) map[string]any {
	if h, ok := root["hooks"].(map[string]any); ok {
		return h
	}
	return map[string]any{}
}

func buildPreToolUseEntry(platform string) []any {
	return []any{
		map[string]any{
			"hooks": []any{
				map[string]any{
					"type":    "command",
					"command": fmt.Sprintf("aicfg hooks --platform %s", platform),
				},
			},
		},
	}
}
