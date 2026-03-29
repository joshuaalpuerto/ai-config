package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadPlatforms reads and validates platforms.yaml.
func LoadPlatforms(path string) (PlatformsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var cfg PlatformsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	for platform, pc := range cfg {
		for typeName, tc := range pc.Types {
			if tc.Path == "" {
				return nil, fmt.Errorf("%s: platform %q type %q missing required field \"path\"", path, platform, typeName)
			}
			if tc.Suffix == "" {
				return nil, fmt.Errorf("%s: platform %q type %q missing required field \"suffix\"", path, platform, typeName)
			}
		}
	}
	return cfg, nil
}

// LoadTargets reads targets.yaml.
func LoadTargets(path string) (TargetsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var cfg TargetsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return cfg, nil
}

// LoadToolMap reads tool-map.yaml.
func LoadToolMap(path string) (ToolMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var tm ToolMap
	if err := yaml.Unmarshal(data, &tm); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return tm, nil
}
