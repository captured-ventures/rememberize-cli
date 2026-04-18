package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteMCPConfig_CreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")

	err := writeMCPConfig(path, "uid.secret", "https://rememberize.app")
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	servers, ok := cfg["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("mcpServers missing: %v", cfg)
	}
	rm, ok := servers["rememberize"].(map[string]any)
	if !ok {
		t.Fatalf("rememberize entry missing")
	}
	// Remote MCP shape: type + url + headers (streamable HTTP transport).
	// type:"http" is required — without it Claude Code defaults to stdio
	// and /doctor errors on the entry.
	if rm["type"] != "http" {
		t.Errorf("type = %v, want http", rm["type"])
	}
	if rm["url"] != "https://rememberize.app/mcp" {
		t.Errorf("url = %v, want https://rememberize.app/mcp", rm["url"])
	}
	headers, ok := rm["headers"].(map[string]any)
	if !ok {
		t.Fatalf("headers missing")
	}
	if headers["Authorization"] != "Bearer uid.secret" {
		t.Errorf("Authorization header = %v, want 'Bearer uid.secret'", headers["Authorization"])
	}
	// The old stdio-wrapper fields must NOT be present.
	if _, exists := rm["command"]; exists {
		t.Errorf("legacy `command` field should not be present in remote MCP config")
	}
	if _, exists := rm["env"]; exists {
		t.Errorf("legacy `env` field should not be present in remote MCP config")
	}
}

func TestWriteMCPConfig_MergesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".mcp.json")

	existing := `{
  "mcpServers": {
    "other-server": {
      "command": "node",
      "args": ["other.js"]
    }
  }
}`
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	err := writeMCPConfig(path, "uid.secret", "https://rememberize.app")
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	data, _ := os.ReadFile(path)
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal written config: %v", err)
	}

	servers := cfg["mcpServers"].(map[string]any)
	if _, ok := servers["other-server"]; !ok {
		t.Errorf("existing server lost")
	}
	if _, ok := servers["rememberize"]; !ok {
		t.Errorf("rememberize not added")
	}
}
