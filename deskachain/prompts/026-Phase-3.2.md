Kamu sedang bekerja pada project Go monorepo IndoChain.

Struktur project:

* node/

  * cmd/indochain/
  * cmd/indominer/
  * cmd/indoservice/
  * internal/
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

* Phase 1 sampai Phase 3.1 sudah selesai dan valid.
* go work sync berhasil.
* go test ./node/... pass.
* Address final sudah aktif:

  * wallet baru menghasilkan address `iND...`
  * format address: `iND` + Base58Check
  * private key raw 32-byte hex
* Dynamic difficulty sudah aktif.
* Coinbase maturity sudah aktif.
* Standalone CPU miner sudah aktif:

  * indominer bisa mine via /miner/template dan /miner/submit.
* Node hardening sudah aktif:

  * /health
  * public RPC mode
  * wallet RPC disabled saat public mode
  * graceful shutdown
* Service node simulation sudah aktif:

  * service register
  * service heartbeat
  * service challenge create/submit
  * service score
  * service rewards
  * indoservice agent
  * service points tidak mengubah dIDR balance
  * service points tidak mengubah total supply
  * public RPC default service_rpc=false

Nama patch:
IndoChain Phase 3.2 — Staking Module & Service Node Collateral

Tujuan:
Menambahkan staking/collateral module untuk mengunci dIDR sebagai syarat/eligibility service node.

Prinsip penting:

* Ini BUKAN Proof-of-Stake consensus.
* Ini BUKAN validator set.
* Ini BUKAN block proposer selection.
* Ini BUKAN APY.
* Ini BUKAN reward dIDR otomatis.
* Ini tidak mengubah PoW sebagai consensus utama.
* PoW tetap satu-satunya pembuat block canonical.
* Staking hanya mengunci dIDR agar tidak bisa dibelanjakan selama aktif/unbonding.
* Service node boleh memakai active stake sebagai collateral/eligibility.
* Tidak ada slashing dIDR pada Phase ini.
* Tidak ada staking reward dIDR pada Phase ini.
* Tidak ada mint dIDR dari staking.

Aturan penting:

* Jangan ubah address format dIDR.
* Jangan ubah private key format.
* Jangan ubah difficulty adjustment.
* Jangan ubah cumulative work formula.
* Jangan ubah coinbase maturity.
* Jangan ubah PoW consensus.
* Jangan memasukkan staking reward ke coinbase.
* Jangan implement PoS.
* Jangan implement validator.
* Jangan implement delegation.
* Jangan implement slashing.
* Jangan implement GPU miner.
* Jangan implement mining pool.
* Patch incremental.
* Semua command lama harus tetap bekerja.
* Semua test harus tetap pass:

  go work sync
  go test ./node/...

==================================================

1. Consensus params untuk staking
   ==================================================

Tambahkan staking params ke NetworkProfile / ConsensusParams.

Minimal:

type StakingParams struct {
Enabled bool
MinServiceStake Amount
MinStakeAmount Amount
UnbondingPeriodBlocks uint64
MaxActiveStakesPerAddress int
}

Default localnet:

* Enabled: true
* MinServiceStake: 100 dIDR
* MinStakeAmount: 10 dIDR
* UnbondingPeriodBlocks: 10
* MaxActiveStakesPerAddress: 10

Default testnet placeholder:

* Enabled: true
* MinServiceStake: 1000 dIDR
* MinStakeAmount: 100 dIDR
* UnbondingPeriodBlocks: 100
* MaxActiveStakesPerAddress: 20

Default mainnet placeholder:

* Enabled: false or true placeholder, document as TBD.
* MinServiceStake: TBD
* MinStakeAmount: TBD
* UnbondingPeriodBlocks: TBD

Catatan:

* Mainnet params belum final.
* Jangan hardcode angka staking di banyak tempat.
* Semua staking validation harus ambil dari network profile.

==================================================
2. Transaction model untuk staking
==================================

Implementasikan staking secara deterministic agar semua node menghitung state yang sama.

Preferred approach:

* Tambahkan transaction type secara backward-compatible.

TxType:

* transfer
* coinbase
* stake_lock
* stake_unlock

Jika existing transaction sudah punya coinbase bool:

* Pertahankan compatibility.
* Coinbase lama tetap valid.
* Jika tx type kosong pada tx lama:

  * coinbase true => coinbase
  * coinbase false => transfer

Tambahkan optional fields jika perlu:

* tx.type
* tx.stake_id
* tx.memo or data optional

Rules:

* Transfer tx lama tetap valid.
* Existing blocks/tests tetap valid.
* Storage lama tidak rusak.
* JSON backward-compatible.

Jika project belum siap menambah TxType besar:

* Buat minimal optional field `type` dengan default `transfer`.
* Jangan rewrite semua tx logic besar-besaran.

==================================================
3. Stake lock transaction
=========================

Command:

stake lock --address <IND_ADDR> --amount <AMOUNT>

Remote:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8451 stake lock --address <IND_ADDR> --amount 100

Behavior:

* Validate address.
* Validate amount >= MinStakeAmount.
* Check mature/spendable balance.
* Coinbase immature tidak boleh dipakai.
* Pending outgoing harus dikurangi.
* Active/unbonding stake existing harus mengurangi spendable.
* Create signed tx type `stake_lock`.
* Add tx to mempool.
* Broadcast tx.
* Status pending.

Output:
stake lock tx created
tx id: ...
address: iND...
amount: 100 dIDR
status: pending
note: stake becomes active after tx is mined

After mined:

* Stake record becomes active.
* Amount is locked.
* Locked amount reduces spendable balance.
* Confirmed balance stays same.
* Total supply unchanged.

Stake ID:

* Deterministic.
* Recommended:
  stake_id = tx_id
* Or:
  stake_id = hash(address + tx_id + amount)
* Keep simple and deterministic.

==================================================
4. Stake unlock transaction
===========================

Command:

stake unlock --address <IND_ADDR> --stake-id <STAKE_ID>

Remote:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8451 stake unlock --address <IND_ADDR> --stake-id <STAKE_ID>

Behavior:

* Validate address.
* Validate stake exists.
* Stake owner must match address.
* Stake must be active.
* Create signed tx type `stake_unlock`.
* Add tx to mempool.
* Broadcast tx.
* Status pending.

After mined:

* Stake status becomes `unlocking`.
* unlock_height = block height.
* release_height = unlock_height + UnbondingPeriodBlocks.
* Amount remains locked during unbonding.
* After current height >= release_height:

  * stake status considered `released`.
  * amount no longer reduces spendable balance.

No separate withdraw tx needed in Phase 3.2.
Released stake automatically becomes spendable again because it is no longer counted as locked.

Output:
stake unlock tx created
stake id: ...
release height: current + unbonding period

==================================================
5. Stake state derivation
=========================

Staking state must be derived from canonical chain.

Add package:
internal/staking

Core structs:

* StakeRecord
* StakeStatus
* StakeState
* StakingParams

StakeStatus:

* active
* unlocking
* released

StakeRecord fields:

* stake_id
* owner_address
* amount
* lock_tx_id
* lock_height
* unlock_tx_id
* unlock_height
* release_height
* status

Important:

* Do not rely only on mutable JSON store for consensus staking state.
* Staking state must be rebuildable from canonical blocks.
* Reorg must recalculate staking state from canonical chain.
* Orphaned stake txs must not count as active.
* Requeued stake txs in mempool count as pending, not confirmed.

A JSON staking cache/store is allowed for performance, but canonical source of truth must be chain data.

==================================================
6. Spendable balance integration
================================

Update balance details to include staking.

Balance command should show:

confirmed balance: 550 dIDR
mature balance: 250 dIDR
immature balance: 300 dIDR
active stake: 100 dIDR
unlocking stake: 0 dIDR
released stake: 0 dIDR
pending stake lock: 0 dIDR
pending outgoing: 0 dIDR
spendable balance: 150 dIDR

Rules:

* confirmed balance unchanged by staking.
* mature balance unchanged by staking.
* active stake reduces spendable.
* unlocking stake reduces spendable until release height.
* released stake no longer reduces spendable.
* pending stake lock reduces spendable to avoid double spend.
* pending transfer outgoing still reduces spendable.
* normal transfer received is mature/spendable after confirmed.
* immature coinbase cannot be staked.

Formula:
spendable =
mature_balance
- active_stake
- unlocking_stake
- pending_stake_lock
- pending_outgoing_transfer

Clamp at 0 if needed, but validation logic should prevent negative.

==================================================
7. Send validation with active stake
====================================

Update send validation:

* A user cannot send dIDR locked in active/unbonding stake.
* If mature balance 150 and active stake 100:

  * spendable 50.
  * send 60 fails.
  * send 50 succeeds.
* Error:
  insufficient spendable balance: spendable X dIDR, required Y dIDR, active stake Z dIDR, unlocking stake U dIDR

==================================================
8. Stake validation in chain validation
=======================================

Update chain validation so staking rules are deterministic.

For each block:

* Apply transactions sequentially.
* For stake_lock:

  * validate owner signature.
  * validate amount >= min stake.
  * validate mature spendable balance before tx.
  * validate max active stakes per address.
  * create active stake record.
* For stake_unlock:

  * validate owner signature.
  * validate stake exists.
  * validate stake owner.
  * validate stake active.
  * mark unlocking with release height.
* For transfer:

  * validate spendable balance after considering active/unbonding stake.
* Coinbase:

  * unchanged.

Reject invalid:

* stake amount below min.
* stake immature coinbase.
* stake more than spendable.
* unlock non-existing stake.
* unlock someone else's stake.
* unlock already unlocking/released stake.
* duplicate stake id.
* transfer using locked stake.

Error examples:

* invalid stake lock: amount below minimum
* invalid stake lock: insufficient mature spendable balance
* invalid stake unlock: stake not found
* invalid stake unlock: owner mismatch
* invalid transaction: spends locked stake

==================================================
9. Mempool validation
=====================

Update mempool validation:

* stake_lock tx must pass current spendable check.
* stake_unlock tx must reference active stake.
* pending stake lock must reduce spendable for later pending tx.
* pending stake unlock should not immediately release funds.
* duplicate stake lock tx rejected by duplicate tx guard.
* duplicate unlock for same stake rejected.

Example:

* mature 150.
* create pending stake lock 100.
* spendable should become 50.
* send 60 should fail while stake lock pending.
* after stake lock mined, active stake 100 and spendable still 50.

==================================================
10. Reorg compatibility
=======================

Reorg must work with staking.

Rules:

* Rebuild stake state from canonical branch after reorg.
* Stake locks in orphaned blocks are not active anymore.
* Unlocks in orphaned blocks are not applied.
* Orphaned stake txs can be requeued to mempool only if still valid on new branch.
* If requeued stake lock no longer has spendable funds, drop invalid.
* If requeued stake unlock no longer references active stake, drop invalid.
* Mempool pending stake locks must reduce spendable.

Tests must cover at least simple reorg if possible.

==================================================
11. Service node collateral integration
=======================================

Integrate staking with service node module.

Add service collateral fields:

* required_stake
* active_stake
* stake_eligible true/false
* collateral_status:

  * none
  * insufficient
  * eligible
  * unlocking
  * released

Service score should display collateral info.

Command:

service score --address <IND_ADDR>

Output includes:
required stake: 100 dIDR
active stake: 100 dIDR
stake eligible: true
collateral status: eligible

Rules:

* Phase 3.2 should not slash stake.
* Phase 3.2 should not pay dIDR staking rewards.
* Service points may still be simulation only.
* If active stake < required:

  * stake_eligible false.
  * service score can still be calculated, but rewards should be marked ineligible or simulated_points = 0 depending config.
* Recommended:

  * service_score still shown.
  * simulated_points shown.
  * eligibility_note:
    "not eligible for service reward simulation until active stake >= required stake"
  * If eligible, points count as eligible simulated points.

Add config:
RequireStakeForServiceRewards bool

Default localnet:
true

If true:

* rewards command should show points as ineligible if stake below required.
* Do not mutate dIDR.

==================================================
12. Stake commands
==================

Add command group:

stake

Commands:

1. stake info
   Shows network staking params:
   staking enabled
   min stake amount
   min service stake
   unbonding period
   total active stake
   total unlocking stake
   active stake records

2. stake lock
   stake lock --address <IND_ADDR> --amount <AMOUNT>

3. stake unlock
   stake unlock --address <IND_ADDR> --stake-id <STAKE_ID>

4. stake list
   stake list --address <IND_ADDR>
   stake list
   Shows stake records:
   stake id
   owner
   amount
   status
   lock height
   unlock height
   release height

5. stake status
   stake status --stake-id <STAKE_ID>

Local and remote mode:

* Commands should work with --rpc-url.
* Local mode can read datadir if node not running.
* If datadir locked, suggest valid remote command.

==================================================
13. RPC endpoints
=================

Add RPC endpoints:

GET /stake/info
GET /stake/list
GET /stake/list?address=<IND_ADDR>
GET /stake/status?id=<STAKE_ID>
POST /stake/lock
POST /stake/unlock

Request lock:
{
"address": "iND...",
"amount": "100"
}

Request unlock:
{
"address": "iND...",
"stake_id": "..."
}

Responses:

* JSON consistent with existing RPC.
* Clear error messages.

Public RPC mode:

* Read-only stake info/list/status can be enabled.
* Stake lock/unlock are wallet/signing endpoints if they require local wallet private key.
* In public RPC mode, disable stake lock/unlock by default.
* If node supports unsigned tx creation later, public mode can allow broadcast only, but not now.

Error:
endpoint disabled in public RPC mode

==================================================
14. Wallet/signing behavior
===========================

Stake lock/unlock needs owner signature.

If command runs local:

* sign using wallet store.

If command runs remote local/admin mode:

* server signs using wallet stored on node, same as existing send behavior if applicable.

If public RPC:

* disabled by default because wallet private keys must not be exposed/used on public node.

Do not print private key.

==================================================
15. Chain info additions
========================

Update chain info:

Fields:

* staking enabled
* min stake amount
* min service stake
* unbonding period
* total active stake
* total unlocking stake
* active stake count

Do not change total supply/circulating supply due to staking.
Optionally add:

* liquid circulating supply = circulating supply - active stake - unlocking stake

But do not replace existing circulating supply.

==================================================
16. Health additions
====================

Update /health with simple fields:

* staking_enabled
* active_stake_count
* total_active_stake

Keep health fast.

==================================================
17. Docs
========

Create/update:

docs/Staking.md

Content:

* Staking in Phase 3.2 is collateral only.
* It is not PoS.
* It does not create validators.
* It does not create dIDR rewards.
* It does not affect block production.
* Locked stake reduces spendable balance.
* Unlock starts unbonding period.
* Released stake becomes spendable again.
* Service node collateral requires active stake.
* No slashing in Phase 3.2.
* No APY promise.
* No guaranteed profit.

Update docs/ServiceNode.md and docs/ServiceAgent.md:

* Service node can become stake-eligible if active stake >= required stake.
* Service points still simulation only.

Update README/Roadmap:

* Current phase:
  Phase 3.2 — Staking Module & Service Node Collateral

==================================================
18. Tests wajib
===============

Add/update tests:

1. Stake params:

* localnet params enabled.
* min stake amount 10 dIDR.
* min service stake 100 dIDR.
* unbonding period 10.

2. Stake lock immature rejected:

* mine 3 blocks.
* confirmed 150, mature 0.
* stake lock 10 rejected insufficient mature spendable.

3. Stake lock after maturity success:

* mine 11 blocks.
* mature 50.
* stake lock 10 creates pending tx.
* after mining, active stake 10.

4. Stake lock below min rejected:

* amount below min rejected.

5. Active stake reduces spendable:

* mature 50.
* active stake 10.
* spendable 40.

6. Transfer cannot spend active stake:

* mature 50 active stake 40.
* send 20 fails if spendable only 10.

7. Pending stake lock reduces spendable:

* pending stake lock 40.
* send requiring more than remaining spendable fails before stake mined.

8. Stake unlock success:

* active stake exists.
* unlock tx created.
* after mined, status unlocking.
* release height = unlock height + unbonding period.

9. Unlocking stake still locked:

* during unbonding, spendable still reduced.

10. Released stake spendable:

* mine until release height.
* stake status released.
* spendable increases.

11. Unlock invalid stake rejected:

* non-existing stake id rejected.

12. Unlock owner mismatch rejected:

* address B cannot unlock address A stake.

13. Duplicate unlock rejected:

* stake already unlocking cannot unlock again.

14. Chain validate rejects invalid stake lock:

* block with stake lock over spendable fails.

15. Chain validate rejects transfer using locked funds:

* invalid chain fails.

16. Reorg stake lock:

* stake lock in orphaned branch removed from active state after reorg.

17. Reorg requeue stake tx:

* orphaned valid stake lock can requeue if still spendable.
* invalid requeued stake lock dropped.

18. Service collateral not eligible:

* active stake below min service stake.
* service score shows stake_eligible false.

19. Service collateral eligible:

* active stake >= min service stake.
* service score shows stake_eligible true.

20. Service reward simulation with require stake:

* below required stake => ineligible note or eligible_points 0.
* above required stake => eligible simulated points.

21. No dIDR reward from staking:

* stake lock/unlock does not change total supply.
* no staking reward dIDR minted.

22. Balance fields:

* balance output has active stake, unlocking stake, spendable.

23. Stake RPC:

* /stake/info works.
* /stake/list works.
* /stake/lock works in local/admin mode.
* /stake/unlock works in local/admin mode.

24. Public RPC disables stake lock/unlock:

* public-rpc true.
* stake info works.
* stake lock returns endpoint disabled.

25. Existing tests still pass:

* address
* crypto
* ledger
* mempool
* p2p
* rpc
* cpuminer
* serviceagent
* servicenode
* wallet
* chain
* cli
* indominer

==================================================
19. Expected final commands
===========================

After patch:

go work sync
go mod tidy
go test ./node/...

Manual basic:

go run ./node/cmd/indochain --datadir ./testdata/stake dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/stake init
go run ./node/cmd/indochain --datadir ./testdata/stake wallet new

Start node:

go run ./node/cmd/indochain --datadir ./testdata/stake node start --rpc :8471 --p2p :9471 --advertise-p2p http://127.0.0.1:9471

Mine until mature:

go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8471 --address <IND_ADDR> --threads 4 --max-blocks 11

Balance:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 balance <IND_ADDR>

Expected:
confirmed balance: 550
mature balance: 50
spendable balance: 50

Stake info:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 stake info

Expected:
staking enabled: true
min stake amount: 10 dIDR
min service stake: 100 dIDR
unbonding period: 10 blocks

Stake lock 10:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 stake lock --address <IND_ADDR> --amount 10

Expected:
stake lock tx created
status: pending

Mine 1 block:

go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8471 --address <IND_ADDR> --threads 4 --once

Stake list:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 stake list --address <IND_ADDR>

Expected:
amount: 10 dIDR
status: active

Balance:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 balance <IND_ADDR>

Expected:
active stake: 10 dIDR
spendable balance reduced by 10

Try send over spendable:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 wallet new
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 send --from <IND_ADDR> --to <ADDR_B> --amount 45

Expected:
fail if spendable below 45 after active stake

Unlock:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 stake unlock --address <IND_ADDR> --stake-id <STAKE_ID>

Mine 1 block:

go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8471 --address <IND_ADDR> --threads 4 --once

Stake list:
Expected:
status: unlocking
release height shown

Mine until release height:
Expected:
status: released
spendable balance increases

Service collateral test:

* Stake 100 dIDR after enough maturity.
* Register service.
* Run indoservice once.
* service score should show stake eligible true.

Consensus check:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 chain info
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 chain validate

Expected:
chain valid
total supply unchanged by staking
no staking reward dIDR minted

Public RPC check:

go run ./node/cmd/indochain --datadir ./testdata/stake_pub node start --rpc :8481 --p2p :9481 --advertise-p2p http://127.0.0.1:9481 --public-rpc

Then:
stake info should work.
stake lock should fail:
endpoint disabled in public RPC mode

Jangan over-engineer.
Fokus Phase 3.2 hanya:

* staking lock/unlock,
* spendable balance integration,
* unbonding,
* service node collateral eligibility,
* docs/tests,
* no PoS,
* no staking reward dIDR,
* no slashing yet.
