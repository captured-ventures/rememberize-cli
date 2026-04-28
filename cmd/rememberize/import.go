package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/captured-ventures/rememberize-cli/internal/transfer"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import memories from a file",
	Long:  "Import memories from Claude MEMORY.md, ChatGPT JSON, or CSV files",
	Args:  cobra.ExactArgs(1),
	RunE:  runImport,
}

var (
	importNamespace string
	importFormat    string
	importDryRun    bool
)

func init() {
	importCmd.Flags().StringVar(&importNamespace, "namespace", "", "Override namespace for all imported memories")
	importCmd.Flags().StringVar(&importFormat, "format", "", "Force format (memory-md, chatgpt, csv, json)")
	importCmd.Flags().BoolVar(&importDryRun, "dry-run", false, "Parse and show count without importing")
}

func runImport(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	var data []byte
	var filename string
	var err error
	if filePath == "-" {
		data, err = io.ReadAll(os.Stdin)
		filename = "stdin"
	} else {
		data, err = os.ReadFile(filePath)
		filename = filepath.Base(filePath)
	}
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	if importDryRun {
		var format transfer.Format
		if importFormat != "" {
			format = transfer.Format(importFormat)
		} else {
			format = transfer.DetectFormat(filename, data)
		}
		parser := transfer.GetParser(format)
		if parser == nil {
			return fmt.Errorf("unsupported format: %s", format)
		}
		memories, err := parser.Parse(data)
		if err != nil {
			return fmt.Errorf("parse: %w", err)
		}
		fmt.Printf("Dry run: would import %d memories (format: %s)\n", len(memories), format)
		return nil
	}

	client, err := NewClient()
	if err != nil {
		return err
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("write form part: %w", err)
	}
	if importNamespace != "" {
		if err := writer.WriteField("namespace", importNamespace); err != nil {
			return fmt.Errorf("write namespace field: %w", err)
		}
	}
	if importFormat != "" {
		if err := writer.WriteField("format", importFormat); err != nil {
			return fmt.Errorf("write format field: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := client.newRequest("POST", "/api/import", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("import request: %w", err)
	}
	defer resp.Body.Close()

	var result transfer.ImportResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if jsonOutput {
		return printJSON(result)
	}

	fmt.Printf("Imported: %d created, %d skipped\n", result.Created, result.Skipped)
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Printf("  error: %s\n", e)
		}
	}
	return nil
}
