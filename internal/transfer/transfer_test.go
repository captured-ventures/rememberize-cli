package transfer_test

import (
	"strings"
	"testing"

	"github.com/captured-ventures/rememberize-cli/internal/transfer"
)

// ---------------------------------------------------------------------------
// Format detection tests
// ---------------------------------------------------------------------------

func TestDetectFormat_MemoryMD(t *testing.T) {
	content := "---\nname: test\ntype: user\n---\nSome content"
	got := transfer.DetectFormat("MEMORY.md", []byte(content))
	if got != transfer.FormatMemoryMD {
		t.Errorf("DetectFormat(MEMORY.md) = %q, want %q", got, transfer.FormatMemoryMD)
	}
}

func TestDetectFormat_CSV(t *testing.T) {
	content := "title,content,namespace\nmy mem,hello,default"
	got := transfer.DetectFormat("export.csv", []byte(content))
	if got != transfer.FormatCSV {
		t.Errorf("DetectFormat(export.csv) = %q, want %q", got, transfer.FormatCSV)
	}
}

func TestDetectFormat_ChatGPT(t *testing.T) {
	content := `[{"title":"ChatGPT","create_time":1234}]`
	got := transfer.DetectFormat("conversations.json", []byte(content))
	if got != transfer.FormatChatGPT {
		t.Errorf("DetectFormat(conversations.json) = %q, want %q", got, transfer.FormatChatGPT)
	}
}

func TestDetectFormat_JSON(t *testing.T) {
	content := `[{"id":"abc","content":"hello","type":"semantic"}]`
	got := transfer.DetectFormat("export.json", []byte(content))
	if got != transfer.FormatJSON {
		t.Errorf("DetectFormat(export.json) = %q, want %q", got, transfer.FormatJSON)
	}
}

func TestDetectFormat_Unknown(t *testing.T) {
	got := transfer.DetectFormat("file.xyz", []byte("random data"))
	if got != transfer.FormatUnknown {
		t.Errorf("DetectFormat(file.xyz) = %q, want %q", got, transfer.FormatUnknown)
	}
}

// ---------------------------------------------------------------------------
// MEMORY.md parser/formatter tests
// ---------------------------------------------------------------------------

func TestMemoryMDParser_SingleEntry(t *testing.T) {
	input := `---
name: user role
description: Brad is a technical founder
type: user
---

Brad is a technical founder at captured.ventures.
`
	parser := &transfer.MemoryMDParser{}
	memories, err := parser.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(memories) != 1 {
		t.Fatalf("got %d memories, want 1", len(memories))
	}
	if memories[0].Type != "semantic" {
		t.Errorf("type = %q, want semantic", memories[0].Type)
	}
}

func TestMemoryMDParser_BodyContentSurvives(t *testing.T) {
	// Real-world shape: frontmatter with name/description/type/originSessionId,
	// then a multi-line body containing prose with colons (e.g., "**Why:**").
	// The parser must keep the body as Content, not collapse to "name: description".
	input := `---
name: dogfooding-readiness
description: Brad wants to start dogfooding rememberize
type: project
originSessionId: abc123
---
Brad plans to start dogfooding the platform once connection setup is made less klunky.

**Why:** Currently the connection flow requires manual JSON entry.

**How to apply:** This is the implicit success criterion for SP2.
`
	parser := &transfer.MemoryMDParser{}
	memories, err := parser.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(memories) != 1 {
		t.Fatalf("got %d memories, want 1", len(memories))
	}
	content := memories[0].Content
	if !strings.Contains(content, "Brad plans to start dogfooding the platform") {
		t.Errorf("Content missing body opening line.\nContent = %q", content)
	}
	if !strings.Contains(content, "**Why:** Currently the connection flow") {
		t.Errorf("Content missing **Why:** body line (frontmatter parser likely swallowed it).\nContent = %q", content)
	}
	if !strings.Contains(content, "**How to apply:** This is the implicit success criterion") {
		t.Errorf("Content missing **How to apply:** body line.\nContent = %q", content)
	}
}

func TestMemoryMDParser_MultipleEntries(t *testing.T) {
	input := `---
name: memory one
description: first memory
type: user
---

First memory content.

---
name: memory two
description: second memory
type: feedback
---

Second memory content.
`
	parser := &transfer.MemoryMDParser{}
	memories, err := parser.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(memories) != 2 {
		t.Errorf("got %d memories, want 2", len(memories))
	}
}

func TestMemoryMDParser_IndexOnly(t *testing.T) {
	input := `# Memory Index

## User
- [user_brad.md](user_brad.md) — Brad: technical founder

## Feedback
- [feedback_testing.md](feedback_testing.md) — Use real DBs, not mocks
`
	parser := &transfer.MemoryMDParser{}
	memories, err := parser.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(memories) < 2 {
		t.Errorf("got %d memories, want >= 2", len(memories))
	}
}

func TestMemoryMDFormatter(t *testing.T) {
	memories := []transfer.Memory{
		{ID: "1", Content: "First memory", Type: "semantic", Namespace: "default"},
		{ID: "2", Content: "Second memory", Type: "episodic", Namespace: "work"},
	}
	formatter := &transfer.MemoryMDFormatter{}
	output, err := formatter.Format(memories)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	s := string(output)
	if !strings.Contains(s, "First memory") {
		t.Error("output missing 'First memory'")
	}
	if !strings.Contains(s, "Second memory") {
		t.Error("output missing 'Second memory'")
	}
	if !strings.Contains(s, "---") {
		t.Error("output missing frontmatter delimiters")
	}
}

func TestMemoryMD_RoundTrip(t *testing.T) {
	original := []transfer.Memory{
		{Content: "Round trip test", Type: "semantic", Namespace: "default"},
	}

	formatter := &transfer.MemoryMDFormatter{}
	exported, err := formatter.Format(original)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	parser := &transfer.MemoryMDParser{}
	imported, err := parser.Parse(exported)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(imported) != 1 {
		t.Fatalf("got %d memories, want 1", len(imported))
	}
	if !strings.Contains(imported[0].Content, "Round trip test") {
		t.Errorf("content = %q, want it to contain 'Round trip test'", imported[0].Content)
	}
}

// ---------------------------------------------------------------------------
// ChatGPT parser tests
// ---------------------------------------------------------------------------

func TestChatGPTParser(t *testing.T) {
	input := `[
		{
			"title": "Project discussion",
			"create_time": 1700000000.0,
			"mapping": {
				"msg1": {
					"message": {
						"author": {"role": "user"},
						"content": {"parts": ["Remember that I prefer Go over Python"]}
					}
				},
				"msg2": {
					"message": {
						"author": {"role": "assistant"},
						"content": {"parts": ["Noted! You prefer Go."]}
					}
				}
			}
		}
	]`
	parser := &transfer.ChatGPTParser{}
	memories, err := parser.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(memories) == 0 {
		t.Fatal("got 0 memories, want at least 1")
	}
	found := false
	for _, m := range memories {
		if strings.Contains(m.Content, "prefer Go over Python") {
			found = true
		}
	}
	if !found {
		t.Error("should contain user message content about preferring Go")
	}
}

func TestChatGPTParser_Empty(t *testing.T) {
	parser := &transfer.ChatGPTParser{}
	_, err := parser.Parse([]byte("[]"))
	if err == nil {
		t.Error("expected error for empty export, got nil")
	}
}

// ---------------------------------------------------------------------------
// CSV parser/formatter tests
// ---------------------------------------------------------------------------

func TestCSVParser(t *testing.T) {
	input := "title,content,namespace,type,metadata\nmy memory,hello world,default,semantic,\n"
	parser := &transfer.CSVParser{}
	memories, err := parser.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(memories) != 1 {
		t.Fatalf("got %d memories, want 1", len(memories))
	}
	if memories[0].Content != "hello world" {
		t.Errorf("content = %q, want 'hello world'", memories[0].Content)
	}
	if memories[0].Namespace != "default" {
		t.Errorf("namespace = %q, want 'default'", memories[0].Namespace)
	}
}

func TestCSVParser_MinimalColumns(t *testing.T) {
	input := "title,content\nmy mem,some content\n"
	parser := &transfer.CSVParser{}
	memories, err := parser.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(memories) != 1 {
		t.Fatalf("got %d memories, want 1", len(memories))
	}
	if memories[0].Content != "some content" {
		t.Errorf("content = %q, want 'some content'", memories[0].Content)
	}
	if memories[0].Namespace != "default" {
		t.Errorf("namespace = %q, want 'default'", memories[0].Namespace)
	}
	if memories[0].Type != "semantic" {
		t.Errorf("type = %q, want 'semantic'", memories[0].Type)
	}
}

func TestCSVFormatter(t *testing.T) {
	memories := []transfer.Memory{
		{Content: "hello", Namespace: "default", Type: "semantic"},
	}
	formatter := &transfer.CSVFormatter{}
	output, err := formatter.Format(memories)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	s := string(output)
	if !strings.Contains(s, "hello") {
		t.Error("output missing 'hello'")
	}
	if !strings.Contains(s, "title,content,namespace,type,metadata") {
		t.Error("output missing CSV header")
	}
}

func TestCSV_RoundTrip(t *testing.T) {
	original := []transfer.Memory{
		{Content: "round trip csv", Namespace: "work", Type: "episodic"},
	}
	formatter := &transfer.CSVFormatter{}
	exported, err := formatter.Format(original)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}

	parser := &transfer.CSVParser{}
	imported, err := parser.Parse(exported)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(imported) != 1 {
		t.Fatalf("got %d memories, want 1", len(imported))
	}
	if imported[0].Content != "round trip csv" {
		t.Errorf("content = %q, want 'round trip csv'", imported[0].Content)
	}
	if imported[0].Namespace != "work" {
		t.Errorf("namespace = %q, want 'work'", imported[0].Namespace)
	}
}
