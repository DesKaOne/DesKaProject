Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 4.7 sudah valid full.
* Phase 4.6 Public Testnet Release Candidate & Operator Checklist valid full.
* Phase 4.7 RC1 GitHub Release & External Tester Onboarding valid full.
* Version RC:

  * v0.4.6-testnet-rc1
* Release artifacts sudah siap:

  * deskachain-v0.4.6-testnet-rc1-windows-amd64.zip
  * deskachain-v0.4.6-testnet-rc1-linux-amd64.tar.gz
  * deskachain-v0.4.6-testnet-rc1-linux-arm64.tar.gz
  * SHA256SUMS.txt
* Release docs sudah siap:

  * GitHub release notes
  * tester onboarding
  * operator checklist
  * feedback checklist
  * seed operator guide
  * announcement draft
* Public RPC safety tetap aman.
* Explorer API/UI read-only.
* Testnet DKC has no monetary value.
* Mainnet does not exist yet.
* PoW remains the only block-production consensus.
* Staking remains collateral-only.
* Service points remain simulation-only.

Patch name:
DesKaChain Phase 4.8 — Public Testnet RC1 Post-Release Monitoring & Feedback Loop

Goal:
Add lightweight post-release monitoring, feedback tracking, issue triage docs, and RC2 planning workflow for Public Testnet RC1.

This phase should help operators/testers answer:

* Is the seed node reachable?
* Are peers syncing?
* Is the explorer alive?
* Are public RPC safety flags correct?
* Are testers reporting common issues?
* Which issues block RC2?
* What needs to be fixed before wider public testnet?

This phase is operational/docs/tooling only.
Do not change consensus or testnet genesis.

Non-goals:

* Do not launch mainnet.
* Do not change testnet genesis.
* Do not change localnet genesis.
* Do not change consensus.
* Do not change block/tx format.
* Do not promise price/profit/rewards.
* Do not give testnet DKC monetary value.
* Do not add staking APY.
* Do not make service points spendable.
* Do not expose wallet/admin RPC publicly.
* Do not add explorer write actions.
* Do not add centralized trust assumptions.
* Do not require external paid monitoring services.

==================================================

1. Post-release monitoring docs
   ==================================================

Add:

docs/PostReleaseMonitoring.md

Include sections:

A. Purpose

* Monitor RC1 seed/testnet health.
* Catch sync/explorer/RPC/miner/service issues early.
* Collect tester feedback.
* Prepare RC2 fixes.

B. What to monitor

* Seed P2P reachability.
* RPC `/health`.
* Explorer `/explorer/status`.
* Explorer UI `/explorer-ui/`.
* Current height.
* Tip hash.
* Peer count.
* Mempool count.
* Public RPC flags:

  * wallet_rpc must be false on public nodes.
  * admin_rpc must be false on public nodes.
  * public_rpc true when intentionally public.
* Chain validate result.
* Disk usage.
* Process uptime.
* Recent error logs.

C. What is not monitored

* No financial value.
* No mining profitability.
* No staking yield.
* No service point value.

D. Suggested monitoring cadence

* First 24 hours: every 1–2 hours.
* First week: at least daily.
* After stabilization: daily or as needed.

E. Escalation levels

* Info: cosmetic/UI/docs.
* Warning: one tester issue, recoverable sync issue, slow node.
* Critical: seed unreachable, chain validation fails, consensus mismatch, public wallet/admin exposure, reproducible panic.

==================================================
2. Health check script
======================

Add scripts:

scripts/testnet-health.ps1
scripts/testnet-health.sh

Purpose:
Check a running node or public seed endpoint.

PowerShell usage:

powershell -ExecutionPolicy Bypass -File .\scripts\testnet-health.ps1 -RpcUrl http://127.0.0.1:9311

Bash usage:

bash ./scripts/testnet-health.sh http://127.0.0.1:9311

Checks:

* GET `/health`
* GET `/explorer/status`
* GET `/explorer/blocks?limit=1`
* GET `/explorer-ui/`
* Optional:

  * chain info via CLI if BinDir is provided.
  * chain validate via CLI if BinDir and Datadir are provided.

Output:

* ok/fail summary.
* network.
* network_id.
* chain_id.
* height.
* tip_hash.
* peer_count.
* pending_tx_count.
* wallet_rpc/admin_rpc flags.
* explorer ok.
* warnings.

Fail conditions:

* `/health` unreachable.
* wrong network_id if expected provided.
* wrong chain_id if expected provided.
* wallet_rpc true on public check unless explicitly allowed.
* admin_rpc true on public check unless explicitly allowed.
* explorer status not ok.
* chain validate fails if checked.

Options:

* `-ExpectedNetwork testnet`
* `-ExpectedNetworkID dkc-testnet-1`
* `-ExpectedChainID 777101`
* `-AllowWalletRPC`
* `-AllowAdminRPC`
* `-Json`
* `-TimeoutSeconds 10`

Keep dependencies minimal:

* PowerShell built-in `Invoke-RestMethod`.
* Bash with `curl`; avoid requiring jq if possible, or make jq optional.

==================================================
3. Seed peer monitoring checklist
=================================

Add:

docs/SeedMonitoringChecklist.md

Include:

* Confirm seed process running.
* Confirm P2P port reachable from outside.
* Confirm peer check works from another node.
* Confirm seed advertises correct URL.
* Confirm network ID:

  * dkc-testnet-1
* Confirm chain ID:

  * 777101
* Confirm genesis hash:

  * db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4
* Confirm public RPC safety:

  * wallet/admin disabled.
* Confirm explorer UI if RPC intentionally public.
* Confirm logs have no repeated panic/errors.
* Confirm disk space OK.
* Confirm restart recovery OK.

Add examples:

* LAN
* Tailscale
* VPS/public

==================================================
4. Known issues tracker doc
===========================

Add:

docs/KnownIssues.md

Initial sections:

* RC1 known limitations.
* Open issues.
* Workarounds.
* Fixed in next RC.
* Not a bug / expected behavior.

Seed with expected known limitations:

* Mainnet is not available.
* Testnet DKC has no monetary value.
* Explorer uses simple scan mode.
* No persistent explorer database yet.
* No wallet web UI.
* No mobile app.
* No Stratum/pool mining.
* Service points are simulation-only.
* Staking is collateral-only.
* Faucet must be deliberately operated/funded.
* Public seed availability depends on operators.

Add template entry:

## ISSUE-YYYYMMDD-001 — Title

Status:
Severity:
Affected version:
Affected OS:
Summary:
Steps to reproduce:
Workaround:
Fix plan:
Linked GitHub issue:

==================================================
5. Issue triage guide
=====================

Add:

docs/IssueTriage.md

Include:

* Severity categories:

  * S0 security/safety
  * S1 consensus/data loss
  * S2 networking/sync
  * S3 explorer/UI
  * S4 docs/usability
* Labels:

  * testnet-rc1
  * bug
  * docs
  * explorer
  * p2p
  * mining
  * faucet
  * staking
  * service-node
  * safety
  * needs-repro
  * rc2-candidate
* Triage flow:

  1. Confirm version.
  2. Confirm network/genesis.
  3. Ask for command/logs.
  4. Reproduce locally.
  5. Classify severity.
  6. Add workaround if available.
  7. Mark RC2 candidate if needed.
* Safety rule:

  * Never ask testers to paste private keys or wallet files.
  * Redact IPs if needed.
  * Do not request secrets.
* Close rules:

  * duplicate.
  * unsupported old build.
  * wrong network/datadir.
  * fixed in newer RC.

==================================================
6. Feedback summary template
============================

Add:

docs/FeedbackSummaryTemplate.md

Template for daily/weekly RC1 feedback summary:

# DesKaChain RC1 Feedback Summary — YYYY-MM-DD

## Overall status

* Seed status:
* Current public height:
* Explorer status:
* Known critical issues:
* Recommended action:

## Tester reports

* Total reports:
* New bugs:
* Docs issues:
* Sync issues:
* Explorer issues:
* Mining issues:
* Faucet issues:
* Service/staking issues:

## Confirmed issues

| ID | Severity | Area | Summary | Status | RC2 candidate |
| -- | -------- | ---- | ------- | ------ | ------------- |

## Common user mistakes

* Wrong datadir.
* Localnet vs testnet mismatch.
* RPC bind/firewall issue.
* Wallet/admin exposed accidentally.
* Using old artifact.

## RC2 candidates

* ...

## No monetary value reminder

Testnet DKC has no monetary value. Mainnet is not available.

==================================================
7. RC2 planning doc
===================

Add:

docs/RC2Planning.md

Sections:

* RC2 goal.
* RC2 triggers:

  * consensus bug.
  * seed sync bug.
  * public RPC safety issue.
  * explorer usability blocker.
  * packaging/checksum issue.
  * serious docs/onboarding confusion.
* RC2 non-triggers:

  * cosmetic UI issue.
  * minor docs typo.
  * expected faucet rate limit.
  * expected coinbase maturity confusion if docs can clarify.
* Candidate fixes table:
  | Priority | Area | Issue | Fix | Test needed |
* RC2 release process:

  * patch.
  * tests.
  * smoke.
  * build/package.
  * checksums.
  * release notes.
  * pre-release.
* Version suggestion:

  * v0.4.6-testnet-rc2 if only RC fixes.
  * v0.4.7-testnet-rc1 if new features are added.

==================================================
8. Tester response snippets
===========================

Add:

docs/TesterResponseSnippets.md

Include copy-paste responses for:

* asking for version:

  * `deskachain version`
* asking for health:

  * `/health`
* asking for chain info:

  * `deskachain --rpc-url ... chain info`
* asking for peer list:

  * `deskachain --rpc-url ... peer list`
* asking for logs:

  * PowerShell terminal output.
  * systemd `journalctl -u deskachain-testnet -n 200 --no-pager`
* warning not to paste private keys.
* explaining testnet DKC no monetary value.
* explaining coinbase maturity.
* explaining localnet/testnet mismatch.
* explaining public RPC safety.

==================================================
9. README updates
=================

Update:

* README.md
* README-ID.md
* docs/PublicTestnetQuickstart.md
* docs/OperatorChecklist.md

Add links to:

* PostReleaseMonitoring.md
* SeedMonitoringChecklist.md
* KnownIssues.md
* IssueTriage.md
* FeedbackSummaryTemplate.md
* RC2Planning.md

Add short section:
“After installing RC1”

* run health check,
* verify explorer,
* report issues using template,
* do not expose wallet/admin RPC,
* remember testnet DKC has no monetary value.

==================================================
10. Optional GitHub issue labels docs
=====================================

Add:

.github/labels.yml

or docs-only:

docs/GitHubLabels.md

Suggested labels:

* testnet-rc1
* rc2-candidate
* needs-repro
* safety
* p2p
* explorer
* mining
* faucet
* staking
* service-node
* docs
* packaging

Do not require automation if not currently used.

==================================================
11. Tests and validation
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

Run health check:
powershell -ExecutionPolicy Bypass -File .\scripts\testnet-health.ps1 -RpcUrl http://127.0.0.1:9311

Build/package:
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.6-testnet-rc1 -SkipTests
powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.4.6-testnet-rc1 -SkipTests

==================================================
12. Manual validation
=====================

Manual post-release style validation:

1. Start RC1 node.
2. Run `scripts/testnet-health.ps1`.
3. Confirm output shows:

   * network testnet.
   * network_id dkc-testnet-1.
   * chain_id 777101.
   * wallet_rpc false.
   * admin_rpc false.
   * explorer status ok.
4. Open explorer UI.
5. Mine one block.
6. Re-run health check.
7. Confirm height changed.
8. Simulate tester bug report using issue template.
9. Confirm docs contain no profit/monetary value claims.
10. Confirm README links work.

==================================================
13. Done criteria
=================

Phase 4.8 valid if:

* post-release monitoring docs exist.
* seed monitoring checklist exists.
* known issues doc exists.
* issue triage guide exists.
* feedback summary template exists.
* RC2 planning doc exists.
* tester response snippets exist.
* health check scripts exist.
* health check script works against local node.
* README/README-ID updated.
* all tests pass.
* smoke test passes.
* build/package still works.
* public safety remains intact.
* docs clearly say testnet DKC has no monetary value and mainnet is unavailable.
