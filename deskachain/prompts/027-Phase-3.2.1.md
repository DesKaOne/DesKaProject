Kamu sedang bekerja pada project Go monorepo DesKaChain.

Struktur project:

* node/

  * cmd/deskachain/
  * cmd/dkcminer/
  * cmd/dkcservice/
  * internal/

    * staking/
    * chain/
    * ledger/
    * mempool/
    * rpc/
    * servicenode/
    * serviceagent/
  * go.mod
  * go.sum
* root repo punya:

  * go.work
  * README.md
  * Roadmap.md
  * docs/
  * testdata/
  * data/

Status saat ini:

* Phase 1 sampai Phase 3.2 sudah selesai dan valid.
* go work sync berhasil.
* go test ./node/... pass.
* Address final sudah aktif:

  * wallet baru menghasilkan address `DKC...`
  * format address: `DKC` + Base58Check
  * private key raw 32-byte hex
* Dynamic difficulty sudah aktif.
* Coinbase maturity sudah aktif.
* Standalone CPU miner sudah aktif.
* Node public hardening sudah aktif.
* Service node simulation sudah aktif.
* dkcservice agent sudah aktif.
* Staking collateral sudah aktif:

  * stake lock
  * stake unlock
  * unbonding period
  * released stake balik spendable
  * active stake mengurangi spendable
  * pending outgoing mengurangi spendable
  * service collateral eligibility
  * public RPC stake lock disabled
  * chain validate tetap pass
* Catatan penting:

  * internal/staking saat ini masih `[no test files]`.
  * Phase ini harus menambah regression tests agar staking tidak gampang rusak di patch berikutnya.

Nama patch:
DesKaChain Phase 3.2.1 — Staking Regression Tests & Consensus Safety

Tujuan:
Menambahkan test coverage permanen untuk staking/collateral agar aman sebelum lanjut fitur baru.

Phase ini fokus pada:

* unit tests internal/staking,
* integration tests staking + balance,
* staking + mempool pending rules,
* staking + chain validate,
* staking + service eligibility,
* staking + public RPC safety,
* no consensus mutation,
* no PoS,
* no staking reward DKC.

Prinsip penting:

* Jangan ubah consensus behavior yang sudah valid.
* Jangan ubah address format DKC.
* Jangan ubah private key format.
* Jangan ubah difficulty adjustment.
* Jangan ubah coinbase maturity.
* Jangan ubah PoW consensus.
* Jangan implement PoS.
* Jangan implement validator.
* Jangan implement slashing.
* Jangan implement delegation.
* Jangan implement staking reward DKC.
* Jangan rewrite besar.
* Patch ini mayoritas test dan safety fix kecil saja.
* Semua command lama harus tetap bekerja.
* Semua test harus pass:

  go work sync
  go test ./node/...
  go test -count=1 ./node/...

==================================================

1. Tambahkan test untuk internal/staking
   ==================================================

Saat ini internal/staking masih belum punya test file.

Tambahkan test file:
node/internal/staking/staking_test.go

Minimal test coverage:

1. TestLocalnetStakingParams

* staking enabled true.
* min stake amount = 10 DKC.
* min service stake = 100 DKC.
* unbonding period = 10 blocks.
* max active stakes per address sesuai config.

2. TestStakeRecordStatusLifecycle

* create stake active.
* unlock at height H.
* release height = H + unbonding.
* before release => unlocking.
* at release height => released.

3. TestLockedAmountByAddress

* active stake dihitung sebagai locked.
* unlocking stake dihitung sebagai locked.
* released stake tidak dihitung sebagai locked.

4. TestTotalActiveAndUnlockingStake

* total active stake benar.
* total unlocking stake benar.
* released tidak dihitung active/unlocking.

5. TestStakeBelowMinimumRejected

* amount < min stake rejected.

6. TestDuplicateStakeIDRejected

* stake id sama tidak boleh double active.

7. TestUnlockNonExistingStakeRejected

* unlock stake id yang tidak ada harus error.

8. TestUnlockOwnerMismatchRejected

* owner B tidak bisa unlock stake owner A.

9. TestUnlockAlreadyUnlockingRejected

* stake yang sudah unlocking tidak bisa unlock lagi.

10. TestReleasedStakeNotLocked

* released stake tidak mengurangi spendable lagi.

==================================================
2. Balance + staking regression tests
=====================================

Tambahkan test di package yang paling tepat:

* internal/ledger
  atau
* internal/chain
  atau
* internal/rpc integration test

Tujuan:
Memastikan staking memengaruhi spendable dengan benar.

Tests:

1. TestStakeLockImmatureCoinbaseRejected

* mine kurang dari maturity.
* confirmed ada, mature 0.
* stake lock ditolak.
* error mengandung insufficient mature/spendable.

2. TestStakeLockAfterMaturityAccepted

* mine sampai height minimal 11.
* mature 50 DKC.
* stake lock 10 DKC accepted.
* after mined, active stake 10 DKC.

3. TestActiveStakeReducesSpendable

* mature 100.
* active stake 40.
* spendable 60.

4. TestUnlockingStakeReducesSpendable

* active stake 40.
* unlock mined.
* status unlocking.
* spendable tetap berkurang 40 sampai release height.

5. TestReleasedStakeRestoresSpendable

* unlock at height H.
* mine until release height.
* released stake tidak mengurangi spendable.
* spendable naik sesuai amount.

6. TestPendingStakeLockReducesSpendable

* mature 100.
* pending stake lock 70.
* spendable efektif 30.
* send 40 harus gagal.
* send 30 harus sukses atau minimal valid.

7. TestPendingOutgoingAndStakeLockBothReduceSpendable

* mature 100.
* active stake 20.
* pending outgoing 30.
* spendable 50.
* tx baru 51 harus gagal.
* tx baru 50 harus sukses jika nonce/pending logic mendukung.

8. TestSendCannotSpendActiveStake

* mature 100.
* active stake 90.
* spendable 10.
* send 11 rejected.

9. TestSendAtExactSpendableSucceeds

* mature 105.
* unlocking stake 10.
* spendable 95.
* send 95 succeeds.
* ini penting karena manual test membuktikan exact limit valid.

==================================================
3. Mempool staking regression tests
===================================

Tambahkan atau update tests di:
internal/mempool
atau integration test dengan RPC.

Tests:

1. TestMempoolAcceptsValidStakeLock

* valid mature spendable.
* stake lock added.
* pending count increment.

2. TestMempoolRejectsStakeLockOverSpendable

* stake amount > spendable.
* reject.

3. TestMempoolRejectsStakeLockBelowMinimum

* amount below min.
* reject.

4. TestMempoolPendingStakeLockPreventsOverspendTransfer

* pending stake lock 80.
* mature 100.
* transfer 30 rejected karena remaining 20.

5. TestMempoolRejectsDuplicateStakeUnlock

* pending unlock for stake exists.
* duplicate unlock rejected.

6. TestMempoolUnlockDoesNotReleaseImmediately

* pending unlock tx exists.
* stake remains locked until mined and unbonding done.

==================================================
4. Chain validation staking tests
=================================

Tambahkan/update tests di internal/chain.

Tests:

1. TestChainValidateStakeLockBlock

* block berisi coinbase + stake_lock valid.
* chain validate pass.
* stake active after block.

2. TestChainValidateRejectStakeLockOverSpendable

* buat block invalid berisi stake_lock lebih dari mature spendable.
* chain validate fails.

3. TestChainValidateRejectStakeLockImmature

* stake lock memakai immature coinbase.
* chain validate fails.

4. TestChainValidateRejectStakeUnlockMissingStake

* block dengan stake_unlock stake id tidak ada.
* chain validate fails.

5. TestChainValidateRejectStakeUnlockOwnerMismatch

* owner mismatch.
* chain validate fails.

6. TestChainValidateRejectTransferSpendingLockedStake

* active stake mengurangi spendable.
* transfer yang menghabiskan locked funds harus invalid.
* chain validate fails.

7. TestChainValidateStakeUnlockAndRelease

* stake lock mined.
* unlock mined.
* before release height locked.
* at release height released.
* chain validate pass.

8. TestStakeDoesNotChangeTotalSupply

* before stake total supply X.
* after lock/unlock/release total supply still based on coinbase only.

==================================================
5. RPC staking tests
====================

Tambahkan/update tests di internal/rpc.

Tests:

1. TestStakeInfoRPC

* GET /stake/info returns staking enabled, min amount, min service stake, unbonding period.

2. TestStakeLockRPCAdminMode

* local/admin mode.
* POST /stake/lock valid.
* returns tx id and stake id.

3. TestStakeUnlockRPCAdminMode

* active stake exists.
* POST /stake/unlock returns tx id and release height.

4. TestStakeListRPC

* list all.
* list by address.

5. TestStakeStatusRPC

* returns stake by id.

6. TestPublicRPCStakeInfoAllowed

* public-rpc true.
* /stake/info works.

7. TestPublicRPCStakeLockDisabled

* public-rpc true.
* /stake/lock returns endpoint disabled in public RPC mode.

8. TestPublicRPCStakeUnlockDisabled

* public-rpc true.
* /stake/unlock returns endpoint disabled in public RPC mode.

9. TestStakeRPCBodyLimit

* oversized request rejected 413 if body limit middleware available.

==================================================
6. Service collateral regression tests
======================================

Tambahkan/update tests di internal/servicenode atau integration test.

Tests:

1. TestServiceCollateralBelowRequired

* active stake 0.
* required stake 100.
* service score shows:

  * stake_eligible false
  * collateral_status insufficient or none
  * eligible points 0 or ineligible note.

2. TestServiceCollateralEligible

* active stake 100.
* service score shows:

  * required stake 100 DKC
  * active stake 100 DKC
  * stake_eligible true
  * collateral_status eligible
  * eligible simulated points equals simulated points.

3. TestServiceCollateralUnlockingNotEligible

* active stake 100.
* unlock mined.
* stake status unlocking.
* service score should show:

  * stake_eligible false
  * collateral_status unlocking
  * eligible points 0 or ineligible.

4. TestServiceCollateralReleasedNotEligible

* released stake does not count as active collateral.
* stake_eligible false.

5. TestServicePointsStillNotDKC

* service score/rewards do not change DKC balance.
* total supply unchanged.

==================================================
7. Reorg staking safety tests
=============================

Jika project sudah punya reorg test helper, tambahkan minimal:

1. TestReorgRemovesStakeLockFromOrphanedBranch

* branch A includes stake lock.
* branch B becomes canonical without stake lock.
* active stake from branch A removed after reorg.
* balance spendable recalculated.

2. TestReorgRemovesStakeUnlockFromOrphanedBranch

* branch A includes stake unlock.
* branch B becomes canonical without unlock.
* stake returns active if lock still canonical and unlock orphaned.

3. TestReorgRequeuesValidStakeTx

* orphaned valid stake lock can re-enter mempool if still valid.

4. TestReorgDropsInvalidStakeTx

* orphaned stake lock dropped if no longer spendable.
* orphaned unlock dropped if stake not active on new branch.

Jika reorg helper belum tersedia atau terlalu besar:

* tambahkan TODO jelas di docs/tests.
* minimal jangan skip core staking tests.
* Tapi usahakan ada setidaknya 1 reorg staking regression jika infrastruktur sudah ada.

==================================================
8. CLI regression tests
=======================

Jika internal/cli punya test command harness, tambahkan:

1. stake info output contains:

* staking enabled
* min stake amount
* min service stake
* unbonding period

2. stake lock output contains:

* stake lock tx created
* tx id
* stake id
* status pending

3. stake list output contains:

* stake id
* owner
* amount
* status
* lock height

4. stake unlock output contains:

* stake unlock tx created
* release height
* status pending

5. balance output contains:

* active stake
* unlocking stake
* released stake
* pending stake lock
* spendable balance

==================================================
9. Bug fix rules
================

Jika tests menemukan bug kecil, boleh patch dengan syarat:

* tidak rewrite besar,
* tidak mengubah public API tanpa perlu,
* tidak mengubah format address/key,
* tidak mengubah PoW consensus,
* tidak menambah staking reward,
* tidak menambah slashing.

Bug kecil yang boleh diperbaiki:

* missing spendable check.
* missing pending stake lock calculation.
* wrong released stake calculation.
* wrong service eligibility if unlocking.
* wrong public RPC guard.
* wrong error message.
* wrong chain validation for stake tx.

==================================================
10. Docs update
===============

Update:

* docs/Staking.md
* docs/ServiceNode.md
* Roadmap.md

Tambahkan note:

* Phase 3.2.1 adds staking regression tests and consensus safety checks.
* Staking remains collateral only.
* No PoS.
* No validator set.
* No staking APY.
* No DKC staking reward.
* No slashing in this phase.

==================================================
11. Expected final commands
===========================

After patch:

go work sync
go mod tidy
go test ./node/...
go test -count=1 ./node/...

Package-specific:

go test ./node/internal/staking -v
go test ./node/internal/chain -run Stake -v
go test ./node/internal/rpc -run Stake -v
go test ./node/internal/servicenode -run Collateral -v
go test ./node/internal/mempool -run Stake -v

Expected:

* internal/staking no longer says `[no test files]`.
* all staking tests pass.
* all old tests pass.

Manual quick regression:

go run ./node/cmd/deskachain --datadir ./testdata/stake_reg dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/stake_reg init
go run ./node/cmd/deskachain --datadir ./testdata/stake_reg wallet new

Start node:

go run ./node/cmd/deskachain --datadir ./testdata/stake_reg node start --rpc :8491 --p2p :9491 --advertise-p2p http://127.0.0.1:9491

Mine:

go run ./node/cmd/dkcminer --rpc-url http://127.0.0.1:8491 --address <DKC_ADDR> --threads 4 --max-blocks 12

Stake:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8491 stake lock --address <DKC_ADDR> --amount 100
go run ./node/cmd/dkcminer --rpc-url http://127.0.0.1:8491 --address <DKC_ADDR> --threads 4 --once

Check:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8491 balance <DKC_ADDR>
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8491 stake list --address <DKC_ADDR>
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8491 chain validate

Expected:

* active stake: 100
* spendable reduced
* chain valid

Service eligibility quick:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8491 service register --address <DKC_ADDR> --endpoint http://127.0.0.1:9501
go run ./node/cmd/dkcservice --rpc-url http://127.0.0.1:8491 --address <DKC_ADDR> --endpoint http://127.0.0.1:9501 --once
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8491 service score --address <DKC_ADDR>

Expected:

* required stake: 100 DKC
* active stake: 100 DKC
* stake eligible: true
* collateral status: eligible

Public RPC quick:

go run ./node/cmd/deskachain --datadir ./testdata/stake_reg_pub node start --rpc :8501 --p2p :9501 --advertise-p2p http://127.0.0.1:9501 --public-rpc

Then:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8501 stake info

Expected:

* works

  go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8501 stake lock --address <DKC_ADDR> --amount 10

Expected:

* error: endpoint disabled in public RPC mode

Jangan over-engineer.
Fokus Phase 3.2.1 hanya:

* staking tests,
* consensus safety,
* regression coverage,
* tiny bug fixes if tests reveal them,
* no new economics,
* no PoS,
* no staking rewards,
* no slashing.
