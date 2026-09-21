# Phase 9.4 — Backup, Restore & Data Recovery

## Objective
Define a conservative recovery boundary for persistent IndoChain testnet state without changing consensus behavior.

The backup unit is the node persistent datadir. When stored below that directory, chain/block state, node/P2P identity, mempool state, Explorer index data, and operator-local persistent configuration are treated as one filesystem snapshot.

## Backup boundary
A backup must be taken from a STOPPED node.

Included when stored below the datadir:
- canonical chain/block state;
- persistent node/P2P identity;
- mempool state;
- Explorer index/read-model state;
- operator-local persistent configuration.

Private keys and identity material are sensitive even on testnet. Backups must not enter source control or CI logs.

## Rebuild versus restore

| State | Preferred recovery |
|---|---|
| Explorer index only | Rebuild read model from canonical chain |
| Mempool | Restore only when preserving pending work is required; otherwise discard/repopulate |
| Chain/block state | Restore verified backup or resynchronize from approved peers |
| P2P identity | Restore the node's own identity if continuity is required; never copy it to another node |
| Operator configuration | Restore only after checking network/chain identity and endpoint bindings |

Explorer index data remains non-consensus state.

## Backup procedure
1. Stop the target node cleanly.
2. Confirm the datadir is no longer being written.
3. Create a filesystem backup with the testnet-backup shell or PowerShell script.
4. Record node role, timestamp, network ID, chain ID, height/tip, and backup filename.
5. Store the backup outside the live datadir.
6. Protect identity/private-key material with filesystem permissions.

Example:
    ./scripts/testnet-backup.sh --datadir ./data/testnet-node-a --output ./backups/node-a.tar.gz

## Restore drill
Restore into a NEW EMPTY datadir first. Do not overwrite an unrelated live node.

1. Stop the destination node.
2. Create a fresh empty destination datadir.
3. Extract/copy the backup into that datadir.
4. Verify ownership and permissions.
5. Start with the same intended network/chain configuration.
6. Run chain info and chain validate.
7. Run /health, /peer/health, and /node/metrics.
8. If Explorer is enabled, verify /explorer/status and /explorer/indexer/stats.
9. Run the existing testnet smoke/health checks.
10. Compare restored height/tip with recorded backup evidence.

## Recovery choices

### Explorer-only loss
Rebuild the Explorer index from canonical chain. Do not restore an old index merely to make the UI appear synchronized.

### Chain-state loss
Use a verified backup or approved peer resynchronization. A peer is not automatically a trusted authority; normal chain validation and network invariants remain authoritative.

### Node-identity loss
Provision a new identity if continuity is not required. If continuity is required, restore the node's own identity from protected backup. Never clone that identity into another independent node.

### Mempool loss
Pending transactions may be lost when the mempool is discarded. This is operational state, not canonical committed chain state.

## Recovery evidence
Record backup identifier/timestamp, source role/datadir, network ID, chain ID, recorded height/tip hash, backup location, restore target, restored height/tip, chain validation result, health/metrics result, Explorer/indexer result, and any discarded pending transactions.

## Acceptance criteria
- POSIX and PowerShell backup tooling exists.
- Backup source is explicitly a stopped datadir.
- Restore is documented into a new empty datadir.
- Chain, identity, mempool, and Explorer recovery boundaries are explicit.
- Explorer loss is recoverable by rebuild.
- Recovery evidence records chain identity and tip.
- Backup material does not enter source control or CI logs.
- No consensus behavior changes are required.
