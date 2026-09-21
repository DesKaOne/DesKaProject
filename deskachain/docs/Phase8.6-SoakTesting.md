# Phase 8.6 — Soak Testing

Phase 8.6 validates IndoChain behavior under sustained and repeated activity using the Phase 8.5 multi-node topology. The purpose is to expose state, synchronization, mempool, restart/recovery, peer-churn, and Explorer indexer issues that short functional tests may not reveal.

## Soak lanes

### 8.6.1 Long-running chain
- Repeated block production over an extended test window.
- Track height, tip hash, cumulative work, and chain validation.
- Confirm no unexpected divergence between nodes.

### 8.6.2 Transaction load
- Submit repeated valid transactions through the supported transaction path.
- Track accepted, confirmed, rejected, and pending transactions.
- Confirm balances, nonces, fees, and transaction history remain consistent.

### 8.6.3 Mining load
- Continue block production across the testnet.
- Observe current/next difficulty, target block interval, retarget direction, and last-block age.
- Confirm difficulty transitions do not invalidate the chain.

### 8.6.4 Mempool pressure
- Build bounded pending transaction batches.
- Observe pending count while blocks are produced.
- Confirm confirmed transactions leave the mempool and invalid transactions are rejected or removed according to existing rules.

### 8.6.5 Restart/recovery
For each node:
1. Record height, tip, peer state, and Explorer indexer stats.
2. Stop cleanly.
3. Restart using the same datadir and node identity.
4. Confirm peer metadata and chain state persist.
5. Allow catch-up from another node.
6. Validate the chain.
7. Confirm Explorer indexer catches up independently.

### 8.6.6 Peer churn
- Repeatedly disconnect/reconnect test peers.
- Recreate test P2P servers between sync cycles.
- Confirm synchronization resumes without corrupting canonical state.
- Observe peer health and active peer counts.

### 8.6.7 Explorer indexer recovery
- Run the chain while the Explorer indexer is catching up.
- Observe indexed height, chain height, lag, readiness, sync count, and failure metrics.
- Exercise restart/rebuild paths without treating Explorer data as consensus state.
- Confirm indexed data eventually converges to the canonical chain.

## Automated bounded soak coverage

The P2P suite includes `TestBoundedMultiNodeSoak`, which provides a CI-safe bounded approximation of the long-running testnet pattern:

1. Node A produces an initial 8-block chain.
2. Node B synchronizes from A and validates the chain.
3. A advances another 8 blocks.
4. Node C synchronizes from A.
5. Node B repeats synchronization from A.
6. C produces two additional blocks.
7. A synchronizes from C.
8. A, B, and C are checked for matching tips and validated chains.
9. P2P test servers are recreated between synchronization cycles to exercise peer/session churn.

The P2P suite also includes `TestBoundedMiningLoad`, which performs three sequential block-production cycles on one node and checks block heights, timestamps, difficulty, final tip, elapsed execution, and canonical chain validation.

For mempool pressure, `TestBoundedMempoolPressure` keeps two valid transactions pending from one funded sender, verifies pending-outgoing accounting, confirms duplicate transaction rejection does not change the queue, then drains the transactions across two blocks and validates the final chain state.

For restart/recovery, `TestNodeRestartRecovery` reopens the same persistent paths after block production and verifies that the chain tip and canonical validation remain unchanged across the restart boundary.

For peer churn, `TestBoundedPeerChurn` recreates the P2P server for three synchronization cycles, advances the source node between sessions, and verifies that the peer resumes synchronization and converges to the same canonical tip after each reconnect.

These tests are intentionally bounded for CI. They complement, rather than replace, an extended manual or scheduled soak run.

## Operator measurements

Use the existing testnet health scripts together with:

- `/health`
- `/explorer/status`
- `/explorer/indexer/stats`
- `/peer/health`
- `/peer/list`
- mining status/difficulty endpoints
- `chain info`
- `chain validate`

At minimum record:

| Area | Measurements |
|---|---|
| Chain | height, tip hash, cumulative work |
| Blocks | production rate, interval, last-block age |
| Transactions | accepted/confirmed/rejected/pending |
| Mempool | pending count and drain behavior |
| Peers | known peers, active peers, reconnect behavior |
| Explorer | indexed height, lag, readiness |
| Indexer | sync count, duration, blocks/sec, failures, last error |
| Recovery | restart time, catch-up height, final tip |

## Exit criteria

Phase 8.6 can advance when:
- bounded multi-node soak remains green in CI;
- extended chain runs maintain valid canonical state;
- repeated synchronization converges;
- peer churn does not corrupt state;
- restart/recovery preserves identity and chain state;
- transaction/mempool load remains internally consistent;
- mining/difficulty behavior remains observable and valid;
- Explorer indexer catches up and exposes failures without affecting consensus;
- collected metrics are sufficient to diagnose regressions before Phase 8.7 monitoring work.

Phase 8.7 will formalize the monitoring/statistics layer using these soak measurements.
