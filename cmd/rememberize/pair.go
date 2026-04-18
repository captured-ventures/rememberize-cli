package main

import (
	"bufio"
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

// pairStdin is the reader prompts read from. Swappable so tests can
// feed canned input without touching os.Stdin.
var pairStdin io.Reader = os.Stdin

// pairStdout is the writer prompt text goes to. Swappable for tests.
var pairStdout io.Writer = os.Stdout

const (
	productionServerURL = "https://platform.rememberize.app"
	localDevServerURL   = "http://localhost:8080"
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

	cfg := loadConfig()

	// F4: Resolve the server URL.
	//   1. --server flag
	//   2. config auth.api_url (if non-empty)
	//   3. interactive prompt — persisted before we proceed so a mid-flow
	//      Ctrl-C doesn't strand the user with nothing on retry.
	server := pairServer
	if server == "" && cfg.Auth.APIURL != "" {
		server = cfg.Auth.APIURL
	}
	if server == "" {
		chosen, err := promptForServerURL(pairStdin, pairStdout)
		if err != nil {
			return err
		}
		server = chosen
		if err := setConfigValue(cfg, "auth.api_url", server); err != nil {
			return fmt.Errorf("stash server url: %w", err)
		}
		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
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

	// F4: always persist the server-returned canonical URL. Covers the case
	// where the user typed a URL that redirected (X-Forwarded-Proto, custom
	// domain) — from here on, CLI talks to whatever the server told us is
	// the real origin.
	if result.ServerURL != "" && result.ServerURL != cfg.Auth.APIURL {
		_ = setConfigValue(cfg, "auth.api_url", result.ServerURL)
		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("save server url: %w", err)
		}
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

// promptForServerURL asks the user which server to pair against and
// returns the resolved URL. Default on empty input is the production
// platform. Choice "3" (or any non-1/2) prompts for a custom URL.
func promptForServerURL(in io.Reader, out io.Writer) (string, error) {
	fmt.Fprintln(out, "No server configured. Pair against:")
	fmt.Fprintln(out, "  1. platform.rememberize.app  (production, recommended)")
	fmt.Fprintln(out, "  2. http://localhost:8080      (local dev)")
	fmt.Fprintln(out, "  3. Other (enter URL)")
	fmt.Fprint(out, "Choice [1]: ")

	reader := bufio.NewReader(in)
	choice, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read choice: %w", err)
	}
	choice = strings.TrimSpace(choice)

	switch choice {
	case "", "1":
		return productionServerURL, nil
	case "2":
		return localDevServerURL, nil
	case "3":
		fmt.Fprint(out, "Server URL: ")
		raw, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("read server url: %w", err)
		}
		url := strings.TrimSpace(raw)
		if url == "" {
			return "", fmt.Errorf("server URL required")
		}
		return url, nil
	default:
		// Treat any other input as a pasted URL (common shortcut when the
		// user has the URL on their clipboard and skips the menu).
		if strings.Contains(choice, "://") {
			return choice, nil
		}
		return "", fmt.Errorf("invalid choice: %q (expected 1, 2, 3, or a URL)", choice)
	}
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
