package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var namespacesCmd = &cobra.Command{
	Use:   "namespaces",
	Short: "List namespaces",
	RunE:  runNamespaces,
}

func runNamespaces(cmd *cobra.Command, args []string) error {
	client := NewClient()

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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDESCRIPTION\tCREATED")
	for _, ns := range namespaces {
		fmt.Fprintf(w, "%s\t%s\t%s\n", ns.Name, ns.Description, ns.CreatedAt)
	}
	w.Flush()
	return nil
}
