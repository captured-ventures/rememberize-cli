package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var connectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "List connections",
	RunE:  runConnections,
}

func runConnections(cmd *cobra.Command, args []string) error {
	client := NewClient()

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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tACTIVE\tLAST SEEN\tCREATED")
	for _, c := range connections {
		active := "no"
		if c.IsActive {
			active = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			c.ID, c.Name, c.Type, active, c.LastSeen, c.CreatedAt,
		)
	}
	w.Flush()
	return nil
}
