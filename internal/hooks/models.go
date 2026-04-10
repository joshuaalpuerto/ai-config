package hooks

import "encoding/json"

type EventType string

const (
	EventPreToolUse  EventType = "PreToolUse"
	EventPostToolUse EventType = "PostToolUse"
)

// Event is the JSON payload Claude Code sends via stdin on each tool use event.
type Event struct {
	HookEventName EventType       `json:"hook_event_name"`
	SessionID     string          `json:"session_id"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input,omitempty"`
	ToolResponse  json.RawMessage `json:"tool_response,omitempty"`
	ToolUseID     string          `json:"tool_use_id,omitempty"`
	CWD           string          `json:"cwd,omitempty"`
}

// Response is written as JSON to stdout on allow; Context is injected into the prompt.
type Response struct {
	Continue bool   `json:"continue"`
	Context  string `json:"context,omitempty"`
}

type PolicyMode string

const (
	ModeEnforce PolicyMode = "enforce"
	ModeWarn    PolicyMode = "warn"
	ModeAudit   PolicyMode = "audit"
)

// HooksConfig is the top-level structure of the deployed hooks.yaml.
type HooksConfig struct {
	Version     string    `yaml:"version"`
	PreToolUse  []Rule    `yaml:"PreToolUse"`
	PostToolUse []Rule    `yaml:"PostToolUse"`
	Settings    *Settings `yaml:"settings,omitempty"`
}

// Rule is one policy entry under a hook event (PreToolUse or PostToolUse) in hooks.yaml.
type Rule struct {
	Mode   string   `yaml:"mode,omitempty"`
	Match  Matchers `yaml:"match"`
	Action Actions  `yaml:"action"`
}

// Matchers describes when a rule applies.
type Matchers struct {
	Tools        []string `yaml:"tools,omitempty"`
	CommandMatch string   `yaml:"command_match,omitempty"`
	// Paths is a list of glob patterns matched against the file path.
	// Patterns without a path separator (e.g. "*.go") match the filename only.
	// Patterns ending with "/" (e.g. "src/api/") match any file inside that directory.
	// All other patterns (e.g. "src/**/*.ts") are matched against the full path.
	// Double-star "**" matches zero or more path segments.
	Paths []string `yaml:"paths,omitempty"`
}

// Actions describes what happens when a rule fires.
type Actions struct {
	Block        *bool  `yaml:"block,omitempty"`
	Message      string `yaml:"message,omitempty"`
	Inject       string `yaml:"inject,omitempty"`
	InjectInline string `yaml:"inject_inline,omitempty"`
	Run          string `yaml:"run,omitempty"`
}

// Settings holds global engine options.
type Settings struct {
	FailOpen *bool `yaml:"fail_open,omitempty"`
}
