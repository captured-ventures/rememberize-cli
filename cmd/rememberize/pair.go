package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	pairClient string
	pairConfig string
	pairServer string
)

var pairCmd = &cobra.Command{
	Use:   "pair <code>",
	Short: "Pair a client by exchanging a one-time code for a permanent credential",
	Long: `Exchange a 6-character pairing code (generated in the rememberize web UI)
for a permanent API key, and write the appropriate client config file.

Supported clients:
  claude-code  — writes ./.mcp.json entry
  cursor       — writes Cursor's MCP config
  generic      — prints the key and server URL, no file write`,
	Args: cobra.ExactArgs(1),
	RunE: runPair,
}

func init() {
	pairCmd.Flags().StringVar(&pairClient, "client", "", "client type (claude-code, cursor, generic); autodetects if empty")
	pairCmd.Flags().StringVar(&pairConfig, "config", "", "override default config path")
	pairCmd.Flags().StringVar(&pairServer, "server", "", "override server URL (defaults to config)")
}

func runPair(cmd *cobra.Command, args []string) error {
	code := strings.ToUpper(strings.TrimSpace(args[0]))

	server := pairServer
	if server == "" {
		cfg := loadConfig()
		if cfg != nil && cfg.Auth.APIURL != "" {
			server = cfg.Auth.APIURL
		}
	}
	if server == "" {
		server = "https://rememberize.app"
	}

	// Build client_name from hostname if we can
	hostname, _ := os.Hostname()
	clientName := ""
	if hostname != "" {
		clientName = fmt.Sprintf("%s (%s)", clientTypeOrHint(), hostname)
	}

	// POST to /api/pair/exchange
	body, _ := json.Marshal(map[string]string{
		"code":        code,
		"client_name": clientName,
	})
	resp, err := http.Post(server+"/api/pair/exchange", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	rawBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return humanReadableExchangeError(resp.StatusCode, rawBody)
	}

	var result struct {
		APIKey     string `json:"api_key"`
		ServerURL  string `json:"server_url"`
		Connection struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"connection"`
		Namespaces []struct {
			Name string `json:"name"`
		} `json:"namespaces"`
	}
	if err := json.Unmarshal(rawBody, &result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	// Determine client type and write config
	client := pairClient
	if client == "" {
		client = detectClient()
	}

	switch client {
	case "claude-code":
		path := pairConfig
		if path == "" {
			path = ".mcp.json"
		}
		if err := writeMCPConfig(path, result.APIKey, result.ServerURL); err != nil {
			return fmt.Errorf("write mcp config: %w", err)
		}
		fmt.Printf("Paired claude-code -> %s\n", result.ServerURL)
		fmt.Printf("  Config written: %s\n", path)
		fmt.Printf("  Connection:     %s\n", result.Connection.Name)
	case "cursor":
		path := pairConfig
		if path == "" {
			path = cursorConfigPath()
		}
		if err := writeMCPConfig(path, result.APIKey, result.ServerURL); err != nil {
			return fmt.Errorf("write cursor config: %w", err)
		}
		fmt.Printf("Paired cursor -> %s\n", result.ServerURL)
		fmt.Printf("  Config written: %s\n", path)
	default: // generic
		fmt.Printf("Paired -> %s\n", result.ServerURL)
		fmt.Printf("\n  REMEMBERIZE_API_KEY=%s\n", result.APIKey)
		fmt.Printf("  REMEMBERIZE_SERVER=%s\n\n", result.ServerURL)
		fmt.Println("  Use Authorization: Bearer <key> in REST requests.")
	}

	return nil
}

// clientTypeOrHint returns a short label for the client being paired.
// Used in the client_name string sent with the exchange request.
func clientTypeOrHint() string {
	if pairClient != "" {
		return pairClient
	}
	detected := detectClient()
	if detected != "" {
		return detected
	}
	return "client"
}

// detectClient inspects the current directory for hints about which client
// is being paired. Returns "claude-code" if .mcp.json exists; otherwise
// falls back to "generic".
func detectClient() string {
	if _, err := os.Stat(".mcp.json"); err == nil {
		return "claude-code"
	}
	return "generic"
}

// cursorConfigPath returns the platform-appropriate Cursor MCP config path.
func cursorConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".cursor-mcp.json"
	}
	return filepath.Join(home, ".cursor", "mcp.json")
}

// writeMCPConfig merges a rememberize entry into an MCP JSON config file.
// Creates the file if missing; preserves any existing mcpServers entries.
//
// Emits the streamable-HTTP remote-MCP shape: {url, headers}. This is the
// transport exposed by the rememberize server — no local process spawn, no
// npm package. A prior version of this function emitted a stdio wrapper
// pointing at an npm package that never existed; the server has since grown
// native HTTP MCP support.
func writeMCPConfig(path, apiKey, serverURL string) error {
	var cfg map[string]any

	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("parse existing config: %w", err)
		}
	}
	if cfg == nil {
		cfg = map[string]any{}
	}

	servers, _ := cfg["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}

	servers["rememberize"] = map[string]any{
		"type": "http",
		"url":  strings.TrimRight(serverURL, "/") + "/mcp",
		"headers": map[string]any{
			"Authorization": "Bearer " + apiKey,
		},
	}
	cfg["mcpServers"] = servers

	// Ensure parent dir exists
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
	}

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// humanReadableExchangeError maps HTTP response codes to user-facing messages.
func humanReadableExchangeError(status int, body []byte) error {
	switch status {
	case http.StatusNotFound:
		return fmt.Errorf("pairing code not found — check the code in the rememberize web UI and try again")
	case http.StatusGone:
		return fmt.Errorf("pairing code expired or locked — regenerate one in the rememberize web UI")
	case http.StatusConflict:
		return fmt.Errorf("pairing code already used")
	case http.StatusBadRequest:
		return fmt.Errorf("invalid pairing code format")
	case http.StatusTooManyRequests:
		return fmt.Errorf("too many attempts — wait a minute and try again")
	default:
		return fmt.Errorf("unexpected error (status %d): %s", status, string(body))
	}
}
