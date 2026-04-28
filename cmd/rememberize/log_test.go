package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/log"
)

// TestConfigureLogger_LevelMatrix exercises the verbose/quiet/json
// precedence rules in configureLogger. It rewires `logger` to a
// captured buffer, sets the global flag vars, then probes which
// levels emit.
func TestConfigureLogger_LevelMatrix(t *testing.T) {
	cases := []struct {
		name        string
		verbose     int
		quiet       bool
		json        bool
		wantDebug   bool
		wantInfo    bool
		wantWarn    bool
		wantError   bool
	}{
		{name: "default", verbose: 0, wantWarn: true, wantError: true},
		{name: "v1_info", verbose: 1, wantInfo: true, wantWarn: true, wantError: true},
		{name: "v2_debug", verbose: 2, wantDebug: true, wantInfo: true, wantWarn: true, wantError: true},
		{name: "v3_still_debug", verbose: 3, wantDebug: true, wantInfo: true, wantWarn: true, wantError: true},
		{name: "quiet_overrides_v2", verbose: 2, quiet: true, wantError: true},
		{name: "quiet_alone", quiet: true, wantError: true},
		{name: "json_silences_default", json: true, wantError: false},
		{name: "json_with_v1_logs_at_info", verbose: 1, json: true, wantInfo: true, wantWarn: true, wantError: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Save + restore globals so test order is irrelevant.
			origVerbose, origQuiet, origJSON := verboseCount, quiet, jsonOutput
			origLogger := logger
			t.Cleanup(func() {
				verboseCount, quiet, jsonOutput = origVerbose, origQuiet, origJSON
				logger = origLogger
			})

			verboseCount, quiet, jsonOutput = tc.verbose, tc.quiet, tc.json
			var buf bytes.Buffer
			logger = log.NewWithOptions(&buf, log.Options{
				Level:           log.WarnLevel,
				ReportTimestamp: false,
			})
			configureLogger()

			logger.Debug("dbg-marker")
			logger.Info("info-marker")
			logger.Warn("warn-marker")
			logger.Error("err-marker")

			out := buf.String()
			check := func(label, marker string, want bool) {
				got := strings.Contains(out, marker)
				if got != want {
					t.Errorf("%s: want=%v got=%v\noutput:\n%s", label, want, got, out)
				}
			}
			check("debug", "dbg-marker", tc.wantDebug)
			check("info", "info-marker", tc.wantInfo)
			check("warn", "warn-marker", tc.wantWarn)
			check("error", "err-marker", tc.wantError)
		})
	}
}
