package transpiler

import (
	"sort"
	"strings"

	"github.com/joshuaalpuerto/ai-config/internal/config"
	"github.com/joshuaalpuerto/ai-config/internal/frontmatter"
)

// ResolveField returns the value of a named scalar field for a platform,
// checking the platform override first, then the top-level value.
func ResolveField(fm frontmatter.Frontmatter, platform, field string) string {
	override, hasOverride := fm.Overrides[platform]

	getOverride := func() string {
		if !hasOverride {
			return ""
		}
		switch field {
		case "name":
			return override.Name
		case "description":
			return override.Description
		case "model":
			return override.Model
		case "context":
			return override.Context
		case "agent":
			return override.Agent
		case "path":
			return override.Path
		case "applyTo":
			return override.ApplyTo
		case "argument-hint":
			return override.ArgumentHint
		}
		return ""
	}

	getTop := func() string {
		switch field {
		case "name":
			return fm.Name
		case "description":
			return fm.Description
		case "model":
			return fm.Model
		case "context":
			return fm.Context
		case "agent":
			return fm.Agent
		case "path":
			return fm.Path
		case "applyTo":
			return fm.ApplyTo
		case "argument-hint":
			return fm.ArgumentHint
		}
		return ""
	}

	if v := getOverride(); v != "" {
		return v
	}
	return getTop()
}

// ResolveTools returns the resolved, deduplicated tool list for a platform+field combo.
// Handles override priority, canonical->platform mapping, and scope syntax: Bash(npm test).
func ResolveTools(
	fm frontmatter.Frontmatter,
	platform, field string,
	toolMap config.ToolMap,
) []string {
	if override, ok := fm.Overrides[platform]; ok {
		var overrideTools []string
		switch field {
		case "tools":
			overrideTools = override.Tools
		case "allowed-tools":
			overrideTools = override.AllowedTools
		}
		if len(overrideTools) > 0 {
			return overrideTools
		}
	}

	var canonical []string
	switch field {
	case "tools":
		canonical = fm.Tools
	case "allowed-tools":
		canonical = fm.AllowedTools
	}
	if len(canonical) == 0 {
		return nil
	}

	if platform == "claude" {
		return canonical
	}

	platformMap := toolMap[platform]
	seen := make(map[string]bool)
	result := make([]string, 0, len(canonical))

	for _, tool := range canonical {
		base, scope, hasParen := strings.Cut(tool, "(")
		var mapped string
		if platformMap != nil {
			mapped = platformMap[base]
		}
		if mapped == "" {
			continue
		}
		var fullTool string
		if hasParen {
			fullTool = mapped + "(" + scope
		} else {
			fullTool = mapped
		}
		if !seen[fullTool] {
			seen[fullTool] = true
			result = append(result, fullTool)
		}
	}
	return result
}

// resolveDisableModelInvocation returns the bool value for disable-model-invocation.
func resolveDisableModelInvocation(fm frontmatter.Frontmatter, platform string) bool {
	if override, ok := fm.Overrides[platform]; ok && override.DisableModelInvocation {
		return true
	}
	return fm.DisableModelInvocation
}

// resolvePaths returns the paths list, checking override first.
func resolvePaths(fm frontmatter.Frontmatter, platform string) []string {
	if override, ok := fm.Overrides[platform]; ok && len(override.Paths) > 0 {
		return override.Paths
	}
	return fm.Paths
}

// extraFieldsSortedKeys returns the keys of the extra_fields map in sorted order.
func extraFieldsSortedKeys(extra map[string]string) []string {
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
