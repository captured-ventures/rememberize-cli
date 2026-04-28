package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestRenderTable_NonTTYFallback verifies the tabwriter fallback path.
// A *bytes.Buffer is never a TTY, so renderTable must produce parseable
// tab-aligned output (not lipgloss borders) suitable for cut/awk/etc.
func TestRenderTable_NonTTYFallback(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"ID", "NAME", "TYPE"}
	rows := [][]string{
		{"01ABC", "alpha", "semantic"},
		{"02DEF", "beta", "episodic"},
	}

	renderTable(&buf, headers, rows)

	out := buf.String()

	// Header row + 2 data rows means the tabwriter rendered.
	if !strings.Contains(out, "ID") || !strings.Contains(out, "NAME") || !strings.Contains(out, "TYPE") {
		t.Errorf("missing headers in output:\n%s", out)
	}
	if !strings.Contains(out, "01ABC") || !strings.Contains(out, "02DEF") {
		t.Errorf("missing data rows in output:\n%s", out)
	}

	// Tabwriter output should NOT contain lipgloss border glyphs.
	for _, glyph := range []string{"╭", "╮", "╰", "╯", "│", "─"} {
		if strings.Contains(out, glyph) {
			t.Errorf("non-TTY output should not contain border glyph %q:\n%s", glyph, out)
		}
	}

	// Two-space padding between columns is the tabwriter signature.
	// "ID" header should be followed by spaces, then "NAME".
	if !strings.Contains(out, "ID  ") && !strings.Contains(out, "ID\tNAME") {
		t.Errorf("expected tabwriter spacing between ID and NAME columns:\n%s", out)
	}
}

// TestRenderTable_EmptyRows passes zero rows. Should still emit the
// header row without panicking.
func TestRenderTable_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	renderTable(&buf, []string{"A", "B"}, nil)
	out := buf.String()
	if !strings.Contains(out, "A") || !strings.Contains(out, "B") {
		t.Errorf("empty-rows table missing headers:\n%s", out)
	}
}

// TestIsTTY_BytesBufferIsNotTTY guards the type-assertion path.
// *bytes.Buffer is not *os.File so isTTY must short-circuit to false.
func TestIsTTY_BytesBufferIsNotTTY(t *testing.T) {
	var buf bytes.Buffer
	if isTTY(&buf) {
		t.Error("isTTY should return false for *bytes.Buffer")
	}
}
