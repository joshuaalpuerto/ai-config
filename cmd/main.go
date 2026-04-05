package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/joshuacalpuerto/ai-config/internal/cleaner"
	"github.com/joshuacalpuerto/ai-config/internal/config"
	"github.com/joshuacalpuerto/ai-config/internal/hooks"
	"github.com/joshuacalpuerto/ai-config/internal/settings"
	"github.com/joshuacalpuerto/ai-config/internal/transpiler"
	"github.com/joshuacalpuerto/ai-config/internal/validator"
)

type rootOpts struct {
	srcDir  string
	rootDir string
	cfg     config.AicfgConfig
}

func main() {
	opts := &rootOpts{}

	rootCmd := &cobra.Command{
		Use:   "aicfg",
		Short: "Build, validate, and clean AI configuration files",
	}

	rootCmd.PersistentFlags().StringVar(&opts.rootDir, "root", "", "repo root directory (default: directory of binary or cwd)")

	rootCmd.AddCommand(
		buildCmd(opts),
		validateCmd(opts),
		cleanCmd(opts),
		hooksCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadConfigs(opts *rootOpts) error {
	if opts.rootDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolving root dir: %w", err)
		}
		opts.rootDir = cwd
	}

	// Load config first so src_dir can influence source directory resolution.
	var err error
	opts.cfg, err = config.LoadConfig(filepath.Join(opts.rootDir, "aicfg.yaml"))
	if err != nil {
		return err
	}

	// Resolve source directory from src_dir in aicfg.yaml.
	if filepath.IsAbs(opts.cfg.SrcDir) {
		opts.srcDir = opts.cfg.SrcDir
	} else {
		opts.srcDir = filepath.Join(opts.rootDir, opts.cfg.SrcDir)
	}

	if _, err := os.Stat(opts.srcDir); err != nil {
		return fmt.Errorf("source directory %q not found: %w", opts.srcDir, err)
	}
	return nil
}

func buildCmd(opts *rootOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Transpile all source files for all platforms",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return loadConfigs(opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println()
			fmt.Println("Building ai-config...")
			fmt.Println()
			if err := transpiler.TranspileAll(
				opts.srcDir,
				opts.cfg.Platforms,
				opts.cfg.ToolMap,
				opts.rootDir,
				os.Stdout,
			); err != nil {
				return err
			}

			if claudePlatform, ok := opts.cfg.Platforms["claude"]; ok && claudePlatform.Hooks != nil {
				claudeDir := filepath.Join(opts.rootDir, ".claude")
				if err := settings.MergeClaudeSettings(claudeDir); err != nil {
					return fmt.Errorf("updating .claude/settings.json: %w", err)
				}
				fmt.Println("  settings.json: PreToolUse hook registered")
			}
			fmt.Println()
			fmt.Println("Done.")
			return nil
		},
	}
}

func validateCmd(opts *rootOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate all source files",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return loadConfigs(opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			result := validator.ValidateAll(opts.srcDir, opts.cfg.Platforms, opts.cfg.ToolMap, os.Stdout)
			hookResult := validator.ValidateHooks(opts.rootDir, opts.cfg.Platforms, os.Stdout)
			result.Errors += hookResult.Errors
			result.Warnings += hookResult.Warnings
			if result.Errors > 0 {
				fmt.Printf("\n%d error(s), %d warning(s) found.\n", result.Errors, result.Warnings)
				os.Exit(1)
			}
			fmt.Printf("All source files valid. (%d warning(s))\n", result.Warnings)
			return nil
		},
	}
}

func cleanCmd(opts *rootOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Remove all generated output directories",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return loadConfigs(opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleaner.CleanAll(opts.rootDir, opts.cfg.Platforms, os.Stdout)
		},
	}
}

func hooksCmd() *cobra.Command {
	var hooksFilePath string

	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Evaluate PreToolUse hook event from stdin",
		Long:  "Reads a Claude Code PreToolUse JSON event from stdin, evaluates configured rules, and exits 0 (allow) or 2 (block).",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHooks(hooksFilePath)
		},
	}
	cmd.Flags().StringVar(&hooksFilePath, "config", ".claude/hooks.yaml", "path to deployed hooks.yaml")
	return cmd
}

// runHooks reads a PreToolUse JSON event from stdin, evaluates rules from the deployed
// hooks.yaml, and exits 0 (allow) or 2 (block). Errors are fail-open: diagnostic written
// to stderr, exit 0, so Claude is never blocked by a broken hook configuration.
func runHooks(hooksFilePath string) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aicfg hooks: reading stdin: %v\n", err)
		return nil // fail-open
	}

	var event hooks.Event
	if err := json.Unmarshal(data, &event); err != nil {
		fmt.Fprintf(os.Stderr, "aicfg hooks: parsing event JSON: %v\n", err)
		return nil // fail-open
	}

	cfg, failOpen, err := hooks.LoadConfig(hooksFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aicfg hooks: loading config: %v\n", err)
		if failOpen {
			return nil
		}
		os.Exit(2)
	}

	resp, err := hooks.Evaluate(event, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aicfg hooks: evaluating rules: %v\n", err)
		return nil // fail-open on engine errors
	}

	if !resp.Continue {
		fmt.Fprintln(os.Stderr, resp.Context)
		os.Exit(2)
	}

	if resp.Context != "" {
		out, _ := json.Marshal(resp)
		fmt.Println(string(out))
	}

	return nil
}
