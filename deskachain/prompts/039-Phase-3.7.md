Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 3.6 sudah valid full.
* Phase 3.6.1 Chain Info Snapshot Consistency After P2P Import valid.
* go work sync pass.
* go test ./node/... pass.
* go test -count=1 ./node/... pass.
* Public testnet node mode valid.
* Public RPC safety valid.
* Seed peer bootstrap registry valid.
* Peer normalization/dedupe valid.
* P2P import chain info snapshot fixed.
* Faucet testnet valid.
* Faucet -> stake 1000 -> service eligible E2E valid.
* Staking remains collateral-only.
* Service points remain simulation-only.
* PoW remains the only block-production consensus.
* Testnet DKC has no monetary value.
* Mainnet does not exist yet.

Patch name:
DesKaChain Phase 3.7 — Public Testnet Release Build & Binary Packaging

Goal:
Prepare DesKaChain for public testnet binary distribution.

This phase should make it possible to build and package:

* deskachain CLI/node binary,
* dkcminer binary,
* dkcservice binary,
  for Windows and Linux, with version metadata, checksums, release archives, and quickstart docs.

Non-goals:

* Do not launch mainnet.
* Do not change testnet genesis.
* Do not change localnet genesis.
* Do not change consensus.
* Do not implement installer/updater.
* Do not implement explorer.
* Do not implement Stratum/pool mining.
* Do not implement staking rewards/APY.
* Do not implement slashing/PoS.
* Do not make service points spendable.
* Do not expose wallet/admin RPC publicly.
* Do not add real-price/profit language.
* Do not include private keys, wallet files, runtime datadirs, or faucet state in release artifacts.

==================================================

1. Version command
   ==================================================

Add or finalize version output for all binaries.

Commands:

deskachain version
dkcminer --version
dkcservice --version

or equivalent consistent style if project already has one.

Version output should include:

* app name
* version
* git commit
* build date
* Go version
* OS/ARCH
* network support: localnet/testnet
* warning: mainnet not available, if appropriate

Example:

DesKaChain
version: v0.3.7-testnet
commit: <commit>
built: <date>
go: go1.xx
os/arch: windows/amd64
networks: localnet,testnet
mainnet: not available

Use ldflags variables if possible:

* Version
* Commit
* BuildDate

Add tests:

* version command prints version.
* missing ldflags falls back to dev/unknown safely.
* miner/service version does not require node/rpc.

==================================================
2. Build scripts
================

Add build scripts for local release builds.

Recommended structure:

scripts/build.ps1
scripts/build.sh

Build outputs:

dist/
windows-amd64/
deskachain.exe
dkcminer.exe
dkcservice.exe
linux-amd64/
deskachain
dkcminer
dkcservice

Support at least:

* windows/amd64
* linux/amd64

Optional if easy:

* linux/arm64

PowerShell script should work on Windows.

Shell script should work on Linux/macOS style shell.

Scripts should:

* clean/create dist directory,
* run tests unless `-SkipTests` / `SKIP_TESTS=1`,
* build binaries with ldflags,
* include version/commit/date,
* print output paths,
* fail fast on errors.

PowerShell example:

./scripts/build.ps1 -Version v0.3.7-testnet

Linux example:

./scripts/build.sh v0.3.7-testnet

Do not require external tools beyond Go and standard shell/PowerShell for core build.

Add docs explaining:

* Go version requirement.
* how to run build.
* where binaries are located.

==================================================
3. Release packaging scripts
============================

Add release packaging script.

Recommended:

scripts/package.ps1
scripts/package.sh

Output examples:

dist/releases/
deskachain-v0.3.7-testnet-windows-amd64.zip
deskachain-v0.3.7-testnet-linux-amd64.tar.gz
SHA256SUMS.txt

Each archive should include:

* deskachain binary
* dkcminer binary
* dkcservice binary
* README or QUICKSTART
* LICENSE if exists
* docs links or copied minimal docs if desired
* example configs/systemd files if already present

Do not include:

* testdata/
* wallets/
* chain data/
* faucet_state.json
* dkcservice-state.json
* private keys
* .env files with secrets
* .git

Checksum:

* generate SHA256 checksums.
* PowerShell should use Get-FileHash.
* Shell should use sha256sum or shasum fallback.

Add tests or script validation if practical:

* package archive exists.
* checksum file exists.
* runtime state files are not packaged.
* expected binaries are included.

==================================================
4. Build metadata package
=========================

If not already present, add small internal build info package.

Example:

internal/version

or existing location.

Fields:

* AppName
* Version
* Commit
* BuildDate

Functions:

* Info()
* String()
* JSON maybe optional

Avoid circular imports.

All three binaries should use same build metadata source.

==================================================
5. Binary smoke tests
=====================

Add smoke tests that can run without starting long-lived services.

Examples:

* `deskachain version`
* `deskachain --help`
* `dkcminer --version`
* `dkcminer --help`
* `dkcservice --version`
* `dkcservice --help`

If tests run compiled binary, keep them fast and cross-platform.

If compiled-binary tests are too much, add CLI command tests only.

==================================================
6. Release quickstart docs
==========================

Add or update:

docs/Release.md

Include:

* install from archive.
* verify checksum.
* run `deskachain version`.
* init testnet.
* start public testnet node.
* create miner wallet in separate datadir.
* mine using dkcminer.
* run dkcservice.
* safe public RPC notes.
* seed peer usage.
* systemd usage for Linux.
* Windows PowerShell usage.

Example binary workflow:

Windows:

.\deskachain.exe --datadir .\data\testnet --network testnet init

.\deskachain.exe --datadir .\data\testnet node start --rpc :9011 --p2p :10011 --advertise-p2p http://127.0.0.1:10011 --public-rpc --enable-miner-rpc

.\deskachain.exe --datadir .\data\miner --network testnet init
.\deskachain.exe --datadir .\data\miner wallet new

.\dkcminer.exe --rpc-url http://127.0.0.1:9011 --address <ADDR> --threads 2 --once

Linux:

./deskachain --datadir ./data/testnet --network testnet init

./deskachain --datadir ./data/testnet node start --rpc :9011 --p2p :10011 --advertise-p2p http://127.0.0.1:10011 --public-rpc --enable-miner-rpc

Safety notes:

* Testnet DKC has no monetary value.
* Mainnet is not available.
* Do not expose wallet/admin RPC publicly.
* Use separate datadir for miner reward wallet if node datadir is locked.
* Back up wallet files.
* Seed peers are not trusted authorities; network/genesis validation still applies.

==================================================
7. README updates
=================

Update README.md and README-ID.md:

* Add Release Build section.
* Add binary names:

  * deskachain
  * dkcminer
  * dkcservice
* Add build commands.
* Add package commands.
* Link docs/Release.md and docs/Operator.md.
* Mention Phase 3.7.

Keep wording conservative:

* “public testnet preparation”
* not “mainnet launch”
* not “investment/profit/mining income”.

==================================================
8. .gitignore and release safety
================================

Ensure .gitignore excludes:

* dist/
* release artifacts if generated locally, unless project wants scripts only
* runtime state:

  * testdata/
  * faucet_state.json
  * dkcservice-state.json
  * peer store runtime files if needed
  * wallet runtime data if under examples accidentally

But do not ignore source examples/docs.

Add release safety check in packaging script:

* refuse to package if archive source includes obvious runtime/private files.
* at minimum exclude them.

==================================================
9. Optional GitHub Actions
==========================

Optional only if simple:
Add GitHub Actions workflow:

.github/workflows/build.yml

It should:

* run go test ./node/...
* build windows/linux binaries.
* upload artifacts for manual workflow dispatch or tag.

Do not require secrets.
Do not auto-publish releases unless explicitly designed.

If added, keep it conservative:

* workflow_dispatch
* push to main maybe tests only
* tags maybe build artifacts

If this is too much, skip until next phase.

==================================================
10. Required tests after patch
==============================

Run:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/cli -run "Version|Build|Help|Release" -count=1 -v
go test ./node/internal/config -run "Profile|Testnet|Localnet" -count=1 -v

If version package has tests:

go test ./node/internal/version -count=1 -v

If packaging/build scripts have Go tests or script checks, run them too.

Manual build validation on Windows:

powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.3.7-testnet -SkipTests

Expected:

* dist/windows-amd64/deskachain.exe exists
* dist/windows-amd64/dkcminer.exe exists
* dist/windows-amd64/dkcservice.exe exists
* version commands work

Manual commands:

.\dist\windows-amd64\deskachain.exe version
.\dist\windows-amd64\dkcminer.exe --version
.\dist\windows-amd64\dkcservice.exe --version

Manual binary smoke:

.\dist\windows-amd64\deskachain.exe --datadir .\testdata\release_node dev reset --yes
.\dist\windows-amd64\deskachain.exe --datadir .\testdata\release_node --network testnet init
.\dist\windows-amd64\deskachain.exe --datadir .\testdata\release_wallet --network testnet init
.\dist\windows-amd64\deskachain.exe --datadir .\testdata\release_wallet wallet new

Start node:

.\dist\windows-amd64\deskachain.exe --datadir .\testdata\release_node node start --rpc :9211 --p2p :10211 --advertise-p2p http://127.0.0.1:10211 --public-rpc --enable-miner-rpc

Mine once:

.\dist\windows-amd64\dkcminer.exe --rpc-url http://127.0.0.1:9211 --address <ADDR> --threads 2 --once

Check:

.\dist\windows-amd64\deskachain.exe --rpc-url http://127.0.0.1:9211 chain info
.\dist\windows-amd64\deskachain.exe --rpc-url http://127.0.0.1:9211 chain validate

Expected:

* miner submit accepted.
* chain height 1.
* chain valid.

Package validation:

powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.3.7-testnet -SkipTests

Expected:

* release archive created.
* SHA256SUMS.txt created.
* archive does not include runtime datadir/private state.
* archive includes binaries and quickstart docs.

==================================================
11. Done criteria
=================

Phase 3.7 valid if:

* all tests pass.
* version command works for all binaries.
* Windows build script works.
* Linux build script exists and is reasonable.
* package script creates archive and checksum.
* release docs exist.
* README links release docs.
* binary smoke test can init testnet, start node, mine once, and validate chain.
* release artifacts exclude private/runtime state.
* public testnet safety from prior phases remains intact.
