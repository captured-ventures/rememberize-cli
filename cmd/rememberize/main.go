package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

// version is the CLI version string. Defaults to "dev" for local builds;
// release builds inject the tag via -ldflags "-X main.version=v0.1.0".
var version = "dev"

// jsonOutput is the persistent --json flag, read by every command's RunE
// to switch between human-readable and machine-readable output.
var jsonOutput bool

func main() {
	if err := fang.Execute(context.Background(), rootCmd, fang.WithVersion(version)); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:           "rememberize",
	Short:         "CLI client for the rememberize memory system",
	Long:          "A portable, multi-directional memory system for AI.\nManage memories, search, and configure connections from the command line.",
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output raw JSON (for scripting/piping)")
	rootCmd.PersistentFlags().CountVarP(&verboseCount, "verbose", "v", "increase log verbosity (-v info, -vv debug)")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "suppress all logs except errors")

	cobra.OnInitialize(configureLogger)

	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(recallCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(namespacesCmd)
	rootCmd.AddCommand(connectionsCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(keysCmd)
	rootCmd.AddCommand(pairCmd)
}
