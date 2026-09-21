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

## 8.7.5 Transaction and mempool statistics

`/node/metrics` now exposes read-only mempool statistics:

- pending transaction count
- aggregate pending fees
- aggregate native dIDR amount pending
- aggregate issued-asset amount pending, kept separate because issued assets are distinct from native dIDR
- oldest and newest pending transaction timestamps
- pending transaction counts by transaction type

The endpoint derives these values directly from the persisted mempool snapshot at request time. It does not mutate, reorder, admit, reject, or clear mempool transactions.

### Acceptance

- A pending transaction is reflected in the mempool count, fee total, native amount, and type count.
- Issued-asset amounts remain separated from native dIDR amounts.
- Empty mempool remains represented by zero counts/totals without requiring special endpoint behavior.
- CI must remain green before advancing to 8.7.6.

## 8.7.6 Explorer indexer statistics

`/node/metrics` carries the same persistent explorer indexer statistics exposed by `/explorer/indexer/stats`, including:

- indexed height, chain height, and lag
- ready/sync status and schema version
- indexed block, transaction, address-history, and issued-asset event counts
- sync count and last sync timestamp/duration/block count
- observed blocks-per-second
- sync failure count and last sync error when present

The node metrics endpoint reads the explorer indexer's existing read model; it does not trigger indexing, rebuild the database, or alter chain state. Explorer indexer failures remain isolated to the monitoring payload.

### Acceptance

- A synchronized explorer index is reflected consistently in `/node/metrics.indexer.stats`.
- Indexer counts and cursor height match the persistent explorer indexer's statistics.
- Ready/sync status and schema version are exposed with stable JSON values.
- Monitoring remains read-only and does not participate in consensus.
- CI must remain green before advancing to 8.7.7.

## 8.7.7 Soak-test runtime metrics

The /node/metrics endpoint exposes a `runtime` snapshot for bounded and long-running soak observation:

- `observed_at_unix` — wall-clock observation timestamp
- `uptime_seconds` — current node process uptime
- `chain_height` — chain height observed at the same metrics request
- `active_peer_count` — active peer count observed at the same request
- `mempool_pending_count` — pending transaction count observed at the same request

The runtime snapshot is intentionally lightweight and read-only. It gives operators a consistent point-in-time sample that can be polled during soak tests without introducing a separate mutable consensus metric store.

### Acceptance

- Runtime metrics are present with stable JSON types.
- Observation timestamps do not move backwards between successive samples.
- Uptime is non-negative.
- Chain, peer, and mempool runtime counts reflect the same request snapshot.
- Runtime monitoring does not alter consensus or node state.
- CI must remain green before advancing to 8.7.8.

## 8.7.8 Monitoring endpoint/API contract

The public monitoring contract is hardened around `GET /node/metrics`:

- the payload uses schema version `v1`
- the endpoint exposes the consolidated node, chain, peer, mempool, mining, explorer-indexer, and soak-runtime sections
- the endpoint is read-only and rejects non-GET writes
- the contract is suitable for polling by external monitoring/operations tooling without introducing a mutable consensus metric store

### Acceptance

- Public RPC can read `GET /node/metrics` with a stable `v1` schema marker.
- All consolidated monitoring sections are present in the public response.
- `POST /node/metrics` is rejected with HTTP 405.
- Monitoring remains observational and does not alter consensus state.
- CI must remain green before advancing to 8.7.9.
