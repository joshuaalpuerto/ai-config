package transpiler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/joshuacalpuerto/ai-config/internal/config"
	"github.com/joshuacalpuerto/ai-config/internal/hooks"
	"github.com/joshuacalpuerto/ai-config/schemas"
	"gopkg.in/yaml.v3"
)

// TranspileHooks validates, transforms, and deploys hooks.yaml for one platform.
// Silently skips when hooks.yaml is absent (it is optional).
// Transformation: canonical tool names → platform-specific via toolMap;
// inject paths (relative to rootDir) → absolute platform-prefixed paths.
func TranspileHooks(
	rootDir string,
	platform string,
	hooksCfg config.HooksConfig,
	toolMap config.ToolMap,
	targetRoot string,
	w io.Writer,
) error {
	srcFile := hooksCfg.SrcHooksFile
	srcPath := filepath.Join(rootDir, srcFile)
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("reading hooks.yaml: %w", err)
	}

	if err := schemas.ValidateHooksSchema(data, srcPath); err != nil {
		return fmt.Errorf("validating hooks.yaml: %w", err)
	}

	var src hooks.HooksConfig
	if err := yaml.Unmarshal(data, &src); err != nil {
		return fmt.Errorf("parsing hooks.yaml: %w", err)
	}

	platformToolMap := toolMap[platform]
	contextDirAbs := filepath.Join(targetRoot, hooksCfg.ContextDir)

	for i := range src.Rules {
		src.Rules[i].Matchers.Tools = translateTools(src.Rules[i].Matchers.Tools, platformToolMap)
		src.Rules[i].Actions.Inject = translateInjectPath(src.Rules[i].Actions.Inject, contextDirAbs)
	}

	out, err := yaml.Marshal(&src)
	if err != nil {
		return fmt.Errorf("serialising transformed hooks.yaml: %w", err)
	}

	destPath := filepath.Join(targetRoot, hooksCfg.HooksFile)
	if err := atomicWrite(destPath, out); err != nil {
		return fmt.Errorf("writing hooks.yaml to %s: %w", destPath, err)
	}

	fmt.Fprintf(w, "  hooks: deployed to %s\n", destPath)
	return nil
}

// translateTools maps canonical tool names to platform-specific equivalents.
// If no mapping exists, the canonical name is kept (Claude uses the same names).
func translateTools(canonical []string, platformMap map[string]string) []string {
	if len(canonical) == 0 {
		return canonical
	}
	out := make([]string, len(canonical))
	for i, name := range canonical {
		if mapped, ok := platformMap[name]; ok {
			out[i] = mapped
		} else {
			out[i] = name
		}
	}
	return out
}

// translateInjectPath rewrites a source-relative inject path to the platform's
// deployed context directory. Only the filename is preserved — subdirectory
// nesting within context/ is not supported in this version.
func translateInjectPath(injectPath, contextDirAbs string) string {
	if injectPath == "" {
		return ""
	}
	return filepath.Join(contextDirAbs, filepath.Base(injectPath))
}

// CopyContextDir copies all files from rootDir/context/ to the platform's context_dir.
// Silently skips when context/ is absent (it is optional).
func CopyContextDir(
	rootDir string,
	hooksCfg config.HooksConfig,
	targetRoot string,
	w io.Writer,
) error {
	srcDir := filepath.Join(rootDir, "context")
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return nil
	}

	destDir := filepath.Join(targetRoot, hooksCfg.ContextDir)
	count := 0

	err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(srcDir, path)
		if relErr != nil {
			return fmt.Errorf("resolving relative path for %s: %w", path, relErr)
		}
		if copyErr := copyFile(path, filepath.Join(destDir, rel)); copyErr != nil {
			return fmt.Errorf("copying context file %s: %w", rel, copyErr)
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}

	if count > 0 {
		fmt.Fprintf(w, "  context: %d file(s) copied to %s\n", count, destDir)
	}
	return nil
}
