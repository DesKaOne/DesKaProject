Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status:

* Phase 3.6 automated tests pass.
* Seed peer config, parsing, normalization, dedupe, peer store source metadata, and public RPC safety tests pass.
* Manual two-node seed flow revealed one blocker.

Observed manual bug:
Node A testnet mined block height 1.
Node B testnet received/imported the block through seed/P2P path.

On Node B:

chain info:
height: 0
tip hash: genesis
total supply: 50 DKC
blocks: 2
coinbase blocks: 1

chain validate:
chain valid
height: 1
blocks: 2
total supply: 50 DKC

This is inconsistent.

Expected:
If blocks=2 and total supply=50 DKC after importing height 1, then chain info must report:
height: 1
tip hash: imported block hash
tip difficulty: 4
total supply: 50 DKC
blocks: 2
chain validate agrees with chain info

Patch name:
DesKaChain Phase 3.6.1 — Chain Info Snapshot Consistency After P2P Import

Goal:
Ensure `/chain/info` and CLI `chain info` always report a consistent canonical chain snapshot after:

* local mining,
* miner submit,
* P2P block import,
* peer sync import,
* seed peer/background import,
* restart.

Non-goals:

* Do not change testnet genesis.
* Do not change localnet genesis.
* Do not change PoW validation.
* Do not bypass P2P validation.
* Do not expose peer sync in public RPC if it is intentionally disabled.
* Do not weaken public RPC safety.
* Do not change seed peer behavior except where needed to update runtime chain snapshot.

==================================================

1. Diagnose source of inconsistency
   ==================================================

Audit chain info implementation.

Find where chain info gets:

* height
* tip hash
* difficulty
* tip difficulty
* next difficulty
* cumulative work
* supply stats
* block/tx counts

Likely bug:

* height/tip are read from cached runtime head.
* supply stats/blocks are read from storage/ledger.
* P2P import updates storage but does not refresh cached runtime head.

Fix must ensure chain info uses one consistent canonical source.

Preferred fix:

* Add a single canonical snapshot function, for example:
  ChainSnapshot / RuntimeChainInfo / LoadCanonicalTipAndStats
* It should load tip/head and stats from the same chain/store state.
* `/chain/info` should call this function instead of mixing stale cached fields with fresh stats.

Alternative acceptable fix:

* Ensure every successful block import path updates runtime head/tip cache atomically.
* Also ensure chain info falls back to storage canonical tip if cache is missing/stale.

==================================================
2. Update all block commit/import paths
=======================================

Ensure these paths update the same canonical runtime state:

* local `/mine`
* `/miner/submit`
* P2P `/p2p/block`
* peer sync import loop
* reorg apply
* startup chain load
* restart from datadir

After successful import/commit:

* canonical height updated.
* canonical tip hash updated.
* cumulative work updated.
* difficulty fields consistent.
* chain info immediately reflects imported block.

==================================================
3. Tests
========

Add focused regression tests.

Required tests:

1. TestChainInfoAfterP2PBlockImportReflectsCanonicalTip
   Flow:

* start node A and node B testnet.
* mine or construct valid block height 1 on A.
* import/broadcast block to B through the same P2P path used in runtime.
* call B chain info.
* assert:

  * height == 1
  * tip hash == imported block hash
  * blocks == 2
  * total supply == 50 DKC
  * coinbase blocks == 1
* call B chain validate.
* assert validate height == chain info height.

2. TestChainInfoAfterPeerSyncReflectsCanonicalTip
   Flow:

* A has height > 0.
* B syncs/imports from A through peer sync/import path.
* B chain info must match B chain validate.

3. TestChainInfoAfterRestartReflectsCanonicalTip
   Flow:

* import/mine block.
* stop/reload runtime from same datadir.
* chain info still height/tip correct.

4. TestChainInfoSnapshotConsistentFields
   Flow:

* create chain with height 1 or more.
* call chain info.
* assert no combination like:

  * height 0 with blocks > 1
  * genesis tip with total supply > 0
  * coinbase blocks > 0 while height 0
* unless genesis-only chain.

5. TestPublicRPCChainInfoAfterP2PImport

* same as above but with public RPC mode.
* read-only chain info must be accurate.

==================================================
4. Manual validation after patch
================================

Run:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/rpc -run "ChainInfo|P2P|PeerSync|Public" -count=1 -v
go test ./node/internal/p2p -run "Sync|Import|ChainInfo|Peer|Seed" -count=1 -v
go test ./node/internal/cli -run "ChainInfo|Peer|Seed" -count=1 -v
go test ./node/internal/chain -run "Tip|Snapshot|Validate|Supply" -count=1 -v

Manual reproduction:

Clean:
go run ./node/cmd/deskachain --datadir ./testdata/seed_a dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/seed_b dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/seed_miner dev reset --yes

Init:
go run ./node/cmd/deskachain --datadir ./testdata/seed_a --network testnet init
go run ./node/cmd/deskachain --datadir ./testdata/seed_b --network testnet init
go run ./node/cmd/deskachain --datadir ./testdata/seed_miner --network testnet init

Start A:
go run ./node/cmd/deskachain --datadir ./testdata/seed_a node start --rpc :9111 --p2p :10111 --advertise-p2p http://127.0.0.1:10111 --public-rpc --enable-miner-rpc

Start B with seed peer:
go run ./node/cmd/deskachain --datadir ./testdata/seed_b node start --rpc :9112 --p2p :10112 --advertise-p2p http://127.0.0.1:10112 --public-rpc --enable-miner-rpc --seed-peer http://127.0.0.1:10111/

Create miner wallet:
go run ./node/cmd/deskachain --datadir ./testdata/seed_miner wallet new

Mine on A:
go run ./node/cmd/dkcminer --rpc-url http://127.0.0.1:9111 --address <MINER_ADDR> --threads 2 --once

Check B:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:9112 chain info
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:9112 chain validate

Expected B:
height: 1
tip hash: mined block hash
total supply: 50 DKC
blocks: 2
coinbase blocks: 1
chain valid height: 1

Important:
`peer sync` may remain disabled in public RPC mode. That is okay.
This bug is not about public peer sync. It is about chain info stale after block was already imported/accepted.

==================================================
5. Done criteria
================

Phase 3.6.1 valid if:

* chain info and chain validate agree after P2P import.
* no height 0 + blocks 2 + supply 50 inconsistency.
* public RPC chain info remains read-only and accurate.
* all tests pass.
* manual two-node seed flow reports B height 1 after A mines block.
