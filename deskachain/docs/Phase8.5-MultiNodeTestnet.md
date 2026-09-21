# Phase 8.5 — Multi-Node Testnet

Phase 8.5 validates IndoChain testnet behavior across multiple independent nodes. The goal is convergence and operational observability, not a change to consensus.

## Scope
- Node A/B/C run with independent datadirs and persistent node identities.
- Seed/bootstrap peers are used only for discovery and synchronization.
- Peers validate network ID, chain ID, protocol compatibility, and genesis identity.
- Nodes converge to the same canonical height and tip after synchronization.
- Peer metadata survives restart and supports health/retry diagnostics.
- Node health exposes known/active peer counts.
- Public RPC remains read-only; wallet/admin RPC must stay disabled on public-safe nodes.

## Automated Acceptance

The P2P test suite now includes a three-node convergence test:
1. Create three independent nodes.
2. Mine blocks on Node A.
3. Synchronize Node B from Node A.
4. Synchronize Node C from Node B.
5. Assert all three nodes have the same height and tip hash.
6. Validate the resulting chains on Node B and Node C.

Existing P2P coverage also validates persistent node identity, handshake network/genesis/protocol rejection, peer introduction, peer-store persistence, peer reputation/latency/cooldown, block synchronization, transaction/block broadcast, and peer health/status behavior.

## Manual Multi-Node Matrix

| Check | Node A | Node B | Node C | Expected |
|---|---|---|---|---|
| Network ID | testnet | testnet | testnet | identical |
| Chain ID | testnet chain | testnet chain | testnet chain | identical |
| Genesis | canonical | canonical | canonical | identical |
| Peer discovery | seed | seed/introduction | seed/introduction | reachable |
| Height | H | H | H | converge |
| Tip hash | T | T | T | identical |
| Chain validation | pass | pass | pass | all pass |
| Public RPC safety | read-only | read-only | read-only | wallet/admin disabled |
| Restart recovery | pass | pass | pass | state persists |
| Explorer indexer | ready/catching up | ready/catching up | ready/catching up | no silent stale state |

## Synchronization Checks

Run peer and chain diagnostics on every node using the existing `peer list`, `peer health`, `chain info`, and `chain validate` commands.

The existing `scripts/testnet-health.sh` and `scripts/testnet-health.ps1` can enforce expected network identity, minimum peers, peer-list visibility, RPC safety, and mining freshness.

## Restart / Recovery

For each node: record height/tip/peer count/indexer height; stop cleanly; restart with the same datadir; confirm node identity and peer metadata persist; confirm catch-up after another node advances; run chain validation; and confirm explorer indexing catches up independently.

A failed explorer indexer must not invalidate the canonical chain. Explorer data remains a rebuildable read model.

## Fork / Reorg Observation

Phase 8.5 observes fork/reorg behavior without changing consensus rules. During controlled testing, record competing tips and heights, peer status, and synchronization events; confirm existing canonical-chain/reorg validation rules are applied; confirm explorer indexing does not silently retain a stale canonical tip; and confirm all nodes converge to the same canonical tip after propagation.

A temporary tip difference during propagation is not by itself a consensus failure.

## Operational Health

Use `/health`, `/explorer/status`, `/explorer/indexer/stats`, `/peer/health`, and `/peer/list` together. Observe active peer count, chain height, tip hash, explorer indexed height/lag/readiness, indexer sync failures, latest sync error, and mining freshness where applicable.

## Exit Criteria

Phase 8.5 is ready to move to Phase 8.6 when three-node synchronization converges in automated tests; peer identity and network/genesis validation remain green; the intended LAN/Tailscale/VPS topology works; restart recovery preserves node/peer state; controlled fork/reorg observations converge; public-safe nodes keep wallet/admin RPC disabled; and health plus explorer metrics provide sufficient evidence for soak testing.

Phase 8.6 will use this topology as the baseline for long-running soak, transaction load, mining load, mempool pressure, restart/recovery, peer churn, and explorer indexer recovery.
## 8.5.4 Operator health/runbook validation

The testnet health scripts are the operator-facing acceptance check for a running node. In addition to chain, peer, RPC-safety, mining, and Explorer checks, they now validate the Explorer indexer through `/explorer/indexer/stats`.

Recommended acceptance command:

```sh
./scripts/testnet-health.sh http://127.0.0.1:9311 \\
  --expected-network testnet \\
  --expected-network-id ind-testnet-1 \\
  --expected-chain-id 777101 \\
  --min-peers 2 \\
  --check-peer-list \\
  --require-indexer-ready \\
  --max-indexer-lag 2
```

The PowerShell equivalent exposes `-RequireIndexerReady` and `-MaxIndexerLag`.

The indexer check is intentionally operational/read-model validation: indexer readiness or lag does not alter consensus state. A node can therefore be healthy at the chain layer while its Explorer read model is still catching up; operators can make readiness/lag a hard acceptance gate when required.

### 8.5.4 exit criteria

- `/health` succeeds and reports the expected network identity.
- `/explorer/status` succeeds.
- Public RPC safety is preserved; wallet/admin RPC are not exposed unless explicitly allowed by the operator check.
- Peer health/list checks can confirm the expected topology.
- `/explorer/indexer/stats` is reachable and, for acceptance, reports `ready=true` with lag within the configured threshold.
- The same checks are available from both shell and PowerShell runbooks.
- Controlled fork/reorg and three-node convergence tests remain green in CI.

## 8.5.5 Phase completion

Phase 8.5 is considered complete after the operator/runbook checks above are validated in CI and the multi-node acceptance matrix covers convergence, health identity, restart/recovery, and controlled fork/reorg behavior. The next stage is Phase 8.6 soak testing.
