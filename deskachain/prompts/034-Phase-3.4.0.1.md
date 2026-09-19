DesKaChain Phase 3.4.0.1 — Testnet Wallet Address Profile & Miner Submit Timeout Fix

Context:
Phase 3.4 faucet sudah dipatch, tetapi validasi belum lulus.

Observed failures:

1. `go test -count=1 ./node/...` gagal di package `deskachain/internal/rpc`.
   Failing test:

   * TestMinerSubmitInvalidPoWStaleAndWrongDifficulty
     Error:
   * context deadline exceeded while POST /miner/submit

2. Manual testnet faucet flow:
   Datadir testnet dibuat dengan:
   deskachain --datadir ./testdata/faucet_tn --network testnet init

   Wallet dibuat dengan:
   deskachain --datadir ./testdata/faucet_tn wallet new

   Address yang keluar:
   DKC...

   Tetapi saat digunakan pada node testnet, balance/replay error:
   invalid coinbase recipient: invalid address: wrong network version

3. Faucet info menerima faucet address, miner bisa mulai mining, tetapi balance command menolak address wrong network version. Ini menunjukkan validation tidak konsisten antar:

   * wallet new,
   * balance,
   * miner template,
   * miner submit,
   * coinbase validation,
   * faucet address validation,
   * remote CLI.

Goal:
Menutup blocker Phase 3.4 agar testnet wallet/faucet/miner memakai active network profile secara konsisten, dan test RPC miner submit tidak timeout.

Non-goals:

* Jangan implement public faucet.
* Jangan ubah address prefix DKC.
* Jangan ubah localnet genesis.
* Jangan ubah testnet genesis kecuali benar-benar wajib.
* Jangan disable network-version validation.
* Jangan bypass tx/block validation.
* Jangan membuat faucet mint langsung.
* Jangan ubah PoW consensus.
* Jangan ubah staking behavior.

==================================================

1. Fix wallet new agar pakai datadir network metadata
   ==================================================

Problem:
`deskachain --datadir <testnet_dir> wallet new` kemungkinan masih memakai default localnet profile jika flag `--network testnet` tidak diberikan.

Expected:
Jika datadir sudah init testnet, semua command local datadir harus resolve profile dari datadir metadata.

Fix:

* Audit CLI wallet command.
* Wallet generation must use active profile resolved from:

  1. explicit --network flag if provided and matches datadir,
  2. datadir network metadata if datadir exists,
  3. default localnet only for brand new/no metadata backward-compatible case.

Commands:
deskachain --datadir ./testdata/faucet_tn --network testnet init
deskachain --datadir ./testdata/faucet_tn wallet new

Expected:

* wallet new creates testnet-valid address.
* balance on testnet node accepts it.
* miner template on testnet accepts it.
* faucet address validation accepts it.

Add tests:

1. TestWalletNewUsesDatadirTestnetProfile

* init datadir as testnet.
* run wallet new without --network.
* address validates with config.Testnet().
* address does not validate with wrong network if version differs.

2. TestWalletNewNetworkMismatchRejected

* init datadir testnet.
* run wallet new with --network localnet.
* expect clear mismatch error.

3. TestWalletNewLocalnetBackwardCompatible

* localnet still works.

==================================================
2. Fix remote balance address validation profile
================================================

Problem:
`balance <DKC_ADDR>` on testnet returns:
invalid coinbase recipient: invalid address: wrong network version

Need determine if:
A. address is actually localnet because wallet new bug, or
B. balance/replay uses wrong profile.

Fix both sides:

* RPC balance/replay must use active node profile.
* CLI remote balance should not validate address using local default before sending unless it fetches remote profile first.
* If CLI validates locally, it must use:

  * datadir profile for local mode,
  * remote `/health` or `/chain/info` profile for remote mode,
  * or avoid pre-validation and let RPC validate.

Add tests:

1. TestRemoteBalanceUsesRemoteTestnetProfile

* start testnet RPC.
* testnet address accepted.
* localnet-version address rejected clearly.

2. TestBalanceRejectsWrongNetworkAddress

* localnet node rejects testnet-version address if versions differ.
* testnet node rejects localnet-version address if versions differ.

==================================================
3. Fix miner template / submit coinbase address validation
==========================================================

Problem:
Dkcminer was able to mine/submit using address later rejected by balance as wrong network version.

Expected:

* `/miner/template?address=<addr>` must reject wrong-network reward address immediately.
* `/miner/submit` must reject block with wrong-network coinbase recipient.
* Chain should never commit a block whose coinbase recipient is invalid for active profile.

Fix:

* Audit miner template handler.
* Audit miner submit handler.
* Audit chain validation for coinbase recipient.
* Ensure active profile is passed into validation.
* Do not use config.Localnet() in this path.
* Do not use default address validation without profile.

Add tests:

1. TestMinerTemplateRejectsWrongNetworkRewardAddress

* testnet node + localnet-version address.
* expect wrong network version.

2. TestMinerSubmitRejectsWrongNetworkCoinbaseRecipient

* construct block with coinbase to wrong-network address.
* submit rejected.
* chain height unchanged.

3. TestMinerSubmitAcceptsTestnetRewardAddress

* testnet address accepted.
* submit accepted.

4. TestCommittedBlocksAlwaysValidateCoinbaseRecipientWithProfile

* chain validate catches wrong-network coinbase if constructed manually.

==================================================
4. Fix faucet address validation consistency
============================================

Faucet must validate:

* faucet address with active profile.
* recipient address with active profile.
* tx signing source address with active profile.

Expected:

* enabling faucet with wrong-network faucet address fails at node start or faucet info/request.
* faucet request with wrong-network recipient fails.
* faucet request with correct testnet recipient proceeds to balance check.

Add tests:

1. TestFaucetRejectsWrongNetworkFaucetAddress
2. TestFaucetRejectsWrongNetworkRecipient
3. TestFaucetAcceptsTestnetFaucetAndRecipient
4. TestFaucetRequestInsufficientMatureBalanceAfterValidAddress

* Important: with valid addresses but no mature balance, error should be insufficient mature balance, not wrong network version.

==================================================
5. Fix RPC miner submit timeout test
====================================

Failing test:
TestMinerSubmitInvalidPoWStaleAndWrongDifficulty
Error:
Post /miner/submit: context deadline exceeded

Observed log:

* invalid PoW rejected.
* wrong difficulty rejected.
* then test timed out waiting for stale submit or another invalid case.

Audit test and handler:

* Handler must always respond on invalid/stale/wrong difficulty paths.
* No lock should remain held after rejection.
* No deadlock on template lookup/mempool/chain validation.
* HTTP response should be quick and deterministic.

Likely areas:

* miner submit handler locks chain/template/mempool.
* stale template path may wait on lock or validation.
* wrong difficulty path might not write response in some branch.
* test timeout too aggressive only if handler now does expensive PoW/difficulty validation.

Fix:

* Ensure all rejection branches return JSON/error response.
* Ensure defer unlocks are correct.
* Avoid long validation when request is already stale by template id/height.
* Increase test timeout only if necessary, but prefer fixing handler.
* Keep previous Phase 2.9.1 no-deadlock guarantees.

Add tests:

1. TestMinerSubmitInvalidPoWReturnsQuickly
2. TestMinerSubmitStaleTemplateReturnsQuickly
3. TestMinerSubmitWrongDifficultyReturnsQuickly
4. TestMinerSubmitInvalidCasesDoNotDeadlockNextTemplate

* after invalid submits, `/miner/template` still responds.

Target:
go test ./node/internal/rpc -run TestMinerSubmitInvalidPoWStaleAndWrongDifficulty -count=1 -v

must pass.

==================================================
6. Add faucet package tests
===========================

Currently:
deskachain/internal/faucet [no test files]

Add tests for faucet state if feasible:

1. TestFaucetStateRateLimit
2. TestFaucetStatePendingDuplicate
3. TestFaucetStatePersists
4. TestFaucetStateCorruptHandled
5. TestFaucetStateAddressTotals

If state logic is tested through RPC already, package tests are still recommended because faucet is new critical state.

==================================================
7. Manual cleanup after patch
=============================

Because current `./testdata/faucet_tn` may contain blocks mined to wrong-network coinbase address, reset it after patch.

Manual commands after patch:
go run ./node/cmd/deskachain --datadir ./testdata/faucet_tn dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/faucet_tn --network testnet init
go run ./node/cmd/deskachain --datadir ./testdata/faucet_tn wallet new
go run ./node/cmd/deskachain --datadir ./testdata/faucet_tn wallet new

Start:
go run ./node/cmd/deskachain --datadir ./testdata/faucet_tn node start --rpc :8811 --p2p :9811 --advertise-p2p http://127.0.0.1:9811 --enable-faucet-rpc --faucet-address <FAUCET_ADDR> --faucet-amount 100 --faucet-min-interval 1m

Check immediately:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8811 faucet info
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8811 balance <FAUCET_ADDR>
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8811 balance <RECIPIENT_ADDR>

Expected before mining:

* no wrong network version error.
* balance valid with 0 DKC.

Mine enough:
go run ./node/cmd/dkcminer --rpc-url http://127.0.0.1:8811 --address <FAUCET_ADDR> --threads 4 --max-blocks 105

Check:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8811 balance <FAUCET_ADDR>

Expected:

* mature balance > 0.
* no wrong network version error.

Request:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8811 faucet request --address <RECIPIENT_ADDR>

Expected:

* faucet tx created.
* status pending.

Mine confirmation:
go run ./node/cmd/dkcminer --rpc-url http://127.0.0.1:8811 --address <FAUCET_ADDR> --threads 4 --once

Check recipient:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8811 balance <RECIPIENT_ADDR>

Expected:

* confirmed balance 100 DKC.
* no wrong network version error.

==================================================
8. Required test commands
=========================

After patch:
go work sync
go test ./node/...
go test -count=1 ./node/...

Target:
go test ./node/internal/rpc -run "Faucet|MinerSubmit|Balance" -count=1 -v
go test ./node/internal/cli -run "Faucet|Wallet|Balance" -count=1 -v
go test ./node/internal/wallet -run "Network|Address|Testnet" -count=1 -v
go test ./node/internal/faucet -v

Expected:

* no FAIL.
* no timeout.
* no wrong network version for testnet-generated address.
