#!/usr/bin/env sh
set -eu

usage() {
  cat <<'EOF'
Usage: bash ./scripts/testnet-smoke.sh http://127.0.0.1:9311 [options]

Runs a read-only operator smoke check against an already running testnet node.
No mining, wallet, admin, or consensus-changing RPC is performed.
Options:
  --expected-network testnet
  --expected-network-id ind-testnet-1
  --expected-chain-id 777101
  --min-peers 1
  --require-indexer-ready
  --max-indexer-lag 2
  --check-mining
  --json
  --timeout-seconds 10
EOF
}

[ "$#" -gt 0 ] || { usage; exit 2; }

RPC_URL=""
EXPECTED_NETWORK=""
EXPECTED_NETWORK_ID=""
EXPECTED_CHAIN_ID=""
MIN_PEERS=0
REQUIRE_INDEXER_READY=0
MAX_INDEXER_LAG=0
CHECK_MINING=0
JSON_OUT=0
TIMEOUT_SECONDS=10

while [ "$#" -gt 0 ]; do
  case "$1" in
    --help|-h) usage; exit 0 ;;
    --expected-network) EXPECTED_NETWORK="$2"; shift 2 ;;
    --expected-network-id) EXPECTED_NETWORK_ID="$2"; shift 2 ;;
    --expected-chain-id) EXPECTED_CHAIN_ID="$2"; shift 2 ;;
    --min-peers) MIN_PEERS="$2"; shift 2 ;;
    --require-indexer-ready) REQUIRE_INDEXER_READY=1; shift ;;
    --max-indexer-lag) MAX_INDEXER_LAG="$2"; shift 2 ;;
    --check-mining) CHECK_MINING=1; shift ;;
    --json) JSON_OUT=1; shift ;;
    --timeout-seconds) TIMEOUT_SECONDS="$2"; shift 2 ;;
    *)
      if [ -z "$RPC_URL" ]; then RPC_URL="$1"; shift
      else echo "unknown argument: $1" >&2; exit 2
      fi
      ;;
  esac
done

[ -n "$RPC_URL" ] || { usage; exit 2; }
RPC_URL="${RPC_URL%/}"

fetch() {
  curl -fsS --max-time "$TIMEOUT_SECONDS" "$RPC_URL$1"
}

tmp="${TMPDIR:-/tmp}/indochain-operator-smoke-$$"
mkdir -p "$tmp"
trap 'rm -rf "$tmp"' EXIT INT TERM

errors=""
add_error() { errors="${errors}${errors:+; }$1"; }

health_file="$tmp/health.json"
metrics_file="$tmp/metrics.json"
peer_file="$tmp/peer.json"
indexer_file="$tmp/indexer.json"
mining_file="$tmp/mining.json"
ui_file="$tmp/ui.html"

fetch "/health" >"$health_file" || add_error "health endpoint failed"
fetch "/node/metrics" >"$metrics_file" || add_error "node metrics endpoint failed"
fetch "/peer/health" >"$peer_file" || add_error "peer health endpoint failed"
fetch "/explorer/indexer/stats" >"$indexer_file" || add_error "explorer indexer stats endpoint failed"
fetch "/explorer-ui/" >"$ui_file" || add_error "explorer UI endpoint failed"

if [ "$CHECK_MINING" = "1" ]; then
  fetch "/mining/status" >"$mining_file" || add_error "mining status endpoint failed"
fi

json_value() {
  key="$1"; file="$2"
  if command -v jq >/dev/null 2>&1; then
    jq -r ".$key // empty" "$file"
  else
    sed -n "s/.*\"$key\"[[:space:]]*:[[:space:]]*\\"\\([^\\"]*\\)\\".*/\\1/p; s/.*\"$key\"[[:space:]]*:[[:space:]]*\\([^,}]*\\).*/\\1/p" "$file" | head -n 1 | tr -d ' '
  fi
}

network="$(json_value network "$metrics_file")"
network_id="$(json_value network_id "$metrics_file")"
chain_id="$(json_value chain_id "$metrics_file")"
height="$(json_value 'chain.height' "$metrics_file")"
active_peers="$(json_value 'peers.active' "$metrics_file")"
indexer_ready="$(json_value 'indexer.stats.ready' "$metrics_file")"
indexer_lag="$(json_value 'indexer.stats.lag' "$metrics_file")"
schema="$(json_value schema_version "$metrics_file")"
health_ok="$(json_value ok "$health_file")"

[ -z "$EXPECTED_NETWORK" ] || [ "$network" = "$EXPECTED_NETWORK" ] || add_error "network mismatch: expected $EXPECTED_NETWORK got $network"
[ -z "$EXPECTED_NETWORK_ID" ] || [ "$network_id" = "$EXPECTED_NETWORK_ID" ] || add_error "network_id mismatch: expected $EXPECTED_NETWORK_ID got $network_id"
[ -z "$EXPECTED_CHAIN_ID" ] || [ "$chain_id" = "$EXPECTED_CHAIN_ID" ] || add_error "chain_id mismatch: expected $EXPECTED_CHAIN_ID got $chain_id"
[ "$health_ok" = "true" ] || add_error "health returned ok=false"
[ "$schema" = "v1" ] || add_error "node metrics schema is not v1: $schema"
[ "${active_peers:-0}" -ge "$MIN_PEERS" ] || add_error "active peers below minimum: expected >= $MIN_PEERS got ${active_peers:-0}"
if [ "$REQUIRE_INDEXER_READY" = "1" ] && [ "$indexer_ready" != "true" ]; then
  add_error "explorer indexer is not ready"
fi
if [ "$MAX_INDEXER_LAG" -gt 0 ] && [ "${indexer_lag:-0}" -gt "$MAX_INDEXER_LAG" ]; then
  add_error "explorer indexer lag exceeds maximum: expected <= $MAX_INDEXER_LAG got ${indexer_lag:-0}"
fi
grep -q "IndoChain Explorer" "$ui_file" || add_error "explorer UI content check failed"

if [ "$CHECK_MINING" = "1" ]; then
  mining_ok="$(json_value ok "$mining_file")"
  [ "$mining_ok" = "true" ] || add_error "mining status returned ok=false"
fi

post_status="$(curl -sS -o /dev/null -w '%{http_code}' --max-time "$TIMEOUT_SECONDS" -X POST "$RPC_URL/node/metrics" || true)"
[ "$post_status" = "405" ] || add_error "POST /node/metrics expected 405 got $post_status"

ok=false
[ -z "$errors" ] && ok=true

if [ "$JSON_OUT" = "1" ]; then
  printf '{"ok":%s,"network":"%s","network_id":"%s","chain_id":%s,"height":%s,"active_peers":%s,"indexer_ready":%s,"indexer_lag":%s,"metrics_schema":"%s","errors":"%s"}\n'     "$ok" "$network" "$network_id" "${chain_id:-0}" "${height:-0}" "${active_peers:-0}" "${indexer_ready:-false}" "${indexer_lag:-0}" "$schema" "$errors"
else
  [ "$ok" = "true" ] && echo "ok: testnet operator smoke passed" || echo "fail: testnet operator smoke failed"
  echo "network: $network"
  echo "network_id: $network_id"
  echo "chain_id: $chain_id"
  echo "height: $height"
  echo "active_peers: $active_peers"
  echo "indexer_ready: $indexer_ready"
  echo "indexer_lag: $indexer_lag"
  echo "metrics_schema: $schema"
  [ -z "$errors" ] || echo "error: $errors"
fi

[ "$ok" = "true" ] || exit 1
