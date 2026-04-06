package hooks

import (
	"encoding/json"
	"fmt"
)

// Platform identifies which AI assistant runtime is consuming hook output.
// Each platform has a distinct JSON wire format for hook responses.
type Platform string

const (
	PlatformClaude = Platform("claude")
	PlatformGitHub = Platform("github")
	PlatformGemini = Platform("gemini")
)

// claudePreToolUseOutput is the wire format Claude Code expects on stdout
// when a PreToolUse hook exits 0 with context to inject.
type claudePreToolUseOutput struct {
	HookSpecificOutput claudeHookSpecificOutput `json:"hookSpecificOutput"`
}

type claudeHookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// FormatOutput converts an engine Response into platform-specific wire format.
// It returns the bytes to write to stdout, a message to write to stderr,
// and the process exit code. Non-zero exitCode with a non-empty stderr means
// something stopped execution — whether a rule block or a broken configuration.
func FormatOutput(platform Platform, event Event, resp Response) (stdout []byte, stderr string, exitCode int) {
	switch platform {
	case PlatformClaude:
		return formatClaude(event, resp)
	case PlatformGitHub, PlatformGemini:
		return nil, fmt.Sprintf("platform %q hook output format not yet supported", platform), 2
	default:
		return nil, fmt.Sprintf("unsupported platform %q", platform), 2
	}
}

func formatClaude(event Event, resp Response) (stdout []byte, stderr string, exitCode int) {
	if !resp.Continue {
		return nil, resp.Context, 2
	}

	if resp.Context == "" {
		return nil, "", 0
	}

	out, err := json.Marshal(claudePreToolUseOutput{
		HookSpecificOutput: claudeHookSpecificOutput{
			HookEventName:     string(event.HookEventName),
			AdditionalContext: resp.Context,
		},
	})
	if err != nil {
		return nil, fmt.Sprintf("serialising claude hook output: %v", err), 2
	}

	return out, "", 0
}
