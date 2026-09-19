#!/usr/bin/env sh
set -eu

usage() {
  cat <<'EOF'
Usage: bash ./scripts/testnet-health.sh http://127.0.0.1:9311 [options]

Options:
  --expected-network testnet
  --expected-network-id idr-testnet-1
  --expected-chain-id 777101
  --allow-wallet-rpc
  --allow-admin-rpc
  --min-peers 1
  --expected-min-height 100
  --expected-max-height-lag 10
  --check-peer-list
  --check-seed http://host:10311
  --fail-on-zero-peers
  --check-mining
  --check-difficulty
  --warn-if-no-recent-block-minutes 10
  --fail-if-no-recent-block-minutes 30
  --json
  --timeout-seconds 10
  --bin-dir ./dist/linux-amd64
  --datadir ./data/testnet
EOF
}

[ "$#" -gt 0 ] || { usage; exit 2; }

RPC_URL=""
EXPECTED_NETWORK=""
EXPECTED_NETWORK_ID=""
EXPECTED_CHAIN_ID=""
ALLOW_WALLET_RPC=0
ALLOW_ADMIN_RPC=0
JSON_OUT=0
TIMEOUT_SECONDS=10
BIN_DIR=""
DATADIR=""
MIN_PEERS=-1
EXPECTED_MIN_HEIGHT=0
EXPECTED_MAX_HEIGHT_LAG=0
CHECK_PEER_LIST=0
CHECK_SEEDS=""
FAIL_ON_ZERO_PEERS=0
CHECK_MINING=0
CHECK_DIFFICULTY=0
WARN_IF_NO_RECENT_BLOCK_MINUTES=0
FAIL_IF_NO_RECENT_BLOCK_MINUTES=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --help|-h)
      usage
      exit 0
      ;;
    --expected-network)
      EXPECTED_NETWORK="$2"
      shift 2
      ;;
    --expected-network-id)
      EXPECTED_NETWORK_ID="$2"
      shift 2
      ;;
    --expected-chain-id)
      EXPECTED_CHAIN_ID="$2"
      shift 2
      ;;
    --allow-wallet-rpc)
      ALLOW_WALLET_RPC=1
      shift
      ;;
    --allow-admin-rpc)
      ALLOW_ADMIN_RPC=1
      shift
      ;;
    --json)
      JSON_OUT=1
      shift
      ;;
    --min-peers)
      MIN_PEERS="$2"
      shift 2
      ;;
    --expected-min-height)
      EXPECTED_MIN_HEIGHT="$2"
      shift 2
      ;;
    --expected-max-height-lag)
      EXPECTED_MAX_HEIGHT_LAG="$2"
      shift 2
      ;;
    --check-peer-list)
      CHECK_PEER_LIST=1
      shift
      ;;
    --check-seed)
      CHECK_SEEDS="${CHECK_SEEDS}${CHECK_SEEDS:+ }$2"
      shift 2
      ;;
    --fail-on-zero-peers)
      FAIL_ON_ZERO_PEERS=1
      shift
      ;;
    --check-mining)
      CHECK_MINING=1
      shift
      ;;
    --check-difficulty)
      CHECK_DIFFICULTY=1
      shift
      ;;
    --warn-if-no-recent-block-minutes)
      WARN_IF_NO_RECENT_BLOCK_MINUTES="$2"
      shift 2
      ;;
    --fail-if-no-recent-block-minutes)
      FAIL_IF_NO_RECENT_BLOCK_MINUTES="$2"
      shift 2
      ;;
    --timeout-seconds)
      TIMEOUT_SECONDS="$2"
      shift 2
      ;;
    --bin-dir)
      BIN_DIR="$2"
      shift 2
      ;;
    --datadir)
      DATADIR="$2"
      shift 2
      ;;
    *)
      if [ -z "$RPC_URL" ]; then
        RPC_URL="$1"
        shift
      else
        echo "unknown argument: $1" >&2
        exit 2
      fi
      ;;
  esac
done

[ -n "$RPC_URL" ] || { usage; exit 2; }
RPC_URL="${RPC_URL%/}"

errors=""
warnings=""

add_error() {
  errors="${errors}${errors:+; }$1"
}

add_warning() {
  warnings="${warnings}${warnings:+; }$1"
}

fetch() {
  curl -fsS --max-time "$TIMEOUT_SECONDS" "$RPC_URL$1"
}

json_value() {
  key="$1"
  file="$2"
  if command -v jq >/dev/null 2>&1; then
    jq -r ".$key // empty" "$file"
  else
    sed -n "s/.*\"$key\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p; s/.*\"$key\"[[:space:]]*:[[:space:]]*\\([^,}]*\\).*/\\1/p" "$file" | head -n 1 | tr -d ' '
  fi
}

tmpdir="${TMPDIR:-/tmp}/deskachain-health-$$"
mkdir -p "$tmpdir"
trap 'rm -rf "$tmpdir"' EXIT INT TERM

health_file="$tmpdir/health.json"
status_file="$tmpdir/status.json"
blocks_file="$tmpdir/blocks.json"
ui_file="$tmpdir/ui.html"
peer_health_file="$tmpdir/peer_health.json"
peer_list_file="$tmpdir/peer_list.json"
mining_status_file="$tmpdir/mining_status.json"
mining_difficulty_file="$tmpdir/mining_difficulty.json"

health_ok=false
status_ok=false
blocks_ok=false
ui_ok=false

if fetch "/health" > "$health_file" 2>"$tmpdir/health.err"; then
  health_ok=true
else
  add_error "health unreachable: $(cat "$tmpdir/health.err")"
fi

if fetch "/explorer/status" > "$status_file" 2>"$tmpdir/status.err"; then
  if [ "$(json_value ok "$status_file")" = "true" ]; then
    status_ok=true
  else
    add_error "explorer status returned ok=false"
  fi
else
  add_error "explorer status unreachable: $(cat "$tmpdir/status.err")"
fi

if fetch "/explorer/blocks?limit=1" > "$blocks_file" 2>"$tmpdir/blocks.err"; then
  blocks_ok=true
else
  add_warning "explorer blocks check failed: $(cat "$tmpdir/blocks.err")"
fi

if fetch "/explorer-ui/" > "$ui_file" 2>"$tmpdir/ui.err"; then
  if grep -q "DesKaChain Explorer" "$ui_file"; then
    ui_ok=true
  else
    add_warning "explorer UI did not return expected content"
  fi
else
  add_warning "explorer UI check failed: $(cat "$tmpdir/ui.err")"
fi

source_file="$status_file"
[ "$status_ok" = "true" ] || source_file="$health_file"

network="$(json_value network "$source_file")"
network_id="$(json_value network_id "$source_file")"
chain_id="$(json_value chain_id "$source_file")"
height="$(json_value height "$source_file")"
tip_hash="$(json_value tip_hash "$source_file")"
peer_count="$(json_value peer_count "$source_file")"
pending_tx_count="$(json_value pending_tx_count "$source_file")"
public_rpc="$(json_value public_rpc "$source_file")"
wallet_rpc="$(json_value wallet_rpc "$source_file")"
admin_rpc="$(json_value admin_rpc "$source_file")"

if [ -z "$peer_count" ]; then peer_count="$(json_value peers "$source_file")"; fi
if [ -z "$pending_tx_count" ]; then pending_tx_count="$(json_value mempool "$source_file")"; fi

[ -z "$EXPECTED_NETWORK" ] || [ "$network" = "$EXPECTED_NETWORK" ] || add_error "network mismatch: expected $EXPECTED_NETWORK got $network"
[ -z "$EXPECTED_NETWORK_ID" ] || [ "$network_id" = "$EXPECTED_NETWORK_ID" ] || add_error "network_id mismatch: expected $EXPECTED_NETWORK_ID got $network_id"
[ -z "$EXPECTED_CHAIN_ID" ] || [ "$chain_id" = "$EXPECTED_CHAIN_ID" ] || add_error "chain_id mismatch: expected $EXPECTED_CHAIN_ID got $chain_id"
[ "$EXPECTED_MIN_HEIGHT" = "0" ] || [ "${height:-0}" -ge "$EXPECTED_MIN_HEIGHT" ] || add_error "height below expected minimum: expected >= $EXPECTED_MIN_HEIGHT got $height"
[ "$EXPECTED_MAX_HEIGHT_LAG" = "0" ] || add_warning "expected max height lag check is informational in shell health mode"
[ "$ALLOW_WALLET_RPC" = "1" ] || [ "$wallet_rpc" != "true" ] || add_error "wallet_rpc is true on a public safety check"
[ "$ALLOW_ADMIN_RPC" = "1" ] || [ "$admin_rpc" != "true" ] || add_error "admin_rpc is true on a public safety check"
[ "$public_rpc" = "true" ] || add_warning "public_rpc is not true; this may be expected for private/local checks"

known_peer_count=""
active_peer_count="$peer_count"
if [ "$CHECK_PEER_LIST" = "1" ] || [ "$MIN_PEERS" -ge 0 ] || [ "$FAIL_ON_ZERO_PEERS" = "1" ] || [ -n "$CHECK_SEEDS" ]; then
  if fetch "/peer/health" > "$peer_health_file" 2>"$tmpdir/peer_health.err"; then
    known_peer_count="$(json_value known_peer_count "$peer_health_file")"
    active_peer_count="$(json_value active_peer_count "$peer_health_file")"
  else
    add_error "peer health check failed: $(cat "$tmpdir/peer_health.err")"
  fi
  if fetch "/peer/list" > "$peer_list_file" 2>"$tmpdir/peer_list.err"; then
    :
  elif [ "$CHECK_PEER_LIST" = "1" ]; then
    add_error "peer list check failed: $(cat "$tmpdir/peer_list.err")"
  else
    add_warning "peer list check failed: $(cat "$tmpdir/peer_list.err")"
  fi
fi

[ -n "$known_peer_count" ] || known_peer_count=0
[ -n "$active_peer_count" ] || active_peer_count=0
if [ "$FAIL_ON_ZERO_PEERS" = "1" ] && [ "$active_peer_count" -eq 0 ]; then
  add_error "active peer count is zero"
elif [ "$active_peer_count" -eq 0 ]; then
  add_warning "active peer count is zero"
fi
if [ "$MIN_PEERS" -ge 0 ] && [ "$active_peer_count" -lt "$MIN_PEERS" ]; then
  add_error "active peer count below minimum: expected >= $MIN_PEERS got $active_peer_count"
fi
for seed in $CHECK_SEEDS; do
  if [ -f "$peer_list_file" ] && grep -q "\"url\"[[:space:]]*:[[:space:]]*\"${seed%/}\"" "$peer_list_file"; then
    :
  else
    add_warning "seed not found in peer list: $seed"
  fi
done

mining_current_difficulty=""
mining_next_difficulty=""
mining_target_block_time=""
mining_blocks_until_retarget=""
mining_last_block_age=""
mining_average_interval=""
mining_retarget_direction=""
if [ "$CHECK_MINING" = "1" ] || [ "$CHECK_DIFFICULTY" = "1" ] || [ "$WARN_IF_NO_RECENT_BLOCK_MINUTES" -gt 0 ] || [ "$FAIL_IF_NO_RECENT_BLOCK_MINUTES" -gt 0 ]; then
  if fetch "/mining/status" > "$mining_status_file" 2>"$tmpdir/mining_status.err"; then
    mining_current_difficulty="$(json_value current_difficulty "$mining_status_file")"
    mining_next_difficulty="$(json_value next_difficulty "$mining_status_file")"
    mining_target_block_time="$(json_value target_block_time_seconds "$mining_status_file")"
    mining_blocks_until_retarget="$(json_value blocks_until_retarget "$mining_status_file")"
    mining_last_block_age="$(json_value last_block_age_seconds "$mining_status_file")"
    mining_average_interval="$(json_value average_interval_seconds "$mining_status_file")"
    mining_retarget_direction="$(json_value projected_retarget_direction "$mining_status_file")"
  else
    add_error "mining status check failed: $(cat "$tmpdir/mining_status.err")"
  fi
  if [ "$CHECK_DIFFICULTY" = "1" ]; then
    if fetch "/mining/difficulty" > "$mining_difficulty_file" 2>"$tmpdir/mining_difficulty.err"; then
      :
    else
      add_error "mining difficulty check failed: $(cat "$tmpdir/mining_difficulty.err")"
    fi
  fi
fi
if [ -n "$mining_last_block_age" ]; then
  if [ "$WARN_IF_NO_RECENT_BLOCK_MINUTES" -gt 0 ] && [ "$mining_last_block_age" -gt $((WARN_IF_NO_RECENT_BLOCK_MINUTES * 60)) ]; then
    add_warning "no recent block within warning window: age ${mining_last_block_age}s"
  fi
  if [ "$FAIL_IF_NO_RECENT_BLOCK_MINUTES" -gt 0 ] && [ "$mining_last_block_age" -gt $((FAIL_IF_NO_RECENT_BLOCK_MINUTES * 60)) ]; then
    add_error "no recent block within failure window: age ${mining_last_block_age}s"
  fi
fi

cli_chain_info=""
cli_chain_validate=""
if [ -n "$BIN_DIR" ]; then
  deskachain="$BIN_DIR/deskachain"
  [ -x "$deskachain" ] || deskachain="$BIN_DIR/deskachain.exe"
  if [ -x "$deskachain" ]; then
    if cli_chain_info="$("$deskachain" --rpc-url "$RPC_URL" chain info 2>&1)"; then
      :
    else
      add_warning "chain info via CLI failed"
    fi
    if [ -n "$DATADIR" ]; then
      if cli_chain_validate="$("$deskachain" --rpc-url "$RPC_URL" chain validate 2>&1)"; then
        :
      else
        add_error "chain validate via CLI failed"
      fi
    fi
  else
    add_warning "deskachain binary not found in $BIN_DIR"
  fi
fi

explorer_ok=false
if [ "$status_ok" = "true" ] && [ "$blocks_ok" = "true" ] && [ "$ui_ok" = "true" ]; then
  explorer_ok=true
fi

ok=false
[ -z "$errors" ] && ok=true

if [ "$JSON_OUT" = "1" ]; then
  printf '{"ok":%s,"rpc_url":"%s","network":"%s","network_id":"%s","chain_id":%s,"height":%s,"tip_hash":"%s","peer_count":%s,"known_peer_count":%s,"active_peer_count":%s,"pending_tx_count":%s,"public_rpc":%s,"wallet_rpc":%s,"admin_rpc":%s,"explorer_ok":%s,"mining_current_difficulty":"%s","mining_next_difficulty":"%s","mining_last_block_age_seconds":"%s","warnings":"%s","errors":"%s"}\n' \
    "$ok" "$RPC_URL" "$network" "$network_id" "${chain_id:-0}" "${height:-0}" "$tip_hash" "${peer_count:-0}" "${known_peer_count:-0}" "${active_peer_count:-0}" "${pending_tx_count:-0}" "${public_rpc:-false}" "${wallet_rpc:-false}" "${admin_rpc:-false}" "$explorer_ok" "$mining_current_difficulty" "$mining_next_difficulty" "$mining_last_block_age" "$warnings" "$errors"
else
  if [ "$ok" = "true" ]; then
    echo "ok: testnet health check passed"
  else
    echo "fail: testnet health check failed"
  fi
  echo "rpc_url: $RPC_URL"
  echo "network: $network"
  echo "network_id: $network_id"
  echo "chain_id: $chain_id"
  echo "height: $height"
  echo "tip_hash: $tip_hash"
  echo "peer_count: $peer_count"
  echo "known_peer_count: $known_peer_count"
  echo "active_peer_count: $active_peer_count"
  echo "pending_tx_count: $pending_tx_count"
  echo "public_rpc: $public_rpc"
  echo "wallet_rpc: $wallet_rpc"
  echo "admin_rpc: $admin_rpc"
  echo "explorer_ok: $explorer_ok"
  if [ -n "$mining_current_difficulty" ]; then
    echo "mining_current_difficulty: $mining_current_difficulty"
    echo "mining_next_difficulty: $mining_next_difficulty"
    echo "mining_target_block_time: ${mining_target_block_time}s"
    echo "mining_blocks_until_retarget: $mining_blocks_until_retarget"
    echo "mining_last_block_age: ${mining_last_block_age}s"
    echo "mining_average_interval: ${mining_average_interval}s"
    echo "mining_retarget_direction: $mining_retarget_direction"
  fi
  [ -z "$warnings" ] || echo "warning: $warnings"
  [ -z "$errors" ] || echo "error: $errors"
fi

[ "$ok" = "true" ] || exit 1
