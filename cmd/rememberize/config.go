package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all CLI configuration values.
type Config struct {
	Auth     AuthConfig     `toml:"auth"`
	Defaults DefaultsConfig `toml:"defaults"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	APIURL string `toml:"api_url"`
	APIKey string `toml:"api_key"`
}

// DefaultsConfig holds default values for commands.
type DefaultsConfig struct {
	Namespace string `toml:"namespace"`
	Type      string `toml:"type"`
	Format    string `toml:"format"`
}

// configDir returns ~/.rememberize.
func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".rememberize"
	}
	return filepath.Join(home, ".rememberize")
}

// configPath returns ~/.rememberize/config.toml.
func configPath() string {
	return filepath.Join(configDir(), "config.toml")
}

// defaultConfig returns a Config with sensible defaults.
func defaultConfig() *Config {
	return &Config{
		Auth: AuthConfig{
			APIURL: "http://localhost:8080",
			APIKey: "",
		},
		Defaults: DefaultsConfig{
			Namespace: "default",
			Type:      "semantic",
			Format:    "table",
		},
	}
}

// loadConfig reads config from file and applies env overrides.
func loadConfig() *Config {
	cfg := defaultConfig()

	// Read from file if it exists.
	path := configPath()
	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not parse config file: %v\n", err)
		}
	}

	// Env overrides take precedence.
	if v := os.Getenv("REMEMBERIZE_API_URL"); v != "" {
		cfg.Auth.APIURL = v
	}
	if v := os.Getenv("REMEMBERIZE_API_KEY"); v != "" {
		cfg.Auth.APIKey = v
	}

	return cfg
}

// saveConfig writes the config to disk, creating the directory if needed.
func saveConfig(cfg *Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	f, err := os.Create(configPath())
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// setConfigValue updates a single config key by dotted path (e.g. "auth.api_key").
func setConfigValue(cfg *Config, key, value string) error {
	switch key {
	case "auth.api_url":
		cfg.Auth.APIURL = value
	case "auth.api_key":
		cfg.Auth.APIKey = value
	case "defaults.namespace":
		cfg.Defaults.Namespace = value
	case "defaults.type":
		cfg.Defaults.Type = value
	case "defaults.format":
		cfg.Defaults.Format = value
	default:
		return fmt.Errorf("unknown config key: %s\nvalid keys: auth.api_url, auth.api_key, defaults.namespace, defaults.type, defaults.format", key)
	}
	return nil
}
