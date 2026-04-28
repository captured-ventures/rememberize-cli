package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single memory by ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runGet,
}

func runGet(cmd *cobra.Command, args []string) error {
	client := NewClient()

	mem, err := client.GetMemory(args[0])
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(mem)
	}

	fmt.Fprintf(os.Stdout, "ID:        %s\n", mem.ID)
	fmt.Fprintf(os.Stdout, "Namespace: %s\n", mem.Namespace)
	fmt.Fprintf(os.Stdout, "Type:      %s\n", mem.Type)
	fmt.Fprintf(os.Stdout, "Version:   %d\n", mem.Version)
	fmt.Fprintf(os.Stdout, "Created:   %s\n", mem.CreatedAt)
	fmt.Fprintf(os.Stdout, "Updated:   %s\n", mem.UpdatedAt)
	if mem.ExpiresAt != nil {
		fmt.Fprintf(os.Stdout, "Expires:   %s\n", *mem.ExpiresAt)
	}
	if mem.Metadata != nil {
		fmt.Fprintf(os.Stdout, "Metadata:  %s\n", *mem.Metadata)
	}
	if mem.SourceID != nil {
		fmt.Fprintf(os.Stdout, "Source ID: %s\n", *mem.SourceID)
	}
	fmt.Fprintf(os.Stdout, "\n%s\n", mem.Content)
	return nil
}
