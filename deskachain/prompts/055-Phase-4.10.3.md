Patch DesKaChain Phase 4.10.3 — Upstream Seed Backfill & Isolated Mining Guard

Context:
DesKaChain public testnet Phase 4.10 is already valid full:

* Mining stability works.
* Reorg false-positive network mismatch fixed.
* Max reorg depth configurable.
* Deep reorg recovery works.
* Multi-node convergence works after manual sync.

Current topology issue:
VPS is intended to be the public head node / public seed / explorer node.
Mini PC and Windows are mining nodes.

Problem:
If VPS starts without `--seed-peer`, it has:

* peers=0
* seed_peers=0

Mini PC and Windows can mine ahead, but VPS explorer stays stale because VPS does not automatically pull from Mini PC/Windows.

Desired topology:

* VPS remains the head/public seed/explorer node.
* VPS does not need to know Mini PC/Windows as seed-peer.
* Mining nodes push/backfill blocks upstream to VPS.
* VPS stays updated even if it has no seed peers.
* VPS is not consensus authority; PoW cumulative work remains consensus source.

Add new feature:

* `--upstream-peer <url>`
* repeatable like `--seed-peer`
* optional config/env support if project has env config:

  * `DKC_UPSTREAM_PEERS=http://100.86.152.39:10311,http://...`

Definition:

* `--seed-peer` means this node can pull/sync/discover from that peer.
* `--upstream-peer` means this node should push/backfill its accepted blocks to that peer.

Use case:
Mini PC:
--seed-peer http://100.86.152.39:10311
--upstream-peer http://100.86.152.39:10311

Windows:
--seed-peer http://100.86.152.39:10311
--seed-peer http://100.101.251.7:10311
--upstream-peer http://100.86.152.39:10311

VPS:
no miner RPC required
no seed-peer required
receives block backfill from upstream push

Non-goals:

* Do not change testnet genesis.
* Do not change localnet genesis.
* Do not change PoW consensus.
* Do not make VPS a trusted authority.
* Do not bypass validation.
* Do not accept invalid blocks.
* Do not expose wallet/admin RPC publicly.
* Do not give testnet DKC monetary value.
* Do not add mainnet.

==================================================

1. Upstream peer config
   ==================================================

Add CLI flags:
--upstream-peer <url>
--upstream-peers <comma-separated-urls>
--upstream-file <path> optional if easy

Optional env:
DKC_UPSTREAM_PEERS=[http://100.86.152.39:10311,http://100.101.251.7:10311](http://100.86.152.39:10311,http://100.101.251.7:10311)

Normalize and dedupe URLs using existing peer URL normalization.

Startup logs must include:

* upstream_peers count
* upstream peer source
* normalized upstream peer URLs when verbose/startup summary prints peers

Example startup output:
upstream peers: 1
upstream peer source: flag

==================================================
2. Upstream backfill logic
==========================

When local node accepts a new block from:

* miner submit
* local mining
* p2p import
* reorg apply

Then schedule upstream sync/backfill to each upstream peer.

Do not block miner submit response too long.
Use bounded timeout and background goroutine/worker if needed.

For each upstream peer:

1. GET `/p2p/handshake` or `/p2p/status`
2. Validate:

   * network_id matches
   * chain_id matches
   * genesis_hash matches
   * protocol compatible
3. Compare upstream state with local state:

   * If upstream same height and same tip: do nothing.
   * If upstream lower and local branch extends it: push missing blocks.
   * If upstream lower but forked: use common ancestor and cumulative work policy.
   * If upstream higher: local node should pull/sync from upstream if upstream has more cumulative work.
   * If same work different tip: do not force; log fork tie.
4. Backfill:

   * If upstream height is 360 and local height is 393, push blocks 361..393.
   * Push in order.
   * Stop on first reject.
   * Log accepted/rejected count.

Use existing `/p2p/block` endpoint if it already validates and imports blocks.
If missing parent error occurs, retry by finding common ancestor and pushing from ancestor+1.

Important:
A single latest block is not enough if upstream is missing parents.
Must backfill all missing blocks.

==================================================
3. Upstream push/backfill status logs
=====================================

Add clear logs:

On success:
upstream backfill complete peer=http://100.86.152.39:10311 from_height=361 to_height=393 pushed=33 accepted=33

On already synced:
upstream already up to date peer=http://... height=393

On upstream ahead:
upstream ahead peer=http://... local_height=360 peer_height=393 decision=local_sync_from_upstream

On fork tie:
upstream fork tie peer=http://... local_work=... peer_work=... decision=no_force

On reject:
upstream backfill failed peer=http://... height=... reason="..."

Include:

* local_height
* local_tip
* peer_height
* peer_tip
* local_work
* peer_work
* common ancestor if relevant
* pushed count
* failed height if any

==================================================
4. Upstream CLI diagnostics
===========================

Add CLI commands if easy:

deskachain upstream list
deskachain upstream status
deskachain upstream push <peer-url>
deskachain upstream push-all

Alternative if CLI grouping is too much:

* extend `peer list` to show upstream source
* add `peer push <peer-url>` or `peer backfill <peer-url>`

Commands should work only on private/local RPC if they trigger writes.
Public RPC must reject manual push/backfill write actions.

Read-only upstream status may be allowed in public RPC.

==================================================
5. Isolated mining guard
========================

Add protection to prevent isolated forks.

New flags:
--min-mining-peers <int>
--allow-isolated-mining

Optional env:
DKC_MIN_MINING_PEERS=1
DKC_ALLOW_ISOLATED_MINING=false

Suggested defaults:

* localnet:
  min_mining_peers = 0
  allow_isolated_mining = true

* testnet:
  min_mining_peers = 1
  allow_isolated_mining = false

Behavior:
If `--enable-miner-rpc` is active and network is testnet:

* before returning miner template, check active peer count OR upstream peer availability.
* If active peers < min_mining_peers and no reachable upstream peer:
  reject miner template with clear error:
  miner RPC temporarily disabled: insufficient active peers/upstream unavailable; isolated mining is disabled
* miner submit should also reject if the block was produced while isolated policy fails, unless submit path can validate that upstream is reachable.

Rationale:

* A mining node must have at least one active peer or reachable upstream peer.
* This prevents mining on isolated node with stale chain.

Startup warning:
If miner_rpc=true and active peers are unknown at startup:
warning: miner RPC enabled; mining templates require at least 1 active peer or reachable upstream peer on testnet

Health/mining status should include:

* min_mining_peers
* active_peer_count
* upstream_peer_count
* upstream_reachable_count if available
* isolated_mining_allowed
* mining_template_enabled true/false
* mining_template_disabled_reason if false

==================================================
6. Faucet/write guard
=====================

Add similar guard for faucet and other chain-mutating public/testnet endpoints if applicable.

New flags:
--min-write-peers <int>
--allow-isolated-writes

Suggested defaults:

* localnet:
  min_write_peers = 0
  allow_isolated_writes = true

* testnet:
  min_write_peers = 1
  allow_isolated_writes = false

For `--enable-faucet-rpc`:

* faucet request should reject if isolated and no upstream reachable.
* Reason:
  faucet temporarily disabled: insufficient active peers/upstream unavailable; isolated writes are disabled

For admin/wallet write endpoints:

* Public RPC already disables these.
* Private RPC can warn, but testnet should preferably use the same guard unless explicitly overridden.

==================================================
7. Public RPC safety
====================

Preserve:

* public RPC wallet_rpc=false
* public RPC admin_rpc=false
* peer sync/write disabled in public mode
* manual upstream push/backfill write disabled in public mode
* read-only health/explorer/mining/peer status allowed

Do not make `--upstream-peer` expose any private action publicly.

==================================================
8. VPS head-node behavior
=========================

VPS can run:

./deskachain --datadir ./data/testnet --network testnet node start 
--rpc 0.0.0.0:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://100.86.152.39:10311 
--public-rpc 
--max-reorg-depth 128

Mini PC can run:

./deskachain --datadir ./data/testnet --network testnet node start 
--rpc 0.0.0.0:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://100.101.251.7:10311 
--public-rpc 
--enable-miner-rpc 
--seed-peer http://100.86.152.39:10311 
--upstream-peer http://100.86.152.39:10311 
--min-mining-peers 1 
--max-reorg-depth 128

Windows can run:

.\deskachain.exe --datadir .\data\testnet --network testnet node start `    --rpc 0.0.0.0:9312`
--p2p 0.0.0.0:10312 `    --advertise-p2p http://100.83.159.107:10312`
--public-rpc `    --enable-miner-rpc`
--seed-peer http://100.86.152.39:10311 `    --seed-peer http://100.101.251.7:10311`
--upstream-peer http://100.86.152.39:10311 `    --min-mining-peers 1`
--max-reorg-depth 128

Expected:

* VPS may have no seed peers.
* VPS explorer still updates because miners push/backfill blocks upstream.
* Mini PC/Windows mining is blocked if neither active peer nor upstream peer is reachable.

==================================================
9. Tests
========

Add tests:

Config/CLI:

* parses one `--upstream-peer`
* parses repeated `--upstream-peer`
* parses `--upstream-peers`
* normalizes and dedupes upstream peers
* startup summary includes upstream peer count
* parses `--min-mining-peers`
* parses `--allow-isolated-mining`

P2P/upstream:

* upstream lower same chain receives missing blocks via backfill
* upstream missing parent gets parent backfill before latest block
* upstream already up-to-date does nothing
* upstream higher causes local sync/pull when upstream has more cumulative work
* upstream wrong network rejected
* upstream wrong genesis rejected
* upstream fork same work does not force
* upstream fork higher local work does not downgrade local

Mining guard:

* testnet miner template rejected when active peers=0 and upstream unavailable
* testnet miner template allowed when active peers>=1
* testnet miner template allowed when active peers=0 but upstream reachable if policy permits upstream as connectivity
* localnet isolated mining still allowed by default
* `--allow-isolated-mining` allows testnet isolated mining only when explicitly set
* mining status includes guard fields

Public safety:

* public RPC manual upstream push/backfill rejected
* public RPC read-only upstream/mining status allowed
* wallet/admin remain disabled under public RPC

Regression:

* existing Phase 4.10 reorg tests still pass
* stale vs duplicate submit tests still pass
* chain validate passes after upstream backfill

Run:
go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:
go test ./node/internal/p2p -run "Upstream|Backfill|Push|Reorg|Fork|Sync|NetworkMismatch|GenesisMismatch" -count=1 -v

go test ./node/internal/rpc -run "Upstream|MiningGuard|Miner|Template|Public|Safety|Faucet|Write" -count=1 -v

go test ./node/internal/cli -run "Upstream|MinMiningPeers|Isolated|Seed|Peer|Public" -count=1 -v

go test ./node/internal/config -run "Upstream|MiningPeers|Isolated|Profile|Testnet" -count=1 -v

==================================================
10. Manual validation
=====================

Scenario A: VPS starts without seed peer.

1. Start VPS:
   ./deskachain --datadir ./data/testnet --network testnet node start 
   --rpc 0.0.0.0:9311 
   --p2p 0.0.0.0:10311 
   --advertise-p2p http://100.86.152.39:10311 
   --public-rpc 
   --max-reorg-depth 128

2. Start Mini PC with upstream VPS:
   ./deskachain --datadir ./data/testnet --network testnet node start 
   --rpc 0.0.0.0:9311 
   --p2p 0.0.0.0:10311 
   --advertise-p2p http://100.101.251.7:10311 
   --public-rpc 
   --enable-miner-rpc 
   --seed-peer http://100.86.152.39:10311 
   --upstream-peer http://100.86.152.39:10311 
   --min-mining-peers 1 
   --max-reorg-depth 128

3. Mine blocks on Mini PC.

4. Expected:

   * Mini PC height increases.
   * Mini PC pushes/backfills blocks to VPS.
   * VPS height catches up.
   * VPS explorer shows latest height.
   * chain validate passes on both.

Scenario B: VPS is behind by 30 blocks.

1. Let Mini PC height become 393.
2. VPS is height 360.
3. Start Mini PC with upstream VPS.
4. Expected:

   * Mini pushes blocks 361..393.
   * VPS height becomes 393.
   * No manual VPS seed-peer needed.

Scenario C: Upstream unavailable.

1. Stop VPS.
2. Start Mini PC with upstream VPS and miner RPC.
3. Request miner template.
4. Expected:

   * miner template rejected or warning depending policy.
   * no isolated mining by default on testnet.

Scenario D: Localnet compatibility.

1. Start localnet isolated.
2. Miner template still works by default.
3. Existing localnet tests pass.

==================================================
11. Docs
========

Update:

* docs/Mining.md
* docs/MiningStability.md
* docs/TestnetTopology.md
* docs/OperatorChecklist.md
* docs/PublicTestnetQuickstart.md
* README.md
* README-ID.md

Explain:

* VPS can be head/public seed/explorer.
* Mining nodes should use `--upstream-peer` to push blocks to VPS.
* `--seed-peer` is for pulling/discovery.
* `--upstream-peer` is for pushing/backfill.
* Mining nodes must not mine isolated on testnet.
* Testnet DKC has no monetary value.
* Mainnet is not available.
* PoW remains the only block production consensus.

==================================================
12. Build/package
=================

Run:
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.10-testnet-rc2 -SkipTests

powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.4.10-testnet-rc2 -SkipTests

Expected artifacts:

* deskachain-v0.4.10-testnet-rc2-windows-amd64.zip
* deskachain-v0.4.10-testnet-rc2-linux-amd64.tar.gz
* deskachain-v0.4.10-testnet-rc2-linux-arm64.tar.gz
* SHA256SUMS.txt

No wallet/private key/datadir/runtime state in packages.

Done criteria:

* VPS can start without seed-peer.
* Mini PC/Windows can upstream-push/backfill blocks to VPS.
* VPS explorer catches up after being behind.
* testnet isolated mining is blocked by default.
* localnet isolated mining still works.
* public RPC safety preserved.
* all tests pass.
* build/package pass.
