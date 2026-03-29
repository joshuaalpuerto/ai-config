package frontmatter

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// ParseFile splits a Markdown file at the YAML frontmatter delimiters.
// Returns: parsed Frontmatter, raw body string, and whether frontmatter was present.
// If no valid frontmatter is found, returns zero Frontmatter, full file content as body, false.
func ParseFile(path string) (fm Frontmatter, body string, hasFrontmatter bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fm, "", false, fmt.Errorf("reading %s: %w", path, err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	// First line must be exactly "---"
	if !scanner.Scan() {
		return fm, string(data), false, nil
	}
	firstLine := scanner.Text()
	if firstLine != "---" {
		return fm, string(data), false, nil
	}

	// Collect frontmatter lines until second "---"
	var fmLines []string
	foundClose := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			foundClose = true
			break
		}
		fmLines = append(fmLines, line)
	}

	if !foundClose {
		// No closing delimiter — treat as passthrough
		return fm, string(data), false, nil
	}

	// Parse YAML frontmatter
	fmYAML := strings.Join(fmLines, "\n")
	if err := yaml.Unmarshal([]byte(fmYAML), &fm); err != nil {
		return fm, "", false, fmt.Errorf("parsing frontmatter in %s: %w", path, err)
	}

	// Collect remaining lines as body, skipping bare "---" lines.
	// The shell's extract_body uses awk that skips every line matching /^---$/,
	// which strips horizontal rules from the body as a side effect.
	var bodyLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			continue
		}
		bodyLines = append(bodyLines, line)
	}
	// Trim trailing newlines to match shell command-substitution behavior.
	body = strings.TrimRight(strings.Join(bodyLines, "\n"), "\n")

	return fm, body, true, nil
}

// ParseFileValidated is like ParseFile but validates the extracted frontmatter
// YAML block against schemas/<typeName>.schema.json before unmarshalling.
// schemasDir must be the absolute path to the schemas/ directory at the repo root.
// Files with no frontmatter skip schema validation and are passed through unchanged.
func ParseFileValidated(path, typeName, schemasDir string) (Frontmatter, string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Frontmatter{}, "", false, fmt.Errorf("reading %s: %w", path, err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	if !scanner.Scan() {
		return Frontmatter{}, string(data), false, nil
	}
	if scanner.Text() != "---" {
		return Frontmatter{}, string(data), false, nil
	}

	var fmLines []string
	foundClose := false
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			foundClose = true
			break
		}
		fmLines = append(fmLines, line)
	}
	if !foundClose {
		return Frontmatter{}, string(data), false, nil
	}

	fmYAML := strings.Join(fmLines, "\n")
	schemaFile := filepath.Join(schemasDir, typeName+".schema.json")
	if err := validateFrontmatter([]byte(fmYAML), path, schemaFile); err != nil {
		return Frontmatter{}, "", false, err
	}

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(fmYAML), &fm); err != nil {
		return Frontmatter{}, "", false, fmt.Errorf("parsing frontmatter in %s: %w", path, err)
	}

	var bodyLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			continue
		}
		bodyLines = append(bodyLines, line)
	}
	body := strings.TrimRight(strings.Join(bodyLines, "\n"), "\n")

	return fm, body, true, nil
}

// validateFrontmatter validates rawFMYAML against the JSON Schema at schemaFile.
func validateFrontmatter(rawFMYAML []byte, srcFile, schemaFile string) error {
	var doc any
	if err := yaml.Unmarshal(rawFMYAML, &doc); err != nil {
		return fmt.Errorf("parsing frontmatter in %s: %w", srcFile, err)
	}
	c := jsonschema.NewCompiler()
	sch, err := c.Compile(schemaFile)
	if err != nil {
		return fmt.Errorf("compiling schema %s: %w", schemaFile, err)
	}
	if err := sch.Validate(doc); err != nil {
		return fmt.Errorf("%s: frontmatter schema violation: %w", srcFile, err)
	}
	return nil
}
