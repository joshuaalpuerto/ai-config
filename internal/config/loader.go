package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joshuaalpuerto/ai-config/schemas"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"gopkg.in/yaml.v3"
)

// validateAgainstSchema validates rawYAML against the embedded schema identified by schemaID.
func validateAgainstSchema(rawYAML []byte, yamlPath, schemaID string) error {
	var doc any
	if err := yaml.Unmarshal(rawYAML, &doc); err != nil {
		return fmt.Errorf("parsing %s: %w", yamlPath, err)
	}
	c, err := schemas.NewCompiler()
	if err != nil {
		return fmt.Errorf("initialising schema compiler: %w", err)
	}
	sch, err := c.Compile(schemaID)
	if err != nil {
		return fmt.Errorf("compiling schema %s: %w", schemaID, err)
	}
	if err := sch.Validate(doc); err != nil {
		return err
	}
	return nil
}

// LoadConfig reads and validates aicfg.yaml against the embedded aicfg.schema.json,
// then returns the parsed AicfgConfig.
func LoadConfig(path string) (AicfgConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AicfgConfig{}, fmt.Errorf("reading %s: %w", path, err)
	}
	if err := validateAgainstSchema(data, path, "aicfg.schema.json"); err != nil {
		return AicfgConfig{}, formatConfigError(err, path)
	}
	var cfg AicfgConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return AicfgConfig{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return cfg, nil
}

// docsURL is the reference for valid aicfg.yaml fields, included in config error messages.
const docsURL = "https://github.com/joshuaalpuerto/ai-config#project-structure"

// formatConfigError rewrites jsonschema validation errors into user-readable messages.
func formatConfigError(err error, filePath string) error {
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return fmt.Errorf("%s: %w", filePath, err)
	}
	msgs := collectMessages(ve, filePath)
	if len(msgs) == 0 {
		return fmt.Errorf("%s: %w", filePath, err)
	}
	return fmt.Errorf("%s", strings.Join(msgs, "\n"))
}

func collectMessages(ve *jsonschema.ValidationError, filePath string) []string {
	var out []string
	for _, cause := range ve.Causes {
		out = append(out, collectMessages(cause, filePath)...)
	}
	if len(out) > 0 {
		return out
	}

	loc := strings.Join(ve.InstanceLocation, ".")
	if loc == "" {
		loc = "(root)"
	}

	switch k := ve.ErrorKind.(type) {
	case *kind.AdditionalProperties:
		var msgs []string
		for _, prop := range k.Properties {
			msg := fmt.Sprintf("%s: unknown field %q at %s — see %s for valid fields", filePath, prop, loc, docsURL)
			msgs = append(msgs, msg)
		}
		return msgs
	case *kind.Required:
		var msgs []string
		for _, prop := range k.Missing {
			msgs = append(msgs, fmt.Sprintf("%s: %q is missing required field %q — see %s", filePath, loc, prop, docsURL))
		}
		return msgs
	case *kind.Type:
		return []string{fmt.Sprintf("%s: field at %q has wrong type — got %s, want %s", filePath, loc, k.Got, strings.Join(k.Want, " or "))}
	default:
		p := message.NewPrinter(language.English)
		return []string{fmt.Sprintf("%s: %s (at %s)", filePath, ve.ErrorKind.LocalizedString(p), loc)}
	}
}
