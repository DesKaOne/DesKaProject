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

## 8.7.4 Mining statistics

The existing mining observation data is included in `/node/metrics` under `mining` and is derived from the canonical chain and current network profile. It includes:

- current and next difficulty
- target block time and retarget window
- blocks remaining until retarget
- latest block timestamp and age
- recent block intervals with average/minimum/maximum interval
- recent difficulty samples
- projected retarget direction
- pending transaction and peer counts relevant to mining
- mining guard state added by the RPC handler

The mining statistics are observational only. They do not alter difficulty, mining eligibility, block production, or consensus rules.

### Acceptance

- A mined block is reflected by the `/node/metrics` mining height and latest-block fields.
- Difficulty/retarget and interval fields are present with stable JSON types.
- Mining statistics are derived from canonical chain state rather than fabricated counters.
- CI must remain green before advancing to 8.7.5.
