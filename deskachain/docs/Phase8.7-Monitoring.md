# Phase 8.7 — Monitoring & Statistics

Phase 8.7 adds read-only operational observability without changing consensus state.

## 8.7.1 Node metrics

`GET /node/metrics` exposes a versioned monitoring payload:

- network and chain identity
- node uptime
- chain height and tip hash
- block and pending-transaction counts
- peer known/active/best-height information
- mempool pending count
- mining metrics and guard state
- explorer indexer status/statistics

The endpoint remains read-only and is allowed by the mainnet launch gate.

## 8.7.2 Chain, transaction, and peer statistics

The node metrics payload also exposes derived operational statistics:

- current tip difficulty
- cumulative chain work
- total transaction count across the local canonical chain
- known peer count
- active peer count
- failed/offline/cooldown/bad peer count
- best known peer height
- best peer lag relative to the local tip

These values are derived from local chain, mempool, and peer-store state at request time. They are monitoring/read-model data and do not participate in consensus, block validation, transaction validity, or fork choice.

## Acceptance

- `/node/metrics` remains read-only.
- Mainnet read-only RPC gate permits `GET /node/metrics`.
- Monitoring fields are derived from the node's current canonical chain and local peer state.
- Monitoring failures do not mutate consensus state.
- CI must remain green before advancing to the next 8.7 subphase.
