#!/bin/sh
# bundle_test.sh — basic harness for bundle.sh
#
# Exits 0 on all-pass, 1 on any failure.
# Prints PASS/FAIL per check.

set -u

HERE=$(cd "$(dirname "$0")" && pwd)
BUNDLE="$HERE/bundle.sh"

if [ ! -f "$BUNDLE" ]; then
  echo "FAIL: bundle.sh not found at $BUNDLE"
  exit 1
fi

FAILS=0

assert_contains() {
  # $1 = file, $2 = pattern (fixed string), $3 = label
  if LC_ALL=C grep -qF -- "$2" "$1"; then
    echo "PASS: $3"
  else
    echo "FAIL: $3 — pattern '$2' not found in $1"
    FAILS=$((FAILS + 1))
  fi
}

assert_not_contains() {
  if LC_ALL=C grep -qF -- "$2" "$1"; then
    echo "FAIL: $3 — unexpected pattern '$2' in $1"
    FAILS=$((FAILS + 1))
  else
    echo "PASS: $3"
  fi
}

assert_exit_ok() {
  if [ "$1" -eq 0 ]; then
    echo "PASS: $2"
  else
    echo "FAIL: $2 — exit=$1"
    FAILS=$((FAILS + 1))
  fi
}

TMP=$(mktemp -d 2>/dev/null || mktemp -d -t bundletest)
trap 'rm -rf "$TMP"' EXIT INT TERM

# ---------------------------------------------------------------------------
# Case 1: fake Claude-project layout, branch A
# ---------------------------------------------------------------------------

PROJ="$TMP/projA"
mkdir -p "$PROJ/memory"

cat > "$PROJ/memory/MEMORY.md" <<'EOF'
# Memory Index
- [one.md](one.md) — first entry
- [two.md](two.md) — second entry
EOF

cat > "$PROJ/memory/one.md" <<'EOF'
---
name: one
type: feedback
---

Entry one body.
EOF

cat > "$PROJ/memory/two.md" <<'EOF'
---
name: two
type: project
---

Entry two body.
EOF

# A non-frontmatter file — should be skipped in branch A.
cat > "$PROJ/memory/plain.md" <<'EOF'
just a plain markdown note, no frontmatter.
EOF

OUT1="$TMP/out1.md"
sh "$BUNDLE" --claude-project "$PROJ" --out "$OUT1" >"$TMP/out1.log" 2>"$TMP/out1.err"
RC=$?
assert_exit_ok "$RC" "case1: branch A exits 0"

if [ -f "$OUT1" ]; then
  assert_contains "$OUT1" "rememberize-bundle" "case1: header present"
  assert_contains "$OUT1" "name: one" "case1: one.md frontmatter kept"
  assert_contains "$OUT1" "name: two" "case1: two.md frontmatter kept"
  assert_contains "$OUT1" "Entry one body." "case1: one.md body kept"
  assert_contains "$OUT1" "Entry two body." "case1: two.md body kept"
  assert_not_contains "$OUT1" "Memory Index" "case1: MEMORY.md excluded"
  assert_not_contains "$OUT1" "just a plain markdown note" "case1: non-frontmatter skipped"
else
  echo "FAIL: case1 output file missing"
  FAILS=$((FAILS + 1))
fi

# ---------------------------------------------------------------------------
# Case 2: multi-source with dedupe
# ---------------------------------------------------------------------------

PROJ2="$TMP/projB"
mkdir -p "$PROJ2/memory"
cat > "$PROJ2/memory/dup.md" <<'EOF'
---
name: one
type: feedback
---

Duplicate of one — should be dropped.
EOF

cat > "$PROJ2/memory/three.md" <<'EOF'
---
name: three
type: project
---

Entry three body.
EOF

OUT2="$TMP/out2.md"
sh "$BUNDLE" \
  --claude-project "$PROJ" \
  --claude-project "$PROJ2" \
  --out "$OUT2" >"$TMP/out2.log" 2>"$TMP/out2.err"
RC=$?
assert_exit_ok "$RC" "case2: branch B exits 0"

if [ -f "$OUT2" ]; then
  assert_contains "$OUT2" "name: one" "case2: one.md present"
  assert_contains "$OUT2" "name: two" "case2: two.md present"
  assert_contains "$OUT2" "name: three" "case2: three.md present"
  assert_not_contains "$OUT2" "Duplicate of one" "case2: dup body dropped"
  assert_contains "$TMP/out2.err" "duplicate name" "case2: dedupe warning emitted"
else
  echo "FAIL: case2 output file missing"
  FAILS=$((FAILS + 1))
fi

# ---------------------------------------------------------------------------
# Case 3: missing memory dir — clear error
# ---------------------------------------------------------------------------

OUT3="$TMP/out3.md"
sh "$BUNDLE" --claude-project "$TMP/does-not-exist" --out "$OUT3" >"$TMP/out3.log" 2>"$TMP/out3.err"
RC=$?
if [ "$RC" -ne 0 ]; then
  echo "PASS: case3: missing project returns non-zero"
else
  echo "FAIL: case3: missing project should fail (rc=$RC)"
  FAILS=$((FAILS + 1))
fi

# ---------------------------------------------------------------------------
# Case 4: glob matches zero files — warn, but continue with the other source
# ---------------------------------------------------------------------------

OUT4="$TMP/out4.md"
sh "$BUNDLE" \
  --claude-project "$PROJ" \
  --source "$TMP/does/not/exist/*.md" \
  --out "$OUT4" >"$TMP/out4.log" 2>"$TMP/out4.err"
RC=$?
assert_exit_ok "$RC" "case4: zero-match glob does not fail the run"
assert_contains "$TMP/out4.err" "glob matched zero files" "case4: zero-match warning emitted"

# ---------------------------------------------------------------------------

echo ""
if [ "$FAILS" -eq 0 ]; then
  echo "All checks passed."
  exit 0
else
  echo "$FAILS check(s) failed."
  exit 1
fi
