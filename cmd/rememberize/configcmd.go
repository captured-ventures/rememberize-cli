package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or update CLI configuration",
	Long:  "Show current configuration, or use 'config set <key> <value>' to update.",
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Long:  "Set a config value. Keys: auth.api_url, auth.api_key, defaults.namespace, defaults.type, defaults.format",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

func init() {
	configCmd.AddCommand(configSetCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()

	if jsonOutput {
		return printJSON(cfg)
	}

	fmt.Fprintf(os.Stdout, "Config file: %s\n\n", configPath())
	fmt.Fprintln(os.Stdout, "[auth]")
	fmt.Fprintf(os.Stdout, "  api_url = %s\n", cfg.Auth.APIURL)
	if cfg.Auth.APIKey != "" {
		masked := strings.Repeat("*", max(0, len(cfg.Auth.APIKey)-4)) + cfg.Auth.APIKey[max(0, len(cfg.Auth.APIKey)-4):]
		fmt.Fprintf(os.Stdout, "  api_key = %s\n", masked)
	} else {
		fmt.Fprintln(os.Stdout, "  api_key = (not set)")
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "[defaults]")
	fmt.Fprintf(os.Stdout, "  namespace = %s\n", cfg.Defaults.Namespace)
	fmt.Fprintf(os.Stdout, "  type      = %s\n", cfg.Defaults.Type)
	fmt.Fprintf(os.Stdout, "  format    = %s\n", cfg.Defaults.Format)

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	cfg := loadConfig()

	key, value := args[0], args[1]
	if err := setConfigValue(cfg, key, value); err != nil {
		return err
	}

	if err := saveConfig(cfg); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Set %s = %s\n", key, value)
	return nil
}
