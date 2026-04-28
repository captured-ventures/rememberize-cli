package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export memories to a file",
	Long:  "Export memories as Claude MEMORY.md, JSON, or CSV",
	RunE:  runExport,
}

var (
	exportFormat    string
	exportNamespace string
	exportOutput    string
)

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "Export format (memory-md, json, csv)")
	exportCmd.Flags().StringVar(&exportNamespace, "namespace", "", "Export only this namespace")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output file path (defaults to stdout)")
}

func runExport(cmd *cobra.Command, args []string) error {
	client, err := NewClient()
	if err != nil {
		return err
	}

	url := "/api/export?format=" + exportFormat
	if exportNamespace != "" {
		url += "&namespace=" + exportNamespace
	}

	req, err := client.newRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("export request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("export failed (HTTP %d): %s", resp.StatusCode, respBody)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if exportOutput != "" {
		if err := os.WriteFile(exportOutput, data, 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
		fmt.Printf("Exported to %s\n", exportOutput)
	} else {
		_, _ = os.Stdout.Write(data)
	}
	return nil
}
