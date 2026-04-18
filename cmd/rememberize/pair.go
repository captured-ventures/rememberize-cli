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
for a permanent API key. Which local config file (if any) is written is
decided server-side from the pairing code's stored client hint:

  claude-code  — writes ./.mcp.json
  cursor       — writes ~/.cursor/mcp.json
  cli          — writes only the CLI's own ~/.rememberize/config.toml
  generic      — prints an MCP config block to stdout`,
	Args: cobra.ExactArgs(1),
	RunE: runPair,
}

func init() {
	pairCmd.Flags().StringVar(&pairConfig, "config", "", "override default config path for the written MCP file")
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

	// F7+F9: send the hostname as a discrete field and mark client_name
	// with the "@auto" sentinel. The server composes the display name from
	// its stored client_hint ("Claude Code", "Cursor", "CLI", ...) + this
	// hostname, and includes a config_target in the response telling us
	// which local config file (if any) to write.
	hostname, _ := os.Hostname()
	body, _ := json.Marshal(map[string]string{
		"code":        code,
		"client_name": "@auto",
		"hostname":    hostname,
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
			Name         string `json:"name"`
			Type         string `json:"type"`
			ConfigTarget string `json:"config_target"`
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

	// F12: always persist the API key to the CLI's own config first, before
	// any MCP-file write or namespace prompt. Losing power mid-prompt used
	// to mean the user had no key anywhere and had to re-pair; now the key
	// survives whatever happens after this point.
	if result.APIKey != "" {
		_ = setConfigValue(cfg, "auth.api_key", result.APIKey)
		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("save api key: %w", err)
		}
	}

	// F7+F9: dispatch by the server-returned config_target. The server
	// knows, from the pairing code's stored client_hint, which integration
	// the user was targeting — trust it instead of re-sniffing cwd.
	target := result.Connection.ConfigTarget
	if target == "" {
		target = "generic"
	}

	switch target {
	case "claude-code":
		path := pairConfig
		if path == "" {
			path = ".mcp.json"
		}
		if err := writeMCPConfig(path, result.APIKey, result.ServerURL); err != nil {
			return fmt.Errorf("write mcp config: %w", err)
		}
		fmt.Fprintf(pairStdout, "Paired as %s -> %s\n", result.Connection.Name, result.ServerURL)
		fmt.Fprintf(pairStdout, "  Config written: %s\n", path)
	case "cursor":
		path := pairConfig
		if path == "" {
			path = cursorConfigPath()
		}
		if err := writeMCPConfig(path, result.APIKey, result.ServerURL); err != nil {
			return fmt.Errorf("write cursor config: %w", err)
		}
		fmt.Fprintf(pairStdout, "Paired as %s -> %s\n", result.Connection.Name, result.ServerURL)
		fmt.Fprintf(pairStdout, "  Config written: %s\n", path)
	case "cli":
		// The CLI's own config gets the API key below (F12); no MCP file.
		fmt.Fprintf(pairStdout, "Paired as %s -> %s\n", result.Connection.Name, result.ServerURL)
	default: // "generic"
		// Print the MCP config block so the user can paste it anywhere.
		fmt.Fprintf(pairStdout, "Paired as %s -> %s\n", result.Connection.Name, result.ServerURL)
		fmt.Fprintln(pairStdout, "\nNo known integration — paste this into your MCP client:")
		fmt.Fprintf(pairStdout, "{\n  \"mcpServers\": {\n    \"rememberize\": {\n      \"type\": \"http\",\n      \"url\": %q,\n      \"headers\": { \"Authorization\": \"Bearer %s\" }\n    }\n  }\n}\n",
			strings.TrimRight(result.ServerURL, "/")+"/mcp", result.APIKey)
	}

	// F6: offer to set a default namespace. Names are collected from the
	// exchange response so the user doesn't have to type them. Invalid or
	// empty input is a silent skip — this is opt-in, not a gate.
	if len(result.Namespaces) > 0 {
		names := make([]string, len(result.Namespaces))
		for i, ns := range result.Namespaces {
			names[i] = ns.Name
		}
		if picked := promptForDefaultNamespace(pairStdin, pairStdout, names); picked != "" {
			_ = setConfigValue(cfg, "defaults.namespace", picked)
			if err := saveConfig(cfg); err != nil {
				return fmt.Errorf("save default namespace: %w", err)
			}
			fmt.Fprintf(pairStdout, "Default namespace set to %q.\n", picked)
		}
	}

	return nil
}

// promptForDefaultNamespace lists the user's namespaces and asks which
// (if any) to use as the CLI's default. Returns the chosen name, or ""
// if the user skipped or gave invalid input.
func promptForDefaultNamespace(in io.Reader, out io.Writer, names []string) string {
	fmt.Fprintln(out, "\nAvailable namespaces:")
	for i, name := range names {
		fmt.Fprintf(out, "  %d. %s\n", i+1, name)
	}
	fmt.Fprintf(out, "Set default namespace? [1-%d / enter to skip]: ", len(names))

	reader := bufio.NewReader(in)
	raw, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return ""
	}
	choice := strings.TrimSpace(raw)
	if choice == "" {
		return ""
	}
	// Accept either a 1-based index or the literal namespace name.
	for i, name := range names {
		if choice == name {
			return name
		}
		if choice == fmt.Sprintf("%d", i+1) {
			return name
		}
	}
	return ""
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
