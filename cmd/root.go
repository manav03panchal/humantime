// Package cmd provides the CLI commands for Humantime.
//
// This software is a derivative work based on Zeit (https://github.com/mrusme/zeit)
// Original work copyright (c) マリウス (mrusme)
// Modifications copyright (c) Manav Panchal
//
// Licensed under the SEGV License, Version 1.0
// See LICENSE file for full license text.
package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/output"
	"github.com/manav03panchal/humantime/internal/runtime"
)

// Version information (set at build time via ldflags).
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// Global flags.
var (
	flagFormat string
	flagColor  string
	flagDebug  bool
)

// ctx is the shared runtime context.
var ctx *runtime.Context

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "humantime",
	Short: "A next-generation CLI time tracking tool",
	Long: `Humantime is a powerful command-line time tracking tool that helps you
track your work with natural language commands.

Examples:
  humantime start on myproject
  humantime start on clientwork/bugfix with note 'fixing login issue'
  humantime stop
  humantime blocks this week
  humantime stats on clientwork from last month`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip initialization for completion and help commands (but allow __complete for dynamic completions)
		if cmd.Name() == "completion" || cmd.Name() == "help" {
			return nil
		}

		// Parse format flag
		var format output.Format
		switch flagFormat {
		case "json":
			format = output.FormatJSON
		case "plain":
			format = output.FormatPlain
		default:
			format = output.FormatCLI
		}

		// Parse color flag
		var colorMode output.ColorMode
		switch flagColor {
		case "always":
			colorMode = output.ColorAlways
		case "never":
			colorMode = output.ColorNever
		default:
			colorMode = output.ColorAuto
		}

		// Create runtime context
		opts := runtime.DefaultOptions()
		opts.Format = format
		opts.ColorMode = colorMode
		opts.Debug = flagDebug

		var err error
		ctx, err = runtime.New(opts)
		if err != nil {
			return err
		}

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if ctx != nil {
			return ctx.Close()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default behavior: show current status
		return runStatus(cmd, args)
	},
}

// runStatus shows the current tracking status.
func runStatus(cmd *cobra.Command, args []string) error {
	block, err := ctx.ActiveBlockRepo.GetActiveBlock(ctx.BlockRepo)
	if err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.JSONFormatter().PrintStatus(block)
	}

	ctx.CLIFormatter().PrintStatus(block)
	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&flagFormat, "format", "f", "cli",
		"Output format: cli, json, plain")
	rootCmd.PersistentFlags().StringVar(&flagColor, "color", "auto",
		"Color output: auto, always, never")
	rootCmd.PersistentFlags().BoolVar(&flagDebug, "debug", false,
		"Enable debug output")

	// Add commands
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(versionCmd)
}

// versionCmd shows version information.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("humantime %s\n", Version)
		cmd.Printf("  commit: %s\n", Commit)
		cmd.Printf("  built: %s\n", BuildTime)
		cmd.Println("")
		cmd.Println("Based on Zeit (https://github.com/mrusme/zeit)")
		cmd.Println("Licensed under SEGV License v1.0")
	},
}

// Die prints an error and exits.
func Die(err error) {
	if ctx != nil && ctx.IsJSON() {
		ctx.JSONFormatter().PrintError("error", err.Error(), runtime.GetSuggestion(err))
	} else {
		os.Stderr.WriteString("Error: " + runtime.FormatError(err) + "\n")
	}
	os.Exit(1)
}
