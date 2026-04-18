package transfer

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ChatGPTParser parses ChatGPT export JSON files.
// ChatGPT exports are an array of conversation objects with nested message mappings.
type ChatGPTParser struct{}

// chatgptConversation represents a single conversation in the export.
type chatgptConversation struct {
	Title      string                       `json:"title"`
	CreateTime float64                      `json:"create_time"`
	Mapping    map[string]chatgptMsgWrapper `json:"mapping"`
}

type chatgptMsgWrapper struct {
	Message *chatgptMessage `json:"message"`
}

type chatgptMessage struct {
	Author  chatgptAuthor  `json:"author"`
	Content chatgptContent `json:"content"`
}

type chatgptAuthor struct {
	Role string `json:"role"`
}

type chatgptContent struct {
	Parts []any `json:"parts"`
}

func (p *ChatGPTParser) Parse(data []byte) ([]Memory, error) {
	var conversations []chatgptConversation
	if err := json.Unmarshal(data, &conversations); err != nil {
		return nil, fmt.Errorf("parse chatgpt export: %w", err)
	}

	var memories []Memory

	for _, conv := range conversations {
		createdAt := time.Unix(int64(conv.CreateTime), 0).UTC().Format(time.RFC3339)

		for _, wrapper := range conv.Mapping {
			if wrapper.Message == nil {
				continue
			}
			if wrapper.Message.Author.Role != "user" {
				continue
			}

			var textParts []string
			for _, part := range wrapper.Message.Content.Parts {
				if s, ok := part.(string); ok && strings.TrimSpace(s) != "" {
					textParts = append(textParts, s)
				}
			}

			if len(textParts) == 0 {
				continue
			}

			content := strings.Join(textParts, "\n")
			meta := fmt.Sprintf(`{"source":"chatgpt","conversation":"%s"}`, conv.Title)

			memories = append(memories, Memory{
				Content:   content,
				Type:      "episodic",
				Namespace: "default",
				Metadata:  &meta,
				CreatedAt: createdAt,
			})
		}
	}

	if len(memories) == 0 {
		return nil, fmt.Errorf("no user messages found in ChatGPT export")
	}

	return memories, nil
}
