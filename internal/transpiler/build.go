package transpiler

import (
	"fmt"
	"strings"

	"github.com/joshuaalpuerto/ai-config/internal/config"
	"github.com/joshuaalpuerto/ai-config/internal/frontmatter"
)

// allFields is the ordered list of all known frontmatter field names.
// This order determines output field order, matching source declaration order
// in the Override struct definition.
var allFields = []string{
	"name",
	"description",
	"model",
	"context",
	"agent",
	"path",
	"applyTo",
	"argument-hint",
	"disable-model-invocation",
	"tools",
	"allowed-tools",
	"paths",
}

// sequenceFields is the set of fields that hold YAML sequences.
var sequenceFields = map[string]bool{
	"tools":         true,
	"allowed-tools": true,
	"paths":         true,
}

// BuildFrontmatter produces the output YAML frontmatter string for a file+platform+type combo.
// The returned string ends with a newline and does not include the "---" delimiters.
func BuildFrontmatter(
	fm frontmatter.Frontmatter,
	platform string,
	typeCfg config.TypeConfig,
	platCfg config.PlatformConfig,
	toolMap config.ToolMap,
) (string, error) {
	var buf strings.Builder

	// Build a set of drop fields for fast lookup.
	dropSet := make(map[string]bool, len(platCfg.DropFields))
	for _, f := range platCfg.DropFields {
		dropSet[f] = true
	}

	// Build set of extra_fields keys to skip during regular field iteration.
	extraKeySet := make(map[string]bool, len(typeCfg.ExtraFields))
	for k := range typeCfg.ExtraFields {
		extraKeySet[k] = true
	}

	// 1. Inject extra_fields first (sorted to match yq's alphabetical key order).
	for _, k := range extraFieldsSortedKeys(typeCfg.ExtraFields) {
		buf.WriteString(k + ": " + typeCfg.ExtraFields[k] + "\n")
	}

	// 2. Iterate all known fields in struct definition order.
	for _, field := range allFields {
		// Skip fields already emitted as extra_fields.
		if extraKeySet[field] {
			continue
		}

		if sequenceFields[field] {
			// Sequence field handling.
			var items []string
			switch field {
			case "tools":
				items = ResolveTools(fm, platform, "tools", toolMap)
			case "allowed-tools":
				items = ResolveTools(fm, platform, "allowed-tools", toolMap)
			case "paths":
				if dropSet["paths"] {
					// Check if override provides a value.
					if override, ok := fm.Overrides[platform]; !ok || len(override.Paths) == 0 {
						continue
					}
				}
				items = resolvePaths(fm, platform)
			}
			if len(items) == 0 {
				continue
			}
			buf.WriteString(field + ":\n")
			for _, item := range items {
				buf.WriteString("  - " + yamlSeqItem(item) + "\n")
			}
			continue
		}

		if field == "disable-model-invocation" {
			if dropSet[field] {
				if override, ok := fm.Overrides[platform]; !ok || !override.DisableModelInvocation {
					continue
				}
			}
			if resolveDisableModelInvocation(fm, platform) {
				buf.WriteString("disable-model-invocation: true\n")
			}
			continue
		}

		// Scalar field.
		if dropSet[field] {
			// Check if a platform override provides a value despite the drop.
			override, hasOverride := fm.Overrides[platform]
			if !hasOverride {
				continue
			}
			var overrideVal string
			switch field {
			case "name":
				overrideVal = override.Name
			case "description":
				overrideVal = override.Description
			case "model":
				overrideVal = override.Model
			case "context":
				overrideVal = override.Context
			case "agent":
				overrideVal = override.Agent
			case "path":
				overrideVal = override.Path
			case "applyTo":
				overrideVal = override.ApplyTo
			case "argument-hint":
				overrideVal = override.ArgumentHint
			}
			if overrideVal == "" {
				continue
			}
		}

		val := ResolveField(fm, platform, field)
		if val == "" {
			continue
		}
		buf.WriteString(field + ": " + val + "\n")
	}

	result := buf.String()
	if result == "" {
		return "", fmt.Errorf("BuildFrontmatter produced empty output for platform %q", platform)
	}
	return result, nil
}

// yamlSeqItem returns the YAML representation of a sequence item string,
// adding single quotes when the value would be ambiguous as a plain YAML scalar.
// This matches yq v4's quoting behavior for sequence items.
func yamlSeqItem(s string) string {
	if s == "" {
		return "''"
	}
	// Plain scalars in YAML cannot start with these indicator characters.
	const indicators = "-?:,[]{}#&*!|>'\"%@`"
	if strings.ContainsRune(indicators, rune(s[0])) {
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
	// Could be misinterpreted as null/bool/specific YAML keywords.
	switch strings.ToLower(s) {
	case "null", "true", "false", "~", "yes", "no", "on", "off":
		return "'" + s + "'"
	}
	// Check for embedded colon-space or hash-space which would break plain scalars.
	if strings.Contains(s, ": ") || strings.Contains(s, " #") {
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
	return s
}
