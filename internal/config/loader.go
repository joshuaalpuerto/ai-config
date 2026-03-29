package config

import (
	"fmt"
	"os"
	"path/filepath"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// validateAgainstSchema validates rawYAML against the JSON Schema at schemaPath.
// schemaPath is resolved relative to the YAML file's directory so $ref and $id work correctly.
func validateAgainstSchema(rawYAML []byte, yamlPath, schemaPath string) error {
	var doc any
	if err := yaml.Unmarshal(rawYAML, &doc); err != nil {
		return fmt.Errorf("parsing %s: %w", yamlPath, err)
	}

	absSchema := schemaPath
	if !filepath.IsAbs(absSchema) {
		absSchema = filepath.Join(filepath.Dir(yamlPath), absSchema)
	}
	c := jsonschema.NewCompiler()
	sch, err := c.Compile(absSchema)
	if err != nil {
		return fmt.Errorf("compiling schema %s: %w", absSchema, err)
	}

	if err := sch.Validate(doc); err != nil {
		return fmt.Errorf("%s: %w", yamlPath, err)
	}
	return nil
}

// LoadPlatforms reads and validates platforms.yaml.
// Validation is performed against platforms.schema.json (same directory) before unmarshalling.
func LoadPlatforms(path string) (PlatformsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	schemaPath := filepath.Join(filepath.Dir(path), "..", "schemas", "platforms.schema.json")
	if err := validateAgainstSchema(data, path, schemaPath); err != nil {
		return nil, err
	}
	var cfg PlatformsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return cfg, nil
}

// LoadToolMap reads and validates tool-map.yaml.
// Validation is performed against tool-map.schema.json (same directory) before unmarshalling.
func LoadToolMap(path string) (ToolMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	schemaPath := filepath.Join(filepath.Dir(path), "..", "schemas", "tool-map.schema.json")
	if err := validateAgainstSchema(data, path, schemaPath); err != nil {
		return nil, err
	}
	var tm ToolMap
	if err := yaml.Unmarshal(data, &tm); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return tm, nil
}
