package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var recallCmd = &cobra.Command{
	Use:   "recall <query>",
	Short: "Semantic search for memories",
	Long:  "Search memories using vector/semantic similarity.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRecall,
}

var (
	recallNamespace string
	recallLimit     int
)

func init() {
	recallCmd.Flags().StringVar(&recallNamespace, "ns", "", "filter by namespace")
	recallCmd.Flags().IntVar(&recallLimit, "limit", 10, "max results to return")
}

func runRecall(cmd *cobra.Command, args []string) error {
	client, err := NewClient()
	if err != nil {
		return err
	}
	query := strings.Join(args, " ")

	results, err := client.SearchMemories(query, recallNamespace, "", recallLimit, true)
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(results)
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stdout, "No results found.")
		return nil
	}

	printSearchResults(results)
	return nil
}
