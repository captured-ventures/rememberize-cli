package transfer

import (
	"fmt"
	"strings"
)

// MemoryMDParser parses Claude MEMORY.md format files.
// Supports two formats:
// 1. Frontmatter entries: ---\nname: ...\n---\n\ncontent
// 2. Index format: - [title](file.md) — description
type MemoryMDParser struct{}

// memoryMDEntry holds parsed frontmatter fields from a single memory block.
type memoryMDEntry struct {
	Name        string
	Description string
}

func (p *MemoryMDParser) Parse(data []byte) ([]Memory, error) {
	text := string(data)
	trimmed := strings.TrimSpace(text)

	if strings.HasPrefix(trimmed, "---") {
		return p.parseFrontmatter(text)
	}
	return p.parseIndex(text)
}

// parseFrontmatter scans the text block-by-block. Each block begins with a `---`
// line, has key: value lines until the next `---`, and a body that runs until
// the next opening `---` or EOF. Prior implementations merged fields and body
// into one string and lost body content when colon-bearing prose was present.
func (p *MemoryMDParser) parseFrontmatter(text string) ([]Memory, error) {
	lines := strings.Split(text, "\n")
	var memories []Memory
	i := 0

	for i < len(lines) {
		// Skip blank lines between blocks.
		for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
			i++
		}
		if i >= len(lines) {
			break
		}
		// Must be an opening "---".
		if strings.TrimSpace(lines[i]) != "---" {
			// Not a frontmatter block — skip to next line and keep scanning.
			i++
			continue
		}
		i++ // consume opening ---

		// Collect frontmatter fields until the closing ---.
		var fm memoryMDEntry
		for i < len(lines) && strings.TrimSpace(lines[i]) != "---" {
			line := strings.TrimSpace(lines[i])
			if line != "" && strings.Contains(line, ":") {
				key, value, found := strings.Cut(line, ":")
				if found {
					key = strings.TrimSpace(key)
					value = strings.TrimSpace(value)
					switch key {
					case "name":
						fm.Name = value
					case "description":
						fm.Description = value
					}
				}
			}
			i++
		}
		if i < len(lines) {
			i++ // consume closing ---
		}

		// Collect body lines until the next opening --- or EOF.
		var bodyLines []string
		for i < len(lines) && strings.TrimSpace(lines[i]) != "---" {
			bodyLines = append(bodyLines, lines[i])
			i++
		}
		body := strings.TrimSpace(strings.Join(bodyLines, "\n"))

		content := body
		if content == "" && fm.Name != "" {
			content = fm.Name
			if fm.Description != "" {
				content = fm.Name + ": " + fm.Description
			}
		}
		if content == "" {
			continue
		}

		m := Memory{
			Content:   content,
			Type:      "semantic",
			Namespace: "default",
		}
		if fm.Description != "" {
			desc := fm.Description
			m.Metadata = &desc
		}
		memories = append(memories, m)
	}

	if len(memories) == 0 {
		return nil, fmt.Errorf("no memories found in MEMORY.md content")
	}
	return memories, nil
}

func (p *MemoryMDParser) parseIndex(text string) ([]Memory, error) {
	var memories []Memory
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- ") {
			continue
		}

		line = strings.TrimPrefix(line, "- ")

		var content string
		if idx := strings.Index(line, "]("); idx > 0 {
			title := strings.TrimPrefix(line[:idx], "[")
			rest := line[idx+2:]
			if endIdx := strings.Index(rest, ")"); endIdx > 0 {
				after := strings.TrimSpace(rest[endIdx+1:])
				after = strings.TrimPrefix(after, "—")
				after = strings.TrimPrefix(after, "-")
				after = strings.TrimSpace(after)
				if after != "" {
					content = title + ": " + after
				} else {
					content = title
				}
			}
		} else {
			content = line
		}

		if content == "" {
			continue
		}

		memories = append(memories, Memory{
			Content:   content,
			Type:      "semantic",
			Namespace: "default",
		})
	}

	if len(memories) == 0 {
		return nil, fmt.Errorf("no memories found in index content")
	}

	return memories, nil
}

// MemoryMDFormatter formats memories as a Claude MEMORY.md file.
type MemoryMDFormatter struct{}

func (f *MemoryMDFormatter) Format(memories []Memory) ([]byte, error) {
	var sb strings.Builder

	for i, m := range memories {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("---\n")
		sb.WriteString(fmt.Sprintf("name: %s\n", truncate(m.Content, 60)))
		if m.Metadata != nil {
			sb.WriteString(fmt.Sprintf("description: %s\n", *m.Metadata))
		}
		sb.WriteString(fmt.Sprintf("type: %s\n", m.Type))
		sb.WriteString("---\n\n")
		sb.WriteString(m.Content)
		sb.WriteString("\n")
	}

	return []byte(sb.String()), nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
