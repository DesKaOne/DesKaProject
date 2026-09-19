Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 4.2 sudah valid full.
* Phase 4.0 Public Testnet Multi-Host Deployment valid full.
* Phase 4.1 Public Testnet Long-Running Stability & Restart Recovery valid full.
* Phase 4.2 Public Testnet Faucet + Service Node Multi-Host E2E valid full.
* Real multi-host test sudah berhasil:

  * Host A Mini PC Linux sebagai seed/faucet node.
  * Host B Windows sebagai peer/staker/service owner.
  * Faucet request multi-host lolos.
  * Stake lock 1000 DKC lolos.
  * Service register dan dkcservice --once lolos.
  * Service eligible lolos.
  * Host A dan Host B tetap sync.
  * Chain validate pass.
* CI dan release artifact workflow hijau.
* Release binary Windows/Linux valid.
* Public testnet genesis candidate stable:

  * network: testnet
  * network_id: dkc-testnet-1
  * chain_id: 777101
  * genesis hash: db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4
* Testnet DKC has no monetary value.
* Mainnet does not exist yet.
* PoW remains the only block-production consensus.
* Staking remains collateral-only.
* Service points remain simulation-only.

Patch name:
DesKaChain Phase 4.3 — Public Testnet Explorer API & Read-Only Indexer Prep

Goal:
Add a safe read-only explorer/indexer layer for public testnet.

This phase should make it possible to query:

* chain summary,
* recent blocks,
* block detail by height/hash,
* transaction detail by tx id,
* address summary,
* address transaction history,
* address stake summary,
* service node summary,
* faucet/service/staking status as read-only data,
  without exposing wallet/admin/write actions.

This is explorer API + indexer preparation, not full explorer UI.

Non-goals:

* Do not launch mainnet.
* Do not change testnet genesis.
* Do not change localnet genesis.
* Do not change consensus.
* Do not change block/tx format unless absolutely required.
* Do not implement full explorer frontend UI yet.
* Do not implement wallet web UI.
* Do not expose private keys.
* Do not expose wallet/admin RPC.
* Do not add write endpoints to explorer API.
* Do not add staking APY/reward claims.
* Do not make service points spendable.
* Do not promise price/profit/rewards.
* Do not add centralized trust assumptions.

==================================================

1. Explorer API scope
   ==================================================

Add read-only explorer endpoints under a clear namespace.

Recommended prefix:

/explorer/*

or if existing routing style prefers:

/api/explorer/*

Endpoints:

GET /explorer/status
GET /explorer/blocks
GET /explorer/blocks/{height}
GET /explorer/block/{hash}
GET /explorer/tx/{txid}
GET /explorer/address/{address}
GET /explorer/address/{address}/txs
GET /explorer/address/{address}/stakes
GET /explorer/stakes
GET /explorer/services
GET /explorer/service/{address}

Optional:
GET /explorer/mempool
GET /explorer/search?q=<height|hash|txid|address>

All endpoints must be read-only.

No endpoint may:

* create wallet,
* export wallet,
* sign tx,
* send tx,
* lock stake,
* unlock stake,
* register service,
* request faucet,
* mine,
* admin reset,
* expose private key,
* mutate chain state.

==================================================
2. Explorer status
==================

Implement:

GET /explorer/status

Response should include:

* network
* network_id
* chain_id
* genesis_hash
* height
* tip_hash
* difficulty
* next_difficulty
* target_block_time
* retarget_window
* coinbase_maturity
* total_supply
* circulating_supply
* pending_tx_count
* peer_count if easily available
* public_rpc mode flags if already exposed:

  * wallet_rpc
  * admin_rpc
  * miner_rpc
  * faucet_rpc
  * service_rpc
* mainnet_available: false
* testnet_value_warning or equivalent: testnet DKC has no monetary value

Do not expose secrets or local private wallet data.

==================================================
3. Blocks list
==============

Implement:

GET /explorer/blocks?limit=20&offset=0

or cursor style if easier.

Return recent blocks ordered newest first.

Fields:

* height
* hash
* previous_hash
* timestamp
* difficulty
* nonce
* tx_count
* coinbase_tx_id
* miner/reward address if derivable
* reward amount
* cumulative_work if easy
* total_supply_after if easy

Validation:

* limit capped, e.g. max 100.
* default limit 20.
* negative offset rejected.
* invalid params return clear 400.

Do not scan unbounded chain on every request if it can be avoided.
For now, simple scan is acceptable for testnet, but add caps.

==================================================
4. Block detail
===============

Implement:

GET /explorer/blocks/{height}
GET /explorer/block/{hash}

Return:

* block header fields
* tx list summary
* full tx detail if current tx type is simple enough
* coinbase marker
* normal/stake/service tx classification if available
* validation metadata if already available

Error behavior:

* unknown block returns 404.
* invalid height/hash returns 400.
* genesis block returns valid response.

==================================================
5. Transaction detail
=====================

Implement:

GET /explorer/tx/{txid}

Return:

* txid
* block height/hash if confirmed
* status: confirmed/pending/not_found
* timestamp/block time if confirmed
* type:

  * coinbase
  * transfer
  * stake_lock
  * stake_unlock
  * service_register
  * faucet_transfer if distinguishable, otherwise normal transfer
* inputs/outputs or from/to fields based on current model
* amount
* fee if available
* involved addresses
* confirmations

For pending tx:

* include pending status if mempool lookup exists.
* if not in chain or mempool, 404.

==================================================
6. Address summary
==================

Implement:

GET /explorer/address/{address}

Validate address network:

* testnet endpoint/node should reject localnet address.
* localnet node should reject testnet address.
* invalid address returns 400.

Return:

* address
* network
* confirmed_balance
* mature_balance
* immature_balance
* spendable_balance
* active_stake
* unlocking_stake
* total_received if easy
* total_sent if easy
* tx_count
* first_seen_height if easy
* last_seen_height if easy
* service_collateral_required
* service_collateral_eligible
* service_points if service data is local/simulation
* warning that service points are simulation-only if included

If some fields are expensive or unavailable, omit them or set clearly documented null/zero values.
Do not fake data.

==================================================
7. Address transaction history
==============================

Implement:

GET /explorer/address/{address}/txs?limit=20&offset=0

Return txs involving address, newest first.

Fields per item:

* txid
* block height/hash
* status
* type
* amount delta for address if feasible
* timestamp
* confirmations

Cap limit.
Reject invalid address.
Do not include private wallet data.

If full indexing is not implemented yet:

* perform bounded chain scan for now,
* document this is testnet/simple explorer mode,
* create architecture TODO for persistent indexer.

==================================================
8. Stake explorer
=================

Implement read-only stake endpoints:

GET /explorer/address/{address}/stakes
GET /explorer/stakes?limit=20&offset=0&status=active

Return:

* stake id
* owner address
* amount
* status: active/unlocking/released
* lock height
* unlock height if any
* release height if any
* unbonding period
* confirmations if useful

Do not allow stake lock/unlock from explorer endpoint.
Explorer must remain read-only.

==================================================
9. Service explorer
===================

Implement read-only service endpoints:

GET /explorer/services
GET /explorer/service/{address}

Return:

* owner address
* endpoint
* registered status
* last heartbeat if local service store has it
* score
* points
* required stake
* active stake
* collateral eligible
* status: eligible/not_eligible
* simulation_only: true

Important:
If service registry/score is node-local simulation state, document that explorer service endpoint reflects this node’s local service store, not global consensus state.
If service registration is chain-backed, expose confirmed registration height/tx.

Do not claim service points are DKC.
Do not expose service write actions.

==================================================
10. Search endpoint
===================

Optional but useful:

GET /explorer/search?q=<query>

Detect:

* integer height
* block hash
* tx id
* address

Return:

* type
* canonical URL/path
* summary

Rules:

* ambiguous hash can return multiple candidates.
* invalid query returns 400 or empty result.
* no state mutation.

If too much for this phase, skip and document as Phase 4.4/4.5.

==================================================
11. Indexer architecture prep
=============================

Add internal package if useful:

internal/explorer
internal/indexer

Recommended approach:

* Keep Phase 4.3 simple and read-only.
* Use chain storage APIs where possible.
* Avoid duplicating consensus logic.
* Avoid modifying block validation.
* Use helper functions for:

  * list blocks
  * find block by hash
  * find tx by id
  * compute address summary
  * list address txs
  * map stake/service summary

If persistent index is not added yet, add design notes:

docs/Explorer.md

Include:

* current simple scan mode,
* future persistent indexer mode,
* possible SQLite/Postgres index,
* reindex command future,
* no consensus changes.

Do not introduce database dependency unless already cleanly supported.

==================================================
12. Public safety
=================

Explorer endpoints must be safe on public RPC.

Tests:

* explorer endpoints are available in public RPC mode.
* wallet/admin endpoints remain disabled in public RPC.
* explorer endpoints do not require enabling wallet/admin.
* explorer endpoint cannot mutate state.
* faucet request is not available through explorer.
* stake lock/unlock is not available through explorer.
* service register/heartbeat/submit are not available through explorer.

Add explicit negative tests:

* POST to explorer endpoints rejected or method not allowed.
* unknown explorer path returns 404.
* invalid address rejected.
* wrong network address rejected.

==================================================
13. CLI helpers
===============

Optional CLI read-only commands:

deskachain explorer status
deskachain explorer blocks
deskachain explorer block <height|hash>
deskachain explorer tx <txid>
deskachain explorer address <address>

These should call RPC explorer endpoints.
Keep output human-readable.

If CLI work is too much, skip CLI and focus HTTP API.

==================================================
14. Docs
========

Add:

docs/Explorer.md

Include:

* endpoint list,
* examples using curl,
* response examples,
* safety notes,
* public RPC compatibility,
* testnet-only warning,
* service points simulation-only warning,
* staking collateral-only warning,
* no wallet/admin/private key exposure.

Update:

* README.md
* README-ID.md
* docs/MultiHostTestnet.md
* docs/DeployTestnet.md

Mention:

* explorer API is read-only.
* explorer UI is not included yet.
* persistent indexer may come later.

Example curl:

curl http://127.0.0.1:9311/explorer/status
curl "http://127.0.0.1:9311/explorer/blocks?limit=10"
curl http://127.0.0.1:9311/explorer/blocks/1
curl http://127.0.0.1:9311/explorer/tx/<txid>
curl http://127.0.0.1:9311/explorer/address/<ADDRESS>
curl "http://127.0.0.1:9311/explorer/address/<ADDRESS>/txs?limit=10"

==================================================
15. Tests
=========

Run:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/rpc -run "Explorer|Public|ChainInfo|Health|Stake|Service|Faucet|Supply" -count=1 -v

go test ./node/internal/cli -run "Explorer|Public|Wallet|Stake|Service|Faucet" -count=1 -v

go test ./node/internal/chain -run "Block|Tx|Address|Supply|Snapshot" -count=1 -v

go test ./node/internal/servicenode -run "Score|Collateral|Points|Service" -count=1 -v

go test ./node/internal/staking -run "Stake|Status|Total|Unlock" -count=1 -v

Add tests for:

* explorer status at genesis.
* explorer blocks at genesis.
* explorer block detail for genesis and mined block.
* explorer tx detail for coinbase.
* explorer address summary for miner address after coinbase.
* explorer address tx history.
* explorer stake summary after stake lock.
* explorer service summary after service register/score.
* wrong network address rejected.
* invalid address rejected.
* public RPC explorer allowed.
* public wallet/admin still rejected.
* POST/write methods rejected.

==================================================
16. Manual validation
=====================

Use Host A Mini PC or Windows node after mining a few blocks.

Start node with public RPC and miner RPC:

./deskachain --datadir ./data/seed node start 
--rpc 0.0.0.0:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://100.101.251.7:10311 
--public-rpc 
--enable-miner-rpc

Mine a block if needed:

./dkcminer --rpc-url http://127.0.0.1:9311 --address <ADDR> --threads 2 --once

Validate:

curl http://127.0.0.1:9311/explorer/status

curl "http://127.0.0.1:9311/explorer/blocks?limit=5"

curl http://127.0.0.1:9311/explorer/blocks/1

curl http://127.0.0.1:9311/explorer/address/<ADDR>

curl "http://127.0.0.1:9311/explorer/address/<ADDR>/txs?limit=5"

If tx id is available:

curl http://127.0.0.1:9311/explorer/tx/<TXID>

Safety checks:

curl http://127.0.0.1:9311/health

deskachain --rpc-url http://127.0.0.1:9311 wallet new

Expected:

* explorer endpoints work.
* health works.
* wallet new rejected in public RPC mode.

If stake/service data exists from Phase 4.2:

* query address stake endpoint.
* query service endpoint.
* verify simulation-only flags or docs.

==================================================
17. Release/build validation
============================

If any code changed, validate release still builds:

powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.3-testnet -SkipTests

powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.4.3-testnet -SkipTests

Optional Bash:

bash ./scripts/build.sh v0.4.3-testnet --skip-tests
bash ./scripts/package.sh v0.4.3-testnet --skip-tests

Expected:

* Windows/Linux binaries build.
* archives and SHA256SUMS created.
* docs/Explorer.md included in release archive if docs are packaged.
* no runtime/private state in archives.

==================================================
18. Done criteria
=================

Phase 4.3 valid if:

* all tests pass.
* explorer read-only endpoints exist.
* explorer status works.
* block list works.
* block detail by height/hash works.
* tx detail works.
* address summary works.
* address tx history works.
* stake read-only summary works if staking data exists.
* service read-only summary works if service data exists.
* invalid/wrong-network addresses are rejected.
* explorer works in public RPC mode.
* wallet/admin endpoints remain disabled in public RPC.
* explorer endpoints do not mutate state.
* docs/Explorer.md exists.
* README/README-ID mention explorer API.
* manual curl validation passes.
* release build/package still works.
