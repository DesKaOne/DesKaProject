Kamu sedang bekerja pada project Go monorepo IndoChain.

Status saat ini:

* Phase 1 sampai Phase 4.9 sudah valid full.
* Phase 4.6 Public Testnet RC1 valid full.
* Phase 4.7 GitHub Release & External Tester Onboarding valid full.
* Phase 4.8 Post-Release Monitoring & Feedback Loop valid full.
* Phase 4.9 Multi-Seed & Peer Discovery Hardening valid full.
* Multi-seed bootstrap sudah valid:

  * VPS seed: http://100.86.152.39:10311
  * Mini PC seed: http://100.101.251.7:10311
* Peer diagnostics public read-only sudah valid.
* Health script `-CheckPeerList -MinPeers 1` sudah valid.
* Public RPC safety tetap aman:

  * wallet_rpc false
  * admin_rpc false
* Explorer API/UI read-only.
* Testnet dIDR has no monetary value.
* Mainnet does not exist yet.
* PoW remains the only block-production consensus.
* Staking remains collateral-only.
* Service points remain simulation-only.

Patch name:
IndoChain Phase 4.10 — Public Testnet Mining Stability & Difficulty Observation

Goal:
Improve and observe mining stability on public testnet without changing consensus rules.

This phase should make mining safer and easier to monitor across multiple nodes/miners:

* miner reconnect behavior,
* stale job detection,
* duplicate submit handling,
* submit timeout/retry,
* multi-miner observation,
* block interval metrics,
* difficulty retarget observation,
* fork/stale block diagnostics,
* operator mining runbook.

This phase is mining/runtime hardening and observability.
Do not change consensus, genesis, block format, reward rules, or monetary assumptions.

Non-goals:

* Do not launch mainnet.
* Do not change testnet genesis.
* Do not change localnet genesis.
* Do not change PoW consensus.
* Do not change block reward.
* Do not change difficulty formula unless fixing a clearly identified bug.
* Do not add mining pool/Stratum yet.
* Do not add GPU mining.
* Do not promise mining profit.
* Do not give testnet dIDR monetary value.
* Do not expose wallet/admin RPC publicly.
* Do not make staking produce blocks.
* Do not make service points spendable.

==================================================

1. Miner reconnect and retry hardening
   ==================================================

Harden `indominer` runtime behavior.

Requirements:

* Miner should survive temporary RPC failures.
* Miner should retry fetching job with bounded backoff.
* Miner should retry submit only when safe.
* Miner should not spin CPU aggressively when RPC is down.
* Miner should log clear status:

  * rpc unreachable
  * retrying
  * backoff duration
  * new job received
  * block found
  * submit accepted
  * submit rejected
  * stale job
  * duplicate/already known block
  * miner stopped cleanly
* Add flags/env if missing:

  * `--retry`
  * `--retry-delay`
  * `--max-retry-delay`
  * `--submit-timeout`
  * `--job-refresh-interval`
  * `--log-interval`
  * `--duration`
  * `--max-blocks`
* Keep defaults safe and simple.

Behavior:

* RPC down at startup should not crash immediately unless `--once` is used.
* In `--once` mode, return non-zero on unrecoverable RPC failure.
* In continuous mode, keep retrying with bounded backoff.
* On submit timeout, fetch latest chain info before deciding to continue.
* If local tip changed while mining, discard old job and fetch a new one.

==================================================
2. Stale job detection
======================

Add explicit stale job handling.

A job is stale when:

* node tip hash changed,
* job height is no longer next height,
* submit response says stale,
* submit response says parent mismatch,
* chain height already advanced.

Requirements:

* Miner should stop working on stale job quickly.
* Miner should log stale reason.
* Miner should count stale jobs.
* Miner stats should include:

  * jobs_received
  * jobs_stale
  * blocks_found
  * submits_accepted
  * submits_rejected
  * rpc_errors
  * reconnects
  * elapsed
  * approximate_hashrate if available.

Do not accept stale blocks into chain.

==================================================
3. Node submit diagnostics
==========================

Improve miner submit response and logs on node RPC.

When a miner submits a block, node should return clear JSON/text result:

* accepted true/false
* height
* hash
* reason if rejected
* current_height
* current_tip
* expected_parent if relevant
* submitted_parent if relevant
* duplicate true/false if already known.

Node logs should clearly show:

* submit received
* decoded height/tx count
* validating difficulty
* committed height/hash
* rejected reason
* broadcast summary

Do not leak private wallet data.

==================================================
4. Mining metrics endpoints
===========================

Add read-only mining/network observation endpoints if not already present.

Recommended endpoints:

* `GET /mining/status`
* `GET /mining/stats`
* `GET /mining/difficulty`
* `GET /mining/blocks`

Public mode:

* These endpoints are read-only and safe.
* They must not expose wallet/admin operations.
* They may show public chain/mining metrics only.

Metrics should include:

* network
* network_id
* chain_id
* height
* tip_hash
* current_difficulty
* next_difficulty
* target_block_time
* retarget_window
* blocks_until_retarget
* last_block_time
* recent block intervals
* average interval over last N blocks
* min/max interval over last N blocks
* recent difficulties
* total supply
* coinbase maturity
* pending tx count
* peer count if available.

No profitability or fiat value.

==================================================
5. Difficulty observation
=========================

Do not change the difficulty algorithm unless a test proves a bug.

Add observation helpers:

* CLI command:

  * `indochain mining status`
  * `indochain mining difficulty`
  * `indochain mining blocks --limit 30`
* RPC endpoint or CLI output should show:

  * target block time 30s
  * retarget window 30
  * current difficulty
  * next difficulty
  * blocks until retarget
  * average block interval over current window
  * projected retarget direction:

    * up
    * down
    * unchanged
  * note that this is informational only.

Add tests around existing difficulty behavior:

* fast blocks increase difficulty at retarget.
* slow blocks decrease or preserve difficulty within min/max.
* difficulty stays within min/max.
* cumulative work remains valid.
* chain validate passes after retarget.
* reorg respects cumulative work.

==================================================
6. Multi-miner testnet observation
==================================

Add docs and tests for multiple miners submitting to same node and different nodes.

Scenarios:

* one miner to VPS node.
* one miner to Mini PC node.
* two miners to same RPC.
* Windows miner to VPS RPC.
* Mini PC miner to local RPC.
* miner continues after peer broadcast failure.
* miner handles stale job when another miner wins first.
* miner handles node restart.

Automated tests:

* two simulated miners can race, only one accepted per height.
* stale submit rejected clearly.
* duplicate submit rejected/handled clearly.
* node remains valid after concurrent submit attempts.
* mempool tx is included once.
* no duplicate coinbase.
* chain validate passes.

Manual tests:

* Start VPS seed.
* Start Mini PC seed.
* Start Windows peer.
* Run miner A against VPS.
* Run miner B against Mini PC or Windows.
* Observe:

  * blocks advance.
  * peer sync catches up.
  * no chain validation failure.
  * stale blocks are rejected gracefully.
  * difficulty metrics update.

==================================================
7. Health script mining checks
==============================

Update:

* `scripts/testnet-health.ps1`
* `scripts/testnet-health.sh`

Add optional flags:

* `-CheckMining`
* `-ExpectedMinHeight`
* `-ExpectedMaxHeightLag`
* `-CheckDifficulty`
* `-WarnIfNoRecentBlockMinutes`
* `-FailIfNoRecentBlockMinutes`

PowerShell example:
powershell -ExecutionPolicy Bypass -File .\scripts\testnet-health.ps1 `    -RpcUrl http://100.86.152.39:9311`
-CheckPeerList `    -MinPeers 1`
-CheckMining `
-WarnIfNoRecentBlockMinutes 10

Output should include:

* current difficulty
* next difficulty
* target block time
* blocks until retarget
* last block age if available
* average interval if available
* mining status ok/warn/fail.

Zero recent blocks should be warning by default, not failure, unless explicit fail flag is used.

==================================================
8. Mining docs
==============

Add/update:

docs/Mining.md
docs/MiningStability.md
docs/DifficultyObservation.md
docs/TestnetTopology.md
docs/PostReleaseMonitoring.md
docs/OperatorChecklist.md
docs/PublicTestnetQuickstart.md
README.md
README-ID.md

Docs must explain:

* Testnet mining is for testing only.
* Testnet dIDR has no monetary value.
* No mining profit promise.
* Mainnet is not available.
* PoW is the only block-production consensus.
* Staking does not produce blocks.
* How to run miner.
* How to run miner continuously.
* How to stop miner safely.
* How to check mining status.
* How to read difficulty metrics.
* What stale job means.
* What submit rejected means.
* What coinbase maturity 100 means.
* How to avoid exposing wallet/admin RPC.
* Multi-node mining example:

  * VPS seed
  * Mini PC seed
  * Windows miner.

Example commands:

Windows:
.\indominer.exe --rpc-url http://127.0.0.1:9311 --address <IND_ADDRESS> --threads 2

Linux:
./indominer --rpc-url http://127.0.0.1:9311 --address <IND_ADDRESS> --threads 2

Once:
./indominer --rpc-url http://127.0.0.1:9311 --address <IND_ADDRESS> --threads 2 --once

Limited:
./indominer --rpc-url http://127.0.0.1:9311 --address <IND_ADDRESS> --threads 2 --max-blocks 5

==================================================
9. Explorer mining panel
========================

If easy, add a simple read-only Explorer UI section:

* Mining / Difficulty panel.

Show:

* current height
* current difficulty
* next difficulty
* blocks until retarget
* target block time
* recent average block interval
* last block age
* recent block list.

No charts required yet.
No wallet/admin actions.

If UI update is too large, add API first and document UI as next phase.

==================================================
10. Tests
=========

Run:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/cpuminer -run "Miner|Retry|Stale|Submit|Reconnect|Stats" -count=1 -v

go test ./node/internal/rpc -run "Mining|Miner|Submit|Difficulty|Explorer|Public|Health|Safety" -count=1 -v

go test ./node/internal/chain -run "Difficulty|Retarget|Cumulative|Validate|Work" -count=1 -v

go test ./node/internal/cli -run "Mining|Miner|Difficulty|Public|Health|Wallet|Admin" -count=1 -v

go test ./node/internal/p2p -run "Sync|Peer|Broadcast|Reorg|NetworkMismatch|GenesisMismatch" -count=1 -v

Add tests:

* miner retries when RPC initially down.
* miner reconnects after RPC comes back.
* stale job rejected clearly.
* duplicate submit handled clearly.
* concurrent submit accepts only one block for height.
* submit timeout does not hang miner forever.
* miner stats counters update.
* mining status endpoint public read-only allowed.
* mining endpoint does not enable wallet/admin.
* difficulty observation matches chain state.
* retarget boundaries min/max preserved.
* health script mining options documented.

==================================================
11. Manual validation
=====================

Use current public testnet machines:

Known nodes:

* VPS:

  * RPC: http://100.86.152.39:9311
  * P2P: http://100.86.152.39:10311
* Mini PC:

  * RPC: http://100.101.251.7:9311
  * P2P: http://100.101.251.7:10311
* Windows:

  * RPC local: http://127.0.0.1:9312
  * P2P: http://100.83.159.107:10312

A. Check mining status:
.\indochain.exe --rpc-url http://127.0.0.1:9312 mining status
.\indochain.exe --rpc-url http://127.0.0.1:9312 mining difficulty
.\indochain.exe --rpc-url http://127.0.0.1:9312 mining blocks --limit 10

B. Health mining check:
powershell -ExecutionPolicy Bypass -File .\scripts\testnet-health.ps1 `    -RpcUrl http://127.0.0.1:9312`
-CheckPeerList `    -MinPeers 1`
-CheckMining

C. Run one miner:
.\dist\windows-amd64\indominer.exe `    --rpc-url http://127.0.0.1:9312`
--address <IND_ADDRESS> `    --threads 2`
--max-blocks 3

D. Run second miner against VPS or Mini PC:
.\dist\windows-amd64\indominer.exe `    --rpc-url http://100.86.152.39:9311`
--address <IND_ADDRESS_2> `    --threads 2`
--max-blocks 3

E. Validate:
.\indochain.exe --rpc-url http://127.0.0.1:9312 chain info
.\indochain.exe --rpc-url http://127.0.0.1:9312 chain validate
.\indochain.exe --rpc-url http://127.0.0.1:9312 mining status
.\indochain.exe --rpc-url http://127.0.0.1:9312 peer health

F. Restart node during miner retry test:

* Stop node.
* Start miner continuous mode.
* Confirm miner retries.
* Start node.
* Confirm miner reconnects and mines/submits.

G. Stale job test:

* Start two miners.
* Confirm one accepted and stale/duplicate reject is graceful.
* Chain remains valid.

==================================================
12. Build/package validation
============================

Run:

powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.10-testnet-rc1 -SkipTests

powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.4.10-testnet-rc1 -SkipTests

Expected artifacts:

* indochain-v0.4.10-testnet-rc1-windows-amd64.zip
* indochain-v0.4.10-testnet-rc1-linux-amd64.tar.gz
* indochain-v0.4.10-testnet-rc1-linux-arm64.tar.gz
* SHA256SUMS.txt

Archive must include:

* docs/Mining.md
* docs/MiningStability.md
* docs/DifficultyObservation.md
* docs/TestnetTopology.md
* scripts/testnet-health.ps1
* scripts/testnet-health.sh
* README.md
* README-ID.md

No datadir, wallet, private key, runtime state, or secrets.

==================================================
13. Done criteria
=================

Phase 4.10 valid if:

* all tests pass.
* miner retry/reconnect behavior is stable.
* stale job detection works.
* duplicate/stale submit is handled clearly.
* miner/node submit logs are clear.
* mining status/difficulty observation works.
* public mining endpoints are read-only and safe.
* health script supports mining checks.
* multi-miner manual test passes.
* chain validate passes after mining.
* peer sync remains stable after mined blocks.
* difficulty observation matches chain state.
* docs clearly state no monetary value and no mining profit promise.
* build/package works.
