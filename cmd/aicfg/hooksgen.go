package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/joshuaalpuerto/ai-config/internal/config"
	"github.com/joshuaalpuerto/ai-config/internal/hooks"
)

// hooksgen.go holds the "hooks generate" subcommand. Separated from main.go
// for readability since it mirrors the docaudit.go pattern. Can be merged into
// main.go later if preferred.
func hooksGenerateCmd() *cobra.Command {
	var docRoots []string
	var outputPath string
	var projectName string

	cmd := &cobra.Command{
		Use:   "generate <directory>",
		Short: "Generate a hooks-generator skill for a project",
		Long:  "Reads the project's aicfg.yaml (if present) and writes a tailored SKILL.md that instructs the AI how to generate a hooks.yaml policy file for that project.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := args[0]

			abs, err := filepath.Abs(root)
			if err != nil {
				return fmt.Errorf("resolving directory: %w", err)
			}

			if projectName == "" {
				projectName = filepath.Base(abs)
			}

			// Best-effort load of aicfg.yaml for config values.
			var srcDir string
			var excludes []string
			var configDocRoots []string
			var hooksFile string
			if cfg, err := config.LoadConfig(filepath.Join(abs, "aicfg.yaml")); err == nil {
				excludes = cfg.AnalyzeExclude
				configDocRoots = cfg.DocAudit.Paths
				hooksFile = cfg.SrcHooksFile
				if filepath.IsAbs(cfg.SrcDir) {
					srcDir = cfg.SrcDir
				} else {
					srcDir = filepath.Join(abs, cfg.SrcDir)
				}
			}

			// Resolve output path from src_dir.
			if outputPath == "" {
				if srcDir == "" {
					srcDir = filepath.Join(abs, "src")
				}
				outputPath = filepath.Join(srcDir, "skills", "hooks-generator", "SKILL.md")
			}

			// Resolve doc roots: flag > aicfg.yaml > auto-detect.
			if len(docRoots) == 0 {
				if len(configDocRoots) > 0 {
					docRoots = configDocRoots
				} else {
					if info, err := os.Stat(filepath.Join(abs, "docs")); err == nil && info.IsDir() {
						docRoots = append(docRoots, "docs/")
					}
					if _, err := os.Stat(filepath.Join(abs, "README.md")); err == nil {
						docRoots = append(docRoots, "README.md")
					}
				}
			}

			content := hooks.GenerateSkill(hooks.GenerateConfig{
				TargetDir:      abs,
				ProjectName:    projectName,
				DocRoots:       docRoots,
				AnalyzeExclude: excludes,
				HooksFile:      hooksFile,
			})

			if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
				return fmt.Errorf("creating output directory: %w", err)
			}
			if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
				return fmt.Errorf("writing skill file: %w", err)
			}

			fmt.Printf("wrote %s — run \"aicfg build\" to apply\n", outputPath)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&docRoots, "doc-roots", nil, "documentation folders/files to reference (default: auto-detect docs/ and README.md)")
	cmd.Flags().StringVar(&outputPath, "output", "", "path to write the SKILL.md (default: <src_dir>/skills/hooks-generator/SKILL.md)")
	cmd.Flags().StringVar(&projectName, "project-name", "", "project name embedded in the skill (default: directory basename)")
	return cmd
}
