Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 3.8 sudah valid full.
* Phase 3.8 GitHub Actions CI Release Artifacts valid full.
* CI workflow hijau di GitHub.
* Release artifact workflow hijau di GitHub.
* Release packages Windows/Linux + SHA256SUMS sudah valid.
* Public testnet node mode valid.
* Public RPC safety valid.
* Seed peer bootstrap valid.
* Chain info snapshot setelah P2P import valid.
* Faucet testnet valid.
* Faucet -> stake 1000 -> service eligible E2E valid.
* Binary release smoke test valid.
* Testnet IDR has no monetary value.
* Mainnet does not exist yet.
* Staking remains collateral-only.
* Service points remain simulation-only.
* PoW remains the only block-production consensus.

Patch name:
DesKaChain Phase 3.9 — Public Testnet Genesis Candidate & Seed Node Deployment Prep

Goal:
Prepare a clean public testnet candidate deployment using release artifacts, with a reproducible seed node setup, operator checklist, deployment docs, and final safety validation before running on VPS or external devices.

This phase is not mainnet launch.
This phase is public-testnet deployment preparation only.

Non-goals:

* Do not launch mainnet.
* Do not change consensus.
* Do not add real monetary value.
* Do not promise rewards/profit.
* Do not implement staking rewards/APY.
* Do not implement slashing/PoS.
* Do not make service points spendable.
* Do not implement explorer.
* Do not implement Stratum/pool mining.
* Do not auto-publish public seed addresses unless configured by operator.
* Do not expose wallet/admin RPC publicly.
* Do not include private keys, wallet files, faucet state, service state, or runtime datadirs in release artifacts.

==================================================

1. Testnet genesis candidate verification
   ==================================================

Create a formal testnet genesis candidate verification document.

Add or update:

docs/TestnetGenesis.md

It should include:

* network: testnet
* network_id: idr-testnet-1
* chain_id: 777101
* genesis hash: db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4
* target block time: 30s
* retarget window: 30
* min difficulty: 1
* max difficulty: 12
* coinbase maturity: 100
* min stake amount: 100 IDR
* min service stake: 1000 IDR
* unbonding period: 100
* block reward if defined in code/docs
* note: testnet IDR has no monetary value
* note: mainnet not available

Add command examples to verify:

deskachain --datadir ./data/testnet --network testnet init
deskachain --datadir ./data/testnet chain info
deskachain --datadir ./data/testnet chain validate

If local CLI cannot call chain info without node start, document the correct command.

Add tests:

* testnet genesis hash stable.
* testnet network id stable.
* testnet chain id stable.
* testnet consensus params stable.
* docs expected genesis hash matches config/test result if feasible.

==================================================
2. Seed node deployment profiles
================================

Create operator deployment profiles for seed nodes.

Add examples:

examples/testnet/seed-node.env
examples/testnet/public-node.env
examples/testnet/miner-node.env
examples/testnet/faucet-node.env
examples/testnet/service-node.env

Seed node env should include:

* IDR_DATADIR
* IDR_NETWORK=testnet
* IDR_RPC_ADDR=:9011 or :8811
* IDR_P2P_ADDR=:10011 or :9811
* IDR_ADVERTISE_P2P=http://<PUBLIC_HOST>:10011
* IDR_PUBLIC_RPC=true
* IDR_ENABLE_MINER_RPC=true or false depending recommended role
* IDR_ENABLE_FAUCET_RPC=false by default
* IDR_ENABLE_SERVICE_RPC=false by default
* IDR_SEED_PEERS optional

Do not include private keys.
Do not include real wallet addresses unless placeholder.

Add docs explaining:

* seed node is not trusted authority.
* all peers still validate network id and genesis.
* bad/offline seed should not crash node.
* firewall must expose P2P port.
* RPC public exposure must be read-only safe.

==================================================
3. Deployment runbook
=====================

Add or update:

docs/DeployTestnet.md

Include Linux/VPS deployment using release artifact:

1. Download release archive.
2. Verify SHA256.
3. Extract to `/opt/deskachain`.
4. Create system user if desired.
5. Create datadir:
   `/var/lib/deskachain/testnet`
6. Init testnet.
7. Configure env:
   `/etc/deskachain/testnet.env`
8. Install systemd unit.
9. Start node.
10. Check logs.
11. Check health/chain info.
12. Mine from a separate wallet datadir if miner RPC enabled.
13. Add seed peer to another node.

Commands should support Linux binary workflow:

./deskachain version
./deskachain --datadir /var/lib/deskachain/testnet --network testnet init
./deskachain --datadir /var/lib/deskachain/testnet node start ...

Systemd example:

* use installed binary path.
* use env file.
* restart always.
* LimitNOFILE 65535.
* KillSignal SIGINT.
* TimeoutStopSec 30.

Add Windows deployment notes:

* extract zip.
* run PowerShell.
* init datadir.
* start public node.
* create separate miner wallet datadir.
* mine once.

==================================================
4. Preflight checklist
======================

Add a preflight checklist doc and optionally a CLI command if simple.

Docs checklist:

docs/Preflight.md

Checklist:

* binary version correct.
* checksum verified.
* network is testnet.
* genesis hash matches expected.
* datadir is clean or intentionally reused.
* wallet/admin RPC not exposed publicly.
* faucet RPC disabled unless intentionally operating faucet.
* service RPC disabled unless intentionally operating service node endpoint.
* miner RPC enabled only if node accepts public mining.
* advertised P2P URL reachable.
* P2P port open.
* seed peers configured.
* chain validate passes.
* `/health` reachable.
* `chain info` shows expected network and chain id.
* testnet IDR has no monetary value.
* mainnet not available.

Optional CLI command:

deskachain --datadir <dir> node preflight

If implemented, it should check:

* datadir initialized.
* network metadata.
* genesis hash.
* public RPC safety config.
* advertised P2P present if public.
* seed peers parse.
* chain validate if chain exists.

Do not over-engineer. Docs-only checklist is acceptable if CLI command would be too big.

==================================================
5. Seed peer registry candidate
===============================

Prepare a seed peer registry file that can be edited before real deployment.

Add:

config/testnet-seeds.example.txt

Content:

# DesKaChain public testnet seed peers

# Replace placeholders with real public seed nodes before publishing.

# Example:

# http://seed1.example.org:9811

# http://seed2.example.org:9811

Do not hardcode fake public seed domains as active defaults.
Keep placeholders commented.

Update docs to show:

* `--seed-file config/testnet-seeds.txt`
* `--seed-peer http://<HOST>:<P2P_PORT>`
* `IDR_SEED_PEERS=http://host1:port,http://host2:port`

==================================================
6. Public faucet operator prep
==============================

Prepare docs for an optional controlled testnet faucet node.

Update docs/Faucet.md and docs/DeployTestnet.md:

* faucet is disabled by default.
* faucet RPC should only be enabled deliberately.
* faucet wallet must have mature balance.
* recommended faucet amount for service collateral testing: 1000 IDR only for controlled dev/testnet.
* max per address and min interval should be configured.
* faucet uses normal tx, no silent mint.
* faucet state must be backed up if operator wants rate-limit continuity.
* testnet IDR has no monetary value.

Add config example:

examples/testnet/faucet-node.env

Do not include private key or real faucet address.

==================================================
7. Service node operator prep
=============================

Update docs/ServiceNode.md and docs/DeployTestnet.md:

* service node requires testnet IDR collateral for eligibility.
* min service stake: 1000 IDR.
* service points are simulation-only and not spendable.
* staking is collateral-only, no APY.
* service agent should use its own state file.
* service owner wallet should be backed up.

Add config example:

examples/testnet/service-agent.env

Do not include private key.

==================================================
8. Release artifact verification
================================

Add docs section:
“Verifying release artifacts”

Include:

Linux:
sha256sum -c SHA256SUMS.txt

Windows:
Get-FileHash .\deskachain-v0.3.8-testnet-windows-amd64.zip -Algorithm SHA256

Explain:

* checksum verifies download integrity.
* checksum does not replace source review.
* only download from official repository/workflow artifacts.

==================================================
9. Safety and warning banner
============================

Ensure docs and startup summary clearly state:

* Testnet only.
* Testnet IDR has no monetary value.
* Mainnet not available.
* Do not expose wallet/admin RPC.
* Back up wallets.
* Seed peers are not trusted authorities.
* Public RPC is read-only unless explicit miner/service/faucet toggles are enabled.

If startup summary already prints modes, verify it includes enough context.

Add test if easy:

* public node startup summary includes network and access flags.

==================================================
10. Tests
=========

Required:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/config -run "Genesis|Profile|Testnet|Seed|Deploy" -count=1 -v
go test ./node/internal/cli -run "Preflight|Version|Public|Seed|Genesis|Operator" -count=1 -v
go test ./node/internal/rpc -run "Public|Health|ChainInfo|Faucet|Service|Miner" -count=1 -v
go test ./node/internal/p2p -run "Seed|Peer|NetworkMismatch|GenesisMismatch" -count=1 -v

If docs tests exist, update them.
If no matching tests for Deploy docs, okay.

==================================================
11. Manual validation
=====================

Using release binary from Phase 3.7 or rebuilt binary:

Clean:

deskachain --datadir ./testdata/deploy_seed dev reset --yes

Init:

deskachain --datadir ./testdata/deploy_seed --network testnet init

Start seed candidate:

deskachain --datadir ./testdata/deploy_seed node start --rpc :9311 --p2p :10311 --advertise-p2p http://127.0.0.1:10311 --public-rpc --enable-miner-rpc

Check:

deskachain --rpc-url http://127.0.0.1:9311 version
deskachain --rpc-url http://127.0.0.1:9311 chain info
deskachain --rpc-url http://127.0.0.1:9311 chain validate
deskachain --rpc-url http://127.0.0.1:9311 wallet new

Expected:

* version works locally if command supports remote or local version.
* chain info works.
* chain validate works.
* wallet new rejected in public RPC mode.

Start second node with seed peer:

deskachain --datadir ./testdata/deploy_node_b --network testnet init

deskachain --datadir ./testdata/deploy_node_b node start --rpc :9312 --p2p :10312 --advertise-p2p http://127.0.0.1:10312 --public-rpc --enable-miner-rpc --seed-peer http://127.0.0.1:10311/

Check B:
deskachain --rpc-url http://127.0.0.1:9312 peer list
deskachain --rpc-url http://127.0.0.1:9312 peer check http://127.0.0.1:10311

Mine on A and ensure B imports/broadcasts:

deskachain --datadir ./testdata/deploy_miner --network testnet init
deskachain --datadir ./testdata/deploy_miner wallet new

idrminer --rpc-url http://127.0.0.1:9311 --address <ADDR> --threads 2 --once

Check B:
deskachain --rpc-url http://127.0.0.1:9312 chain info
deskachain --rpc-url http://127.0.0.1:9312 chain validate

Expected:

* B height updates to 1.
* chain valid.
* chain info and validate agree.

==================================================
12. Done criteria
=================

Phase 3.9 valid if:

* testnet genesis candidate is documented.
* deploy runbook exists.
* preflight checklist exists.
* seed node config examples exist.
* faucet/service operator examples exist.
* seed peer registry example exists.
* docs include checksum verification.
* public safety warnings are clear.
* all tests pass.
* manual seed candidate deployment works with two nodes.
* node B imports block from seed/broadcast and chain info remains consistent.
