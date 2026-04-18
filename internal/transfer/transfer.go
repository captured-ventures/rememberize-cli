package transfer

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

// Format represents a supported import/export format.
type Format string

const (
	FormatMemoryMD Format = "memory-md"
	FormatChatGPT  Format = "chatgpt"
	FormatCSV      Format = "csv"
	FormatJSON     Format = "json"
	FormatUnknown  Format = "unknown"
)

// ImportResult summarizes an import operation.
type ImportResult struct {
	Created int      `json:"created"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

// Parser converts raw bytes into a slice of Memory structs.
type Parser interface {
	Parse(data []byte) ([]Memory, error)
}

// Formatter converts a slice of Memory structs into raw bytes.
type Formatter interface {
	Format(memories []Memory) ([]byte, error)
}

// DetectFormat auto-detects the format from filename extension and content sniffing.
func DetectFormat(filename string, content []byte) Format {
	ext := strings.ToLower(filepath.Ext(filename))
	base := strings.ToLower(filepath.Base(filename))

	// Filename-based detection
	if base == "memory.md" || base == "memories.md" {
		return FormatMemoryMD
	}
	if ext == ".csv" {
		return FormatCSV
	}
	if ext == ".md" {
		// Check for frontmatter (MEMORY.md style)
		if strings.HasPrefix(strings.TrimSpace(string(content)), "---") {
			return FormatMemoryMD
		}
	}

	// JSON content sniffing
	if ext == ".json" {
		trimmed := strings.TrimSpace(string(content))
		if strings.HasPrefix(trimmed, "[") {
			// Try to detect ChatGPT format vs plain JSON array
			var arr []json.RawMessage
			if json.Unmarshal(content, &arr) == nil && len(arr) > 0 {
				// Check if first item has ChatGPT-specific fields
				var chatgptItem struct {
					Title      string  `json:"title"`
					CreateTime float64 `json:"create_time"`
				}
				if json.Unmarshal(arr[0], &chatgptItem) == nil && chatgptItem.CreateTime > 0 {
					return FormatChatGPT
				}
				return FormatJSON
			}
		}
	}

	return FormatUnknown
}

// GetParser returns the appropriate parser for the given format.
func GetParser(format Format) Parser {
	switch format {
	case FormatMemoryMD:
		return &MemoryMDParser{}
	case FormatChatGPT:
		return &ChatGPTParser{}
	case FormatCSV:
		return &CSVParser{}
	case FormatJSON:
		return &JSONParser{}
	default:
		return nil
	}
}

// GetFormatter returns the appropriate formatter for the given format.
func GetFormatter(format Format) Formatter {
	switch format {
	case FormatMemoryMD:
		return &MemoryMDFormatter{}
	case FormatCSV:
		return &CSVFormatter{}
	case FormatJSON:
		return &JSONFormatter{}
	default:
		return nil
	}
}

// JSONParser parses a JSON array of memory objects.
type JSONParser struct{}

func (p *JSONParser) Parse(data []byte) ([]Memory, error) {
	var memories []Memory
	if err := json.Unmarshal(data, &memories); err != nil {
		return nil, err
	}
	return memories, nil
}

// JSONFormatter formats memories as a JSON array.
type JSONFormatter struct{}

func (f *JSONFormatter) Format(memories []Memory) ([]byte, error) {
	return json.MarshalIndent(memories, "", "  ")
}
