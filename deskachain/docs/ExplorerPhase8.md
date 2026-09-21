# IndoChain Explorer — Phase 8 Contract

Phase 8 upgrades the existing read-only explorer toward the future DesKaScan surface without changing consensus, transaction rules, wallet behavior, or the native dIDR economic model.

## 8.1 Contract goals

chain / mempool → explorer read model → explorer API → Explorer UI / future DesKaScan

The explorer is derived data. Deleting or rebuilding explorer data must never invalidate the chain.

### API contract

Stable contract version: v1.

Explorer collection endpoints use: limit, offset, count, total_count, next_offset, prev_offset.

Universal search returns typed results for block, tx, address, and asset. Each result provides both a UI path and API path.

## 8.2 Indexer boundary

The indexer is explicitly separated from consensus. It exposes indexing mode, indexed height, current chain height, lag, readiness, rebuild-required state, last indexed hash, and schema version.

Canonical lag is chain_height - indexed_height.

### Reorg rule

Before advancing its durable cursor, an indexer must verify the parent/hash relationship. If the indexed tip is no longer canonical, it enters rebuild/recovery mode instead of silently serving stale data.

## 8.3 Search contract

Universal search accepts one query and detects block height, block hash, transaction ID, network-valid address, or asset ID. Ambiguous hashes may return multiple typed results.

## 8.4 Asset/token readiness

Asset exploration remains a read-model concern. Keep these concepts distinct: native asset dIDR, issued assets/tokens, transaction fee asset, and optional authorized paymaster. Issued-token ownership does not redefine the native fee model.

## 8.5 Performance boundary

The UI consumes explorer endpoints rather than scanning raw RPC responses. Future persistent indexes are replaceable behind this contract and must be rebuildable from canonical chain data.

## 8.6 Next slice

- persistent block/transaction indexes
- address history indexes
- asset indexes
- indexer cursor/recovery
- explorer indexer status endpoint
- asset-aware universal search
- testnet soak-test metrics

### Phase 8.3 lifecycle

The persistent explorer indexer runs in the RPC server lifecycle and advances its durable cursor in the background. Indexed read endpoints remain read-only and return an explicit not-ready response until the index catches up.

### Phase 8.4 soak metrics

The persistent indexer exposes `GET /explorer/indexer/stats` with indexed/chain height, lag, readiness, record counts, sync count, last sync timestamp, last sync duration, last sync block count, and blocks-per-second. Sync metrics are persisted in the explorer `meta` bucket so they survive RPC restarts and can be used during testnet soak monitoring. Metrics describe the explorer read model only and do not participate in consensus. Sync failures are also persisted with a failure count and the latest error string so a long-running testnet soak can distinguish healthy idle syncs from repeated indexing failures.
