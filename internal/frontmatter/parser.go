package frontmatter

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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
