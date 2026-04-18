package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// withIsolatedHome redirects the CLI's config location to a temp dir so a
// test cannot write into the user's real ~/.rememberize/. HOME covers
// Unix; USERPROFILE covers Windows (os.UserHomeDir semantics).
func withIsolatedHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	// REMEMBERIZE_* env vars can silently override config values; clear
	// them so the test sees only what it explicitly writes.
	t.Setenv("REMEMBERIZE_API_URL", "")
	t.Setenv("REMEMBERIZE_API_KEY", "")
	return dir
}

// stubExchangeServer returns an httptest server that records the last
// exchange request body and replies with the supplied response struct.
type recordedRequest struct {
	Body map[string]any
}

type exchangeResponse struct {
	APIKey     string                  `json:"api_key"`
	ServerURL  string                  `json:"server_url"`
	Connection exchangeRespConnection  `json:"connection"`
	Namespaces []exchangeRespNamespace `json:"namespaces"`
}

type exchangeRespConnection struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	ConfigTarget string `json:"config_target"`
}

type exchangeRespNamespace struct {
	Name string `json:"name"`
}

func stubExchangeServer(t *testing.T, resp exchangeResponse, rec *recordedRequest) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/pair/exchange", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if rec != nil {
			_ = json.Unmarshal(body, &rec.Body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
	return httptest.NewServer(mux)
}

// resetPairState clears mutable package-level state between tests so they
// don't leak server URLs or stdin readers into each other.
func resetPairState(t *testing.T) {
	t.Helper()
	oldServer := pairServer
	oldConfig := pairConfig
	oldStdin := pairStdin
	oldStdout := pairStdout
	t.Cleanup(func() {
		pairServer = oldServer
		pairConfig = oldConfig
		pairStdin = oldStdin
		pairStdout = oldStdout
	})
	pairServer = ""
	pairConfig = ""
	pairStdin = strings.NewReader("")
	pairStdout = io.Discard
}

// ---------------------------------------------------------------------------
// F4: prompt for server when config empty
// ---------------------------------------------------------------------------

// Prompt resolution is tested as a unit — running the full pair flow
// against the real production/localhost URLs would hit the network.
// Full-flow persistence is covered by TestPair_PromptPersistsChoice.
func TestPair_PromptsForServerWhenConfigEmpty(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantURL string
	}{
		{"default-production-on-empty", "\n", productionServerURL},
		{"explicit-choice-1", "1\n", productionServerURL},
		{"localhost-choice-2", "2\n", localDevServerURL},
		{"pasted-url", "https://my.server.example/\n", "https://my.server.example/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			got, err := promptForServerURL(strings.NewReader(tc.input), &buf)
			if err != nil {
				t.Fatalf("promptForServerURL: %v", err)
			}
			if got != tc.wantURL {
				t.Errorf("input %q: got %q, want %q", tc.input, got, tc.wantURL)
			}
		})
	}
}

// TestPair_PersistsServerReturnedURL verifies that after a successful
// exchange, the canonical server_url returned by the server wins over
// whatever URL the caller typed or the --server flag supplied. This is
// the F4 "redirect-aware" behavior.
func TestPair_PersistsServerReturnedURL(t *testing.T) {
	withIsolatedHome(t)
	resetPairState(t)

	server := stubExchangeServer(t, exchangeResponse{
		APIKey:    "persisted.key",
		ServerURL: "https://canonical.example.com",
		Connection: exchangeRespConnection{
			Name:         "CLI (box)",
			Type:         "cli",
			ConfigTarget: "cli",
		},
	}, nil)
	defer server.Close()

	pairServer = server.URL
	pairStdout = io.Discard

	if err := runPair(pairCmd, []string{"ABC123"}); err != nil {
		t.Fatalf("runPair: %v", err)
	}

	cfg := loadConfig()
	if cfg.Auth.APIURL != "https://canonical.example.com" {
		t.Errorf("config APIURL = %q, want the server-returned canonical URL", cfg.Auth.APIURL)
	}
}

// ---------------------------------------------------------------------------
// F7+F9: hostname + sentinel client_name, use server-returned name and
// config_target
// ---------------------------------------------------------------------------

func TestPair_SendsHostnameAndSentinelClientName(t *testing.T) {
	withIsolatedHome(t)
	resetPairState(t)

	rec := &recordedRequest{}
	server := stubExchangeServer(t, exchangeResponse{
		APIKey:    "k",
		ServerURL: "http://does-not-matter",
		Connection: exchangeRespConnection{
			Name:         "Claude Code (host)",
			Type:         "mcp",
			ConfigTarget: "cli", // write nothing so test is hermetic
		},
	}, rec)
	defer server.Close()

	pairServer = server.URL
	pairStdout = io.Discard

	if err := runPair(pairCmd, []string{"CODE01"}); err != nil {
		t.Fatalf("runPair: %v", err)
	}
	if got := rec.Body["client_name"]; got != "@auto" {
		t.Errorf("client_name = %v, want \"@auto\"", got)
	}
	host, ok := rec.Body["hostname"].(string)
	if !ok || host == "" {
		t.Errorf("hostname missing or empty: %v", rec.Body["hostname"])
	}
}

func TestPair_UsesServerReturnedConnectionName(t *testing.T) {
	withIsolatedHome(t)
	resetPairState(t)

	server := stubExchangeServer(t, exchangeResponse{
		APIKey:    "k",
		ServerURL: "http://server",
		Connection: exchangeRespConnection{
			Name:         "Claude Code (testhost)",
			Type:         "mcp",
			ConfigTarget: "cli",
		},
	}, nil)
	defer server.Close()

	var buf bytes.Buffer
	pairServer = server.URL
	pairStdout = &buf

	if err := runPair(pairCmd, []string{"CODE01"}); err != nil {
		t.Fatalf("runPair: %v", err)
	}
	if !strings.Contains(buf.String(), "Claude Code (testhost)") {
		t.Errorf("output did not contain server-returned connection name.\ngot: %s", buf.String())
	}
}

// ---------------------------------------------------------------------------
// F7+F9: dispatch by config_target
// ---------------------------------------------------------------------------

func TestPair_WritesPerConfigTarget(t *testing.T) {
	type check struct {
		name       string
		target     string
		wantFile   func(home string) string // "" means no file expected
		wantStdout string
	}
	cases := []check{
		{
			name:       "claude-code writes .mcp.json in cwd",
			target:     "claude-code",
			wantFile:   func(home string) string { return "" }, // in cwd, verified separately
			wantStdout: "Config written:",
		},
		{
			name:   "cursor writes ~/.cursor/mcp.json",
			target: "cursor",
			wantFile: func(home string) string {
				return filepath.Join(home, ".cursor", "mcp.json")
			},
			wantStdout: "Config written:",
		},
		{
			name:       "cli writes no MCP file",
			target:     "cli",
			wantFile:   func(home string) string { return "" },
			wantStdout: "Paired as",
		},
		{
			name:       "generic prints paste block",
			target:     "generic",
			wantFile:   func(home string) string { return "" },
			wantStdout: "paste this into your MCP client",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := withIsolatedHome(t)
			resetPairState(t)

			// For claude-code, chdir to a tempdir so .mcp.json lands somewhere hermetic.
			var mcpDir string
			if tc.target == "claude-code" {
				mcpDir = t.TempDir()
				oldWD, _ := os.Getwd()
				if err := os.Chdir(mcpDir); err != nil {
					t.Fatalf("chdir: %v", err)
				}
				t.Cleanup(func() { _ = os.Chdir(oldWD) })
			}

			server := stubExchangeServer(t, exchangeResponse{
				APIKey:    "bearer.key",
				ServerURL: "https://example.test",
				Connection: exchangeRespConnection{
					Name:         "X (host)",
					Type:         "mcp",
					ConfigTarget: tc.target,
				},
			}, nil)
			defer server.Close()

			var buf bytes.Buffer
			pairServer = server.URL
			pairStdout = &buf

			if err := runPair(pairCmd, []string{"CODE01"}); err != nil {
				t.Fatalf("runPair: %v", err)
			}

			if !strings.Contains(buf.String(), tc.wantStdout) {
				t.Errorf("stdout missing %q\ngot: %s", tc.wantStdout, buf.String())
			}

			// Verify file writes.
			switch tc.target {
			case "claude-code":
				data, err := os.ReadFile(filepath.Join(mcpDir, ".mcp.json"))
				if err != nil {
					t.Fatalf("expected .mcp.json in cwd: %v", err)
				}
				if !strings.Contains(string(data), "bearer.key") {
					t.Errorf(".mcp.json missing API key\ngot: %s", data)
				}
			case "cursor":
				path := tc.wantFile(home)
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("expected cursor config at %s: %v", path, err)
				}
				if !strings.Contains(string(data), "bearer.key") {
					t.Errorf("cursor config missing API key\ngot: %s", data)
				}
			case "cli", "generic":
				// No MCP file should exist anywhere in HOME.
				// (The CLI's own config.toml will exist — that's F12, checked separately.)
				if _, err := os.Stat(filepath.Join(home, ".cursor", "mcp.json")); err == nil {
					t.Errorf("unexpected cursor config written for target=%q", tc.target)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// F12: API key persisted to CLI config regardless of target
// ---------------------------------------------------------------------------

func TestPair_PersistsAPIKeyToCLIConfig(t *testing.T) {
	targets := []string{"claude-code", "cursor", "cli", "generic"}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			home := withIsolatedHome(t)
			resetPairState(t)

			// Isolate .mcp.json in cwd for claude-code target.
			if target == "claude-code" {
				cwd := t.TempDir()
				oldWD, _ := os.Getwd()
				_ = os.Chdir(cwd)
				t.Cleanup(func() { _ = os.Chdir(oldWD) })
			}

			server := stubExchangeServer(t, exchangeResponse{
				APIKey:    "persist.this.key",
				ServerURL: "https://example.test",
				Connection: exchangeRespConnection{
					Name:         "X (host)",
					Type:         "mcp",
					ConfigTarget: target,
				},
			}, nil)
			defer server.Close()

			pairServer = server.URL
			pairStdout = io.Discard

			if err := runPair(pairCmd, []string{"CODE01"}); err != nil {
				t.Fatalf("runPair: %v", err)
			}

			cfgPath := filepath.Join(home, ".rememberize", "config.toml")
			data, err := os.ReadFile(cfgPath)
			if err != nil {
				t.Fatalf("expected CLI config at %s: %v", cfgPath, err)
			}
			if !strings.Contains(string(data), "persist.this.key") {
				t.Errorf("CLI config missing API key for target=%q\ngot: %s", target, data)
			}
			_ = runtime.GOOS // keep import clean if we ever need platform branching
		})
	}
}

// ---------------------------------------------------------------------------
// F6: namespace prompt sets default
// ---------------------------------------------------------------------------

func TestPair_NamespacePromptSetsDefault(t *testing.T) {
	home := withIsolatedHome(t)
	resetPairState(t)

	server := stubExchangeServer(t, exchangeResponse{
		APIKey:    "k",
		ServerURL: "https://example.test",
		Connection: exchangeRespConnection{
			Name:         "CLI (host)",
			Type:         "cli",
			ConfigTarget: "cli",
		},
		Namespaces: []exchangeRespNamespace{
			{Name: "work"},
			{Name: "home"},
			{Name: "research"},
		},
	}, nil)
	defer server.Close()

	pairServer = server.URL
	pairStdin = strings.NewReader("2\n") // pick "home"
	pairStdout = io.Discard

	if err := runPair(pairCmd, []string{"CODE01"}); err != nil {
		t.Fatalf("runPair: %v", err)
	}

	cfg := loadConfig()
	if cfg.Defaults.Namespace != "home" {
		t.Errorf("defaults.namespace = %q, want %q", cfg.Defaults.Namespace, "home")
	}
	_ = home
}

func TestPair_NamespacePromptSkipOnEmptyInput(t *testing.T) {
	withIsolatedHome(t)
	resetPairState(t)

	server := stubExchangeServer(t, exchangeResponse{
		APIKey:    "k",
		ServerURL: "https://example.test",
		Connection: exchangeRespConnection{
			Name:         "CLI (host)",
			Type:         "cli",
			ConfigTarget: "cli",
		},
		Namespaces: []exchangeRespNamespace{
			{Name: "work"},
		},
	}, nil)
	defer server.Close()

	pairServer = server.URL
	pairStdin = strings.NewReader("\n") // empty → skip
	pairStdout = io.Discard

	if err := runPair(pairCmd, []string{"CODE01"}); err != nil {
		t.Fatalf("runPair: %v", err)
	}

	cfg := loadConfig()
	if cfg.Defaults.Namespace != "default" {
		t.Errorf("defaults.namespace = %q, want untouched default", cfg.Defaults.Namespace)
	}
}
