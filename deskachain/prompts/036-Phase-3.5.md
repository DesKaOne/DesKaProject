Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 3.4.1 sudah valid full.
* go work sync pass.
* go test ./node/... pass.
* go test -count=1 ./node/... pass.
* Phase 3.4 Dev Faucet & Testnet Funding Flow valid full.
* Phase 3.4.1 Faucet -> Stake 1000 -> Service Eligible E2E valid full.
* Controlled local testnet valid.
* Multi-node sync valid.
* Peer persistence valid.
* Dev faucet testnet valid.
* Wallet testnet profile fixed.
* Miner/faucet/balance profile-aware.
* Faucet creates normal transaction, no silent mint.
* Faucet rate limit valid.
* Faucet-funded owner can stake 1000 IDR.
* Service node becomes eligible when active stake is 1000 IDR.
* Service points remain simulation-only and not spendable IDR.
* Staking remains collateral-only.
* PoW remains the only block-production consensus.

Latest manual E2E checkpoint:

* network: testnet
* chain id: 777101
* height: 122
* difficulty: 6
* total supply: 6100 IDR
* pending tx count: 0
* total transactions: 124
* coinbase transactions: 122
* normal transactions: 2
* staking enabled: true
* min stake amount: 100 IDR
* min service stake: 1000 IDR
* total active stake: 1000 IDR
* active stake count: 1
* chain validate: chain valid

Important observation:
`chain info` printed:
circulating supply: 0 IDR

This may be wrong or undefined because at height 122 with coinbase maturity 100, some coinbase rewards should already be mature, and one account has 1000 IDR active stake. Phase 3.5 must define and test circulating supply semantics clearly.

Patch name:
DesKaChain Phase 3.5 — Public Testnet Node Packaging & Operator Runbook

Goal:
Prepare DesKaChain for a public-testnet style operator workflow without changing consensus or economics.

This phase should make it easy and safe for someone to run:

1. a public read-only testnet node,
2. a miner-enabled testnet node,
3. a private/local wallet node,
4. a service-node agent against a testnet node,
5. optional dev faucet node for controlled testing.

Focus:

* public-safe node profiles,
* config examples,
* operator docs,
* systemd examples,
* Windows PowerShell examples,
* Linux examples,
* preflight checks,
* public RPC hardening,
* startup summary clarity,
* health/status endpoint consistency,
* chain info sanity,
* no risky defaults.

Non-goals:

* Do not launch mainnet.
* Do not change testnet genesis.
* Do not change localnet genesis.
* Do not add real monetary value.
* Do not implement explorer.
* Do not implement Stratum.
* Do not implement pool mining.
* Do not implement staking rewards/APY.
* Do not implement slashing.
* Do not implement PoS/validator consensus.
* Do not make service points spendable.
* Do not make faucet public by default.
* Do not expose wallet/admin RPC in public mode.
* Do not bypass address/network validation.
* Do not weaken miner submit validation.

==================================================

1. Public testnet node mode
   ==================================================

Ensure there is a clean and documented way to start a public-safe testnet node.

Expected behavior:

* network: testnet
* public RPC enabled for safe read-only endpoints.
* wallet RPC disabled.
* admin RPC disabled.
* faucet RPC disabled unless explicitly enabled.
* service RPC disabled unless explicitly enabled.
* miner RPC can be enabled explicitly for public mining tests, but default should be conservative.
* peer/P2P enabled.
* node prints a clear startup summary.

Example command:

deskachain --datadir ./data/testnet-public --network testnet init

deskachain --datadir ./data/testnet-public node start 
--rpc :8811 
--p2p :9811 
--advertise-p2p http://<PUBLIC_HOST>:9811 
--public-rpc 
--enable-miner-rpc

If current flag names differ, use existing flags and update docs accordingly.

Startup summary should clearly print:

* network
* network id
* chain id
* genesis hash
* datadir
* rpc listen address
* p2p listen address
* advertised p2p URL
* public_rpc true/false
* wallet_rpc true/false
* miner_rpc true/false
* admin_rpc true/false
* faucet_rpc true/false
* service_rpc true/false
* bootnodes
* peer count if known

Add tests:

* public node disables wallet/admin RPC.
* public node read-only chain endpoints work.
* public node miner endpoint only works when explicitly enabled.
* public node faucet endpoint disabled by default.
* public node service endpoint disabled by default.
* startup config summary includes correct mode fields if testable.

==================================================
2. Private local wallet node mode
=================================

Document a private node mode for local wallet usage.

Expected:

* wallet RPC allowed only on localhost/local trusted usage.
* admin/dev endpoints not public.
* clear warning if binding wallet RPC to 0.0.0.0.

If the code already prevents unsafe binding, test it.
If not, add a warning or refusal for dangerous combinations such as:

* wallet_rpc true and rpc bind 0.0.0.0 with public_rpc true.
* admin_rpc true and rpc bind 0.0.0.0.

Do not break local development.

Add tests:

* unsafe public wallet/admin mode rejected or loudly warned depending current architecture.
* local wallet mode still works.

==================================================
3. Miner-enabled public testnet node
====================================

Document and test miner-safe RPC mode.

Expected miner endpoints:

* miner template works if miner_rpc enabled.
* miner submit works if miner_rpc enabled.
* miner endpoints reject wrong-network reward address.
* miner endpoints disabled if miner_rpc false.

Add docs with example:

idrminer --rpc-url http://<NODE_HOST>:8811 --address <TESTNET_IDR_ADDR> --threads 4

Add note:

* HTTP RPC mining is solo/direct-node mining.
* Stratum/pool mining is not implemented yet.
* Testnet IDR has no monetary value.

==================================================
4. Bootnode and peer operator workflow
======================================

Create clear docs for running multiple public testnet nodes.

Examples:
Node A bootnode:
deskachain --datadir ./data/node-a --network testnet init
deskachain --datadir ./data/node-a node start --rpc :8811 --p2p :9811 --advertise-p2p http://<A_HOST>:9811 --public-rpc --enable-miner-rpc

Node B with bootnode:
deskachain --datadir ./data/node-b --network testnet init
deskachain --datadir ./data/node-b node start --rpc :8812 --p2p :9812 --advertise-p2p http://<B_HOST>:9812 --bootnode http://<A_HOST>:9811 --public-rpc --enable-miner-rpc

CLI checks:
deskachain --rpc-url http://<B_HOST>:8812 peer list
deskachain --rpc-url http://<B_HOST>:8812 peer check http://<A_HOST>:9811
deskachain --rpc-url http://<B_HOST>:8812 peer sync http://<A_HOST>:9811
deskachain --rpc-url http://<B_HOST>:8812 chain info
deskachain --rpc-url http://<B_HOST>:8812 chain validate

Add tests if missing:

* bootnode URL normalized.
* bootnode persisted.
* restart uses persisted peer.
* public node refuses wrong network peer sync.
* public node refuses wrong genesis peer sync.

Do not duplicate existing tests unless useful.

==================================================
5. Config examples
==================

Add example files under something like:

examples/testnet/public-node.env
examples/testnet/miner-node.env
examples/testnet/faucet-node.env
examples/testnet/service-agent.env
examples/systemd/deskachain-testnet.service
examples/systemd/idrservice-testnet.service

If the project does not use env files yet, examples can be docs-only templates.

Include:

* datadir
* network
* rpc bind
* p2p bind
* advertise p2p
* bootnode
* public rpc mode
* miner rpc toggle
* faucet rpc toggle
* faucet address/amount/max/min interval
* service rpc toggle
* logs path if supported

Do not add secrets to examples.
Do not include private keys.

==================================================
6. Systemd examples
===================

Add Linux systemd sample for node:

[Unit]
Description=DesKaChain Testnet Node
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/deskachain
EnvironmentFile=/etc/deskachain/testnet.env
ExecStart=/opt/deskachain/bin/deskachain --datadir ${IDR_DATADIR} node start ...
Restart=always
RestartSec=5
LimitNOFILE=65535
KillSignal=SIGINT
TimeoutStopSec=30
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target

Add service agent sample if useful.

Docs should explain:
sudo systemctl daemon-reload
sudo systemctl enable --now deskachain-testnet
journalctl -u deskachain-testnet -f

==================================================
7. Windows PowerShell operator examples
=======================================

Add docs for Windows development/testnet operator:

* init testnet datadir
* start node
* run miner
* peer list/check/sync
* faucet flow
* stake/service flow
* stop/restart notes

Use commands compatible with current `go run ./node/cmd/...` workflow and optionally compiled binary workflow.

==================================================
8. Preflight command or checklist
=================================

If simple, add a `node preflight` command or improve existing command.

Optional command:
deskachain --datadir ./data/testnet node preflight

It should check:

* datadir exists/initialized.
* network metadata exists.
* genesis hash matches profile.
* RPC port configured.
* P2P port configured.
* advertised P2P URL present if public.
* public mode does not expose wallet/admin.
* faucet enabled only testnet/dev.
* service enabled only when explicitly requested.
* peer store readable.
* chain validate passes if chain exists.

If too much complexity, skip command and add docs checklist only.

Do not over-engineer.

==================================================
9. Chain info and circulating supply sanity
===========================================

Investigate this observed output after valid Phase 3.4.1 manual E2E:

height: 122
coinbase maturity: 100
total supply: 6100 IDR
normal transactions: 2
total active stake: 1000 IDR
circulating supply: 0 IDR

Define exact semantics for circulating supply.

Recommended semantics:

* total supply = sum of all mined coinbase rewards.
* circulating supply = mature supply, including mature coins that are active stake/unlocking/released, excluding immature coinbase.
* spendable supply is different from circulating supply and excludes active/unlocking stake and pending outgoing.

Under this definition, at height 122 with maturity 100 and reward 50 IDR:

* mature coinbase blocks should be about 22 blocks.
* mature supply should be about 1100 IDR.
* active stake 1000 IDR should still count as circulating because it is mature and owner-controlled collateral, just not spendable.

If the project intentionally defines circulating supply differently, document it clearly and add tests.

Add tests:

1. TestChainInfoCirculatingSupplyWithMaturity

* mine past maturity.
* assert circulating supply equals mature coinbase supply.

2. TestChainInfoCirculatingSupplyIncludesActiveStake

* faucet fund or transfer mature coins.
* stake lock 1000.
* assert active stake does not reduce circulating supply.

3. TestChainInfoSpendableNotEqualCirculating

* active stake reduces spendable but not circulating.

4. TestChainInfoSupplyUnaffectedByFaucetStakeService

* faucet/stake/service do not increase total supply except mined coinbase blocks.

If existing chain info field is wrong, fix it.
If field name should be changed, prefer backward-compatible addition:

* circulating supply
* mature supply
* spendable supply if globally available

Do not hide the field or set it to zero.

==================================================
10. Docs update
===============

Update or add:

docs/Testnet.md
docs/API.md
docs/Faucet.md
docs/Staking.md
docs/ServiceNode.md
docs/Architecture.md
README.md
README-ID.md

Add new doc if useful:

docs/Operator.md

docs/Operator.md should include:

* public testnet node quickstart.
* firewall ports.
* systemd setup.
* Windows PowerShell quickstart.
* miner quickstart.
* peer sync.
* health checks.
* safe public RPC notes.
* faucet operator mode.
* service node operator mode.
* troubleshooting.

Safety notes:

* Testnet IDR has no monetary value.
* Do not expose wallet/admin RPC publicly.
* Service points are simulation-only.
* Staking is collateral-only, no APY/reward.
* Back up wallet files before deleting datadir.
* Do not use mainnet funds; mainnet does not exist yet.

==================================================
11. API docs alignment
======================

Ensure docs/API.md matches actual endpoints and access modes.

Check at least:

* /health
* /chain/info
* /chain/validate
* /balance
* /wallet/new
* /send
* /miner/template
* /miner/submit
* /faucet/info
* /faucet/request
* /stake/info
* /stake/lock
* /stake/list
* /stake/unlock
* /service/register
* /service/score
* /peers
* /p2p/health
* /p2p/status
* /p2p/blocks or actual block sync endpoints

Do not document endpoints that do not exist.
If docs/API.md currently has uncertain endpoint names, correct them.

==================================================
12. Tests
=========

Required after patch:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/rpc -run "Public|Access|Miner|Faucet|Service|ChainInfo|Circulating|Supply" -count=1 -v
go test ./node/internal/cli -run "Public|Preflight|Operator|Wallet|ChainInfo" -count=1 -v
go test ./node/internal/config -run "Profile|Public|Testnet" -count=1 -v
go test ./node/internal/p2p -run "Bootnode|Peer|NetworkMismatch|GenesisMismatch" -count=1 -v
go test ./node/internal/chain -run "Supply|Circulating|Maturity" -count=1 -v

If some packages have no matching tests, that is okay only if the functionality is covered elsewhere. Prefer adding focused tests for new behavior.

==================================================
13. Manual validation
=====================

Manual public node:

go run ./node/cmd/deskachain --datadir ./testdata/public_node dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/public_node --network testnet init

Start:

go run ./node/cmd/deskachain --datadir ./testdata/public_node node start --rpc :9011 --p2p :10011 --advertise-p2p http://127.0.0.1:10011 --public-rpc --enable-miner-rpc

Check:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:9011 chain info
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:9011 chain validate
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:9011 wallet new

Expected:

* chain info works.
* chain validate works.
* wallet new rejected in public RPC mode.

Mine:

go run ./node/cmd/deskachain --datadir ./testdata/public_node wallet new

Use local datadir wallet address as miner reward.

go run ./node/cmd/idrminer --rpc-url http://127.0.0.1:9011 --address <ADDR> --threads 2 --once

Expected:

* submit accepted.

Public faucet check:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:9011 faucet info

Expected:

* disabled unless explicitly enabled.

Service check:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:9011 service score --address <ADDR>

Expected:

* disabled unless explicitly enabled, or safe read-only depending intended design. Document exact expected behavior.

Circulating supply check:

* mine to height > maturity.
* run chain info.
* circulating supply should not stay 0 if mature supply exists, unless intentionally documented.

==================================================
14. Done criteria
=================

Phase 3.5 valid if:

* public testnet node mode is safe and documented.
* wallet/admin RPC not exposed in public mode.
* miner RPC mode is explicit and tested.
* faucet/service RPC modes are explicit and tested.
* operator docs exist and are accurate.
* systemd/examples exist.
* Windows/Linux runbooks exist.
* bootnode/peer operator flow documented.
* chain info circulating supply is defined, tested, and not suspiciously zero after maturity.
* all tests pass.
* manual public node sanity works.
