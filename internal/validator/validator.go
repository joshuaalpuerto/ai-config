package validator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joshuacalpuerto/ai-config/internal/config"
	"github.com/joshuacalpuerto/ai-config/internal/frontmatter"
	"github.com/joshuacalpuerto/ai-config/schemas"
)

// Result holds the error and warning counts from a validation run.
type Result struct {
	Errors   int
	Warnings int
}

// ValidateAll validates all source files and prints findings to w.
func ValidateAll(
	srcDir string,
	platforms config.PlatformsConfig,
	toolMap config.ToolMap,
	w io.Writer,
) Result {
	var result Result

	validPlatforms := make(map[string]bool, len(platforms))
	platformList := make([]string, 0, len(platforms))
	for p := range platforms {
		validPlatforms[p] = true
		platformList = append(platformList, p)
	}
	sort.Strings(platformList)

	validTools := make(map[string]bool)
	if claudeMap, ok := toolMap["claude"]; ok {
		for tool := range claudeMap {
			validTools[tool] = true
		}
	}

	typeSet := make(map[string]bool)
	for _, pc := range platforms {
		for typeName := range pc.Types {
			typeSet[typeName] = true
		}
	}
	typeNames := make([]string, 0, len(typeSet))
	for t := range typeSet {
		typeNames = append(typeNames, t)
	}
	sort.Strings(typeNames)

	for _, typeName := range typeNames {
		typeDir := filepath.Join(srcDir, typeName)
		info, err := os.Stat(typeDir)
		if err != nil || !info.IsDir() {
			continue
		}

		var mdFiles []string
		_ = filepath.WalkDir(typeDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !d.IsDir() && strings.HasSuffix(path, ".md") {
				mdFiles = append(mdFiles, path)
			}
			return nil
		})
		sort.Strings(mdFiles)

		for _, srcFile := range mdFiles {
			relPath, _ := filepath.Rel(filepath.Dir(srcDir), srcFile)

			fm, _, hasFrontmatter, parseErr := frontmatter.ParseFileValidated(srcFile, typeName)
			if parseErr != nil {
				fmt.Fprintf(w, "ERROR: %s — %s\n", relPath, parseErr)
				result.Errors++
				continue
			}

			if !hasFrontmatter {
				continue
			}

			if isEmptyFrontmatter(fm) {
				continue
			}

			for op := range fm.Overrides {
				if !validPlatforms[op] {
					fmt.Fprintf(w, "ERROR: %s — override targets unknown platform '%s' (valid: %s)\n",
						relPath, op, strings.Join(platformList, ", "))
					result.Errors++
				}
			}

			for _, tool := range fm.Tools {
				base, _, _ := strings.Cut(tool, "(")
				if len(validTools) > 0 && !validTools[base] {
					fmt.Fprintf(w, "WARN: %s — tool '%s' has no mapping in tool-map.yaml (will be dropped for non-Claude platforms)\n",
						relPath, base)
					result.Warnings++
				}
			}
		}
	}

	return result
}

// isEmptyFrontmatter returns true if all fields in the frontmatter are zero values.
func isEmptyFrontmatter(fm frontmatter.Frontmatter) bool {
	return fm.Name == "" &&
		fm.Description == "" &&
		fm.Model == "" &&
		fm.Context == "" &&
		fm.Agent == "" &&
		fm.Path == "" &&
		fm.ApplyTo == "" &&
		fm.ArgumentHint == "" &&
		!fm.DisableModelInvocation &&
		len(fm.Tools) == 0 &&
		len(fm.AllowedTools) == 0 &&
		len(fm.Paths) == 0 &&
		len(fm.Overrides) == 0
}

// ValidateHooks validates hooks.yaml against hooks.schema.json when at least one
// platform has hooks configured. Emits a warning (not an error) if hooks are
// configured in aicfg.yaml but hooks.yaml is absent — the file is optional.
func ValidateHooks(rootDir string, platforms config.PlatformsConfig, w io.Writer) Result {
	var result Result

	hasHooksPlatform := false
	for _, platCfg := range platforms {
		if platCfg.Hooks != nil {
			hasHooksPlatform = true
			break
		}
	}
	if !hasHooksPlatform {
		return result
	}

	hooksPath := filepath.Join(rootDir, "hooks.yaml")
	data, err := os.ReadFile(hooksPath)
	if os.IsNotExist(err) {
		fmt.Fprintf(w, "WARNING: hooks configured in aicfg.yaml but hooks.yaml not found at %s\n", hooksPath)
		result.Warnings++
		return result
	}
	if err != nil {
		fmt.Fprintf(w, "ERROR: %s — %s\n", hooksPath, err)
		result.Errors++
		return result
	}

	if err := schemas.ValidateHooksSchema(data, hooksPath); err != nil {
		fmt.Fprintf(w, "ERROR: %s — %s\n", hooksPath, err)
		result.Errors++
	}
	return result
}
