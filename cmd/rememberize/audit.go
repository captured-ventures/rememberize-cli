package main

import (
	"fmt"
	"os"
	"text/tabwriter"

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
	client := NewClient()

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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tACTION\tCONNECTION\tMEMORY\tCREATED")
	for _, e := range entries {
		memID := "-"
		if e.MemoryID != nil {
			memID = *e.MemoryID
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			e.ID, e.Action, e.ConnectionID, memID, e.CreatedAt,
		)
	}
	w.Flush()
	return nil
}
