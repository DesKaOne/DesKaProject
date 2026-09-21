#!/usr/bin/env sh
set -eu
usage(){ cat <<'EOF'
Usage: bash ./scripts/testnet-backup.sh --datadir PATH --output FILE [--format tar.gz|directory]
Creates a filesystem backup of a STOPPED IndoChain node datadir.
EOF
}
DATADIR=""; OUTPUT=""; FORMAT="tar.gz"
while [ "$#" -gt 0 ]; do case "$1" in
--help|-h) usage; exit 0;; --datadir) DATADIR="$2"; shift 2;;
--output) OUTPUT="$2"; shift 2;; --format) FORMAT="$2"; shift 2;;
*) echo "unknown argument: $1" >&2; exit 2;; esac; done
[ -n "$DATADIR" ] && [ -n "$OUTPUT" ] || { usage; exit 2; }
[ -d "$DATADIR" ] || { echo "datadir not found: $DATADIR" >&2; exit 1; }
case "$FORMAT" in
tar.gz)
  case "$OUTPUT" in *.tar.gz) ;; *) OUTPUT="$OUTPUT.tar.gz";; esac
  mkdir -p "$(dirname "$OUTPUT")"; tmp="$OUTPUT.tmp.$$"; trap 'rm -f "$tmp"' EXIT INT TERM
  tar -C "$DATADIR" -czf "$tmp" .; mv "$tmp" "$OUTPUT";;
directory)
  [ ! -e "$OUTPUT" ] || { echo "target already exists: $OUTPUT" >&2; exit 1; }
  mkdir -p "$OUTPUT"; cp -a "$DATADIR"/. "$OUTPUT"/;;
*) echo "unsupported format: $FORMAT" >&2; exit 2;; esac
echo "backup created: $OUTPUT"
echo "source datadir: $DATADIR"
