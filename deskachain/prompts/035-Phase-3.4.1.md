Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 3.4 sudah valid full.
* go work sync pass.
* go test ./node/... pass.
* go test -count=1 ./node/... pass.
* docs/API.md sudah ditambahkan.
* controlled local testnet sudah valid.
* multi-node sync sudah valid.
* peer persistence sudah valid.
* dev faucet testnet sudah valid.
* wallet testnet profile sudah fixed.
* miner/faucet/balance sudah profile-aware.
* faucet request membuat normal pending tx.
* faucet tidak mint langsung.
* faucet rate limit valid.
* faucet total supply safety valid.
* service node testnet valid.
* staking collateral testnet valid.
* testnet params:

  * network: testnet
  * network_id: dkc-testnet-1
  * chain_id: 777101
  * genesis hash: db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4
  * coinbase maturity: 100
  * min stake amount: 100 DKC
  * min service stake: 1000 DKC
  * unbonding period: 100 blocks
* Staking tetap collateral-only.
* Service points tetap simulation-only.
* PoW tetap satu-satunya consensus block production.

Nama patch:
DesKaChain Phase 3.4.1 — Faucet → Stake 1000 → Service Eligible E2E Scenario

Tujuan:
Menambahkan skenario end-to-end testnet:
faucet funding -> recipient receives testnet DKC -> stake lock 1000 DKC -> mine stake tx -> service register -> dkcservice once -> service score eligible.

Fokus:

* E2E scenario tests,
* CLI/RPC workflow hardening,
* faucet funding helper UX jika diperlukan,
* stake/service eligibility verification,
* docs runbook,
* no new economics.

Non-goals:

* Jangan implement public faucet.
* Jangan implement explorer.
* Jangan implement mainnet.
* Jangan implement staking APY.
* Jangan implement staking reward.
* Jangan implement slashing.
* Jangan implement PoS.
* Jangan implement validator set.
* Jangan ubah testnet genesis.
* Jangan ubah localnet genesis.
* Jangan bypass mempool/chain validation.
* Jangan membuat faucet auto-mint.
* Jangan membuat service points spendable.
* Jangan ubah address format DKC.

==================================================

1. End-to-end target flow
   ==================================================

Buat skenario resmi:

1. Init testnet datadir.
2. Create faucet wallet.
3. Create service owner wallet.
4. Start testnet node with faucet enabled.
5. Mine enough blocks to mature faucet wallet.
6. Request faucet funds to service owner.
7. Mine one block to confirm faucet tx.
8. Verify service owner balance.
9. Repeat faucet request or use configured amount until owner has at least 1000 DKC spendable.
10. Stake lock 1000 DKC from service owner.
11. Mine one block to confirm stake tx.
12. Verify stake status active.
13. Verify balance shows active stake 1000 and spendable reduced.
14. Register service node for service owner.
15. Run dkcservice --once or equivalent service agent flow.
16. Check service score.
17. Expected:

    * required stake: 1000 DKC
    * active stake: 1000 DKC
    * stake eligible: true
    * collateral status: eligible
    * eligible simulated points > 0
    * service points are simulation only and are not spendable DKC
18. Verify total supply changed only from mined coinbase blocks, not from faucet/stake/service.

==================================================
2. Faucet funding amount UX
===========================

Current faucet default amount may be 100 DKC.
Testnet min service stake is 1000 DKC.

Options:
A. Keep faucet amount 100 and request 10 times with test helper/time control.
B. Allow dev/testnet faucet start with `--faucet-amount 1000`.
C. Use existing optional request amount if supported:
faucet request --address <ADDR> --amount 1000
but only if max amount config allows it.

Recommended for Phase 3.4.1:

* Support `--faucet-amount 1000` in manual scenario.
* Keep max per address configurable.
* Keep rate limit behavior.
* Do not disable rate limit globally.
* For tests, use short min interval or test clock/helper.

Manual node start can use:

--enable-faucet-rpc
--faucet-address <FAUCET_ADDR>
--faucet-amount 1000
--faucet-min-interval 1s
--faucet-max-per-address 2000

If `--faucet-max-per-address` does not exist yet but internal config has max per address, add CLI flag if simple.

==================================================
3. E2E tests
============

Add tests in appropriate package:

* internal/rpc
* internal/cli
* or new integration-style test package if project has one.

Required tests:

1. TestFaucetToStakeToServiceEligibleE2E
   Flow:

* create testnet node/runtime in test.
* create faucet address and owner address using testnet profile.
* fund faucet with mature balance using helper or mined blocks.
* call faucet request to owner for 1000 DKC.
* mine/commit block containing faucet tx.
* assert owner confirmed/spendable >= 1000.
* call stake lock 1000.
* mine/commit block containing stake tx.
* assert stake list shows active 1000.
* call service register.
* run service challenge/score path or service agent once equivalent.
* assert service score response:

  * required stake = 1000 DKC
  * active stake = 1000 DKC
  * stake_eligible = true
  * collateral_status = eligible
  * eligible_simulated_points > 0
* assert service points not spendable.
* assert total supply did not change from faucet/stake/service except coinbase.

2. TestServiceNotEligibleBeforeStakeE2E

* owner receives faucet funds but does not stake.
* service register/score should show:

  * required stake 1000
  * active stake 0
  * stake_eligible false
  * collateral_status none/insufficient
  * eligible_simulated_points 0.

3. TestServiceNotEligibleWithInsufficientStakeE2E

* owner stakes 100 DKC only.
* service score:

  * active stake 100
  * required 1000
  * stake_eligible false.

4. TestServiceEligibilityLostAfterUnlock

* owner stakes 1000.
* eligible true.
* unlock stake.
* mine unlock tx.
* status unlocking.
* service score:

  * stake_eligible false
  * collateral_status unlocking or insufficient.
  * eligible points 0.
* Mine until release if test helper supports.
* released stake still not active collateral.

5. TestFaucetStakeDoesNotMutateSupplyDirectly

* capture total supply before faucet request.
* faucet request pending: supply unchanged.
* faucet tx mined: supply increases only by coinbase reward.
* stake tx mined: supply increases only by coinbase reward.
* service score/challenge: supply unchanged.

6. TestE2EProfileConsistency

* all wallet addresses generated from testnet datadir validate as testnet.
* faucet/miner/stake/service all accept the same address.
* wrong-network address rejected in faucet/stake/service.

==================================================
4. CLI E2E test
===============

Add CLI-level test if feasible:

TestCLI_FaucetStakeServiceE2E

It should exercise command handlers or remote CLI wrappers:

* faucet info
* faucet request
* balance
* stake lock
* stake list
* service register
* service score

Output assertions:

* faucet tx created.
* balance shows confirmed/spendable 1000.
* stake lock tx created.
* stake list status active.
* service score shows:

  * required stake: 1000 DKC
  * active stake: 1000 DKC
  * stake eligible: true
  * collateral status: eligible
  * eligible simulated points.

Avoid making this test too slow:

* use helper mining/commit if available.
* use low-difficulty test profile helper if existing.
* Do not change production testnet params just for test.

==================================================
5. Optional scenario command
============================

If useful, add a dev-only command:

dev scenario faucet-stake-service

or:

testnet scenario faucet-stake-service

This is optional. Do not over-engineer.

If implemented:

* dev/testnet only.
* refuses public RPC/mainnet.
* prints step-by-step status.
* does not expose private key.
* no silent mint.
* still uses normal faucet tx, stake tx, service flow.

But if this adds too much complexity, skip. Tests + docs are enough.

==================================================
6. Improve faucet docs for service collateral
=============================================

Update docs/Faucet.md:

* Add section: “Using faucet funds for service-node collateral”.
* Explain testnet min service stake = 1000 DKC.
* Example start faucet with amount 1000.
* Example request 1000 DKC.
* Mine confirmation.
* Stake 1000 DKC.
* Register service node.
* Run dkcservice.
* Check eligible simulated points.
* Note:

  * testnet DKC has no monetary value.
  * service points are simulation only.
  * staking is collateral only, not APY.

Update docs/ServiceNode.md:

* Add testnet collateral flow using faucet.
* Add expected output:

  * required stake: 1000 DKC
  * active stake: 1000 DKC
  * stake eligible: true
  * collateral status: eligible.

Update docs/Staking.md:

* Add note that faucet-funded testnet DKC can be used for testing stake lock.
* No staking rewards.

Update docs/Testnet.md:

* Add full runbook for faucet -> stake -> service.

Update docs/API.md:

* Add/verify endpoint references:

  * faucet info/request
  * stake lock/list
  * service register/score
* Add E2E example if short.

Update README.md and README-ID.md:

* Add link to faucet/service E2E docs if not too noisy.

==================================================
7. Manual validation commands
=============================

After patch:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/rpc -run "Faucet.*Stake|Stake.*Service|Service.*Eligible|E2E" -count=1 -v
go test ./node/internal/cli -run "Faucet.*Stake|Stake.*Service|Service.*Eligible|E2E" -count=1 -v
go test ./node/internal/servicenode -run "Eligible|Collateral" -count=1 -v
go test ./node/internal/staking -run "Stake|Unlock" -count=1 -v
go test ./node/internal/faucet -v

Manual E2E:

Clean:

go run ./node/cmd/deskachain --datadir ./testdata/e2e_service dev reset --yes

Init:

go run ./node/cmd/deskachain --datadir ./testdata/e2e_service --network testnet init

Create faucet wallet:

go run ./node/cmd/deskachain --datadir ./testdata/e2e_service wallet new

Save:
<FAUCET_ADDR>

Create service owner wallet:

go run ./node/cmd/deskachain --datadir ./testdata/e2e_service wallet new

Save:
<OWNER_ADDR>

Start node:

go run ./node/cmd/deskachain --datadir ./testdata/e2e_service node start --rpc :8911 --p2p :9911 --advertise-p2p http://127.0.0.1:9911 --enable-faucet-rpc --faucet-address <FAUCET_ADDR> --faucet-amount 1000 --faucet-min-interval 1s --faucet-max-per-address 2000

Fund faucet:

go run ./node/cmd/dkcminer --rpc-url http://127.0.0.1:8911 --address <FAUCET_ADDR> --threads 4 --max-blocks 105

Check faucet:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 balance <FAUCET_ADDR>

Expected:

* mature balance enough for 1000.

Request faucet:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 faucet request --address <OWNER_ADDR>

Expected:

* faucet tx created
* amount 1000 DKC
* status pending

Mine faucet confirmation:

go run ./node/cmd/dkcminer --rpc-url http://127.0.0.1:8911 --address <FAUCET_ADDR> --threads 4 --once

Check owner:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 balance <OWNER_ADDR>

Expected:

* confirmed balance 1000
* mature balance 1000
* spendable balance 1000

Stake lock:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 stake lock --address <OWNER_ADDR> --amount 1000

Expected:

* stake lock tx created
* amount 1000 DKC
* status pending

Mine stake confirmation:

go run ./node/cmd/dkcminer --rpc-url http://127.0.0.1:8911 --address <FAUCET_ADDR> --threads 4 --once

Check stake:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 stake list --address <OWNER_ADDR>
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 balance <OWNER_ADDR>

Expected:

* stake status active.
* active stake 1000.
* spendable reduced to 0 if exactly 1000 funded.

Register service:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 service register --address <OWNER_ADDR> --endpoint http://127.0.0.1:9971

Run service agent:

go run ./node/cmd/dkcservice --rpc-url http://127.0.0.1:8911 --address <OWNER_ADDR> --endpoint http://127.0.0.1:9971 --once

Check score:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 service score --address <OWNER_ADDR>

Expected:

* required stake: 1000 DKC
* active stake: 1000 DKC
* stake eligible: true
* collateral status: eligible
* eligible simulated points > 0
* note: service points are simulation only and are not spendable DKC

Check chain:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 chain validate
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 chain info

Expected:

* chain valid.
* normal tx count includes faucet tx + stake tx.
* total supply only from coinbase blocks.

==================================================
8. Negative manual check
========================

Before stake, service score should be not eligible.

Optional manual:

* after faucet confirmation but before stake:
  service register owner
  service score owner

Expected:

* active stake 0
* stake eligible false
* eligible simulated points 0

After stake:

* same service score becomes eligible.

==================================================
9. Done criteria
================

Phase 3.4.1 valid if:

* faucet-funded wallet can stake 1000 DKC on testnet.
* stake tx mines normally.
* service score sees active stake 1000.
* service eligibility true.
* eligible simulated points > 0.
* insufficient/no stake remains ineligible.
* unlocking/released stake not eligible if tested.
* total supply not mutated by faucet/stake/service except coinbase.
* all tests pass.
* docs updated.
