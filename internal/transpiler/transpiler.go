package transpiler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joshuaalpuerto/ai-config/internal/config"
	"github.com/joshuaalpuerto/ai-config/internal/frontmatter"
)

// TranspileFile processes a single source file for a single platform+type combo.
func TranspileFile(
	srcFile, platform, typeName string,
	typeCfg config.TypeConfig,
	platCfg config.PlatformConfig,
	toolMap config.ToolMap,
	outDir string,
) error {
	fm, body, hasFrontmatter, err := frontmatter.ParseFileValidated(srcFile, typeName)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", srcFile, err)
	}

	// Determine destination path (caller already set outDir; srcFile is flat, no subdir here).
	filename := filepath.Base(srcFile)
	base := strings.TrimSuffix(filename, ".md")
	destFile := filepath.Join(outDir, base+typeCfg.Suffix)

	if !hasFrontmatter {
		return copyFile(srcFile, destFile)
	}

	fmStr, err := BuildFrontmatter(fm, platform, typeCfg, platCfg, toolMap)
	if err != nil {
		return fmt.Errorf("building frontmatter for %s: %w", srcFile, err)
	}

	content := "---\n" + fmStr + "---\n" + body + "\n"
	return atomicWrite(destFile, []byte(content))
}

// TranspileSubdirFile processes a source file that lives in a subdirectory
// (e.g. src/skills/github/SKILL.md). The relative path is preserved under outDir
// with no suffix applied, matching the shell behavior.
func TranspileSubdirFile(
	srcFile, relPath, platform, typeName string,
	typeCfg config.TypeConfig,
	platCfg config.PlatformConfig,
	toolMap config.ToolMap,
	outDir string,
) error {
	fm, body, hasFrontmatter, err := frontmatter.ParseFileValidated(srcFile, typeName)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", srcFile, err)
	}

	destFile := filepath.Join(outDir, relPath)

	if !hasFrontmatter {
		return copyFile(srcFile, destFile)
	}

	fmStr, err := BuildFrontmatter(fm, platform, typeCfg, platCfg, toolMap)
	if err != nil {
		return fmt.Errorf("building frontmatter for %s: %w", srcFile, err)
	}

	content := "---\n" + fmStr + "---\n" + body + "\n"
	return atomicWrite(destFile, []byte(content))
}

// TranspileType walks src/<typeName>/ and transpiles each .md for one platform.
// Returns the count of files processed.
func TranspileType(
	srcDir, typeName, platform string,
	typeCfg config.TypeConfig,
	platCfg config.PlatformConfig,
	toolMap config.ToolMap,
	targetRoot string,
) (int, error) {
	typeDir := filepath.Join(srcDir, typeName)
	info, err := os.Stat(typeDir)
	if err != nil || !info.IsDir() {
		return 0, nil
	}

	outDir := filepath.Join(targetRoot, typeCfg.Path)

	// Collect .md files sorted for deterministic output order.
	var mdFiles []string
	err = filepath.WalkDir(typeDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".md") {
			mdFiles = append(mdFiles, path)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("walking %s: %w", typeDir, err)
	}
	sort.Strings(mdFiles)

	for _, srcFile := range mdFiles {
		relPath, err := filepath.Rel(typeDir, srcFile)
		if err != nil {
			return 0, err
		}

		if err := os.MkdirAll(filepath.Join(outDir, filepath.Dir(relPath)), 0o755); err != nil {
			return 0, fmt.Errorf("creating output directory: %w", err)
		}

		if strings.Contains(relPath, string(filepath.Separator)) {
			// Subdirectory file — preserve relative path, no suffix.
			if err := TranspileSubdirFile(srcFile, relPath, platform, typeName, typeCfg, platCfg, toolMap, outDir); err != nil {
				return 0, err
			}
		} else {
			if err := TranspileFile(srcFile, platform, typeName, typeCfg, platCfg, toolMap, outDir); err != nil {
				return 0, err
			}
		}
	}

	return len(mdFiles), nil
}

// TranspileAll runs all platforms × all types.
func TranspileAll(
	srcDir string,
	platforms config.PlatformsConfig,
	toolMap config.ToolMap,
	rootDir string,
	w io.Writer,
) error {
	// Sort platform names for deterministic output.
	platformNames := make([]string, 0, len(platforms))
	for p := range platforms {
		platformNames = append(platformNames, p)
	}
	sort.Strings(platformNames)

	for _, platform := range platformNames {
		platCfg := platforms[platform]
		fmt.Fprintf(w, "[%s]\n", platform)

		targetRoot := platCfg.Target
		if targetRoot == "" || targetRoot == "null" {
			targetRoot = "."
		}
		if !filepath.IsAbs(targetRoot) {
			targetRoot = filepath.Join(rootDir, targetRoot)
		}

		// Sort type names for deterministic output.
		typeNames := make([]string, 0, len(platCfg.Types))
		for t := range platCfg.Types {
			typeNames = append(typeNames, t)
		}
		sort.Strings(typeNames)

		for _, typeName := range typeNames {
			typeCfg := platCfg.Types[typeName]
			count, err := TranspileType(srcDir, typeName, platform, typeCfg, platCfg, toolMap, targetRoot)
			if err != nil {
				return fmt.Errorf("transpiling %s/%s: %w", platform, typeName, err)
			}
			if count > 0 {
				fmt.Fprintf(w, "  %s: %d file(s)\n", typeName, count)
			}
		}
	}
	return nil
}

// copyFile copies src to dst, creating parent directories as needed.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", dst, err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading %s: %w", src, err)
	}
	return atomicWrite(dst, data)
}

// atomicWrite writes data to path using a temp file + rename to avoid partial writes.
func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating directory for %s: %w", path, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("writing temp file %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming %s to %s: %w", tmp, path, err)
	}
	return nil
}
