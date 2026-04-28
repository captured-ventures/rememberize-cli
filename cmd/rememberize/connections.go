package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var connectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "List connections",
	RunE:  runConnections,
}

func runConnections(cmd *cobra.Command, args []string) error {
	client, err := NewClient()
	if err != nil {
		return err
	}

	connections, err := client.ListConnections()
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(connections)
	}

	if len(connections) == 0 {
		fmt.Fprintln(os.Stdout, "No connections found.")
		return nil
	}

	headers := []string{"ID", "NAME", "TYPE", "ACTIVE", "LAST SEEN", "CREATED"}
	rows := make([][]string, 0, len(connections))
	for _, c := range connections {
		active := "no"
		if c.IsActive {
			active = "yes"
		}
		rows = append(rows, []string{c.ID, c.Name, c.Type, active, c.LastSeen, c.CreatedAt})
	}
	renderTable(os.Stdout, headers, rows)
	return nil
}
