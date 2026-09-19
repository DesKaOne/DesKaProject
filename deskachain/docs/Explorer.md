# DesKaChain Explorer

Phase 4.5 hardens the safe read-only explorer web UI and API for public-testnet use. It adds `/explorer/search`, pagination metadata, friendlier API errors, and UI pagination/search polish.

Testnet IDR has no monetary value. Staking remains collateral-only. Service points are simulation-only and are not spendable IDR.

## Safety Model

- The explorer web UI is served at `/explorer-ui/`.
- Explorer endpoints are under `/explorer/*`.
- Explorer UI/API endpoints are `GET` only.
- Explorer UI/API endpoints do not create wallets, export private keys, sign transactions, send transactions, mine, request faucet funds, lock/unlock stake, register services, submit service challenges, clear mempool, reset datadirs, or run admin actions.
- Explorer UI/API endpoints are available in public RPC mode without enabling wallet/admin RPC.
- Public RPC should still keep wallet/admin disabled.

## Web UI

Open the UI from the node RPC host:

```text
http://127.0.0.1:9311/explorer-ui/
http://100.101.251.7:9311/explorer-ui/
```

The UI is a small embedded HTML/CSS/JavaScript single-page app. It has no npm build step, no external CDN, no tracking, and no wallet/admin/write actions.

Views included:

- dashboard,
- recent blocks with limit and paging controls,
- mining/difficulty panel,
- block detail by height or hash,
- transaction detail,
- address summary with recent transaction history and stake records,
- stake records,
- service-node simulation summary,
- search by height, block hash, transaction id, or IDR address.
- copy buttons for long hashes and addresses.

The UI fetches the same-host API by default, for example `/explorer/status` and `/explorer/blocks?limit=20`. An optional `api` query parameter can point at another RPC base URL only when the target node's CORS settings allow it:

```text
http://127.0.0.1:9311/explorer-ui/?api=http://127.0.0.1:9311
```

## Current Indexer Mode

The current explorer uses simple bounded scans over chain storage and mempool. This is acceptable for public-testnet rehearsal and keeps consensus unchanged.

Future persistent indexer work may add:

- a reindex command,
- a SQLite or Postgres read model,
- address and transaction indexes,
- block search indexes,
- reorg-aware index updates.

Those future indexes must remain derived data only; they must not change block validation or consensus rules.

## Endpoints

```text
GET /explorer/status
GET /explorer/search?q=<query>
GET /explorer/blocks?limit=20&offset=0
GET /explorer/blocks/{height}
GET /explorer/block/{hash}
GET /explorer/tx/{txid}
GET /explorer/address/{address}
GET /explorer/address/{address}/txs?limit=20&offset=0
GET /explorer/address/{address}/stakes
GET /explorer/stakes?limit=20&offset=0&status=active
GET /explorer/services?limit=20&offset=0
GET /explorer/service/{address}
```

Limits default to `20` and are capped at `100`. Offset defaults to `0`. Negative, zero, or non-integer limits return `400`; negative or non-integer offsets return `400`. Too-large limits are capped to `100`.

Paginated responses include:

- `limit`
- `offset`
- `count`, the returned page size
- `total_count`, the number of records found by the bounded scan
- `next_offset`, when another page can be inferred
- `prev_offset`, when `offset > 0`

Explorer API errors use JSON:

```json
{
  "ok": false,
  "error": "invalid_limit",
  "message": "limit must be between 1 and 100"
}
```

Invalid input returns `400`. Missing blocks, transactions, and search results return `404`. Unsupported methods on registered GET routes return `405`.

## Examples

```sh
curl http://127.0.0.1:9311/explorer/status
curl "http://127.0.0.1:9311/explorer/search?q=126"
curl "http://127.0.0.1:9311/explorer/search?q=<BLOCK_HASH>"
curl "http://127.0.0.1:9311/explorer/search?q=<TXID>"
curl "http://127.0.0.1:9311/explorer/search?q=<ADDRESS>"
curl "http://127.0.0.1:9311/explorer/blocks?limit=10"
curl http://127.0.0.1:9311/explorer/blocks/1
curl http://127.0.0.1:9311/explorer/block/<BLOCK_HASH>
curl http://127.0.0.1:9311/explorer/tx/<TXID>
curl http://127.0.0.1:9311/explorer/address/<ADDRESS>
curl "http://127.0.0.1:9311/explorer/address/<ADDRESS>/txs?limit=10"
curl http://127.0.0.1:9311/explorer/address/<ADDRESS>/stakes
curl "http://127.0.0.1:9311/explorer/stakes?status=active"
curl "http://127.0.0.1:9311/explorer/services?limit=10&offset=0"
curl http://127.0.0.1:9311/explorer/service/<ADDRESS>
```

## Search

`/explorer/search?q=<query>` detects:

- block height integer,
- 64-character block hash,
- 64-character transaction id,
- `IDR...` address for the active network.

The response contains typed results with a UI `path` and API `api_path`. Hash searches try block and transaction matches and can return more than one typed result if an id is ambiguous.

## Status Response

`/explorer/status` returns network identity, tip, difficulty, maturity, supply, pending transaction count, peer count, RPC mode flags, `mainnet_available: false`, and a warning that testnet IDR has no monetary value.

## Blocks And Transactions

Block list is newest first. Block detail includes header fields and transaction summaries. Transaction detail reports confirmed or pending status, block metadata when confirmed, transaction type, from/to fields, amount, fee, involved addresses, and confirmations.

Unknown block or transaction returns `404`. Invalid height, hash, or tx id returns `400`.

## Address Views

Address endpoints validate the node's active network. A testnet node rejects localnet addresses and invalid addresses.

Address summary returns public ledger-derived data such as confirmed, mature, immature, spendable balance, stake amounts, tx count, first/last seen height, service collateral fields, and local simulation service points when present.

No private wallet data is exposed.

## Stake Views

Stake endpoints are read-only. They expose stake id, owner, amount, status, lock height, unlock height, release height, unbonding period, and confirmations.

Stake lock/unlock remains wallet RPC and is disabled by default in public RPC mode.

## Service Views

Service endpoints expose local service simulation state:

- owner address,
- endpoint,
- registered status,
- last heartbeat,
- score,
- simulated points,
- required stake,
- active stake,
- collateral eligibility.

Important: service registration, challenge samples, and simulated rewards are local to this RPC node's service store. They are not global consensus state. Stake collateral itself is chain-backed and syncs between peers.

Service points are simulation-only and are not IDR.
