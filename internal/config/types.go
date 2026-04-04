package config

// TypeConfig represents one type entry under a platform in platforms.yaml.
type TypeConfig struct {
	Path        string            `yaml:"path"`
	Suffix      string            `yaml:"suffix"`
	ExtraFields map[string]string `yaml:"extra_fields,omitempty"`
}

// PlatformConfig represents one platform block in platforms.yaml.
type PlatformConfig struct {
	Target     string                `yaml:"target"`
	Types      map[string]TypeConfig `yaml:"types"`
	DropFields []string              `yaml:"drop_fields"`
}

// PlatformsConfig is the top-level structure of platforms.yaml.
type PlatformsConfig map[string]PlatformConfig

// ToolMap is the top-level structure of the tool_map section.
// Shape: map[platform]map[canonicalTool]mappedTool
type ToolMap map[string]map[string]string

// AicfgConfig is the top-level structure of aicfg.yaml.
type AicfgConfig struct {
	Platforms PlatformsConfig `yaml:"platforms"`
	ToolMap   ToolMap         `yaml:"tool_map"`
}
