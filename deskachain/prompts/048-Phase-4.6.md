Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 4.5 sudah valid full.
* Phase 4.3 Explorer API valid full.
* Phase 4.4 Explorer Web UI MVP valid full.
* Phase 4.5 Explorer UX Polish + API Pagination/Search Hardening valid full.
* Public testnet multi-host sudah valid.
* Long-running restart/catch-up sudah valid.
* Faucet -> stake -> service eligible multi-host sudah valid.
* Explorer UI/API sudah public read-only.
* Release artifacts v0.4.5-testnet sudah berhasil dibuat.
* Public RPC safety tetap aman:

  * wallet_rpc=false
  * admin_rpc=false
  * faucet_rpc=false by default
  * service_rpc=false by default
* Testnet DKC has no monetary value.
* Mainnet does not exist yet.
* PoW remains the only block-production consensus.
* Staking remains collateral-only.
* Service points remain simulation-only.

Patch name:
DesKaChain Phase 4.6 — Public Testnet Release Candidate & Operator Checklist

Goal:
Prepare DesKaChain public testnet as a release candidate that can be shared with external testers/operators.

This phase should produce:

* release candidate versioning,
* release notes,
* known limitations,
* final operator checklist,
* smoke test scripts,
* artifact verification docs,
* public seed node checklist,
* safe testnet onboarding guide,
* final safety copy for public testers.

This phase should not change consensus or testnet genesis.

Non-goals:

* Do not launch mainnet.
* Do not change testnet genesis.
* Do not change localnet genesis.
* Do not change consensus.
* Do not introduce a new coin economics model.
* Do not promise price/profit/rewards.
* Do not add staking APY.
* Do not make service points spendable.
* Do not add wallet web UI.
* Do not expose wallet/admin RPC publicly.
* Do not add explorer write actions.
* Do not implement production explorer indexer database yet.
* Do not implement Stratum/pool mining.
* Do not add installer/updater unless trivial.

==================================================

1. Version and release candidate naming
   ==================================================

Prepare version:

v0.4.6-testnet-rc1

Ensure version output can show RC version from build script:

deskachain version
dkcminer --version
dkcservice --version

Expected:

* version: v0.4.6-testnet-rc1
* commit
* built time
* go version
* os/arch
* networks: localnet,testnet
* mainnet: not available

Do not hardcode RC version permanently in source if build ldflags already handle it.

Update docs/examples to reference RC version where appropriate.

==================================================
2. Release notes
================

Add:

docs/ReleaseNotes-v0.4.6-testnet-rc1.md

Include sections:

A. Summary

* Public testnet release candidate.
* Multi-host testnet validated.
* Explorer API and Web UI included.
* Faucet/staking/service simulation included.

B. Included binaries

* deskachain
* dkcminer
* dkcservice

C. Supported platforms

* windows-amd64
* linux-amd64
* linux-arm64

D. Features included

* Testnet/localnet profiles.
* PoW mining.
* Public RPC read-only mode.
* P2P seed peer sync.
* Faucet operator mode.
* Staking collateral.
* Service node simulation.
* Explorer API.
* Explorer Web UI.
* Release artifacts and checksums.

E. Safety warnings

* Testnet DKC has no monetary value.
* Mainnet is not available.
* No profit/reward promises.
* Do not expose wallet/admin RPC publicly.
* Back up wallet files.
* Use separate datadirs for roles.

F. Known limitations

* Simple scan explorer/indexer.
* No persistent explorer database yet.
* No explorer UI write actions.
* No mainnet.
* No Stratum/pool mining.
* No mobile/desktop wallet app yet.
* Service points are simulation-only.
* Staking is collateral-only, no APY.
* Public seed list may be operator-supplied.
* Faucet must be operated deliberately and funded manually.

G. Upgrade notes

* Existing testnet datadirs may be reused if same genesis/network.
* Operators should verify genesis hash.
* Operators should verify SHA256SUMS.
* If switching from older localnet/testnet test datadir, reset/reinit intentionally.

H. Smoke validation

* commands to verify binary, init, start node, mine block, open explorer.

==================================================
3. Operator checklist
=====================

Add:

docs/OperatorChecklist.md

Checklist sections:

A. Before running

* Download release artifact.
* Verify SHA256.
* Confirm binary version.
* Choose mode:

  * private local test,
  * LAN,
  * Tailscale,
  * public VPS.
* Choose datadir.
* Confirm testnet genesis.
* Confirm firewall plan.

B. Network setup

* P2P port reachable.
* RPC binding intentional.
* Advertise P2P address reachable.
* Seed peers configured.
* Wallet/admin RPC not public.

C. Node startup

* Init testnet.
* Start node.
* Check /health.
* Check /explorer/status.
* Check chain info.
* Check chain validate.
* Check peer list.

D. Mining

* Create separate miner wallet/datadir.
* Mine once.
* Confirm chain height increases.
* Confirm explorer block list updates.

E. Faucet operator

* Create faucet wallet.
* Mine mature funds.
* Start with faucet RPC explicitly.
* Configure amount/min interval/max per address.
* Confirm faucet state path.
* Confirm faucet does not expose wallet/admin.

F. Service node

* Confirm owner has 1000 DKC.
* Lock stake.
* Confirm active stake.
* Enable service RPC deliberately.
* Run dkcservice --once.
* Confirm service eligible.
* Confirm service points simulation-only.

G. Explorer

* Open /explorer-ui/.
* Test search by height/hash/tx/address.
* Confirm no write controls.
* Confirm public safety banner.

H. Restart/recovery

* Restart service/node.
* Confirm chain height/tip persists.
* Confirm peer reconnect.
* Confirm chain validate.

I. Before sharing with testers

* Publish seed peer URL.
* Publish checksum.
* Publish warnings.
* Publish known limitations.
* Do not publish private wallets/datadirs.

==================================================
4. Public tester quickstart
===========================

Add:

docs/PublicTestnetQuickstart.md

Audience: external tester/operator.

Keep it short and safe.

Include:

1. Download artifact.
2. Verify SHA256.
3. Run version.
4. Init testnet.
5. Start peer node using public seed.
6. Open explorer UI.
7. Optional mine once.
8. Optional faucet request if faucet URL is provided.
9. Safety warnings.

Example Linux:

tar -xzf deskachain-v0.4.6-testnet-rc1-linux-amd64.tar.gz
cd deskachain-v0.4.6-testnet-rc1-linux-amd64
sha256sum -c SHA256SUMS.txt
./deskachain version
./deskachain --datadir ./data/testnet --network testnet init
./deskachain --datadir ./data/testnet node start 
--rpc 127.0.0.1:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://<YOUR_REACHABLE_IP>:10311 
--public-rpc 
--seed-peer http://<SEED_HOST>:10311/

Example Windows PowerShell:
Expand archive.
.\deskachain.exe version
.\deskachain.exe --datadir .\data\testnet --network testnet init
.\deskachain.exe --datadir .\data\testnet node start ...

Warnings:

* Testnet only.
* Testnet DKC has no monetary value.
* Mainnet unavailable.
* No wallet/admin public exposure.

==================================================
5. Smoke scripts
================

Add scripts:

scripts/smoke-test.ps1
scripts/smoke-test.sh

Purpose:
Quickly validate a release binary locally.

Smoke script should:

* accept binary directory or default `dist/windows-amd64` / `dist/linux-amd64`
* create temporary datadir
* run `deskachain version`
* init testnet
* start node in background
* wait for `/health`
* create miner wallet/datadir
* mine one block with `dkcminer --once`
* call:

  * chain info
  * chain validate
  * /explorer/status
  * /explorer/blocks?limit=1
  * /explorer-ui/
* stop node cleanly
* cleanup temp datadir unless `--keep-data`

PowerShell example:

powershell -ExecutionPolicy Bypass -File .\scripts\smoke-test.ps1 -BinDir .\dist\windows-amd64

Bash example:

bash ./scripts/smoke-test.sh ./dist/linux-amd64

Tests:

* scripts have help output.
* scripts fail fast.
* scripts do not use production datadir.
* scripts do not require secrets.

==================================================
6. Release artifact verification
================================

Update:

docs/Release.md
docs/DeployTestnet.md
README.md
README-ID.md

Add explicit artifact verification:

Windows:
Get-FileHash .\deskachain-v0.4.6-testnet-rc1-windows-amd64.zip -Algorithm SHA256

Linux:
sha256sum -c SHA256SUMS.txt

Explain:

* checksum verifies download integrity.
* checksum does not replace source review.
* download only from official repo/workflow release artifact.
* do not run random binaries from unknown sources.

==================================================
7. Public seed configuration
============================

Update:

config/testnet-seeds.example.txt
docs/MultiHostTestnet.md
docs/PublicTestnetQuickstart.md

Keep real seed URLs configurable, not hardcoded unless operator explicitly chooses.

Use placeholders:

# Public testnet RC seed peers

# Replace with operator-published seeds.

# http://<SEED_HOST>:10311

If using Tailscale/LAN:

# http://100.101.251.7:10311

If using VPS:

# http://<VPS_PUBLIC_IP>:10311

Do not auto-connect to unknown seeds by default unless explicitly configured.

==================================================
8. Release archive content
==========================

Ensure package scripts include:

* binaries
* README.md
* README-ID.md
* QUICKSTART.md
* docs/Release.md
* docs/ReleaseNotes-v0.4.6-testnet-rc1.md
* docs/OperatorChecklist.md
* docs/PublicTestnetQuickstart.md
* docs/Explorer.md
* docs/MultiHostTestnet.md
* docs/DeployTestnet.md
* examples/systemd/*
* examples/testnet/*
* config/testnet-seeds.example.txt

Ensure package scripts exclude:

* dist intermediate data
* testdata
* data
* wallets
* private keys
* faucet state
* service state
* dkcservice-state.json
* peers runtime store
* .env with secrets
* .git

Add or keep package content printout.

==================================================
9. Release safety audit
=======================

Add a small release safety checklist in docs or script output.

Check:

* no runtime datadir in archive.
* no wallet files in archive.
* no faucet_state.json.
* no dkcservice-state.json.
* no private keys.
* no `.env` containing secrets.
* no wallet/admin public enablement in examples.
* example env files contain placeholders only.

If feasible, package script should scan archive contents and fail on forbidden patterns:

* private_key
* wallet
* faucet_state.json
* dkcservice-state.json
* testdata/
* data/
* .env if not explicitly examples with safe placeholders

Be careful not to fail because docs mention “private key” as warning text. Prefer archive path/name checks rather than content grep unless precise.

==================================================
10. GitHub Actions release candidate workflow
=============================================

Update existing release workflow if needed:

* allow manual input version `v0.4.6-testnet-rc1`
* build artifacts
* run smoke script on linux-amd64 artifact if feasible
* upload release archives and SHA256SUMS
* do not auto-publish GitHub Release unless explicitly manual guarded.

Optional:

* add workflow_dispatch input:

  * version
  * run_smoke
  * publish_release default false

If publish_release is implemented:

* default false.
* require tag.
* attach artifacts.
* release title includes “testnet RC”.
* release notes include warnings.

Do not require secrets.

==================================================
11. Docs safety language
========================

Ensure all public-facing docs say:

* Testnet DKC has no monetary value.
* Mainnet is not available.
* Do not treat testnet DKC as investment.
* No mining income/profit promise.
* Staking is collateral-only.
* Service points are simulation-only and not spendable.
* Do not expose wallet/admin RPC publicly.
* Back up wallets.
* Use at your own risk; testnet may reset.

==================================================
12. Tests
=========

Run:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/rpc -run "Explorer|Public|Health|UI|Static|Wallet|Admin|Release|Smoke" -count=1 -v

go test ./node/internal/cli -run "Version|Release|Public|Wallet|Admin|Seed|Smoke" -count=1 -v

go test ./node/internal/config -run "Profile|Testnet|Seed|Genesis" -count=1 -v

go test ./node/internal/p2p -run "Peer|Sync|Seed|NetworkMismatch|GenesisMismatch" -count=1 -v

If smoke scripts are testable:
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-test.ps1 -BinDir .\dist\windows-amd64

bash ./scripts/smoke-test.sh ./dist/linux-amd64

==================================================
13. Build/package validation
============================

Run:

powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.6-testnet-rc1 -SkipTests

powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.4.6-testnet-rc1 -SkipTests

Optional Bash:
bash ./scripts/build.sh v0.4.6-testnet-rc1 --skip-tests
bash ./scripts/package.sh v0.4.6-testnet-rc1 --skip-tests

Expected artifacts:

* deskachain-v0.4.6-testnet-rc1-windows-amd64.zip
* deskachain-v0.4.6-testnet-rc1-linux-amd64.tar.gz
* deskachain-v0.4.6-testnet-rc1-linux-arm64.tar.gz
* SHA256SUMS.txt

==================================================
14. Manual validation
=====================

Manual RC validation:

A. Fresh binary test

* Extract artifact to clean folder.
* Run version.
* Init testnet.
* Start public RPC node.
* Open explorer UI.
* Mine once.
* Confirm explorer updates.
* Confirm chain validate.

B. Multi-host smoke

* Host A seed node.
* Host B join with seed peer.
* Mine on one host.
* Confirm same height/tip on both.

C. Public safety

* Public wallet new rejected.
* Admin/dev write rejected.
* Explorer works.
* /health works.
* Faucet/service disabled unless explicitly enabled.

D. Release archive safety

* Inspect archive contents.
* Confirm no datadir/wallet/private state.

==================================================
15. Done criteria
=================

Phase 4.6 valid if:

* all tests pass.
* release notes exist.
* operator checklist exists.
* public testnet quickstart exists.
* smoke scripts exist and work.
* build/package v0.4.6-testnet-rc1 works.
* release archives and SHA256SUMS exist.
* release archives exclude runtime/private state.
* explorer UI works from release binary.
* public safety remains intact.
* docs clearly state testnet/no monetary value/mainnet unavailable.
* external tester/operator can follow docs to join testnet safely.
