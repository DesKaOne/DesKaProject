Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 4.6 sudah valid full.
* Phase 4.6 Public Testnet Release Candidate & Operator Checklist valid full.
* Version RC sudah valid:

  * v0.4.6-testnet-rc1
* Smoke test binary sudah valid:

  * deskachain version
  * init testnet
  * start node
  * mine one block
  * chain info
  * chain validate
  * explorer status/UI
* Release artifacts sudah dibuat:

  * deskachain-v0.4.6-testnet-rc1-windows-amd64.zip
  * deskachain-v0.4.6-testnet-rc1-linux-amd64.tar.gz
  * deskachain-v0.4.6-testnet-rc1-linux-arm64.tar.gz
  * SHA256SUMS.txt
* Docs sudah ada:

  * Release.md
  * ReleaseNotes-v0.4.6-testnet-rc1.md
  * OperatorChecklist.md
  * PublicTestnetQuickstart.md
  * Explorer.md
  * MultiHostTestnet.md
  * DeployTestnet.md
* Public RPC safety tetap aman.
* Explorer API/UI read-only.
* Faucet/stake/service multi-host sudah valid.
* Testnet IDR has no monetary value.
* Mainnet does not exist yet.
* PoW remains the only block-production consensus.
* Staking remains collateral-only.
* Service points remain simulation-only.

Patch name:
DesKaChain Phase 4.7 — Public Testnet RC1 GitHub Release & External Tester Onboarding

Goal:
Prepare and publish DesKaChain Public Testnet RC1 for limited external testers/operators.

This phase should produce:

* GitHub Release checklist,
* release description,
* artifact attachment verification,
* seed node/operator publishing guide,
* external tester onboarding guide,
* bug report template,
* feedback form/checklist,
* public announcement draft,
* tester safety warnings,
* post-release monitoring checklist.

This phase is about release/onboarding readiness.
Do not change consensus or testnet genesis.

Non-goals:

* Do not launch mainnet.
* Do not give testnet IDR monetary value.
* Do not promise price/profit/rewards.
* Do not add staking APY.
* Do not make service points spendable.
* Do not add wallet web UI.
* Do not expose wallet/admin RPC publicly.
* Do not add explorer write actions.
* Do not change block/tx format.
* Do not require centralized trust.
* Do not auto-publish real seed URLs unless operator chooses.

==================================================

1. GitHub Release RC1 checklist
   ==================================================

Add:

docs/GitHubReleaseChecklist.md

Checklist for publishing:

A. Before release

* Confirm branch clean.
* Confirm tag exists:

  * v0.4.6-testnet-rc1
* Confirm CI green.
* Confirm release artifact workflow green.
* Confirm smoke test passed.
* Confirm SHA256SUMS.txt exists.
* Confirm release archives contain expected docs/examples.
* Confirm no datadir/wallet/private state in archives.
* Confirm docs warnings are present.

B. Artifacts to attach

* deskachain-v0.4.6-testnet-rc1-windows-amd64.zip
* deskachain-v0.4.6-testnet-rc1-linux-amd64.tar.gz
* deskachain-v0.4.6-testnet-rc1-linux-arm64.tar.gz
* SHA256SUMS.txt

C. Release title
DesKaChain Public Testnet RC1 — v0.4.6-testnet-rc1

D. Release type

* Pre-release: true
* Latest release: false, if GitHub allows.
* Mark clearly as testnet RC.

E. Post publish

* Download artifacts from GitHub Release.
* Verify SHA256 from downloaded artifacts.
* Run smoke script from downloaded artifact if practical.
* Open Explorer UI.
* Confirm quickstart docs render correctly.

==================================================
2. GitHub Release notes
=======================

Add:

docs/GitHubRelease-v0.4.6-testnet-rc1.md

This should be copy-paste ready for GitHub Release body.

Required sections:

# DesKaChain Public Testnet RC1 — v0.4.6-testnet-rc1

## Status

Public testnet release candidate for limited testing.

## Important warnings

* Testnet IDR has no monetary value.
* Mainnet is not available.
* Do not treat testnet IDR as investment.
* No mining income/profit promise.
* Staking is collateral-only.
* Service points are simulation-only and not spendable IDR.
* Testnet may reset.
* Back up wallet files.
* Do not expose wallet/admin RPC publicly.

## What is included

* deskachain node/CLI.
* idrminer CPU miner.
* idrservice service-node simulation agent.
* Public testnet profile.
* P2P seed peer sync.
* Public read-only RPC mode.
* Explorer API.
* Explorer Web UI.
* Faucet operator mode.
* Staking collateral.
* Service node simulation.
* Windows/Linux release artifacts.
* SHA256 checksums.

## Downloads

List expected artifacts.

## Verify checksums

Windows:
Get-FileHash .\deskachain-v0.4.6-testnet-rc1-windows-amd64.zip -Algorithm SHA256

Linux:
sha256sum -c SHA256SUMS.txt

## Quick start

Link to:

* PublicTestnetQuickstart.md
* OperatorChecklist.md
* Explorer.md

## Seed peers

Use placeholder:
http://<SEED_HOST>:10311

Say:
Operator-published seeds will be announced separately.
Seed peers are not trusted authorities. Nodes validate network ID and genesis hash.

## Known limitations

* No mainnet.
* No persistent explorer database yet.
* Explorer currently uses simple scan mode.
* No wallet web UI.
* No Stratum/pool mining.
* No mobile wallet app yet.
* Faucet must be manually operated and funded.
* Service node scoring is simulation-only.
* Staking is collateral-only.

## Feedback requested

Ask testers to report:

* OS/architecture.
* binary version.
* node mode: LAN/Tailscale/VPS.
* whether node synced.
* explorer UI issues.
* mining issues.
* faucet issues.
* service node issues.
* logs and commands used.

==================================================
3. External tester onboarding guide
===================================

Add:

docs/TesterOnboarding.md

Audience:
Non-core testers who want to try public testnet safely.

Sections:

A. What you need

* Windows x64 or Linux x64/arm64.
* Basic terminal.
* Network connection.
* Optional Tailscale or LAN/VPS.

B. What this is

* Public testnet RC.
* Testnet IDR has no monetary value.
* Mainnet unavailable.

C. Download and verify

* Download artifact.
* Verify checksum.
* Run version.

D. Join as read-only/explorer tester

* Run node.
* Add seed peer.
* Open explorer UI.
* Check blocks.

E. Join as miner tester

* Create miner wallet.
* Mine one block.
* Check explorer/address page.

F. Join as faucet/staking/service tester

* Request faucet if faucet URL is provided.
* Lock stake only if you understand this is testnet.
* Run service agent.
* Check service score.
* Reminder: service points are simulation-only.

G. What to report

* Sync issues.
* Wrong height/tip.
* Explorer UI errors.
* Crash/panic.
* Firewall issues.
* Faucet rate-limit issues.
* Service agent errors.

H. What not to do

* Do not expose wallet/admin RPC publicly.
* Do not use important/private machine without understanding testnet risk.
* Do not put real funds/private keys into DesKaChain testnet.
* Do not trust random binaries.

==================================================
4. Bug report template
======================

Add GitHub issue template:

.github/ISSUE_TEMPLATE/testnet-bug-report.yml

or markdown:

.github/ISSUE_TEMPLATE/testnet-bug-report.md

Fields:

* Version:
* OS:
* Architecture:
* Binary used:
* Node role:

  * seed
  * peer
  * miner
  * faucet
  * service
  * explorer-only
* Network mode:

  * LAN
  * Tailscale
  * VPS/public
  * local only
* Command used:
* Expected behavior:
* Actual behavior:
* Logs:
* Chain info:
* Peer list:
* Explorer URL if relevant:
* Screenshots if relevant:
* Did wallet/admin RPC remain private? yes/no/unknown
* Additional context:

Add warning:
Do not paste private keys, wallet files, or secrets.

==================================================
5. Feedback checklist
=====================

Add:

docs/TestnetFeedbackChecklist.md

Checklist for testers:

Node sync:

* Node started.
* Seed peer reachable.
* Peer list active.
* Chain info height matches seed.
* Chain validate passes.

Mining:

* Miner wallet created.
* Miner submits block.
* Explorer block list updates.
* Address page shows coinbase.

Explorer:

* Dashboard loads.
* Blocks page works.
* Search works.
* Address page works.
* Tx page works.
* Mobile view usable.

Faucet:

* Request succeeds.
* Rate limit works.
* Faucet tx appears.
* No supply mint beyond coinbase.

Service:

* Stake lock works.
* Service register works.
* idrservice --once works.
* Service eligible shown.
* Points clearly simulation-only.

Safety:

* Wallet/admin RPC not public.
* Testnet warnings visible.
* No write buttons in explorer UI.

==================================================
6. Public announcement draft
============================

Add:

docs/Announcement-v0.4.6-testnet-rc1.md

Write a short announcement draft.

Tone:

* Clear, modest, no hype.
* Testnet only.
* No monetary value.
* Looking for technical testers.

Include:

* What DesKaChain is.
* What RC1 includes.
* How to download.
* How to verify checksum.
* How to join via seed peer.
* How to open explorer.
* What feedback is needed.
* Safety warnings.

Do not include profit language.

==================================================
7. Seed operator publishing guide
=================================

Add:

docs/SeedOperatorPublish.md

Include:

* How to choose public seed URL.
* LAN/Tailscale/VPS examples.
* Firewall ports.
* P2P advertise address.
* RPC exposure recommendation:

  * P2P public OK.
  * RPC preferably local/trusted only.
  * If public RPC enabled, wallet/admin disabled.
* How to publish seed peer:

  * http://<host>:10311
* How to verify from external host:

  * peer check
  * chain info
  * explorer status if RPC is reachable.
* How to rotate seed URL.
* How to remove bad seed.

==================================================
8. Release workflow polish
==========================

If useful, update GitHub Actions release workflow docs/config:

* input version default to v0.4.6-testnet-rc1 or generic.
* upload artifacts.
* upload SHA256SUMS.
* optional smoke test.
* optional publish release disabled by default.

Do not auto-publish without explicit manual input.

Optional:
Add `.github/release.yml` if using GitHub generated release notes categories.
Skip if not needed.

==================================================
9. README updates
=================

Update:

* README.md
* README-ID.md

Add:

* Public Testnet RC1 section.
* Link:

  * ReleaseNotes-v0.4.6-testnet-rc1.md
  * PublicTestnetQuickstart.md
  * TesterOnboarding.md
  * OperatorChecklist.md
  * GitHubReleaseChecklist.md
  * TestnetFeedbackChecklist.md
* Mention:

  * Explorer UI path `/explorer-ui/`
  * testnet no monetary value
  * mainnet not available

==================================================
10. Tests and validation
========================

Run:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/rpc -run "Explorer|Public|Health|UI|Static|Wallet|Admin|Release|Smoke" -count=1 -v

go test ./node/internal/cli -run "Version|Release|Public|Wallet|Admin|Seed|Smoke" -count=1 -v

go test ./node/internal/config -run "Profile|Testnet|Seed|Genesis" -count=1 -v

go test ./node/internal/p2p -run "Peer|Sync|Seed|NetworkMismatch|GenesisMismatch" -count=1 -v

Run smoke:
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-test.ps1 -BinDir .\dist\windows-amd64

Build/package:
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.6-testnet-rc1 -SkipTests
powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.4.6-testnet-rc1 -SkipTests

If Bash is available:
bash ./scripts/smoke-test.sh ./dist/linux-amd64
bash ./scripts/build.sh v0.4.6-testnet-rc1 --skip-tests
bash ./scripts/package.sh v0.4.6-testnet-rc1 --skip-tests

==================================================
11. Manual release validation
=============================

Manual validation after GitHub Release draft:

1. Create GitHub pre-release draft.
2. Attach artifacts and SHA256SUMS.
3. Paste release notes.
4. Mark pre-release.
5. Download artifact from draft/release.
6. Verify checksum.
7. Extract clean folder.
8. Run:

   * deskachain version
   * init testnet
   * start node
   * open /explorer-ui/
   * mine once
   * chain validate
9. Confirm docs links work.
10. Confirm warnings are visible.
11. Confirm no private/runtime files in archive.

==================================================
12. Done criteria
=================

Phase 4.7 valid if:

* GitHub release checklist exists.
* GitHub release notes copy exists.
* Tester onboarding guide exists.
* Bug report template exists.
* Feedback checklist exists.
* Announcement draft exists.
* Seed operator publish guide exists.
* README/README-ID updated.
* all tests pass.
* smoke test passes.
* build/package still works.
* release artifacts are ready for GitHub Release.
* public safety warnings are clear.
* no monetary value/profit wording.
* external tester can follow docs safely.
