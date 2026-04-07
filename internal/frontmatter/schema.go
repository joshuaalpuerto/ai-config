package frontmatter

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// StringOrSlice is a []string that also accepts a single space-separated
// string during YAML unmarshalling, e.g. "allowed-tools: Read Grep Glob".
type StringOrSlice []string

func (s *StringOrSlice) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		parts := strings.Fields(value.Value)
		*s = parts
		return nil
	case yaml.SequenceNode:
		var items []string
		if err := value.Decode(&items); err != nil {
			return err
		}
		*s = items
		return nil
	default:
		return fmt.Errorf("expected string or list, got %v", value.Kind)
	}
}

// Override mirrors Frontmatter for a specific platform, excluding Overrides itself.
type Override struct {
	Name                   string        `yaml:"name,omitempty"`
	Description            string        `yaml:"description,omitempty"`
	Model                  string        `yaml:"model,omitempty"`
	Context                string        `yaml:"context,omitempty"`
	Agent                  string        `yaml:"agent,omitempty"`
	Path                   string        `yaml:"path,omitempty"`
	ApplyTo                string        `yaml:"applyTo,omitempty"`
	ArgumentHint           string        `yaml:"argument-hint,omitempty"`
	DisableModelInvocation bool          `yaml:"disable-model-invocation,omitempty"`
	UserInvocable          *bool         `yaml:"user-invocable,omitempty"`
	Tools                  StringOrSlice `yaml:"tools,omitempty"`
	AllowedTools           StringOrSlice `yaml:"allowed-tools,omitempty"`
	Paths                  []string      `yaml:"paths,omitempty"`
}

// Frontmatter is the typed schema for all known source frontmatter fields.
type Frontmatter struct {
	Override  `yaml:",inline"`
	Overrides map[string]Override `yaml:"overrides,omitempty"`
}
