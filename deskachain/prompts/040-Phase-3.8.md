Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 3.7 sudah valid full.
* Phase 3.7 Public Testnet Release Build & Binary Packaging valid full.
* go work sync pass.
* go test ./node/... pass.
* go test -count=1 ./node/... pass.
* Windows binary build valid.
* Linux amd64 binary build valid.
* Linux arm64 binary build valid.
* Version metadata valid:

  * deskachain version
  * dkcminer --version
  * dkcservice --version
* Binary smoke test valid:

  * init testnet
  * start node
  * create separate miner wallet
  * mine once
  * chain info height 1
  * chain validate pass
* Release archives valid:

  * deskachain-v0.3.7-testnet-windows-amd64.zip
  * deskachain-v0.3.7-testnet-linux-amd64.tar.gz
  * deskachain-v0.3.7-testnet-linux-arm64.tar.gz
  * SHA256SUMS.txt
* Public RPC safety valid.
* Seed peer bootstrap valid.
* Faucet/staking/service-node testnet E2E valid.
* Testnet DKC has no monetary value.
* Mainnet does not exist yet.

Patch name:
DesKaChain Phase 3.8 — GitHub Actions CI Release Artifacts

Goal:
Add GitHub Actions workflows for CI and release artifact builds.

This phase should make GitHub able to:

1. run tests automatically,
2. build Windows/Linux binaries,
3. package release archives,
4. generate SHA256SUMS.txt,
5. upload artifacts from workflow runs,
6. optionally prepare tag-based testnet release artifacts.

Non-goals:

* Do not launch mainnet.
* Do not publish mainnet release.
* Do not auto-publish GitHub Release unless workflow is manual or clearly guarded.
* Do not require secrets.
* Do not include private keys, wallets, datadirs, faucet state, service state, testdata, or runtime files in artifacts.
* Do not change consensus.
* Do not change testnet genesis.
* Do not change localnet genesis.
* Do not implement explorer.
* Do not implement Stratum/pool mining.
* Do not implement installer/updater.
* Do not add profit/investment wording.

==================================================

1. CI workflow
   ==================================================

Add:

.github/workflows/ci.yml

Triggers:

* pull_request
* push to main/master
* workflow_dispatch

Jobs:

1. test

   * runs-on: ubuntu-latest
   * checkout
   * setup-go
   * go version
   * go work sync
   * go test ./node/...
   * go test -count=1 ./node/...

Optional:

* go test race only for selected packages if not too slow.
* go vet ./node/... if project passes vet.
* gofmt check.

Recommended conservative commands:

go work sync
go test ./node/...
go test -count=1 ./node/...

If `go vet` currently fails because of existing style or generated code, do not force it yet. Add only if it passes.

==================================================
2. Release artifact workflow
============================

Add:

.github/workflows/release-artifacts.yml

Triggers:

* workflow_dispatch with input:

  * version, default v0.3.7-testnet
  * skip_tests, default false
* push tags:

  * "v*"

Behavior:

* checkout
* setup-go
* compute version:

  * if manual input, use input version.
  * if tag, use tag name.
* compute commit short SHA.
* build date UTC.
* run tests unless skip_tests true.
* run scripts/build.sh or equivalent.
* run scripts/package.sh or equivalent.
* upload dist/releases as artifact.

Artifact name example:

deskachain-${version}-release-artifacts

Do not require GitHub secrets.

Do not create GitHub Release automatically unless behind manual input like `publish_release: false` default.
For now, prefer artifact upload only.

==================================================
3. Make scripts CI-friendly
===========================

Ensure existing scripts can run in GitHub Actions:

PowerShell:
scripts/build.ps1
scripts/package.ps1

Shell:
scripts/build.sh
scripts/package.sh

For GitHub ubuntu runner, prefer using shell scripts:
./scripts/build.sh v0.3.7-testnet
./scripts/package.sh v0.3.7-testnet

Requirements:

* scripts must be executable or invoked via bash.
* scripts must create dist/
* scripts must build:

  * windows-amd64
  * linux-amd64
  * linux-arm64
* scripts must package:

  * windows zip
  * linux amd64 tar.gz
  * linux arm64 tar.gz
  * SHA256SUMS.txt
* scripts must not include runtime/private files.

If package.sh does not exist yet, add it using behavior equivalent to package.ps1.

If build.sh/package.sh already exist, verify and fix.

==================================================
4. Release artifact safety check
================================

Add safety guard in package scripts.

Package artifacts must not include:

* testdata/
* data/
* wallets/
* chain db files
* faucet_state.json
* dkcservice-state.json
* peer runtime store if generated
* private keys
* .env
* .git
* dist intermediate directories inside archive

Allowed in archive:

* binaries:

  * deskachain / deskachain.exe
  * dkcminer / dkcminer.exe
  * dkcservice / dkcservice.exe
* README.md or README-ID.md
* LICENSE if present
* docs/Release.md or QUICKSTART.md
* examples/systemd if present
* examples/testnet config templates if they contain no secrets

Add script check:

* after packaging, inspect archive file list.
* fail if forbidden patterns appear.
* print package contents summary.

For PowerShell zip:

* use Expand-Archive into temp dir or use .NET zip inspection if simple.
  For tar.gz:
* use tar -tzf.

If implementing cross-platform inspection is too much, at least ensure packaging source directory is clean and only includes explicitly copied files.

==================================================
5. Version metadata in CI
=========================

Ensure CI build injects:

* Version
* Commit
* BuildDate

Examples:

* version from tag/input.
* commit from `git rev-parse --short HEAD`.
* build date from UTC ISO string.

After build, workflow should run:

./dist/linux-amd64/deskachain version
./dist/linux-amd64/dkcminer --version
./dist/linux-amd64/dkcservice --version

For Windows binary on Linux runner, do not run `.exe` unless using Wine. Just build and package it.
Linux amd64 binary can be smoke-tested on ubuntu runner.

==================================================
6. Linux binary smoke test in workflow
======================================

Add a short smoke test using linux-amd64 binaries.

Commands:

* create temp datadir.
* init testnet.
* create separate miner wallet datadir.
* start node in background with public RPC + miner RPC.
* wait for /health.
* mine once with dkcminer.
* chain info.
* chain validate.
* stop node.

Example:

./dist/linux-amd64/deskachain --datadir ./ci-data/node --network testnet init

./dist/linux-amd64/deskachain --datadir ./ci-data/wallet --network testnet init
ADDR=$(./dist/linux-amd64/deskachain --datadir ./ci-data/wallet wallet new)

./dist/linux-amd64/deskachain --datadir ./ci-data/node node start --rpc :9311 --p2p :10311 --advertise-p2p http://127.0.0.1:10311 --public-rpc --enable-miner-rpc &
NODE_PID=$!

wait until:
./dist/linux-amd64/deskachain --rpc-url http://127.0.0.1:9311 chain info

./dist/linux-amd64/dkcminer --rpc-url http://127.0.0.1:9311 --address "$ADDR" --threads 2 --once

./dist/linux-amd64/deskachain --rpc-url http://127.0.0.1:9311 chain validate

kill $NODE_PID

Keep timeout reasonable.
Ensure cleanup trap kills node if test fails.

==================================================
7. Checksums
============

Ensure workflow uploads:

* release archives
* SHA256SUMS.txt

Ensure SHA256SUMS.txt includes all archives:

* windows-amd64 zip
* linux-amd64 tar.gz
* linux-arm64 tar.gz

Add docs for verifying checksum:

Linux:
sha256sum -c SHA256SUMS.txt

Windows:
Get-FileHash .\deskachain-v0.3.7-testnet-windows-amd64.zip -Algorithm SHA256

==================================================
8. README and docs update
=========================

Update:

docs/Release.md
docs/Operator.md
README.md
README-ID.md

Add section:

* GitHub Actions CI
* How to manually run release artifact workflow
* How to download artifacts from workflow
* How to verify SHA256SUMS
* What artifacts contain
* What artifacts intentionally exclude

Make clear:

* This is public testnet preparation.
* Testnet DKC has no monetary value.
* Mainnet is not available.
* Do not expose wallet/admin RPC publicly.

==================================================
9. Optional badge
=================

Optional:
Add CI badge to README if repo path is known and stable.

If repo path is unknown or likely to change, skip badge.

==================================================
10. Required validation after patch
===================================

Local tests:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/cli -run "Version|Build|Help|Release" -count=1 -v
go test ./node/internal/version -count=1 -v

If no internal/version package exists, skip that package.

Local script validation on Windows:

powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.3.8-testnet -SkipTests

powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.3.8-testnet -SkipTests

Expected:

* dist/windows-amd64 binaries built.
* dist/linux-amd64 binaries built.
* dist/linux-arm64 binaries built.
* dist/releases archives created.
* SHA256SUMS.txt created.

If Bash is available locally, validate:

bash ./scripts/build.sh v0.3.8-testnet --skip-tests
bash ./scripts/package.sh v0.3.8-testnet --skip-tests

GitHub Actions validation:

* push branch.
* confirm ci.yml passes.
* run release-artifacts.yml manually with version v0.3.8-testnet.
* confirm release artifact uploaded.
* confirm archive/checksum names are correct.

==================================================
11. Done criteria
=================

Phase 3.8 valid if:

* CI workflow exists and passes.
* release artifact workflow exists.
* workflows do not require secrets.
* workflows run tests unless explicitly skipped.
* workflows build Windows/Linux binaries.
* workflows package archives and SHA256SUMS.
* linux-amd64 binary smoke test passes in workflow or local equivalent exists.
* artifacts exclude runtime/private files.
* version metadata is injected in CI.
* docs explain how to run/download/verify artifacts.
* all local tests pass.
