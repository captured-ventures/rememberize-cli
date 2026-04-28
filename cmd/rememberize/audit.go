package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View audit log",
	RunE:  runAudit,
}

var (
	auditLimit        int
	auditAction       string
	auditConnectionID string
)

func init() {
	auditCmd.Flags().IntVar(&auditLimit, "limit", 20, "max entries to return")
	auditCmd.Flags().StringVar(&auditAction, "action", "", "filter by action")
	auditCmd.Flags().StringVar(&auditConnectionID, "connection-id", "", "filter by connection ID")
}

func runAudit(cmd *cobra.Command, args []string) error {
	client, err := NewClient()
	if err != nil {
		return err
	}

	entries, err := client.ListAudit(auditLimit, auditAction, auditConnectionID)
	if err != nil {
		return err
	}

	if jsonOutput {
		return printJSON(entries)
	}

	if len(entries) == 0 {
		fmt.Fprintln(os.Stdout, "No audit entries found.")
		return nil
	}

	headers := []string{"ID", "ACTION", "CONNECTION", "MEMORY", "CREATED"}
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		memID := "-"
		if e.MemoryID != nil {
			memID = *e.MemoryID
		}
		rows = append(rows, []string{e.ID, e.Action, e.ConnectionID, memID, e.CreatedAt})
	}
	renderTable(os.Stdout, headers, rows)
	return nil
}
