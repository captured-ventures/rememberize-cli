package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var namespacesCmd = &cobra.Command{
	Use:   "namespaces",
	Short: "List namespaces",
	RunE:  runNamespaces,
}

func runNamespaces(cmd *cobra.Command, args []string) error {
	client, err := NewClient()
	if err != nil {
		return err
	}

	namespaces, err := client.ListNamespaces()
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(namespaces)
	}

	if len(namespaces) == 0 {
		fmt.Fprintln(os.Stdout, "No namespaces found.")
		return nil
	}

	headers := []string{"NAME", "DESCRIPTION", "CREATED"}
	rows := make([][]string, 0, len(namespaces))
	for _, ns := range namespaces {
		rows = append(rows, []string{ns.Name, ns.Description, ns.CreatedAt})
	}
	renderTable(os.Stdout, headers, rows)
	return nil
}
