package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// printJSON marshals v as indented JSON and writes to stdout.
func printJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}

// truncate shortens s to maxLen characters, appending "..." if truncated.
// Newlines are flattened so the output stays single-line for table cells.
func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// printSearchResults renders search results as a table.
//
// The header expands to include FTS/VEC columns when at least one result has
// non-zero component scores — semantic queries always return both, but plain
// FTS responses leave the vector score zeroed and we collapse to a smaller
// table for readability.
func printSearchResults(results []SearchResult) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	hasVec := false
	for _, r := range results {
		if r.VectorScore > 0 || r.FTSScore > 0 {
			hasVec = true
			break
		}
	}
	if hasVec {
		fmt.Fprintln(w, "SCORE\tFTS\tVEC\tID\tNAMESPACE\tTYPE\tCONTENT")
		for _, r := range results {
			fmt.Fprintf(w, "%.4f\t%.4f\t%.4f\t%s\t%s\t%s\t%s\n",
				r.Score, r.FTSScore, r.VectorScore,
				r.Memory.ID,
				r.Memory.Namespace,
				r.Memory.Type,
				truncate(r.Memory.Content, 60),
			)
		}
	} else {
		fmt.Fprintln(w, "SCORE\tID\tNAMESPACE\tTYPE\tCONTENT")
		for _, r := range results {
			fmt.Fprintf(w, "%.4f\t%s\t%s\t%s\t%s\n",
				r.Score,
				r.Memory.ID,
				r.Memory.Namespace,
				r.Memory.Type,
				truncate(r.Memory.Content, 60),
			)
		}
	}
	w.Flush()
}
