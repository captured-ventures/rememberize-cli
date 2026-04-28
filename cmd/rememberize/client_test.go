package main

import (
	"strings"
	"testing"
)

// TestNewClientFromConfig_RequiresAPIKey guards the preflight: without
// a configured key, the CLI must error before any HTTP call goes out.
// Naked unauthenticated requests would either get rejected by the
// server (best case) or, worse, hit an endpoint the server didn't
// mean to expose. The CLI is the last place to catch that.
func TestNewClientFromConfig_RequiresAPIKey(t *testing.T) {
	cfg := &Config{
		Auth: AuthConfig{
			APIURL: "https://platform.rememberize.app",
			APIKey: "",
		},
	}
	c, err := newClientFromConfig(cfg)
	if err == nil {
		t.Fatalf("expected error for missing api key, got nil (client: %+v)", c)
	}
	if !strings.Contains(err.Error(), "no API key configured") {
		t.Errorf("error message should mention missing key, got: %v", err)
	}
	if !strings.Contains(err.Error(), "rememberize pair") {
		t.Errorf("error message should suggest 'rememberize pair', got: %v", err)
	}
	if !strings.Contains(err.Error(), "REMEMBERIZE_API_KEY") {
		t.Errorf("error message should mention env var fallback, got: %v", err)
	}
}

// TestNewClientFromConfig_RequiresAPIURL covers the second precondition.
// A key with no server URL would compose a request against "/api/foo"
// (no host), which fails opaquely. Pre-flight surfaces it cleanly.
func TestNewClientFromConfig_RequiresAPIURL(t *testing.T) {
	cfg := &Config{
		Auth: AuthConfig{
			APIURL: "",
			APIKey: "01ABCDEF.somefakekey",
		},
	}
	c, err := newClientFromConfig(cfg)
	if err == nil {
		t.Fatalf("expected error for missing api url, got nil (client: %+v)", c)
	}
	if !strings.Contains(err.Error(), "no server URL configured") {
		t.Errorf("error message should mention missing url, got: %v", err)
	}
}

// TestNewClientFromConfig_HappyPath verifies a well-formed config
// produces a usable Client and no error.
func TestNewClientFromConfig_HappyPath(t *testing.T) {
	cfg := &Config{
		Auth: AuthConfig{
			APIURL: "https://platform.rememberize.app",
			APIKey: "01ABCDEF.somefakekey",
		},
	}
	c, err := newClientFromConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.BaseURL != cfg.Auth.APIURL {
		t.Errorf("BaseURL: want %q got %q", cfg.Auth.APIURL, c.BaseURL)
	}
	if c.APIKey != cfg.Auth.APIKey {
		t.Errorf("APIKey: want %q got %q", cfg.Auth.APIKey, c.APIKey)
	}
	if c.HTTP == nil {
		t.Error("HTTP client should not be nil")
	}
}
