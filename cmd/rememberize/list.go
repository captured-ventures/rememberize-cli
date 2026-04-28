package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List memories",
	Long:  "List memories with optional namespace and type filters.",
	RunE:  runList,
}

var (
	listNamespace string
	listType      string
	listLimit     int
	listOffset    int
)

func init() {
	listCmd.Flags().StringVar(&listNamespace, "ns", "", "filter by namespace")
	listCmd.Flags().StringVar(&listType, "type", "", "filter by type")
	listCmd.Flags().IntVar(&listLimit, "limit", 20, "max results to return")
	listCmd.Flags().IntVar(&listOffset, "offset", 0, "offset for pagination")
}

func runList(cmd *cobra.Command, args []string) error {
	client := NewClient()

	memories, err := client.ListMemories(listNamespace, listType, listLimit, listOffset)
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(memories)
	}

	if len(memories) == 0 {
		fmt.Fprintln(os.Stdout, "No memories found.")
		return nil
	}

	headers := []string{"ID", "NAMESPACE", "TYPE", "CONTENT", "CREATED"}
	rows := make([][]string, 0, len(memories))
	for _, m := range memories {
		rows = append(rows, []string{
			m.ID,
			m.Namespace,
			m.Type,
			truncate(m.Content, 60),
			m.CreatedAt,
		})
	}
	renderTable(os.Stdout, headers, rows)
	return nil
}
