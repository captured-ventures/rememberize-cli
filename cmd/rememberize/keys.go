package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage API keys",
	Long:  "Create, list, and revoke API keys for connecting external tools",
}

var keysCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeysCreate,
}

var (
	keysCreateType string
	keysCreatePerm string
)

func init() {
	keysCreateCmd.Flags().StringVar(&keysCreateType, "type", "api", "Connection type (api, mcp, cli)")
	keysCreateCmd.Flags().StringVar(&keysCreatePerm, "permissions", "read,write", "Permissions (read,write,delete,admin)")
	keysCmd.AddCommand(keysCreateCmd, keysListCmd, keysRevokeCmd)
}

func runKeysCreate(cmd *cobra.Command, args []string) error {
	client := NewClient()
	name := args[0]

	body := map[string]string{
		"name":        name,
		"type":        keysCreateType,
		"permissions": keysCreatePerm,
	}

	resp, _, err := client.do("POST", "/api/keys", body)
	if err != nil {
		return err
	}

	if jsonOutput {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		ID     string `json:"id"`
		APIKey string `json:"api_key"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return err
	}

	fmt.Printf("Created API key: %s\n", result.Name)
	fmt.Printf("Key: %s\n", result.APIKey)
	fmt.Printf("ID:  %s\n", result.ID)
	fmt.Println("\nSave this key now -- it will not be shown again.")
	return nil
}

var keysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API keys",
	RunE:  runKeysList,
}

func runKeysList(cmd *cobra.Command, args []string) error {
	client := NewClient()

	resp, _, err := client.do("GET", "/api/keys", nil)
	if err != nil {
		return err
	}

	if jsonOutput {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Keys []Connection `json:"keys"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return err
	}

	if len(result.Keys) == 0 {
		fmt.Println("No API keys found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tPERMISSIONS\tCREATED")
	for _, k := range result.Keys {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", k.ID, k.Name, k.Type, k.Permissions, k.CreatedAt)
	}
	w.Flush()
	return nil
}

var keysRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke an API key",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeysRevoke,
}

func runKeysRevoke(cmd *cobra.Command, args []string) error {
	client := NewClient()

	_, _, err := client.do("DELETE", "/api/keys/"+args[0], nil)
	if err != nil {
		return err
	}

	fmt.Println("Key revoked.")
	return nil
}
