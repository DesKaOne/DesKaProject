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