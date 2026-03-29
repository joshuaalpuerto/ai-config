package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/joshuacalpuerto/ai-config/internal/cleaner"
	"github.com/joshuacalpuerto/ai-config/internal/config"
	"github.com/joshuacalpuerto/ai-config/internal/transpiler"
	"github.com/joshuacalpuerto/ai-config/internal/validator"
)

type rootOpts struct {
	srcDir    string
	rootDir   string
	platforms config.PlatformsConfig
	toolMap   config.ToolMap
}

func main() {
	opts := &rootOpts{}

	rootCmd := &cobra.Command{
		Use:   "aicfg",
		Short: "Build, validate, and clean AI configuration files",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return loadConfigs(opts)
		},
	}

	rootCmd.PersistentFlags().StringVar(&opts.rootDir, "root", "", "repo root directory (default: directory of binary or cwd)")
	rootCmd.PersistentFlags().StringVar(&opts.srcDir, "src", "", "source directory (default: $SRC_DIR env, then <root>/src)")

	rootCmd.AddCommand(
		buildCmd(opts),
		validateCmd(opts),
		cleanCmd(opts),
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

	// Resolve source directory: --src flag > $SRC_DIR env > <root>/src.
	if opts.srcDir == "" {
		if envSrc := os.Getenv("SRC_DIR"); envSrc != "" {
			opts.srcDir = envSrc
		} else {
			opts.srcDir = filepath.Join(opts.rootDir, "src")
		}
	}

	if _, err := os.Stat(opts.srcDir); err != nil {
		return fmt.Errorf("source directory %q not found: %w", opts.srcDir, err)
	}

	var err error
	opts.platforms, err = config.LoadPlatforms(filepath.Join(opts.rootDir, "config", "platforms.yaml"))
	if err != nil {
		return err
	}
	opts.toolMap, err = config.LoadToolMap(filepath.Join(opts.rootDir, "config", "tool-map.yaml"))
	if err != nil {
		return err
	}
	return nil
}

func buildCmd(opts *rootOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Transpile all source files for all platforms",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println()
			fmt.Println("Building ai-config...")
			fmt.Println()
			if err := transpiler.TranspileAll(
				opts.srcDir,
				opts.platforms,
				opts.toolMap,
				opts.rootDir,
				os.Stdout,
			); err != nil {
				return err
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
		RunE: func(cmd *cobra.Command, args []string) error {
			result := validator.ValidateAll(opts.srcDir, opts.platforms, opts.toolMap, os.Stdout)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleaner.CleanAll(opts.rootDir, opts.platforms, os.Stdout)
		},
	}
}
