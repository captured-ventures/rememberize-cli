package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Full-text search for memories",
	Long:  "Search memories using full-text search.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearch,
}

var (
	searchNamespace string
	searchType      string
	searchLimit     int
)

func init() {
	searchCmd.Flags().StringVar(&searchNamespace, "ns", "", "filter by namespace")
	searchCmd.Flags().StringVar(&searchType, "type", "", "filter by type")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 10, "max results to return")
}

func runSearch(cmd *cobra.Command, args []string) error {
	client, err := NewClient()
	if err != nil {
		return err
	}
	query := strings.Join(args, " ")

	results, err := client.SearchMemories(query, searchNamespace, searchType, searchLimit, false)
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
