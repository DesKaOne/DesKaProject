Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 4.1 sudah valid full.
* Phase 4.0 Public Testnet Multi-Host Deployment valid full.
* Phase 4.1 Public Testnet Long-Running Stability & Restart Recovery valid full.
* Real multi-host test sudah berhasil:

  * Host A Mini PC Linux sebagai seed node.
  * Host B Windows sebagai peer node.
  * Host B sync dari Host A via Tailscale seed peer.
  * Host B sempat salah localnet, dan network mismatch safety berhasil menolak sync.
  * Setelah reset/reinit testnet, Host B catch-up sampai height 4.
  * Host A dan Host B punya height sama.
  * Host A dan Host B punya tip hash sama.
  * Chain validate pass di dua host.
* Public testnet genesis candidate stable:

  * network: testnet
  * network_id: idr-testnet-1
  * chain_id: 777101
  * genesis hash: db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4
* CI dan release artifact workflow hijau.
* Release binary Windows/Linux valid.
* Testnet IDR has no monetary value.
* Mainnet does not exist yet.
* PoW remains the only block-production consensus.
* Staking remains collateral-only.
* Service points remain simulation-only.

Patch name:
DesKaChain Phase 4.2 — Public Testnet Faucet + Service Node Multi-Host E2E

Goal:
Validate faucet, staking collateral, and service node simulation across multiple real hosts.

This phase should prove:

* faucet can run on a controlled testnet seed/operator node,
* another host can request testnet IDR,
* faucet creates normal transactions, no silent mint,
* requested faucet funds become available after mining and maturity rules as expected,
* remote host can lock 1000 IDR as service collateral,
* service agent can run from another host,
* service score/eligibility works across public testnet nodes,
* all nodes remain in sync and chain valid,
* service points remain simulation-only,
* staking remains collateral-only.

Non-goals:

* Do not launch mainnet.
* Do not give testnet IDR monetary value.
* Do not promise real mining/service rewards.
* Do not implement staking APY.
* Do not implement slashing/PoS.
* Do not make service points spendable.
* Do not implement explorer.
* Do not implement Stratum/pool mining.
* Do not expose wallet/admin RPC publicly.
* Do not make faucet public by default without operator intent.
* Do not bypass normal transaction rules.
* Do not mint faucet funds silently.

==================================================

1. Multi-host faucet docs
   ==================================================

Add or update:

docs/Faucet.md
docs/MultiHostTestnet.md
docs/LongRunTestnet.md

Document controlled faucet setup:

Host A: Mini PC / faucet seed node

* Runs testnet public node.
* Faucet RPC enabled deliberately.
* Faucet address belongs to operator.
* Faucet wallet must be funded and mature.
* Faucet sends normal transactions.
* Faucet state stores rate limits.
* Testnet IDR has no monetary value.

Host B: Windows/Linux user node

* Joins testnet via seed peer.
* Creates wallet.
* Requests faucet from Host A.
* Mines or waits for mined tx.
* Checks balance.
* Optionally locks 1000 IDR for service collateral.

Warnings:

* Faucet disabled by default.
* Do not expose wallet/admin RPC.
* Faucet RPC should be rate-limited.
* Faucet state should be backed up if continuity matters.
* Faucet private key/wallet should never be committed.
* Faucet is for testnet only.

==================================================
2. Faucet operator runbook
==========================

Add a concrete faucet operator section.

Example flow:

A. Prepare faucet wallet datadir on Host A:

./deskachain --datadir ./data/faucet_wallet --network testnet init
./deskachain --datadir ./data/faucet_wallet wallet new

B. Fund faucet wallet by mining to faucet address:

./idrminer --rpc-url http://127.0.0.1:9311 --address <FAUCET_ADDR> --threads 2 --max-blocks 120

Because testnet coinbase maturity is 100, ensure faucet has mature balance.

C. Start faucet-enabled node on Host A:

./deskachain --datadir ./data/seed node start 
--rpc 0.0.0.0:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://<HOST_A_REACHABLE_IP>:10311 
--public-rpc 
--enable-miner-rpc 
--enable-faucet-rpc 
--faucet-address <FAUCET_ADDR> 
--faucet-amount 1000 
--faucet-min-interval 1h 
--faucet-max-per-address 2000

If current CLI names differ, use existing flags.

D. Request faucet from Host B:

./deskachain --rpc-url http://<HOST_A_RPC_IP>:9311 faucet request --address <USER_ADDR>

or if only local RPC is used, document the correct command pattern.

E. Mine block to confirm faucet tx:

./idrminer --rpc-url http://127.0.0.1:9311 --address <MINER_ADDR> --threads 2 --once

F. Check Host B balance:

./deskachain --rpc-url http://127.0.0.1:9312 wallet balance --address <USER_ADDR>

or correct current balance command.

==================================================
3. Faucet safety tests
======================

Verify or add tests:

* faucet disabled by default.
* faucet cannot run on localnet unless explicitly allowed by existing design.
* faucet rejects wrong-network faucet address.
* faucet rejects wrong-network recipient.
* faucet requires mature balance.
* faucet creates pending normal tx.
* faucet does not change total supply directly.
* faucet rate limit works.
* faucet state persists.
* faucet state corrupt handling works.
* public faucet RPC requires explicit enable flag.

Keep previous faucet tests passing.

==================================================
4. Multi-host stake/service docs
================================

Add or update:

docs/ServiceNode.md
docs/Staking.md
docs/MultiHostTestnet.md

Document E2E service flow:

Host B:

1. Create owner wallet.
2. Request faucet 1000 IDR or receive test funds.
3. Confirm mature/spendable funds if required by transaction model.
4. Lock stake:
   ./deskachain --rpc-url http://127.0.0.1:9312 stake lock --address <OWNER_ADDR> --amount 1000
5. Mine block to confirm stake tx.
6. Register service node:
   ./deskachain --rpc-url http://127.0.0.1:9312 service register --address <OWNER_ADDR> --endpoint http://<HOST_B_REACHABLE_IP>:9971
7. Run service agent:
   ./idrservice --rpc-url http://127.0.0.1:9312 --address <OWNER_ADDR> --endpoint http://<HOST_B_REACHABLE_IP>:9971 --once
8. Check score:
   ./deskachain --rpc-url http://127.0.0.1:9312 service score --address <OWNER_ADDR>

Expected:

* active stake: 1000 IDR.
* required stake: 1000 IDR.
* stake eligible: true.
* service collateral status: eligible.
* service points visible as simulation-only.
* no spendable IDR minted by service score.

Warnings:

* staking is collateral-only.
* no APY.
* no PoS.
* no slashing yet.
* service points are simulation-only.
* service eligibility can be lost if stake is unlocked/released.

==================================================
5. Cross-host service visibility
================================

Ensure service registration/score is visible consistently across nodes.

Scenario:

1. Host B locks stake and registers service.
2. Tx/block propagates to Host A.
3. Host A service score/info for same owner matches Host B.
4. Chain validate pass both nodes.
5. Chain info active stake matches on both nodes.

If service state is node-local and not consensus-backed, document that clearly.
If service registration is consensus tx, ensure it syncs across hosts.
Do not pretend service state is global if it is currently local-only.

Important:

* If current architecture stores service nodes only in local `service_nodes.json`, then Phase 4.2 should explicitly separate:

  * stake collateral on-chain / consensus-backed,
  * service agent score local/simulation store,
  * what must be re-registered per node if needed.
* If service registration already propagates via chain tx, verify cross-host consistency.

Do not invent behavior. Patch docs/tests to match actual architecture.

==================================================
6. Service agent multi-host behavior
====================================

Validate `idrservice` from a host different from seed node.

Cases:

* Host B service agent talks to Host B local RPC.
* Host B service agent can also talk to Host A public RPC if service RPC is enabled there.
* Service RPC disabled should fail clearly.
* Service RPC enabled should allow service challenge/submit/score as designed.
* Wrong address/network should fail clearly.
* Insufficient stake should show not eligible.
* Stake unlocked/released should lose eligibility.

Keep `idrservice --once` fast and deterministic enough for testnet smoke.

==================================================
7. RPC safety with faucet/service
=================================

Public RPC matrix must remain safe:

Public node default:

* wallet_rpc=false
* admin_rpc=false
* miner_rpc optional
* faucet_rpc=false unless explicitly enabled
* service_rpc=false unless explicitly enabled

Faucet operator node:

* wallet_rpc=false
* admin_rpc=false
* miner_rpc maybe true
* faucet_rpc=true explicitly
* service_rpc optional false

Service node API:

* service_rpc=true only when deliberately enabled.
* must not expose wallet/admin actions.

Add tests if needed:

* enabling faucet RPC does not enable wallet/admin.
* enabling service RPC does not enable wallet/admin.
* public wallet new still rejected.
* public stake writes still disabled unless existing intended behavior says otherwise.

==================================================
8. Automated tests
==================

Run:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/rpc -run "Faucet|Stake|Service|Public|Health|ChainInfo|Supply" -count=1 -v

go test ./node/internal/cli -run "Faucet|Stake|Service|Wallet|Public|MultiHost|LongRun" -count=1 -v

go test ./node/internal/servicenode -count=1 -v

go test ./node/internal/serviceagent -count=1 -v

go test ./node/internal/staking -count=1 -v

go test ./node/internal/p2p -run "Sync|Peer|Broadcast|Restart" -count=1 -v

Add or update E2E tests if feasible:

* faucet to stake to service eligible.
* service not eligible before stake.
* insufficient stake not eligible.
* eligibility lost after unlock/release.
* faucet/stake/service do not mutate supply directly.
* public RPC safety with faucet/service toggles.

==================================================
9. Manual multi-host E2E validation
===================================

Use real hosts:

Host A:

* Mini PC Linux
* seed node
* faucet operator
* Tailscale IP example: 100.101.251.7
* RPC: 9311
* P2P: 10311

Host B:

* Windows/Linux peer
* wallet/staker/service owner
* Tailscale IP example: 100.83.159.107
* RPC: 9312
* P2P: 10312
* service endpoint: 9971

Manual proof required:

A. Start Host A faucet-enabled testnet node.
B. Start Host B testnet peer with seed-peer Host A.
C. Host B peer check Host A OK.
D. Host B wallet creates testnet address.
E. Host B requests faucet from Host A.
F. Faucet tx confirmed by mining.
G. Host B balance shows 1000 IDR.
H. Host B locks stake 1000 IDR.
I. Stake tx confirmed by mining.
J. Host B stake list/info shows active stake 1000.
K. Host B registers service endpoint.
L. Run idrservice --once.
M. service score shows eligible.
N. Host A and Host B chain info same height/tip.
O. Host A and Host B chain validate pass.
P. Confirm total supply only reflects mined coinbase, not faucet/service mint.

Expected final:

* owner active stake: 1000 IDR.
* service collateral eligible: true.
* service points > 0 or valid score output.
* chain valid on both hosts.
* same height/tip on both hosts.
* no wallet/admin public exposure.

==================================================
10. Manual command sketch
=========================

Host A, faucet wallet:

./deskachain --datadir ./data/faucet_wallet --network testnet init
FAUCET_ADDR=$(./deskachain --datadir ./data/faucet_wallet wallet new)

Mine mature faucet funds:

./idrminer --rpc-url http://127.0.0.1:9311 --address "$FAUCET_ADDR" --threads 2 --max-blocks 120

Start/restart Host A with faucet enabled:

./deskachain --datadir ./data/seed node start 
--rpc 0.0.0.0:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://100.101.251.7:10311 
--public-rpc 
--enable-miner-rpc 
--enable-faucet-rpc 
--faucet-address "$FAUCET_ADDR" 
--faucet-amount 1000 
--faucet-min-interval 1h 
--faucet-max-per-address 2000

Host B:

./deskachain --datadir ./data/node_b --network testnet init

./deskachain --datadir ./data/node_b node start 
--rpc 0.0.0.0:9312 
--p2p 0.0.0.0:10312 
--advertise-p2p http://100.83.159.107:10312 
--public-rpc 
--enable-miner-rpc 
--enable-service-rpc 
--seed-peer http://100.101.251.7:10311/

Host B wallet:

./deskachain --datadir ./data/wallet_b --network testnet init
OWNER_ADDR=$(./deskachain --datadir ./data/wallet_b wallet new)

Request faucet:

./deskachain --rpc-url http://100.101.251.7:9311 faucet request --address "$OWNER_ADDR"

Mine confirmation:

./idrminer --rpc-url http://127.0.0.1:9311 --address "$FAUCET_ADDR" --threads 2 --once

Check balance:

./deskachain --rpc-url http://127.0.0.1:9312 wallet balance --address "$OWNER_ADDR"

Stake:

./deskachain --rpc-url http://127.0.0.1:9312 stake lock --address "$OWNER_ADDR" --amount 1000

Mine confirmation:

./idrminer --rpc-url http://127.0.0.1:9312 --address "$OWNER_ADDR" --threads 2 --once

Check stake:

./deskachain --rpc-url http://127.0.0.1:9312 stake list --address "$OWNER_ADDR"

Service register:

./deskachain --rpc-url http://127.0.0.1:9312 service register --address "$OWNER_ADDR" --endpoint http://100.83.159.107:9971

Run service once:

./idrservice --rpc-url http://127.0.0.1:9312 --address "$OWNER_ADDR" --endpoint http://100.83.159.107:9971 --once

Score:

./deskachain --rpc-url http://127.0.0.1:9312 service score --address "$OWNER_ADDR"

Final chain checks:

./deskachain --rpc-url http://127.0.0.1:9311 chain info
./deskachain --rpc-url http://127.0.0.1:9312 chain info

./deskachain --rpc-url http://127.0.0.1:9311 chain validate
./deskachain --rpc-url http://127.0.0.1:9312 chain validate

Adjust commands to actual CLI if names/flags differ.

==================================================
11. Release/build validation
============================

If any code changed, validate release still builds:

powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.2-testnet -SkipTests

powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.4.2-testnet -SkipTests

Optional Bash:

bash ./scripts/build.sh v0.4.2-testnet --skip-tests
bash ./scripts/package.sh v0.4.2-testnet --skip-tests

Expected:

* Windows/Linux binaries build.
* archives and SHA256SUMS created.
* no runtime/private state in archives.

==================================================
12. Done criteria
=================

Phase 4.2 valid if:

* all tests pass.
* faucet docs and operator runbook are updated.
* service/staking multi-host docs are updated.
* public RPC safety remains intact.
* faucet disabled by default.
* faucet creates normal tx and does not mint supply directly.
* multi-host faucet request works.
* Host B receives/spends testnet faucet funds.
* Host B locks 1000 IDR stake.
* service node becomes eligible with 1000 IDR active stake.
* service points remain simulation-only.
* Host A and Host B remain synced with same height/tip.
* chain validate passes on both hosts.
* total supply is only from mined coinbase.
