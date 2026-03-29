package frontmatter

// Override mirrors Frontmatter for a specific platform, excluding Overrides itself.
type Override struct {
	Name                   string   `yaml:"name,omitempty"`
	Description            string   `yaml:"description,omitempty"`
	Model                  string   `yaml:"model,omitempty"`
	Context                string   `yaml:"context,omitempty"`
	Agent                  string   `yaml:"agent,omitempty"`
	Path                   string   `yaml:"path,omitempty"`
	ApplyTo                string   `yaml:"applyTo,omitempty"`
	ArgumentHint           string   `yaml:"argument-hint,omitempty"`
	DisableModelInvocation bool     `yaml:"disable-model-invocation,omitempty"`
	Tools                  []string `yaml:"tools,omitempty"`
	AllowedTools           []string `yaml:"allowed-tools,omitempty"`
	Paths                  []string `yaml:"paths,omitempty"`
}

// Frontmatter is the typed schema for all known source frontmatter fields.
type Frontmatter struct {
	Override  `yaml:",inline"`
	Overrides map[string]Override `yaml:"overrides,omitempty"`
}
