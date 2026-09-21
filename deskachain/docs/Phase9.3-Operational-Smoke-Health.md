# Phase 9.3 — Operational Smoke & Health

## Purpose

Phase 9.3 turns the existing health endpoints and local smoke test into a repeatable operator-facing smoke flow for an already running testnet node.

The operator smoke flow is read-only. It does not mine, submit transactions, modify wallet state, rebuild the Explorer index, or change consensus state.

## Operator smoke scripts

- `scripts/testnet-smoke.sh` — POSIX shell implementation.
- `scripts/testnet-smoke.ps1` — PowerShell implementation.

Example:

```sh
./scripts/testnet-smoke.sh http://127.0.0.1:9311 \
  --expected-network testnet \
  --expected-network-id ind-testnet-1 \
  --expected-chain-id 777101 \
  --min-peers 2 \
  --require-indexer-ready \
  --max-indexer-lag 2 \
  --check-mining
```

## Smoke coverage

The operator flow verifies:

1. `/health` is reachable and reports `ok=true`.
2. `/node/metrics` is reachable and remains schema `v1`.
3. Network, network ID, and chain ID match the operator's expected values when supplied.
4. Active peer count meets the configured minimum.
5. Explorer indexer is ready and within the configured lag bound when required.
6. Explorer UI is reachable and serves the expected IndoChain Explorer shell.
7. Optional mining status is reachable and healthy.
8. `POST /node/metrics` is rejected with HTTP 405, proving the monitoring surface remains read-only.

## Relationship to existing health tooling

`scripts/testnet-health.sh` and `scripts/testnet-health.ps1` remain the broader health checks and support additional warnings, seed checks, recent-block checks, CLI validation, and JSON output.

The new operator smoke flow is intentionally smaller: it is the quick pre/post-operation gate for a running node. The existing local `smoke-test` remains a development/CI bootstrap smoke test using temporary datadirs.

## Operational sequence

Recommended order:

1. Start seed/bootstrap connectivity.
2. Start the target node.
3. Run the operator smoke flow with expected network/chain parameters.
4. If the smoke flow fails, inspect `/node/metrics`, `/peer/health`, and `/explorer/indexer/stats`.
5. Run the broader `testnet-health` check when deeper diagnostics are needed.
6. Record the command, timestamp, node endpoint, and result as operational evidence.

## Safety boundaries

- No private keys or wallet secrets are required.
- The script does not call mining or transaction-submission methods.
- Monitoring and Explorer remain read-only.
- A passing smoke check is evidence of operational health at that observation point; it is not a production-readiness or security-audit claim.

## Acceptance criteria

- Shell and PowerShell operator smoke scripts exist.
- Both validate the same core node/P2P/Explorer readiness contract.
- The flow validates the public read-only monitoring contract.
- Expected network/chain identity and peer/indexer thresholds are operator-configurable.
- The flow can be run against an already running testnet without changing node state.
- CI remains green on the Phase 9.3 commits before advancing to 9.4.
