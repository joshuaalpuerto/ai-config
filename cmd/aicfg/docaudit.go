package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/joshuaalpuerto/ai-config/internal/config"
	"github.com/joshuaalpuerto/ai-config/internal/docaudit"
)

func docauditCmd() *cobra.Command {
	var docRoots []string
	var outputPath string
	var projectName string

	cmd := &cobra.Command{
		Use:   "docaudit <directory>",
		Short: "Generate a project-specific doc-audit skill",
		Long:  "Reads the project's aicfg.yaml (if present) and writes a tailored SKILL.md that instructs the AI how to audit contributor-enablement gaps for that project.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := args[0]

			abs, err := filepath.Abs(root)
			if err != nil {
				return fmt.Errorf("resolving directory: %w", err)
			}

			// Default project name to the directory basename.
			if projectName == "" {
				projectName = filepath.Base(abs)
			}

			// Best-effort load of aicfg.yaml for source dir, exclude patterns, and doc roots.
			var srcDir string
			var excludes []string
			var configDocRoots []string
			if cfg, err := config.LoadConfig(filepath.Join(abs, "aicfg.yaml")); err == nil {
				excludes = cfg.AnalyzeExclude
				configDocRoots = cfg.DocAudit.Paths
				if filepath.IsAbs(cfg.SrcDir) {
					srcDir = cfg.SrcDir
				} else {
					srcDir = filepath.Join(abs, cfg.SrcDir)
				}
			}

			// Resolve output path from src_dir, falling back to <root>/src/skills/doc-audit/SKILL.md.
			if outputPath == "" {
				if srcDir == "" {
					srcDir = filepath.Join(abs, "src")
				}
				outputPath = filepath.Join(srcDir, "skills", "doc-audit", "SKILL.md")
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

			content := docaudit.GenerateSkill(docaudit.Config{
				TargetDir:      abs,
				ProjectName:    projectName,
				DocRoots:       docRoots,
				AnalyzeExclude: excludes,
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

	cmd.Flags().StringSliceVar(&docRoots, "doc-roots", nil, "documentation folders/files to audit against (default: auto-detect docs/ and README.md)")
	cmd.Flags().StringVar(&outputPath, "output", "", "path to write the SKILL.md (default: <src_dir>/skills/doc-audit/SKILL.md)")
	cmd.Flags().StringVar(&projectName, "project-name", "", "project name embedded in the skill (default: directory basename)")
	return cmd
}
