package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

// isTTY reports whether w is a real terminal. lipgloss tables only render
// meaningfully in a TTY; piped/redirected output falls back to tabwriter
// so the result stays parseable by `cut`/`awk`/etc.
func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

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

// renderTable picks lipgloss in a TTY, tabwriter when piped. Callers in
// --json mode skip this entirely (presentation contract is JSON only).
//
// Writer is parameterised so tests can pass a *bytes.Buffer and assert
// the tabwriter fallback path deterministically.
func renderTable(w io.Writer, headers []string, rows [][]string) {
	if !isTTY(w) {
		renderTabwriter(w, headers, rows)
		return
	}

	cellStyle := lipgloss.NewStyle().Padding(0, 1)
	headerStyle := cellStyle.Bold(true).Foreground(lipgloss.Color("8"))
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	t := table.New().
		Headers(headers...).
		Rows(rows...).
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

	fmt.Fprintln(w, t)
}

// renderTabwriter is the non-TTY fallback path. Output is identical in
// shape to the prior tabwriter blocks each command used to inline.
func renderTabwriter(w io.Writer, headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	for _, r := range rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
	}
	tw.Flush()
}

// printSearchResults renders search results. Header expands to include
// FTS/VEC columns when at least one result has non-zero component scores.
func printSearchResults(results []SearchResult) {
	hasVec := false
	for _, r := range results {
		if r.VectorScore > 0 || r.FTSScore > 0 {
			hasVec = true
			break
		}
	}

	var headers []string
	rows := make([][]string, 0, len(results))

	if hasVec {
		headers = []string{"SCORE", "FTS", "VEC", "ID", "NAMESPACE", "TYPE", "CONTENT"}
		for _, r := range results {
			rows = append(rows, []string{
				fmt.Sprintf("%.4f", r.Score),
				fmt.Sprintf("%.4f", r.FTSScore),
				fmt.Sprintf("%.4f", r.VectorScore),
				r.Memory.ID,
				r.Memory.Namespace,
				r.Memory.Type,
				truncate(r.Memory.Content, 60),
			})
		}
	} else {
		headers = []string{"SCORE", "ID", "NAMESPACE", "TYPE", "CONTENT"}
		for _, r := range results {
			rows = append(rows, []string{
				fmt.Sprintf("%.4f", r.Score),
				r.Memory.ID,
				r.Memory.Namespace,
				r.Memory.Type,
				truncate(r.Memory.Content, 60),
			})
		}
	}

	renderTable(os.Stdout, headers, rows)
}
