package config

// TypeConfig represents one type entry under a platform in platforms.yaml.
type TypeConfig struct {
	Path        string            `yaml:"path"`
	Suffix      string            `yaml:"suffix"`
	ExtraFields map[string]string `yaml:"extra_fields,omitempty"`
}

// HooksConfig declares where to deploy hooks artifacts for a platform.
type HooksConfig struct {
	SrcHooksFile    string `yaml:"src_hooks_file"`
	HooksFile       string `yaml:"hooks_file"`
	ContextDir      string `yaml:"context_dir"`
	SrcContextDir   string `yaml:"src_context_dir,omitempty"`
}

// PlatformConfig represents one platform block in platforms.yaml.
type PlatformConfig struct {
	Target     string                `yaml:"target"`
	Types      map[string]TypeConfig `yaml:"types"`
	DropFields []string              `yaml:"drop_fields"`
	Hooks      *HooksConfig          `yaml:"hooks,omitempty"`
}

// PlatformsConfig is the top-level structure of platforms.yaml.
type PlatformsConfig map[string]PlatformConfig

// ToolMap is the top-level structure of the tool_map section.
// Shape: map[platform]map[canonicalTool]mappedTool
type ToolMap map[string]map[string]string

// AicfgConfig is the top-level structure of aicfg.yaml.
type AicfgConfig struct {
	SrcDir    string          `yaml:"src_dir"`
	Platforms PlatformsConfig `yaml:"platforms"`
	ToolMap   ToolMap         `yaml:"tool_map"`
}
