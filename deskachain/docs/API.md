# IndoChain API

Status: Phase 3.6  
Network support: `localnet`, `testnet`  
Primary RPC protocol: HTTP JSON  
P2P protocol: HTTP JSON, node-to-node  
Ticker: `dIDR`

This document describes the current IndoChain HTTP RPC and P2P APIs. It is intended for CLI, wallet, miner, service-node, faucet, explorer, and testnet tooling.

> Testnet dIDR has no monetary value. Staking is collateral-only. Service points are simulation-only and are not spendable dIDR. PoW remains the only block production consensus.

## Base URLs

Typical local RPC node:

```text
http://127.0.0.1:8811
```

Typical local P2P endpoint:

```text
http://127.0.0.1:9811
```

Example testnet node start:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/faucet_tn node start --rpc :8811 --p2p :9811 --advertise-p2p http://127.0.0.1:9811
```

Example public-testnet node using seed peers:

```powershell
go run ./node/cmd/indochain --datadir ./data/testnet-public --network testnet node start --rpc :8811 --p2p :9811 --advertise-p2p http://<PUBLIC_HOST>:9811 --public-rpc --seed-file ./examples/testnet/testnet-seeds.txt
```

Seed peers are startup configuration, not a separate public RPC API. They are persisted in the peer store with source `seed`; network ID, chain ID, genesis hash, and protocol compatibility are still enforced by peer validation and sync paths.

Example testnet node with faucet enabled:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/faucet_tn node start --rpc :8811 --p2p :9811 --advertise-p2p http://127.0.0.1:9811 --enable-faucet-rpc --faucet-address <FAUCET_ADDR> --faucet-amount 100 --faucet-min-interval 1m
```

For a faucet-funded service collateral scenario, use `--faucet-amount 1000 --faucet-max-per-address 2000` on a controlled testnet node.

## Common Response Rules

Most successful responses are JSON objects. Some fields are numbers, while human-friendly amounts are usually strings formatted in dIDR.

Errors usually return HTTP `400`, `403`, `413`, or `429`:

```json
{
  "ok": false,
  "error": "error message"
}
```

Body size limits:

| Category | Limit |
|---|---:|
| General JSON body | 1 MiB |
| Miner template body | 128 KiB |
| Service request body | 128 KiB |
| Block submit/body | 8 MiB |

Rate limits may return:

```json
{
  "ok": false,
  "error": "rate limit exceeded"
}
```

## Network Profiles

Current important profiles:

| Network | Network ID | Chain ID | Coinbase maturity | Min stake | Min service stake | Unbonding |
|---|---|---:|---:|---:|---:|---:|
| `localnet` | `ind-local-1` | `777001` | `10` | `10 dIDR` | `100 dIDR` | `10` blocks |
| `testnet` | `ind-testnet-1` | `777101` | `100` | `100 dIDR` | `1000 dIDR` | `100` blocks |

Known genesis hashes:

| Network | Genesis hash |
|---|---|
| `localnet` | `6e1b3fed63a01109cb665c45c76796b7272b9fec08efcf057b6134a6dc71c22c` |
| `testnet` | `db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4` |

## RPC Access Modes

Node startup flags control which RPC groups are available.

| Group | Examples | Notes |
|---|---|---|
| Generic/read | `/health`, `/chain/info`, `/mining/status`, `/balance/{address}`, `/stake/info` | Usually available even in public RPC mode. |
| Admin | `/peers`, `/peers/sync`, `/reorg/apply`, `/mempool/clear`, `/debug/*` | Disabled in public RPC mode unless explicitly enabled. |
| Wallet | `/wallet/new`, `/wallets`, `/stake/lock`, `/stake/unlock` | Disabled in public RPC mode unless explicitly enabled. |
| Miner | `/miner/template`, `/miner/submit`, `/mine` | Controlled by miner RPC flag; disabled by default in public RPC mode unless explicitly enabled. |
| Service | `/service/register`, `/service/heartbeat`, `/service/challenge/*` | Controlled by service RPC flag. |
| Faucet | `/faucet/request` | Disabled by default and testnet-only. |

When an endpoint is disabled, the response is usually:

```json
{
  "ok": false,
  "error": "endpoint disabled in public RPC mode"
}
```

Service-write endpoints can return:

```json
{
  "ok": false,
  "error": "service RPC disabled"
}
```

## Mining Observation

Read-only mining observation endpoints are public-safe and do not enable block template or submit operations:

```text
GET /mining/status
GET /mining/stats
GET /mining/difficulty
GET /mining/blocks?limit=30
```

These endpoints expose public chain metrics such as height, current difficulty, next difficulty, target block time, retarget window, recent block intervals, projected retarget direction, coinbase maturity, pending transaction count, and peer count. They do not expose private keys, wallet data, or admin controls.

In public RPC mode, wallet/admin/miner/faucet/service write endpoints are disabled by default. Enable miner RPC explicitly with `--enable-miner-rpc` only when the node should accept direct solo miner traffic.

## Address And Amount Format

Addresses use `iND...` Base58Check format. Address validation is network-profile aware, even when the visible prefix is still `iND`.

Amounts in public CLI/docs are written as decimal iND strings, for example:

```json
{
  "amount": "100"
}
```

Transaction JSON stores base units as unsigned integers:

```json
{
  "amount": 10000000000
}
```

Current examples use `1 dIDR = 100000000` base units.

## Transaction Object

A transaction object has this shape:

```json
{
  "id": "<TX_ID>",
  "from": "<IND_ADDR_OR_COINBASE>",
  "to": "<IND_ADDR>",
  "amount": 10000000000,
  "fee": 0,
  "nonce": 1,
  "timestamp": 1782050193,
  "signature": "<HEX>",
  "public_key": "<HEX>",
  "coinbase": false,
  "type": "transfer",
  "stake_id": ""
}
```

Transaction types:

| Type | Meaning |
|---|---|
| `transfer` | Normal wallet transfer. |
| `coinbase` | Miner reward transaction. |
| `stake_lock` | Locks mature spendable funds as staking collateral. |
| `stake_unlock` | Starts unbonding for an active stake. |

## Block Object

A block object has this shape:

```json
{
  "height": 197,
  "previous_hash": "<HASH>",
  "timestamp": 1782050193,
  "nonce": 7794265,
  "difficulty": 6,
  "miner_address": "<IND_ADDR>",
  "transactions": [],
  "merkle_root": "<HASH>",
  "hash": "<HASH>",
  "genesis_marker": ""
}
```

---

# RPC API

## Health And Node Status

### `GET /health`

Returns a compact node health snapshot.

Example:

```powershell
Invoke-RestMethod http://127.0.0.1:8811/health
```

Response fields include:

```json
{
  "ok": true,
  "network": "testnet",
  "network_id": "ind-testnet-1",
  "chain_id": 777101,
  "genesis_hash": "db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4",
  "height": 197,
  "tip_hash": "<HASH>",
  "peers": 1,
  "mempool": 0,
  "public_rpc": false,
  "service_rpc": true,
  "service_nodes": 0,
  "staking_enabled": true,
  "active_stake_count": 0,
  "total_active_stake": "0",
  "uptime_seconds": 123
}
```

### `GET /ready`

Readiness check. Returns `200` when the node can read current chain state.

```json
{
  "ok": true
}
```

### `GET /network/info`

Returns the active network profile.

```json
{
  "name": "testnet",
  "network_id": "ind-testnet-1",
  "chain_id": 777101,
  "genesis_hash": "<HASH>"
}
```

### `GET /node/id`

Returns this node's persistent P2P node ID.

```json
{
  "node_id": "<NODE_ID>"
}
```

### `GET /node/status`

Returns local node state, including datadir, network, height, tip, peer count, RPC/P2P listen addresses, mempool count, and lock state.

### `GET /node/compare?peer=<P2P_URL>`

Compares local node status with a peer.

Example:

```powershell
Invoke-RestMethod "http://127.0.0.1:8811/node/compare?peer=http://127.0.0.1:9811"
```

Response:

```json
{
  "in_sync": true,
  "local": {},
  "peer": {}
}
```

## Chain

### `GET /chain/info`

Returns chain, difficulty, supply, transaction, and staking summary.

Example:

```powershell
Invoke-RestMethod http://127.0.0.1:8811/chain/info
```

Important fields:

```json
{
  "network": "testnet",
  "network_id": "ind-testnet-1",
  "chain_id": 777101,
  "genesis_hash": "<HASH>",
  "height": 197,
  "tip_hash": "<HASH>",
  "difficulty": 6,
  "tip_difficulty": 6,
  "next_difficulty": 6,
  "target_block_time": "30s",
  "retarget_window": 30,
  "min_difficulty": 1,
  "max_difficulty": 12,
  "coinbase_maturity": 100,
  "total_supply": "9850 dIDR",
  "circulating_supply": "4850 dIDR",
  "cumulative_work": 123456,
  "pending_tx_count": 0,
  "total_transactions": 197,
  "coinbase_transactions": 197,
  "normal_transactions": 0,
  "staking_enabled": true,
  "min_stake_amount": "100 dIDR",
  "min_service_stake": "1000 dIDR",
  "unbonding_period": 100,
  "total_active_stake": "0 dIDR",
  "total_unlocking_stake": "0 dIDR",
  "active_stake_count": 0
}
```

Supply semantics:

- `total_supply`: all confirmed coinbase rewards.
- `circulating_supply`: mature coinbase supply, including mature coins locked as active/unlocking/released stake.
- Wallet `spendable_balance`: mature balance minus active/unlocking stake, pending stake locks, and pending outgoing transactions.

### `GET /chain/difficulty`

Returns difficulty-related fields.

### `GET /chain/locator`

Returns a block locator for peer sync / common ancestor detection.

Response:

```json
{
  "height": 197,
  "tip_hash": "<HASH>",
  "locator": []
}
```

### `POST /chain/common-ancestor`

Finds the common ancestor for a peer-provided block locator.

Request:

```json
{
  "locator": [
    {"height": 197, "hash": "<HASH>"}
  ]
}
```

Response:

```json
{
  "found": true,
  "height": 196,
  "hash": "<HASH>"
}
```

### `GET /chain/blocks?from=<HEIGHT>&limit=<N>`

Returns blocks starting at `from`.

Example:

```powershell
Invoke-RestMethod "http://127.0.0.1:8811/chain/blocks?from=1&limit=10"
```

### `GET /chain/validate`

Validates the local chain using the active network profile.

Response:

```json
{
  "valid": true,
  "height": 197,
  "blocks": 198,
  "total_supply": "9850 dIDR"
}
```

## Address, Balance, Transactions, And Mempool

### `GET /balance/{address}`

Returns confirmed, mature, immature, staking, and pending balances.

Example:

```powershell
Invoke-RestMethod http://127.0.0.1:8811/balance/INDD...
```

Response:

```json
{
  "address": "INDD...",
  "confirmed_balance": "100",
  "mature_balance": "100",
  "immature_balance": "0",
  "active_stake": "0",
  "unlocking_stake": "0",
  "released_stake": "0",
  "pending_stake_lock": "0",
  "spendable_balance": "100",
  "pending_outgoing": "0",
  "pending_incoming": "0",
  "coinbase_maturity": 100,
  "current_height": 197
}
```

### `GET /address/{address}`

Inspects address activity. Returns balance data plus known confirmed/pending transactions for that address.

### `GET /tx/{tx_id}`

Looks up a transaction in the chain or mempool.

Response fields include:

```json
{
  "found": true,
  "location": "confirmed",
  "block_height": 197,
  "block_hash": "<HASH>",
  "transaction": {}
}
```

### `GET /mempool` and `GET /mempool/list`

Returns pending mempool transactions.

```json
{
  "pending": [],
  "count": 0
}
```

### `POST /mempool/clear`

Admin endpoint. Clears local mempool.

```json
{
  "status": "cleared"
}
```

## Wallet And Transfer

### `GET /wallets`

Wallet RPC endpoint. Lists local wallet addresses from the selected datadir.

```json
{
  "wallets": [
    {
      "address": "INDD...",
      "format": "base58check",
      "network": "testnet",
      "key_curve": "secp256k1",
      "legacy": false
    }
  ],
  "count": 1
}
```

### `POST /wallet/new`

Wallet RPC endpoint. Creates a new wallet using the active network profile.

Response:

```json
{
  "address": "INDD...",
  "format": "base58check",
  "network": "testnet",
  "key_curve": "secp256k1"
}
```

### `POST /send`

Creates and broadcasts a normal pending transfer transaction.

Request:

```json
{
  "from": "<FROM_IND_ADDR>",
  "to": "<TO_IND_ADDR>",
  "amount": "100"
}
```

Response:

```json
{
  "status": "pending",
  "id": "<TX_ID>",
  "tx_id": "<TX_ID>",
  "from": "<FROM_IND_ADDR>",
  "to": "<TO_IND_ADDR>",
  "amount": "100 dIDR",
  "fee": "0 dIDR",
  "nonce": 1,
  "broadcast": {}
}
```

Notes:

- Sender must have enough mature spendable balance.
- Active stake, unlocking stake, pending stake locks, and pending outgoing transactions reduce spendable balance.
- The transaction becomes confirmed only after it is included in a mined block.

## Mining

### `GET /miner/template?address=<IND_ADDR>`

Returns a mining candidate block template for a reward address.

Example:

```powershell
Invoke-RestMethod "http://127.0.0.1:8811/miner/template?address=<IND_ADDR>"
```

Response:

```json
{
  "template_id": "<TEMPLATE_ID>",
  "network": "testnet",
  "network_id": "ind-testnet-1",
  "chain_id": 777101,
  "protocol_version": 1,
  "height": 198,
  "previous_hash": "<HASH>",
  "difficulty": 6,
  "target": "000000ffff...",
  "reward_address": "<IND_ADDR>",
  "coinbase_reward": "50",
  "timestamp": 1782050193,
  "transactions": [],
  "tx_count": 1,
  "selected_txs": 0,
  "header": {},
  "block": {}
}
```

### `POST /miner/template`

Same as `GET /miner/template`, but the address is sent in JSON.

Request:

```json
{
  "address": "<IND_ADDR>"
}
```

### `POST /miner/submit`

Submits a mined block returned from a template.

Request:

```json
{
  "template_id": "<TEMPLATE_ID>",
  "block": {
    "height": 198,
    "previous_hash": "<HASH>",
    "timestamp": 1782050193,
    "nonce": 12345,
    "difficulty": 6,
    "miner_address": "<IND_ADDR>",
    "transactions": [],
    "merkle_root": "<HASH>",
    "hash": "<HASH>"
  }
}
```

Response:

```json
{
  "accepted": true,
  "height": 198,
  "hash": "<HASH>"
}
```

Possible rejection errors:

- `invalid proof of work`
- `stale template`
- `invalid difficulty expected X got Y`
- `invalid coinbase recipient: invalid address: wrong network version`

### `POST /mine`

Built-in RPC miner endpoint. Useful for dev/local testing, not recommended for public mining.

Request:

```json
{
  "address": "<IND_ADDR>",
  "blocks": 1,
  "max_nonce": 0,
  "verbose": false
}
```

Response includes mining job status, mined block count, and committed block hashes.

### `GET /mine/status`

Returns current built-in mining job status.

## Faucet

The faucet is dev/testnet-only and disabled by default. It does not mint silently. It creates a normal transfer from the configured faucet wallet, inserts it into mempool, and requires a mined block for confirmation.

### `GET /faucet/info`

Returns faucet configuration and state.

Response:

```json
{
  "enabled": true,
  "network": "testnet",
  "network_id": "ind-testnet-1",
  "chain_id": 777101,
  "faucet_address": "<FAUCET_ADDR>",
  "amount": "100",
  "max_per_address": "1000",
  "min_interval_seconds": 60,
  "mempool_pending": 0,
  "note": "testnet faucet only; testnet dIDR has no monetary value"
}
```

### `POST /faucet/request`

Requests testnet funds for a recipient address.

Request:

```json
{
  "address": "<RECIPIENT_ADDR>"
}
```

Optional custom amount, capped by faucet settings:

```json
{
  "address": "<RECIPIENT_ADDR>",
  "amount": "100"
}
```

Response:

```json
{
  "tx_id": "<TX_ID>",
  "from": "<FAUCET_ADDR>",
  "to": "<RECIPIENT_ADDR>",
  "amount": "100",
  "status": "pending",
  "note": "mine a block to confirm faucet transaction"
}
```

Common errors:

| Error | Meaning |
|---|---|
| `faucet disabled` | Faucet RPC was not enabled at node start. |
| `faucet is testnet-only` | Faucet was requested on a non-testnet node. |
| `invalid faucet address` | Configured faucet source is missing or wrong network. |
| `invalid address` | Recipient address is invalid or wrong network. |
| `insufficient mature faucet balance` | Faucet wallet does not have enough mature spendable dIDR. |
| `recipient rate limited` | Recipient requested too soon. |
| `pending faucet tx already exists for address` | Recipient already has a pending faucet tx from the faucet. |
| `recipient faucet daily limit exceeded` | Recipient exceeded configured daily limit. |

### Faucet -> Stake -> Service E2E

Short controlled testnet sequence:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 faucet request --address <OWNER_ADDR>
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8911 --address <FAUCET_ADDR> --threads 4 --once
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 stake lock --address <OWNER_ADDR> --amount 1000
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8911 --address <FAUCET_ADDR> --threads 4 --once
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 service register --address <OWNER_ADDR> --endpoint http://127.0.0.1:9971
go run ./node/cmd/indoservice --rpc-url http://127.0.0.1:8911 --address <OWNER_ADDR> --endpoint http://127.0.0.1:9971 --once
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 service score --address <OWNER_ADDR>
```

Expected service metadata: required stake `1000 dIDR`, active stake `1000 dIDR`, `stake_eligible: true`, `collateral_status: "eligible"`, and `eligible_simulated_points > 0`. Faucet, stake, and service flows do not mutate supply directly; supply changes only through mined coinbase rewards.

## Staking

Staking is collateral-only. It does not create PoS consensus, validators, APY, staking rewards, or slashing.

### `GET /stake/info`

Returns staking parameters and summary.

```json
{
  "ok": true,
  "staking_enabled": true,
  "min_stake_amount": "100",
  "min_service_stake": "1000",
  "unbonding_period": 100,
  "total_active_stake": "0",
  "total_unlocking_stake": "0",
  "active_stake_count": 0
}
```

### `GET /stake/list?address=<IND_ADDR>`

Lists stake records. The `address` query is optional.

Response:

```json
{
  "ok": true,
  "stakes": [
    {
      "stake_id": "<STAKE_ID>",
      "owner": "<IND_ADDR>",
      "amount": "1000",
      "status": "active",
      "lock_height": 198,
      "unlock_height": 0,
      "release_height": 0
    }
  ],
  "count": 1
}
```

### `GET /stake/status?id=<STAKE_ID>`

Returns one stake record.

### `POST /stake/lock`

Wallet RPC endpoint. Creates a pending stake lock transaction.

Request:

```json
{
  "address": "<IND_ADDR>",
  "amount": "1000"
}
```

Response:

```json
{
  "ok": true,
  "tx_id": "<TX_ID>",
  "stake_id": "<STAKE_ID>",
  "address": "<IND_ADDR>",
  "amount": "1000",
  "status": "pending",
  "broadcast": {}
}
```

The stake becomes active only after the transaction is mined.

### `POST /stake/unlock`

Wallet RPC endpoint. Creates a pending stake unlock transaction.

Request:

```json
{
  "address": "<IND_ADDR>",
  "stake_id": "<STAKE_ID>"
}
```

Response:

```json
{
  "ok": true,
  "tx_id": "<TX_ID>",
  "stake_id": "<STAKE_ID>",
  "release_height": 298,
  "status": "pending",
  "broadcast": {}
}
```

## Service Node Simulation

Service nodes are simulated/off-chain service-score records. Service points are not spendable dIDR. Active staking collateral can make a service node eligible for simulated rewards/points, but this does not mint dIDR.

### `POST /service/register`

Service RPC endpoint. Registers or updates a service node.

Request:

```json
{
  "address": "<IND_ADDR>",
  "endpoint": "http://127.0.0.1:9701",
  "client_version": "dev",
  "platform": "windows",
  "user_agent": "indoservice"
}
```

Response:

```json
{
  "ok": true,
  "service_node_id": "svc_...",
  "owner_address": "<IND_ADDR>",
  "status": "registered"
}
```

### `POST /service/heartbeat`

Submits heartbeat metrics.

Request fields are the same as `/service/register`.

Response:

```json
{
  "ok": true,
  "status": "active",
  "uptime_score": 100,
  "latency_score": 80,
  "bandwidth_score": 80,
  "reliability_score": 100,
  "abuse_penalty": 0,
  "service_score": 90,
  "flags": [],
  "note": "service points are simulation only and are not spendable dIDR"
}
```

### `POST /service/challenge/create`

Creates a service challenge.

Request:

```json
{
  "address": "<IND_ADDR>"
}
```

Response:

```json
{
  "ok": true,
  "challenge_id": "<ID>",
  "address": "<IND_ADDR>",
  "issued_at": "<TIME>",
  "expires_at": "<TIME>",
  "nonce": "<NONCE>",
  "status": "created"
}
```

### `POST /service/challenge/submit`

Submits challenge result.

Request:

```json
{
  "challenge_id": "<ID>",
  "latency_ms": 50,
  "bytes_up": 10000000,
  "bytes_down": 50000000,
  "success": true
}
```

Response contains updated score fields.

### `GET /service/score?address=<IND_ADDR>`

Returns score and collateral eligibility.

Response:

```json
{
  "ok": true,
  "score": {
    "uptime_score": 100,
    "latency_score": 80,
    "bandwidth_score": 80,
    "reliability_score": 100,
    "abuse_penalty": 0,
    "service_score": 90,
    "simulated_points": 900,
    "required_stake": 100000000000,
    "active_stake": 0,
    "stake_eligible": false,
    "collateral_status": "none",
    "eligible_simulated_points": 0,
    "eligibility_note": "not eligible for service reward simulation until active stake >= required stake",
    "note": "service points are simulation only and are not spendable dIDR"
  }
}
```

### `GET /service/rewards?address=<IND_ADDR>`

Returns simulated reward/point records.

### `GET /service/list`

Returns all known service nodes.

## Peers, Sync, And Reorg

### `GET /peers`

Returns peer store metadata.

Response:

```json
{
  "peers": [
    {
      "url": "http://127.0.0.1:9811",
      "node_id": "<NODE_ID>",
      "network_id": "ind-testnet-1",
      "chain_id": 777101,
      "genesis_hash": "<HASH>",
      "protocol_version": 1,
      "height": 197,
      "tip_hash": "<HASH>",
      "status": "active",
      "score": 100,
      "source": "bootnode",
      "last_seen_at": "<TIME>",
      "last_error": ""
    }
  ],
  "peer_count": 1,
  "max_peers": 128,
  "banned_count": 0
}
```

### `POST /peers`

Admin endpoint. Adds a peer after handshake/profile validation.

Request:

```json
{
  "url": "http://127.0.0.1:9811"
}
```

Response:

```json
{
  "added": true,
  "url": "http://127.0.0.1:9811"
}
```

### `POST /peers/connect`

Admin endpoint. Adds/connects a peer and introduces this node to that peer when possible.

Request:

```json
{
  "url": "http://127.0.0.1:9811"
}
```

Response:

```json
{
  "connected": true,
  "url": "http://127.0.0.1:9811",
  "introduced": true
}
```

### `DELETE /peers`

Admin endpoint. Removes a peer.

Request:

```json
{
  "url": "http://127.0.0.1:9811"
}
```

### `POST /peers/clear`

Admin endpoint. Clears peer store.

Request:

```json
{
  "yes": true
}
```

### `POST /peers/check`

Checks peer profile and updates peer metadata.

Request:

```json
{
  "url": "http://127.0.0.1:9811"
}
```

Success response contains peer handshake/status fields. Network, chain ID, genesis, and protocol mismatches are rejected.

### `POST /peers/status`

Checks all known peers and returns peer status summary.

Request:

```json
{
  "include_bad": false
}
```

### `POST /peers/sync`

Admin endpoint. Syncs blocks from one peer, or from all stored peers if `peer` is omitted.

Request:

```json
{
  "peer": "http://127.0.0.1:9811",
  "include_bad": false,
  "allow_reorg": false,
  "max_reorg_depth": 0,
  "yes": false
}
```

Response:

```json
{
  "synced": true,
  "local_height_before": 196,
  "local_height_after": 197,
  "imported_blocks": 1,
  "peers_checked": 1,
  "results": [
    {
      "peer": "http://127.0.0.1:9811",
      "ok": true,
      "height_before": 196,
      "height_after": 197,
      "imported_blocks": 1,
      "message": "sync complete"
    }
  ]
}
```

Mismatch response example:

```json
{
  "synced": false,
  "error": "peer rejected: network id mismatch"
}
```

### `POST /reorg/preview`

Admin endpoint. Builds a reorg plan from a peer without applying it.

Request:

```json
{
  "peer": "http://127.0.0.1:9811",
  "max_depth": 100
}
```

Response:

```json
{
  "allowed": true,
  "reason": "peer has more cumulative work",
  "network": "testnet",
  "network_id": "ind-testnet-1",
  "chain_id": 777101,
  "local_height": 10,
  "peer_height": 12,
  "common_ancestor_height": 9,
  "disconnect_blocks": 1,
  "connect_blocks": 3,
  "local_work": 100,
  "peer_work": 120
}
```

### `POST /reorg/apply`

Admin endpoint. Applies a reorg plan. Requires `yes: true`.

Request:

```json
{
  "peer": "http://127.0.0.1:9811",
  "max_depth": 100,
  "yes": true
}
```

Response includes old/new height, old/new tip, disconnected/connected block counts, mempool requeue/drop counts, and `chain_valid`.

## Fork Tools

### `POST /fork/check`

Checks fork status against a peer.

Request:

```json
{
  "peer": "http://127.0.0.1:9811"
}
```

### `POST /fork/inspect-datadir`

Admin endpoint. Compares local chain against another datadir.

Request:

```json
{
  "datadir": "./testdata/other_node"
}
```

## Debug Endpoints

### `GET /debug/p2p`

Admin endpoint. Returns node ID, local height/tip, and detailed peer metadata.

### `POST /debug/p2p/ping`

Admin endpoint. Pings a peer URL and stores latency/error metadata.

Request:

```json
{
  "url": "http://127.0.0.1:9811"
}
```

### `GET /debug/locks`

Admin endpoint. Returns mining lock/job status, chain height/tip, mempool count, and peer count.

---

# P2P API

The P2P API is intended for node-to-node communication. Tooling can call it for debugging, but normal apps should prefer RPC endpoints.

## `GET /p2p/health`

Returns compact P2P health:

```json
{
  "ok": true,
  "network": "testnet",
  "network_id": "ind-testnet-1",
  "chain_id": 777101,
  "genesis_hash": "<HASH>",
  "height": 197,
  "tip_hash": "<HASH>",
  "cumulative_work": 123456
}
```

## `GET /p2p/handshake`

Returns profile and node identity used by peer validation.

```json
{
  "network_name": "testnet",
  "network_id": "ind-testnet-1",
  "chain_id": 777101,
  "protocol_version": 1,
  "p2p_protocol_version": "1",
  "min_protocol_version": 1,
  "genesis_hash": "<HASH>",
  "height": 197,
  "tip_hash": "<HASH>",
  "cumulative_work": 123456,
  "node_id": "<NODE_ID>",
  "identity_version": 1,
  "node_public_key": "<ED25519_PUBLIC_KEY_HEX>",
  "node_signature": "<ED25519_SIGNATURE_HEX>",
  "p2p_listen": ":9811",
  "p2p_advertise": "http://127.0.0.1:9811"
}
```

## P2P Message Authentication

Phase 5.15 adds optional application-level Ed25519 authentication for node-to-node HTTP messages.

When a client uses an authenticated node identity, every P2P request carries:

- `X-dIDR-Auth-Version`
- `X-dIDR-Node-ID`
- `X-dIDR-Node-PubKey`
- `X-dIDR-Network-ID`
- `X-dIDR-Chain-ID`
- `X-dIDR-Auth-Timestamp`
- `X-dIDR-Auth-Nonce`
- `X-dIDR-Auth-Signature`

The signature binds the HTTP method, exact request URI, network, chain ID, timestamp, nonce, and SHA-256 digest of the request body. Nonces are replay-protected for the configured five-minute authentication window.

Authenticated responses return the corresponding node identity and a signature bound to the request nonce, HTTP status, timestamp, and response body digest. This protects both request integrity and response integrity at the application layer.

`RequireAuthenticatedNode` is the rollout switch in the active network profile. The current built-in localnet, testnet, and mainnet profiles keep it disabled for compatibility. The handshake identity remains available even while message authentication is optional.

## `GET /p2p/status`

Returns chain/network status with height, tip, difficulty, cumulative work, total supply, and mempool count.

## `GET /p2p/tip`

Returns current tip:

```json
{
  "height": 197,
  "hash": "<HASH>"
}
```

## `GET /p2p/block/{height}`

Returns a single block by height.

## `GET /p2p/blocks?from=<HEIGHT>&limit=<N>`

Returns blocks starting at `from`. Default limit is `100`.

```json
{
  "blocks": []
}
```

## `GET /p2p/headers?from=<HEIGHT>&limit=<N>`

Returns block headers. Default limit is `100`; max limit is `500`.

```json
{
  "headers": [
    {
      "height": 197,
      "hash": "<HASH>",
      "previous_hash": "<HASH>",
      "timestamp": 1782050193,
      "difficulty": 6,
      "merkle_root": "<HASH>",
      "tx_count": 1,
      "miner_address": "<IND_ADDR>"
    }
  ]
}
```

## `GET /p2p/locator`

Returns a block locator for common ancestor discovery.

## `POST /p2p/common-ancestor`

Request:

```json
{
  "locator": [
    {"height": 197, "hash": "<HASH>"}
  ]
}
```

Response:

```json
{
  "found": true,
  "height": 197,
  "hash": "<HASH>"
}
```

## `POST /p2p/tx`

Receives a transaction from a peer and validates it for mempool admission.

Request: transaction object.

Response:

```json
{
  "accepted": true,
  "txid": "<TX_ID>"
}
```

## `POST /p2p/block`

Receives a block from a peer. The block must extend the local tip.

Request: block object.

Response:

```json
{
  "accepted": true,
  "height": 198,
  "hash": "<HASH>"
}
```

If rejected:

```json
{
  "accepted": false,
  "error": "block does not extend local tip",
  "local_height": 197,
  "local_tip": "<HASH>"
}
```

## `POST /p2p/peer`

Receives peer introduction metadata.

Request:

```json
{
  "url": "http://127.0.0.1:9811",
  "node_id": "<NODE_ID>",
  "network_id": "ind-testnet-1",
  "chain_id": 777101
}
```

Response:

```json
{
  "accepted": true
}
```

---

# PowerShell Examples

## Check node

```powershell
Invoke-RestMethod http://127.0.0.1:8811/health
Invoke-RestMethod http://127.0.0.1:8811/chain/info
```

## Create wallet by RPC

```powershell
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8811/wallet/new
```

## Request faucet funds

```powershell
$body = @{ address = "<RECIPIENT_ADDR>" } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8811/faucet/request -ContentType "application/json" -Body $body
```

## Get miner template

```powershell
Invoke-RestMethod "http://127.0.0.1:8811/miner/template?address=<IND_ADDR>"
```

## Submit peer check

```powershell
$body = @{ url = "http://127.0.0.1:9811" } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8811/peers/check -ContentType "application/json" -Body $body
```

---

# Operational Notes

## Public RPC

A public RPC node should keep wallet/admin/service/faucet endpoints disabled unless there is a specific reason to enable them. Public read-only endpoints such as `/health`, `/chain/info`, `/balance/{address}`, `/tx/{id}`, `/mempool`, `/stake/info`, and `/service/score` are safer to expose.

## Faucet Safety

The faucet is for dev/testnet only. It does not create funds directly. It requires:

1. A configured testnet faucet wallet.
2. Mature spendable faucet balance.
3. A valid recipient address for the same network.
4. A mined block to confirm the faucet transaction.

## Mining Safety

`/mine` is a convenience dev endpoint. For normal mining use `indominer` with `/miner/template` and `/miner/submit`.

## P2P Network Guard

Peers are rejected when network ID, chain ID, genesis hash, or protocol version is incompatible. `peer sync` validates the peer before returning `local chain already up to date`, so a wrong-network peer cannot be silently accepted.

## Reorg Safety

Reorg endpoints are admin-only and profile-aware. Use preview first, and require explicit `yes: true` before applying.


## State DB Diagnostics

### `GET /chain/state`

Returns state database metadata without replaying the blockchain:

```json
{
  "available": true,
  "current": true,
  "version": 3,
  "height": 197,
  "state_root": "<STATE_ROOT>",
  "tip_height": 197,
  "tip_hash": "<HASH>"
}
```

`current` is false when the persisted state height/version does not match the canonical tip, or when a canonical block's `StateRoot` does not match the persisted state root.

### `GET /chain/state/validate`

Admin diagnostic endpoint. Replays the canonical chain and compares the result with the persisted state database, including secondary indexes and pending coinbase maturity state.

The endpoint returns HTTP 200 when the state database matches the canonical chain and HTTP 400 with `valid: false` when a mismatch or index corruption is detected.


## Multi-asset RPC

For the v3 asset model:

`GET /asset/info?asset_id=<id>` returns the persisted asset definition.

`GET /asset/balance?address=<address>&asset_id=<id>` returns an address's balance with the asset's decimals plus raw integer units.

The v3 fee asset is native `dIDR`. A token transfer therefore has an asset amount plus an dIDR protocol fee. A non-sender fee payer is represented by the paymaster fields in the transaction envelope and must provide a valid authorization signature.


`GET /asset/balances?address=<address>` returns all non-zero native/token balances indexed for the address.


## Fee / Gas RPC

### `GET /fee/policy`

Returns the active fee policy. In v3 the fee asset is native dIDR and the response includes the gas schedule, minimum fee, gas price, and per-transaction gas limit.

Example:

```json
{
  "ok": true,
  "enabled": true,
  "fee_asset_id": "dIDR",
  "min_fee": 1,
  "min_gas_price": 0,
  "bytes_per_gas": 32,
  "max_gas_per_tx": 100000,
  "base_gas_transfer": 10,
  "base_gas_asset_transfer": 12,
  "base_gas_stake_lock": 12,
  "base_gas_stake_unlock": 8,
  "base_gas_asset_create": 50,
  "base_gas_asset_mint": 30,
  "base_gas_asset_burn": 25,
  "distribution": {
    "model": "block-fee-settlement",
    "recipient": "miner"
  }
}
```

### `POST /fee/estimate`

Estimates gas and the minimum dIDR fee for a transaction envelope.

Request:

```json
{
  "transaction": {
    "version": 3,
    "type": "transfer",
    "from": "<FROM_ADDR>",
    "to": "<TO_ADDR>",
    "asset_id": "asset:example",
    "amount": 100,
    "fee": 5,
    "nonce": 1,
    "timestamp": 1780000000
  }
}
```

Response:

```json
{
  "ok": true,
  "network": "testnet",
  "chain_id": 777101,
  "fee_asset_id": "dIDR",
  "quote": {
    "gas_units": 20,
    "tx_bytes": 160,
    "min_fee": 1,
    "requested_fee": 5,
    "fee_asset_id": "dIDR",
    "sufficient": true
  }
}
```

The exact `gas_units`, `tx_bytes`, and `min_fee` values depend on the active network fee policy and transaction signing envelope.

For v3 issued-asset transfers, the default fee payer is the sender. A different fee payer must provide paymaster authorization, and that payer is charged in native dIDR.
