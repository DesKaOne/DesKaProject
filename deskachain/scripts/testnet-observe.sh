#!/usr/bin/env sh
set -eu
usage() {
  cat <<'EOF'
Usage:
  ./scripts/testnet-observe.sh --node URL [--node URL ...] [options]

Options:
  --node URL             Node RPC endpoint; repeatable.
  --interval-seconds N   Poll interval, default 30.
  --duration-seconds N   Total observation duration, default 3600.
  --output FILE          JSONL output, default ./testnet-observation.jsonl.
  --timeout-seconds N    HTTP timeout, default 10.
EOF
}
nodes=""
interval=30
duration=3600
output="./testnet-observation.jsonl"
timeout=10
while [ "$#" -gt 0 ]; do
  case "$1" in
    --help|-h) usage; exit 0 ;;
    --node)
      if [ -n "$nodes" ]; then nodes="$nodes|$2"; else nodes="$2"; fi
      shift 2 ;;
    --interval-seconds) interval="$2"; shift 2 ;;
    --duration-seconds) duration="$2"; shift 2 ;;
    --output) output="$2"; shift 2 ;;
    --timeout-seconds) timeout="$2"; shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
[ -n "$nodes" ] || { usage; exit 2; }
[ "$interval" -gt 0 ] || { echo "interval must be > 0" >&2; exit 2; }
[ "$duration" -gt 0 ] || { echo "duration must be > 0" >&2; exit 2; }
mkdir -p "$(dirname "$output")"
end=$(( $(date +%s) + duration ))

poll_node() {
  node="$1"
  tmp="${TMPDIR:-/tmp}/indochain-observe-$$"
  mkdir -p "$tmp"
  metrics="$tmp/metrics.json"
  health="$tmp/health.json"
  indexer="$tmp/indexer.json"
  peer="$tmp/peer.json"
  ok=true
  curl -fsS --max-time "$timeout" "$node/node/metrics" >"$metrics" 2>/dev/null || ok=false
  curl -fsS --max-time "$timeout" "$node/health" >"$health" 2>/dev/null || ok=false
  curl -fsS --max-time "$timeout" "$node/explorer/indexer/stats" >"$indexer" 2>/dev/null || ok=false
  curl -fsS --max-time "$timeout" "$node/peer/health" >"$peer" 2>/dev/null || ok=false
  if command -v jq >/dev/null 2>&1 && [ -s "$metrics" ]; then
    jq -cn       --arg observed_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)"       --arg node "$node"       --argjson ok "$ok"       --argjson metrics "$(cat "$metrics")"       --arg health "$(cat "$health" 2>/dev/null || printf '{}')"       --arg indexer "$(cat "$indexer" 2>/dev/null || printf '{}')"       --arg peer "$(cat "$peer" 2>/dev/null || printf '{}')"       '{observed_at:$observed_at,node:$node,ok:$ok,metrics:$metrics,health:($health|fromjson? // null),indexer:($indexer|fromjson? // null),peer:($peer|fromjson? // null)}'
  else
    printf '{"observed_at":"%s","node":"%s","ok":%s}
' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$node" "$ok"
  fi
  rm -rf "$tmp"
}

while [ "$(date +%s)" -lt "$end" ]; do
  oldifs="$IFS"; IFS='|'
  for node in $nodes; do poll_node "$node" >>"$output"; done
  IFS="$oldifs"
  now="$(date +%s)"
  remaining=$((end - now))
  [ "$remaining" -le 0 ] && break
  sleep_for="$interval"
  [ "$remaining" -lt "$sleep_for" ] && sleep_for="$remaining"
  sleep "$sleep_for"
done
echo "observation complete: $output"
