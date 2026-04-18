#!/bin/sh
# bundle.sh — concat Claude auto-memory (and other memory-style markdown)
# into a single upload-ready file for rememberize onboarding.
#
# POSIX sh. Runs on macOS, Linux, Git Bash for Windows, WSL.
#
# Verification steps (manual):
#   1. Create a temp dir with a fake memory/ layout containing 2-3 *.md files
#      starting with `---\nname: foo\n---\nbody`, plus a MEMORY.md index.
#   2. Run: sh bundle.sh --source <tempdir> --out /tmp/out.md
#   3. Confirm /tmp/out.md contains a header block, all source files concatenated,
#      and MEMORY.md was excluded.
#
# See scripts/bundle_test.sh for an automated harness.

set -eu

# ---------------------------------------------------------------------------
# Arg parsing
# ---------------------------------------------------------------------------

SOURCES=""           # newline-separated list of paths (files, dirs, globs)
CLAUDE_PROJECTS=""   # newline-separated list of project dirs
OUT=""
SLUG_HINT=""

usage() {
  cat <<'EOF'
Usage: bundle.sh [options]

Options:
  --source <path>           A file, directory, or glob to include.
                            Directories recurse by default.
                            Repeatable.
  --source-flat <dir>       A directory; only top-level *.md files.
                            Repeatable.
  --claude-project <path>   A Claude project dir (expects <path>/memory/).
                            Reads *.md under memory/ except MEMORY.md.
                            Repeatable.
  --out <path>              Output file. Defaults to
                            $HOME/.claude/bundles/<slug>-<YYYY-MM-DD>.md
  --slug <name>             Override the slug used in the default output filename.
  -h, --help                Show this help.

With no arguments, bundles the current directory's Claude auto-memory
(i.e. $HOME/.claude/projects/<encoded-cwd>/memory/).
EOF
}

# Append a line to a newline-separated variable in a portable way.
append_line() {
  # $1 = var name, $2 = line
  _var=$1
  _line=$2
  eval "_cur=\${$_var}"
  if [ -z "$_cur" ]; then
    eval "$_var=\$_line"
  else
    eval "$_var=\"\${_cur}
\$_line\""
  fi
}

while [ $# -gt 0 ]; do
  case "$1" in
    --source)
      [ $# -ge 2 ] || { echo "error: --source requires a value" >&2; exit 2; }
      append_line SOURCES "R:$2"
      shift 2
      ;;
    --source-flat)
      [ $# -ge 2 ] || { echo "error: --source-flat requires a value" >&2; exit 2; }
      append_line SOURCES "F:$2"
      shift 2
      ;;
    --claude-project)
      [ $# -ge 2 ] || { echo "error: --claude-project requires a value" >&2; exit 2; }
      append_line CLAUDE_PROJECTS "$2"
      shift 2
      ;;
    --out)
      [ $# -ge 2 ] || { echo "error: --out requires a value" >&2; exit 2; }
      OUT=$2
      shift 2
      ;;
    --slug)
      [ $# -ge 2 ] || { echo "error: --slug requires a value" >&2; exit 2; }
      SLUG_HINT=$2
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

# ---------------------------------------------------------------------------
# Default: no args → bundle current cwd's Claude project
# ---------------------------------------------------------------------------

# Encode a path to the Claude project slug convention:
#   /home/u/x      → -home-u-x
#   C:\path\to\x   → C--path-to-x
encode_cwd_slug() {
  # Read cwd via pwd (portable). On Git Bash, prefer `pwd -W` to get the
  # Windows-style path (e.g. C:/path/to/x) because Claude encodes from that.
  if command -v cygpath >/dev/null 2>&1; then
    _p=$(pwd -W 2>/dev/null || pwd)
  else
    _p=$(pwd)
  fi
  # Replace / and \ with -; replace : with -- (for Windows drive letters).
  printf '%s' "$_p" | sed -e 's#:#--#g' -e 's#[/\\]#-#g'
}

if [ -z "$SOURCES" ] && [ -z "$CLAUDE_PROJECTS" ]; then
  _slug=$(encode_cwd_slug)
  _proj="$HOME/.claude/projects/$_slug"
  if [ ! -d "$_proj/memory" ]; then
    echo "error: no sources given, and no Claude project memory dir at:" >&2
    echo "       $_proj/memory" >&2
    echo "Use --source, --claude-project, or run from a directory that has" >&2
    echo "Claude auto-memory enabled." >&2
    exit 1
  fi
  append_line CLAUDE_PROJECTS "$_proj"
fi

# ---------------------------------------------------------------------------
# Resolve the slug for the default output filename
# ---------------------------------------------------------------------------

derive_slug() {
  if [ -n "$SLUG_HINT" ]; then
    printf '%s' "$SLUG_HINT"
    return
  fi

  # Count distinct roots. If exactly one claude-project and no --source*, use
  # its basename. Otherwise "multi".
  _cp_count=0
  if [ -n "$CLAUDE_PROJECTS" ]; then
    _cp_count=$(printf '%s\n' "$CLAUDE_PROJECTS" | grep -c '.')
  fi
  _src_count=0
  if [ -n "$SOURCES" ]; then
    _src_count=$(printf '%s\n' "$SOURCES" | grep -c '.')
  fi

  if [ "$_cp_count" -eq 1 ] && [ "$_src_count" -eq 0 ]; then
    _only=$(printf '%s' "$CLAUDE_PROJECTS" | head -n 1)
    basename "$_only"
  else
    printf 'multi'
  fi
}

if [ -z "$OUT" ]; then
  _slug=$(derive_slug)
  _date=$(date +%Y-%m-%d)
  OUT="$HOME/.claude/bundles/${_slug}-${_date}.md"
fi

# Ensure parent dir exists.
_outdir=$(dirname "$OUT")
mkdir -p "$_outdir"

# ---------------------------------------------------------------------------
# Collect files from all sources
# ---------------------------------------------------------------------------

# We build a list of files (newline-separated) into $FILES, with a prefix
# tag recording their provenance for the summary.
#
# Prefix format: "<tag>\t<path>" where <tag> is a short source identifier.

FILES=""
SOURCE_LABELS=""    # human-readable labels, one per tag, for summary

# Add a source label; returns nothing.
add_label() {
  append_line SOURCE_LABELS "$1: $2"
}

# List *.md files directly under a dir (non-recursive).
list_flat_md() {
  _dir=$1
  # -maxdepth 1 is GNU-specific; use a portable loop.
  ( cd "$_dir" 2>/dev/null && ls -1 2>/dev/null ) | while IFS= read -r _f; do
    case "$_f" in
      *.md) [ -f "$_dir/$_f" ] && printf '%s\n' "$_dir/$_f" ;;
    esac
  done
}

# List *.md files recursively under a dir. Uses `find` which is POSIX.
list_recursive_md() {
  _dir=$1
  find "$_dir" -type f -name '*.md' 2>/dev/null | LC_ALL=C sort
}

# Resolve a glob by handing it to the shell. `sh -c 'ls -d ...'` is the
# portable incantation; we filter failures.
list_glob_md() {
  _pat=$1
  # shellcheck disable=SC2086
  sh -c "ls -d -- $_pat 2>/dev/null" | while IFS= read -r _m; do
    if [ -f "$_m" ]; then
      case "$_m" in
        *.md) printf '%s\n' "$_m" ;;
      esac
    fi
  done
}

# Claude-project sources
if [ -n "$CLAUDE_PROJECTS" ]; then
  _tagi=0
  printf '%s\n' "$CLAUDE_PROJECTS" | while IFS= read -r _proj; do
    [ -z "$_proj" ] && continue
    _memdir="$_proj/memory"
    if [ ! -d "$_memdir" ]; then
      echo "warning: no memory/ dir at $_proj — skipping" >&2
      continue
    fi
    _tagi=$((_tagi + 1))
    _tag="cp${_tagi}"
    # List *.md, exclude MEMORY.md
    list_flat_md "$_memdir" | while IFS= read -r _f; do
      case "$(basename "$_f")" in
        MEMORY.md) continue ;;
      esac
      printf '%s\t%s\n' "$_tag" "$_f"
    done
  done > "$_outdir/.bundle.$$.cp.lst"
  # Write labels (cannot use while-subshell var; re-walk).
  _tagi=0
  printf '%s\n' "$CLAUDE_PROJECTS" | while IFS= read -r _proj; do
    [ -z "$_proj" ] && continue
    _tagi=$((_tagi + 1))
    printf 'cp%d\tclaude-project: %s\n' "$_tagi" "$_proj"
  done > "$_outdir/.bundle.$$.cp.labels"
else
  : > "$_outdir/.bundle.$$.cp.lst"
  : > "$_outdir/.bundle.$$.cp.labels"
fi

# Regular sources (files, dirs recursive, dirs flat, globs)
if [ -n "$SOURCES" ]; then
  _tagi=0
  printf '%s\n' "$SOURCES" | while IFS= read -r _entry; do
    [ -z "$_entry" ] && continue
    _mode=$(printf '%s' "$_entry" | cut -c1)
    _path=$(printf '%s' "$_entry" | cut -c3-)
    _tagi=$((_tagi + 1))
    _tag="s${_tagi}"
    if [ -f "$_path" ]; then
      case "$_path" in
        *.md) printf '%s\t%s\n' "$_tag" "$_path" ;;
        *) echo "warning: non-.md file ignored: $_path" >&2 ;;
      esac
    elif [ -d "$_path" ]; then
      if [ "$_mode" = "F" ]; then
        list_flat_md "$_path" | while IFS= read -r _f; do
          printf '%s\t%s\n' "$_tag" "$_f"
        done
      else
        list_recursive_md "$_path" | while IFS= read -r _f; do
          printf '%s\t%s\n' "$_tag" "$_f"
        done
      fi
    else
      # Treat as glob.
      _before=$(list_glob_md "$_path" | wc -l | tr -d ' ')
      if [ "$_before" -eq 0 ]; then
        echo "warning: glob matched zero files: $_path" >&2
        continue
      fi
      list_glob_md "$_path" | while IFS= read -r _f; do
        printf '%s\t%s\n' "$_tag" "$_f"
      done
    fi
  done > "$_outdir/.bundle.$$.src.lst"

  _tagi=0
  printf '%s\n' "$SOURCES" | while IFS= read -r _entry; do
    [ -z "$_entry" ] && continue
    _mode=$(printf '%s' "$_entry" | cut -c1)
    _path=$(printf '%s' "$_entry" | cut -c3-)
    _tagi=$((_tagi + 1))
    case "$_mode" in
      F) _desc="dir (flat)" ;;
      R) _desc="source" ;;
      *) _desc="source" ;;
    esac
    printf 's%d\t%s: %s\n' "$_tagi" "$_desc" "$_path"
  done > "$_outdir/.bundle.$$.src.labels"
else
  : > "$_outdir/.bundle.$$.src.lst"
  : > "$_outdir/.bundle.$$.src.labels"
fi

# Merge lists
cat "$_outdir/.bundle.$$.cp.lst" "$_outdir/.bundle.$$.src.lst" > "$_outdir/.bundle.$$.all.lst"
cat "$_outdir/.bundle.$$.cp.labels" "$_outdir/.bundle.$$.src.labels" > "$_outdir/.bundle.$$.labels"

# Clean interim files on exit.
trap 'rm -f "$_outdir/.bundle.$$."*' EXIT INT TERM

TOTAL=$(wc -l < "$_outdir/.bundle.$$.all.lst" | tr -d ' ')
if [ "$TOTAL" -eq 0 ]; then
  echo "error: no *.md files found from the given sources" >&2
  exit 1
fi

# ---------------------------------------------------------------------------
# Write the bundle
# ---------------------------------------------------------------------------

# Multi-source flag: true if >1 distinct tag, in which case we dedupe by name:
MULTI=0
_tags=$(awk -F'\t' '{print $1}' "$_outdir/.bundle.$$.all.lst" | sort -u | wc -l | tr -d ' ')
if [ "$_tags" -gt 1 ]; then
  MULTI=1
fi

# Extract `name:` value from first frontmatter block of a file.
# Prints the name (empty if not found / not frontmatter).
extract_name() {
  awk '
    BEGIN { in_fm=0; done=0 }
    NR==1 {
      if ($0 == "---") { in_fm=1; next }
      else { exit }
    }
    in_fm && /^---[[:space:]]*$/ { done=1; exit }
    in_fm && /^name:[[:space:]]*/ {
      sub(/^name:[[:space:]]*/, "")
      sub(/[[:space:]]+$/, "")
      print
      done=1
      exit
    }
  ' "$1"
}

# First-line check: is this a frontmatter file?
is_frontmatter() {
  _first=$(head -n 1 "$1" 2>/dev/null || true)
  [ "$_first" = "---" ]
}

# Build the header block.
DATE=$(date +%Y-%m-%d)
TIME=$(date +%H:%M:%S)

{
  printf '<!--\n'
  printf 'rememberize-bundle\n'
  printf 'generated: %s %s\n' "$DATE" "$TIME"
  printf 'sources:\n'
  while IFS= read -r _lbl; do
    [ -z "$_lbl" ] && continue
    _label=$(printf '%s' "$_lbl" | cut -f2-)
    printf '  - %s\n' "$_label"
  done < "$_outdir/.bundle.$$.labels"
  printf '%s\n\n' '-->'
} > "$OUT"

# Track: files included, files skipped (non-frontmatter + non-utf8 + dedupe),
# per-tag counts, dedupe collisions.
INCLUDED=0
SKIPPED_SHAPE=0
SKIPPED_BINARY=0
DEDUPED=0

# Track seen names in branch B. Portable set = file of sorted names.
_seen="$_outdir/.bundle.$$.seen"
: > "$_seen"

# Per-tag count file.
_counts="$_outdir/.bundle.$$.counts"
: > "$_counts"

incr_tag() {
  _t=$1
  _cur=$(grep "^${_t}	" "$_counts" 2>/dev/null | cut -f2 || true)
  [ -z "$_cur" ] && _cur=0
  _new=$((_cur + 1))
  # Rewrite the counts file atomically.
  grep -v "^${_t}	" "$_counts" 2>/dev/null > "$_counts.tmp" || true
  printf '%s\t%d\n' "$_t" "$_new" >> "$_counts.tmp"
  mv "$_counts.tmp" "$_counts"
}

# Main iteration.
while IFS='	' read -r _tag _path; do
  [ -z "$_path" ] && continue

  # Binary guard: NUL-byte sniff on the first 8KB.
  # Shell vars strip NULs, so we have to pipe through wc directly.
  _peek_raw=$(LC_ALL=C head -c 8192 "$_path" 2>/dev/null | wc -c | tr -d ' ')
  _peek_txt=$(LC_ALL=C head -c 8192 "$_path" 2>/dev/null | LC_ALL=C tr -d '\000' | wc -c | tr -d ' ')
  if [ "$_peek_raw" != "$_peek_txt" ]; then
    echo "warning: skipping non-text file: $_path" >&2
    SKIPPED_BINARY=$((SKIPPED_BINARY + 1))
    continue
  fi

  if ! is_frontmatter "$_path"; then
    # In multi-source mode, we may still wrap: per spec B3, wrap frontmatter-less
    # files using filename as `name:`. In single-source mode (branch A per spec),
    # only frontmatter files are included; report skipped.
    if [ "$MULTI" -eq 1 ]; then
      _fn=$(basename "$_path" .md)
      {
        printf '%s\n' '---'
        printf 'name: %s\n' "$_fn"
        printf 'source-path: %s\n' "$_path"
        printf '%s\n\n' '---'
        cat "$_path"
        # Ensure trailing newline separation.
        _last=$(tail -c 1 "$_path" 2>/dev/null || printf '')
        [ "$_last" != "" ] && printf '\n'
        printf '\n'
      } >> "$OUT"
      INCLUDED=$((INCLUDED + 1))
      incr_tag "$_tag"
    else
      echo "warning: skipping non-frontmatter file: $_path" >&2
      SKIPPED_SHAPE=$((SKIPPED_SHAPE + 1))
    fi
    continue
  fi

  # Dedupe in multi-source mode.
  if [ "$MULTI" -eq 1 ]; then
    _name=$(extract_name "$_path")
    if [ -n "$_name" ]; then
      if grep -Fxq -- "$_name" "$_seen"; then
        echo "warning: duplicate name '$_name' — keeping first, dropping $_path" >&2
        DEDUPED=$((DEDUPED + 1))
        continue
      fi
      printf '%s\n' "$_name" >> "$_seen"
    fi
  fi

  # Append file content + trailing blank separator.
  cat "$_path" >> "$OUT"
  # Ensure trailing newline.
  _last=$(tail -c 1 "$_path" 2>/dev/null || printf '')
  if [ "$_last" != "" ]; then
    printf '\n' >> "$OUT"
  fi
  printf '\n' >> "$OUT"

  INCLUDED=$((INCLUDED + 1))
  incr_tag "$_tag"
done < "$_outdir/.bundle.$$.all.lst"

if [ "$INCLUDED" -eq 0 ]; then
  echo "error: no files were included (all skipped)." >&2
  rm -f "$OUT"
  exit 1
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------

BYTES=$(wc -c < "$OUT" | tr -d ' ')

echo ""
echo "Bundle written: $OUT"
echo "  files included: $INCLUDED"
if [ "$SKIPPED_SHAPE" -gt 0 ]; then
  echo "  skipped (not frontmatter): $SKIPPED_SHAPE"
fi
if [ "$SKIPPED_BINARY" -gt 0 ]; then
  echo "  skipped (non-text): $SKIPPED_BINARY"
fi
if [ "$MULTI" -eq 1 ]; then
  echo ""
  echo "  per-source:"
  while IFS= read -r _lbl; do
    [ -z "$_lbl" ] && continue
    _t=$(printf '%s' "$_lbl" | cut -f1)
    _name=$(printf '%s' "$_lbl" | cut -f2-)
    _c=$(grep "^${_t}	" "$_counts" 2>/dev/null | cut -f2 || true)
    [ -z "$_c" ] && _c=0
    printf '    %s  (%s files)\n' "$_name" "$_c"
  done < "$_outdir/.bundle.$$.labels"
  if [ "$DEDUPED" -gt 0 ]; then
    echo ""
    echo "  dedupe: $DEDUPED duplicate name(s) dropped (first occurrence kept)"
  else
    echo "  dedupe: no conflicts"
  fi
fi
echo "  bytes: $BYTES"
echo ""
echo "Upload this on rememberize onboarding step 3, or via the CLI:"
echo "    rememberize import \"$OUT\""
