#!/usr/bin/env sh
set -eu

usage() {
  cat <<'EOF'
Usage: bash ./scripts/smoke-test.sh [./dist/linux-amd64] [--keep-data]

Runs a local testnet smoke test using temporary datadirs. No secrets are required.
EOF
}

BIN_DIR="./dist/linux-amd64"
KEEP_DATA=0
for arg in "$@"; do
  case "$arg" in
    --help|-h)
      usage
      exit 0
      ;;
    --keep-data)
      KEEP_DATA=1
      ;;
    *)
      BIN_DIR="$arg"
      ;;
  esac
done

ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
case "$BIN_DIR" in
  /*) BIN_PATH="$BIN_DIR" ;;
  *) BIN_PATH="$ROOT/$BIN_DIR" ;;
esac
DESKACHAIN="$BIN_PATH/deskachain"
MINER="$BIN_PATH/idrminer"
[ -x "$DESKACHAIN" ] || { echo "missing executable: $DESKACHAIN" >&2; exit 1; }
[ -x "$MINER" ] || { echo "missing executable: $MINER" >&2; exit 1; }

TMP="${TMPDIR:-/tmp}/deskachain-smoke-$$"
NODE_DIR="$TMP/node"
WALLET_DIR="$TMP/miner-wallet"
RPC="http://127.0.0.1:19311"
NODE_PID=""

cleanup() {
  if [ -n "$NODE_PID" ] && kill -0 "$NODE_PID" >/dev/null 2>&1; then
    kill "$NODE_PID" || true
    wait "$NODE_PID" || true
  fi
  if [ "$KEEP_DATA" = "0" ]; then
    rm -rf "$TMP"
  else
    echo "kept smoke data: $TMP"
  fi
}
trap cleanup EXIT INT TERM

mkdir -p "$TMP"
"$DESKACHAIN" version
"$DESKACHAIN" --datadir "$NODE_DIR" --network testnet init
"$DESKACHAIN" --datadir "$WALLET_DIR" --network testnet init
ADDR="$("$DESKACHAIN" --datadir "$WALLET_DIR" wallet new | tail -n 1)"
[ -n "$ADDR" ] || { echo "wallet new did not return an address" >&2; exit 1; }

# CI smoke test runs a single isolated testnet node. Override isolated-mining guard only for smoke validation. Public testnet nodes should not use this flag.
"$DESKACHAIN" --datadir "$NODE_DIR" --network testnet node start \
  --rpc 127.0.0.1:19311 \
  --p2p 127.0.0.1:19312 \
  --advertise-p2p http://127.0.0.1:19312 \
  --public-rpc \
  --enable-miner-rpc \
  --allow-isolated-mining \
  --min-mining-peers 0 > "$TMP/node.log" 2>&1 &
NODE_PID=$!

ready=0
i=0
while [ "$i" -lt 60 ]; do
  if "$DESKACHAIN" --rpc-url "$RPC" chain info >/dev/null 2>&1; then
    ready=1
    break
  fi
  if ! kill -0 "$NODE_PID" >/dev/null 2>&1; then
    cat "$TMP/node.log" >&2 || true
    echo "node exited before becoming healthy" >&2
    exit 1
  fi
  i=$((i + 1))
  sleep 1
done
[ "$ready" = "1" ] || { cat "$TMP/node.log" >&2 || true; echo "node did not become healthy" >&2; exit 1; }

"$DESKACHAIN" --rpc-url "$RPC" chain info
"$DESKACHAIN" --rpc-url "$RPC" chain validate
"$MINER" --rpc-url "$RPC" --address "$ADDR" --threads 2 --once
"$DESKACHAIN" --rpc-url "$RPC" chain info
"$DESKACHAIN" --rpc-url "$RPC" chain validate

if command -v curl >/dev/null 2>&1; then
  curl -fsS "$RPC/health" >/dev/null
  curl -fsS "$RPC/explorer/status" >/dev/null
  curl -fsS "$RPC/explorer/blocks?limit=1" >/dev/null
  curl -fsS "$RPC/explorer-ui/" | grep -q "DesKaChain Explorer"
else
  "$DESKACHAIN" --rpc-url "$RPC" chain info >/dev/null
fi

echo "smoke test passed using $BIN_PATH"
