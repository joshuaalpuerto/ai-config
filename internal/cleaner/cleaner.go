package cleaner

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/joshuaalpuerto/ai-config/internal/config"
)

// CleanAll removes all generated output directories and prints removed paths to w.
func CleanAll(
	rootDir string,
	platforms config.PlatformsConfig,
	w io.Writer,
) error {
	platformNames := make([]string, 0, len(platforms))
	for p := range platforms {
		platformNames = append(platformNames, p)
	}
	sort.Strings(platformNames)

	for _, platform := range platformNames {
		platCfg := platforms[platform]

		targetRoot := platCfg.Target
		if targetRoot == "" || targetRoot == "null" {
			targetRoot = "."
		}
		if !filepath.IsAbs(targetRoot) {
			targetRoot = filepath.Join(rootDir, targetRoot)
		}

		seen := make(map[string]bool)
		var paths []string
		for _, typeCfg := range platCfg.Types {
			if typeCfg.Path != "" && !seen[typeCfg.Path] {
				seen[typeCfg.Path] = true
				paths = append(paths, typeCfg.Path)
			}
		}
		sort.Strings(paths)

		for _, p := range paths {
			fullPath := filepath.Join(targetRoot, p)
			if _, err := os.Stat(fullPath); err == nil {
				if err := os.RemoveAll(fullPath); err != nil {
					return fmt.Errorf("removing %s: %w", fullPath, err)
				}
				fmt.Fprintf(w, "  removed %s\n", fullPath)
			}
		}

		if platCfg.Hooks != nil {
			hooksDest := filepath.Join(targetRoot, platCfg.Hooks.HooksFile)
			if _, err := os.Stat(hooksDest); err == nil {
				if err := os.Remove(hooksDest); err != nil {
					return fmt.Errorf("removing %s: %w", hooksDest, err)
				}
				fmt.Fprintf(w, "  removed %s\n", hooksDest)
			}
			contextDest := filepath.Join(targetRoot, platCfg.Hooks.ContextDir)
			if _, err := os.Stat(contextDest); err == nil {
				if err := os.RemoveAll(contextDest); err != nil {
					return fmt.Errorf("removing %s: %w", contextDest, err)
				}
				fmt.Fprintf(w, "  removed %s\n", contextDest)
			}
		}
	}

	fmt.Fprintln(w, "Clean complete.")
	return nil
}
