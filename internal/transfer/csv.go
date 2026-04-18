package transfer

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// CSVParser parses CSV files with columns: title, content, namespace, type, metadata.
// Only content is required; others get defaults.
type CSVParser struct{}

func (p *CSVParser) Parse(data []byte) ([]Memory, error) {
	reader := csv.NewReader(bytes.NewReader(data))

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}

	// Map column names to indices
	colIdx := make(map[string]int)
	for i, col := range header {
		colIdx[strings.TrimSpace(strings.ToLower(col))] = i
	}

	// content column is required
	contentIdx, hasContent := colIdx["content"]
	if !hasContent {
		return nil, fmt.Errorf("CSV must have a 'content' column")
	}

	var memories []Memory

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read CSV row: %w", err)
		}

		content := ""
		if contentIdx < len(record) {
			content = strings.TrimSpace(record[contentIdx])
		}
		if content == "" {
			continue // skip empty rows
		}

		m := Memory{
			Content:   content,
			Namespace: "default",
			Type:      "semantic",
		}

		if idx, ok := colIdx["namespace"]; ok && idx < len(record) && record[idx] != "" {
			m.Namespace = strings.TrimSpace(record[idx])
		}
		if idx, ok := colIdx["type"]; ok && idx < len(record) && record[idx] != "" {
			m.Type = strings.TrimSpace(record[idx])
		}
		if idx, ok := colIdx["metadata"]; ok && idx < len(record) && record[idx] != "" {
			meta := strings.TrimSpace(record[idx])
			m.Metadata = &meta
		}

		memories = append(memories, m)
	}

	if len(memories) == 0 {
		return nil, fmt.Errorf("no memories found in CSV")
	}

	return memories, nil
}

// CSVFormatter formats memories as CSV.
type CSVFormatter struct{}

func (f *CSVFormatter) Format(memories []Memory) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	if err := writer.Write([]string{"title", "content", "namespace", "type", "metadata"}); err != nil {
		return nil, fmt.Errorf("write CSV header: %w", err)
	}

	for _, m := range memories {
		title := truncate(m.Content, 60)
		metadata := ""
		if m.Metadata != nil {
			metadata = *m.Metadata
		}
		if err := writer.Write([]string{title, m.Content, m.Namespace, m.Type, metadata}); err != nil {
			return nil, fmt.Errorf("write CSV row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush CSV: %w", err)
	}

	return buf.Bytes(), nil
}
