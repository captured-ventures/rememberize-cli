package main

import (
	"os"

	"github.com/charmbracelet/log"
)

// logger is the package-level structured logger used by every command and
// helper. It writes to stderr — stdout is reserved for command output that
// users may pipe (lists, JSON, success lines).
//
// Configured by configureLogger() via cobra.OnInitialize after flag parse.
// The default below covers any call site that fires before flag parse —
// usually nothing, but a sane fallback beats a nil deref.
var logger = log.NewWithOptions(os.Stderr, log.Options{
	Level:           log.WarnLevel,
	ReportTimestamp: false,
})

// Persistent log flags (registered on rootCmd in main.go).
var (
	verboseCount int
	quiet        bool
)

// configureLogger sets the logger level and formatter from the parsed flag
// values. Called from cobra.OnInitialize so flags are guaranteed resolved.
//
// Level resolution (lowest precedence first):
//
//	default       → WarnLevel
//	-v            → InfoLevel
//	-vv (or more) → DebugLevel
//	--quiet       → ErrorLevel (overrides any -v)
//
// Output discipline:
//
//	--json without -v   → suppress all logs (stdout must stay pristine for piping)
//	--json with -v      → JSON-formatted logs to stderr
//	otherwise           → human-friendly text logs to stderr
func configureLogger() {
	level := log.WarnLevel
	switch {
	case quiet:
		level = log.ErrorLevel
	case verboseCount >= 2:
		level = log.DebugLevel
	case verboseCount == 1:
		level = log.InfoLevel
	}

	if jsonOutput && verboseCount == 0 && !quiet {
		// Pristine stdout is the contract for `--json`. Discard logs entirely
		// unless the user explicitly asked for them with -v.
		logger.SetLevel(log.FatalLevel + 1)
		return
	}

	logger.SetLevel(level)
	if jsonOutput {
		logger.SetFormatter(log.JSONFormatter)
	}
}
