package hooks

import "encoding/json"

type EventType string

const EventPreToolUse EventType = "PreToolUse"

// Event is the JSON payload Claude Code sends via stdin on each tool use.
type Event struct {
	HookEventName EventType       `json:"hook_event_name"`
	SessionID     string          `json:"session_id"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input,omitempty"`
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
	Version  string    `yaml:"version"`
	Rules    []Rule    `yaml:"rules"`
	Settings *Settings `yaml:"settings,omitempty"`
}

// Rule is one policy entry in hooks.yaml.
type Rule struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Mode        string   `yaml:"mode,omitempty"`
	Matchers    Matchers `yaml:"matchers"`
	Actions     Actions  `yaml:"actions"`
}

// Matchers describes when a rule applies.
type Matchers struct {
	Tools        []string `yaml:"tools,omitempty"`
	CommandMatch string   `yaml:"command_match,omitempty"`
	Extensions   []string `yaml:"extensions,omitempty"`
	Directories  []string `yaml:"directories,omitempty"`
}

// Actions describes what happens when a rule fires.
type Actions struct {
	Block        *bool  `yaml:"block,omitempty"`
	BlockMessage string `yaml:"block_message,omitempty"`
	Inject       string `yaml:"inject,omitempty"`
	InjectInline string `yaml:"inject_inline,omitempty"`
}

// Settings holds global engine options.
type Settings struct {
	FailOpen *bool `yaml:"fail_open,omitempty"`
}
