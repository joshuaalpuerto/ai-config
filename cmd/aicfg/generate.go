package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joshuaalpuerto/ai-config/internal/config"
	"github.com/joshuaalpuerto/ai-config/internal/docaudit"
	"github.com/joshuaalpuerto/ai-config/internal/hooks"
)

// generateSkills runs the built-in skill generators listed in
// cfg.GenerateSkills, writing each SKILL.md into srcDir before transpilation.
func generateSkills(cfg config.AicfgConfig, rootDir, srcDir string, w *os.File) error {
	if len(cfg.GenerateSkills) == 0 {
		return nil
	}

	projectName := filepath.Base(rootDir)
	docRoots := resolveDocRoots(cfg, rootDir)

	for _, name := range cfg.GenerateSkills {
		switch name {
		case "docaudit":
			if err := generateDocauditSkill(cfg, rootDir, srcDir, projectName, docRoots); err != nil {
				return fmt.Errorf("generating docaudit skill: %w", err)
			}
			fmt.Fprintf(w, "  generated: skills/doc-audit/SKILL.md\n")

		case "hooks-generator":
			if err := generateHooksSkill(cfg, rootDir, srcDir, projectName, docRoots); err != nil {
				return fmt.Errorf("generating hooks-generator skill: %w", err)
			}
			fmt.Fprintf(w, "  generated: skills/hooks-generator/SKILL.md\n")

		default:
			return fmt.Errorf("unknown generate_skills entry: %q", name)
		}
	}
	fmt.Fprintln(w)
	return nil
}

func generateDocauditSkill(cfg config.AicfgConfig, rootDir, srcDir, projectName string, docRoots []string) error {
	outPath := filepath.Join(srcDir, "skills", "doc-audit", "SKILL.md")

	content := docaudit.GenerateSkill(docaudit.Config{
		TargetDir:      rootDir,
		ProjectName:    projectName,
		DocRoots:       docRoots,
		AnalyzeExclude: cfg.AnalyzeExclude,
	})

	return writeGenerated(outPath, content)
}

func generateHooksSkill(cfg config.AicfgConfig, rootDir, srcDir, projectName string, docRoots []string) error {
	outPath := filepath.Join(srcDir, "skills", "hooks-generator", "SKILL.md")

	content := hooks.GenerateSkill(hooks.GenerateConfig{
		TargetDir:      rootDir,
		ProjectName:    projectName,
		DocRoots:       docRoots,
		AnalyzeExclude: cfg.AnalyzeExclude,
		HooksFile:      cfg.SrcHooksFile,
	})

	return writeGenerated(outPath, content)
}

// resolveDocRoots returns documentation paths from config or auto-detects
// docs/ and README.md in the project root.
func resolveDocRoots(cfg config.AicfgConfig, rootDir string) []string {
	if len(cfg.DocAudit.Paths) > 0 {
		return cfg.DocAudit.Paths
	}

	var roots []string
	if info, err := os.Stat(filepath.Join(rootDir, "docs")); err == nil && info.IsDir() {
		roots = append(roots, "docs/")
	}
	if _, err := os.Stat(filepath.Join(rootDir, "README.md")); err == nil {
		roots = append(roots, "README.md")
	}
	return roots
}

func writeGenerated(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
