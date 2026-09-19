# DesKaChain Roadmap Updated

## Phase 1 — Local Blockchain Core

Status: Done

* Genesis block
* Block validation
* Local mining
* Wallet generate
* Balance check
* Send transaction
* Mempool basic
* Local chain storage
* Chain validate

## Phase 2 — P2P, Sync, Fork, and Reorg Foundation

Status: Done

* Multi-node localnet
* RPC server
* P2P server
* Peer add/connect/list/sync
* Transaction broadcast
* Block broadcast
* Peer score basic
* Handshake and network guard
* Header/locator sync
* Common ancestor detection
* Fork detection
* Fork-safe sync rejection
* Safe reorg preview
* Safe reorg apply with `--yes`
* Reorg mempool recovery
* Orphan transaction handling
* Cumulative work comparison

## Phase 2.6.5 — Runtime Stats & Safety Cleanup

Status: Done

Goals:

* Clean up small runtime bugs before protocol changes.
* Make chain info accurate.
* Make mempool safer before public RPC/testnet.
* Improve project hygiene before codebase grows bigger.

Tasks:

* Fix runtime `/chain/info` transaction stats.
* Ensure normal transactions are counted correctly while node is running.
* Ensure coinbase transactions and normal transactions are separated correctly.
* Improve `.gitignore`.
* Ignore runtime files:

  * `chain.db`
  * `wallets.json`
  * `mempool.json`
  * `peers.json`
  * `node.lock`
  * `node_id`
* Add atomic write for `mempool.json`.
* Add mempool duplicate tx guard.
* Add basic runtime mutex for mempool operations.
* Add TODO or internal structure for stronger public RPC concurrency safety.
* Add HTTP server timeout TODO or initial timeout support.
* Keep behavior backward compatible.

Delivered:

* Canonical block scan for `chain info` transaction stats.
* Atomic `mempool.json` writes and duplicate tx guard.
* Basic per-file mempool mutex.
* Initial HTTP request body limits and server timeout groundwork.
* Runtime safety TODO notes in `docs/RuntimeSafety.md`.

## Phase 2.6.6 — Protocol Spec Freeze & IDR Base58 Address Migration

Status: Done

Goals:

* Freeze core protocol before wallet, explorer, miner, and SDK grow bigger.
* Replace dev address format with final IDR Base58 address.
* Keep private key as raw 32-byte hex.
* Decide key curve direction.
* Add protocol versioning.
* Add localnet/testnet/mainnet network profiles.

Important decision:

DesKaChain now uses secp256k1 for new wallet keys, compressed public keys, and transaction signatures.

Recommended final address format:

* Prefix: `IDR`
* Encoding: Base58Check
* Payload: version byte + public key hash + checksum
* Private key export: raw 32-byte hex

Tasks:

* Add IDR Base58Check address encoder.
* Add IDR Base58Check address decoder.
* Add checksum validation.
* Add address version byte.
* Add network-specific address version.
* Update wallet generation.
* Update wallet import/export.
* Update balance command.
* Update send command.
* Update mining address validation.
* Update RPC address validation.
* Add old dev address support only for migration/local dev if needed.
* Add protocol constants:

  * block version
  * transaction version
  * p2p protocol version
  * rpc api version
* Add network profiles:

  * localnet
  * testnet
  * mainnet
* Define per-network:

  * chain ID
  * network ID
  * genesis config
  * default RPC port
  * default P2P port
  * address prefix/version
  * difficulty params
  * coinbase maturity
  * max reorg depth

Delivered:

* Final `IDR...` Base58Check address encoder/decoder with checksum and leading-zero safe Base58.
* HASH160 address payload from compressed secp256k1 public keys.
* secp256k1 private key generation/import/export using raw 32-byte hex.
* secp256k1 DER transaction signing and verification.
* Localnet/testnet/mainnet network profiles with distinct address versions.
* Localnet-only legacy dev address validation compatibility.
* Wallet inspect/list/export/import metadata for address format, network, and key curve.
* RPC/P2P/node status protocol metadata.
* Protocol reference in `docs/Protocol.md`.

## Phase 2.6.6.1 — Remote Wallet Command UX Fix

Status: Done

Delivered:

* Remote `wallet new` via local/admin RPC.
* Remote `wallet list` without private keys.
* Remote `wallet inspect <address>` metadata output.
* Lock hint now points to a supported remote wallet command.
* Wallet store per-file mutex, duplicate guard, and atomic write path.
* README warning not to expose wallet RPC publicly.

## Phase 2.7 — Difficulty Adjustment & Realistic Cumulative Work

Status: Done

Goals:

* Replace fixed difficulty with dynamic difficulty adjustment.
* Add target block time.
* Add retarget window.
* Validate block difficulty during chain validation.
* Use more realistic cumulative work.
* Keep localnet mining light.

Default localnet params:

* Initial difficulty: 4
* Minimum difficulty: 1
* Maximum difficulty: 8
* Target block time: 10 seconds
* Retarget window: 10 blocks

Tasks:

* Add consensus difficulty params.
* Implement next difficulty calculation.
* Mining uses next difficulty.
* Chain validation checks expected difficulty.
* P2P rejects invalid difficulty blocks.
* Reorg rejects invalid difficulty branches.
* Add `chain difficulty` command.
* Update cumulative work from simple `work = difficulty` to a more realistic model.
* Recommended local model:

  * `work = 16 ^ difficulty`
* Ensure reorg uses cumulative work, not height alone.

Delivered:

* Network difficulty params for localnet/testnet/mainnet.
* Retarget calculation with max +/- 1 difficulty per window.
* Mining uses canonical next difficulty.
* Chain validation checks expected difficulty, PoW, and timestamp sanity.
* Cumulative work now uses `16 ^ difficulty` with overflow clamp.
* `chain difficulty` local and remote command.
* `GET /chain/difficulty` RPC endpoint.
* `chain info` includes target block time, retarget window, min/max difficulty, blocks until retarget, and cumulative work.

## Phase 2.8 — Coinbase Maturity & Mature Balance

Status: Completed

Goals:

* Prevent newly mined rewards from being spent too soon.
* Make reorg safer.
* Prepare wallet/explorer balance display.

Tasks:

* Add coinbase maturity.
* Localnet maturity default: 10 blocks.
* Testnet/mainnet maturity default: 100 blocks.
* Add mature balance.
* Add immature balance.
* Prevent spending immature coinbase reward.
* Update wallet balance output.
* Update send validation.
* Update ledger validation.
* Update reorg tests.
* Update explorer-ready balance fields.
* Define circulating supply as mature coinbase supply.

## Phase 2.9 — Standalone CPU Miner CLI

Status: Completed

Goals:

* Separate mining from node.
* Prepare future desktop miner.
* Prepare mining pool protocol later.

Tasks:

* Add standalone miner command or binary.
* Miner connects to node RPC.
* Miner requests block template.
* Miner submits solved block.
* Add thread count option.
* Add clean cancellation and retry intervals.
* Add mining stats.
* Add hashrate display.
* Add reconnect logic.
* Add Windows/Linux support.
* Add amd64 and arm64 support.
* Node validates and broadcasts accepted standalone-mined blocks.

## Phase 2.10 — Node Public Hardening

Status: Completed

Goals:

* Make public RPC/P2P safer.
* Reduce abuse risk.
* Prepare public testnet nodes.

Tasks:

* Add RPC/P2P HTTP server timeouts and max header size.
* Add bounded RPC request body sizes with JSON 413 responses.
* Add basic in-memory per-IP RPC rate limiting.
* Add public RPC mode with wallet/admin RPC disabled by default.
* Add explicit wallet, miner, and admin RPC enable flags.
* Add CORS allowlist configuration.
* Add max peer defaults per network profile.
* Add graceful shutdown for RPC/P2P servers.
* Add health and readiness endpoints.
* Add structured startup/shutdown logs.
* Add JSON config file foundation and example.

## Phase 3.0 — Bandwidth Mining Research & Reward Simulation

Status: Completed

Important decision:

Bandwidth mining must not become block consensus mining yet.

Consensus mining:

* CPU/GPU PoW creates blocks.

Bandwidth mining:

* Service reward layer.
* Reward based on verified uptime, bandwidth, latency, and useful network service.
* Rewards should come from a defined service reward pool or testnet reward simulation first.

Goals:

* Design Proof-of-Bandwidth / Service Node concept.
* Avoid fake traffic and self-farming.
* Avoid unsafe public exit proxy behavior.
* Keep it testnet-only at first.

Tasks:

* Add service node identity tied to `IDR...` owner address.
* Add service heartbeat and uptime sample tracking.
* Add local challenge verifier simulation.
* Add latency and bandwidth sample scoring.
* Add anti-abuse flags and penalties.
* Add simulated daily service points.
* Add service RPC and CLI commands.
* Keep service points outside IDR balances and consensus.
* Do not pay real/mainnet rewards yet.

## Phase 3.1 — Service Node Agent MVP

Status: Completed

Goals:

* Run service-node simulation as a standalone safe-mode process.
* Keep service agent behavior outside consensus and IDR balances.

Tasks:

* Add `idrservice` binary.
* Add local agent state JSON.
* Add RPC client for Phase 3.0 service endpoints.
* Add safe-mode simulated measurement generator.
* Add automatic register, heartbeat, challenge, submit, and score cycle.
* Add graceful shutdown and state save.
* Add status command for local state.
* Do not implement public proxy, VPN relay, bandwidth selling, staking, or spendable rewards.


## Phase 3.2 — Staking Module & Service Node Collateral

Status: Completed

Goals:

* Add IDR lock/unlock staking collateral without changing PoW consensus.
* Make active stake available as service-node eligibility metadata.
* Keep staking outside coinbase rewards, block production, and total supply changes.

Tasks:

* Add network staking params.
* Add backward-compatible stake transaction types.
* Add canonical stake state derived from chain blocks.
* Add stake lock and unlock commands/RPC.
* Add active, unlocking, released, and pending stake balance accounting.
* Keep active and unbonding stake out of spendable balance.
* Add service-node collateral eligibility fields.
* Do not add delegation, slashing, validators, APY, or staking rewards.

## Phase 3.2.1 — Staking Regression Tests & Consensus Safety

Status: Completed

Goals:

* Add permanent regression tests for staking lifecycle and collateral behavior.
* Verify staking balance accounting, chain validation, RPC safety, and service-node eligibility.
* Keep staking collateral-only with no PoS, no validators, no APY, no IDR staking rewards, and no slashing.

## Phase 3.2.2 — Pre-Testnet Cleanup & Network Profile Plumbing

Status: Completed

Goals:

* Clean runtime state files and ignore local node artifacts.
* Plumb active network profiles through CLI, RPC, P2P, miner, staking, and service collateral paths.
* Add localnet/testnet readiness checks before public bootstrap work.

## Phase 3.2.2.1 — Pre-Testnet Final Cleanup & Profile Reorg Fix

Status: Completed

Goals:

* Remove leftover runtime state files and stale Architecture typo paths.
* Make P2P/RPC/CLI safe reorg validation and apply paths use the active network profile.
* Ensure RPC mining address validation uses the active network profile.
* Keep the testnet profile safe for Phase 3.3 multi-node bootstrap.

## Phase 3.3 — Testnet Genesis/Profile & Multi-node Bootstrap

Status: Completed

Goals:

* Define deterministic testnet genesis/profile distinct from localnet.
* Resolve datadir network metadata automatically when `--network` is omitted.
* Prepare bootnode startup flags and peer-store bootstrap behavior.
* Add multi-node controlled testnet runbook and profile regression tests.

## Phase 3.3.1 — Peer Sync Network Guard Fix

Status: Completed

Goals:

* Ensure `peer sync` validates peer network ID, chain ID, genesis hash, and protocol compatibility before up-to-date checks.
* Support direct positional `peer sync <url>` locally and through `--rpc-url`.
* Add regression coverage for network/genesis mismatch and valid up-to-date peers.

## Phase 3.3.2 — Multi-node Sync Regression Tests

Status: Completed

Goals:

* Normalize peer URLs before persistence and peer operations.
* Persist bootnode peers across restart without duplicate trailing-slash variants.
* Record peer profile fields, source, score, latency, last status check, and `last_error`.
* Mark unavailable peers as offline and invalid/mismatched peers as rejected/bad.
* Keep `peer sync` output explicit about imported blocks, current height, tip hash, up-to-date, and local-ahead cases.
* Add regression coverage for restart sync, offline recovery, network mismatch, local-ahead output, and invalid-block peer penalties.

Long-term research:

* PoS or hybrid consensus remains future research only. Staking is collateral-only in Phase 3.x.

## Phase 3.4 — Dev Faucet & Testnet Funding Flow

Status: Completed

Goals:

* Add a dev/testnet faucet that is disabled by default and enabled explicitly.
* Use a configured faucet wallet as source of normal signed transactions.
* Require mature spendable faucet balance and normal block mining for confirmation.
* Track per-address rate limits and pending request state in `faucet_state.json`.
* Keep faucet disabled on localnet/mainnet and public RPC unless explicitly enabled.
* Document faucet funding, request, mining confirmation, and staking/service test flows.

## Phase 3.4.1 — Faucet Stake/Service Testnet Scenario

Status: Planned

Goals:

* Add an end-to-end controlled testnet scenario that funds a wallet through faucet, locks staking collateral, and verifies service-node eligibility.
* Keep staking collateral-only and service points simulation-only.

## Phase 3.5 — Read-only Explorer API

Status: Planned

Goals:

* Add read-only chain, block, transaction, address, and mempool API surfaces for explorer work.
* Avoid public faucet, wallet, miner, or admin behavior in explorer endpoints.

## Phase 3.6 — Public Testnet Bootstrap Registry & Seed Peer Config

Status: Completed

Goals:

* Add seed peer support to network profiles and startup config.
* Load seeds from flags, config, `IDR_SEED_PEERS`, and seed files.
* Persist normalized seed peers with source metadata while keeping bootnode behavior intact.
* Keep wrong network/genesis rejection in the existing peer validation and sync paths.

## Phase 3.7 — Public Testnet Release Build & Binary Packaging

Status: Completed

Goals:

* Add version metadata to `deskachain`, `idrminer`, and `idrservice`.
* Build Windows/Linux testnet binaries with ldflags.
* Package release archives with quickstart docs and checksums.
* Exclude runtime/private state from release artifacts.

## Phase 3.8 - GitHub Actions CI Release Artifacts

Status: Completed

Goals:

* Run Go tests automatically on pull requests and main/master pushes.
* Build Windows/Linux public-testnet binaries in GitHub Actions.
* Package release archives and `SHA256SUMS.txt`.
* Upload workflow artifacts without requiring secrets or publishing mainnet releases.
* Run Linux amd64 binary version and node/miner smoke tests in CI.

Tasks:

* Add `.github/workflows/ci.yml`.
* Add `.github/workflows/release-artifacts.yml`.
* Make shell packaging scripts CI-friendly.
* Add archive safety checks for runtime/private files.
* Document manual workflow runs, artifact download, and checksum verification.

## Phase 3.9 - Public Testnet Genesis Candidate & Seed Node Deployment Prep

Status: Completed

Goals:

* Document the public-testnet genesis candidate and consensus parameters.
* Prepare seed-node deployment profiles and editable seed registry examples.
* Add VPS/Windows deployment runbooks and preflight checklist.
* Document faucet and service-node operator preparation without changing consensus.
* Keep testnet IDR valueless and mainnet unavailable.

Tasks:

* Add `docs/TestnetGenesis.md`.
* Add `docs/DeployTestnet.md`.
* Add `docs/Preflight.md`.
* Add `config/testnet-seeds.example.txt`.
* Add seed/service deployment environment examples.
* Add regression tests for stable testnet identity and genesis candidate.

## Phase 3.10 - GPU Miner Research

Status: Planned

Goals:

* Explore GPU mining after CPU mining is stable.
* Avoid breaking CPU miner too early.

Tasks:

* Freeze PoW algorithm first.
* Benchmark CPU miner.
* Research GPU implementation.
* Add OpenCL/CUDA only after PoW format is stable.
* Keep GPU optional.
* Do not make mobile GPU mining.

## Phase 3.11 - Mining Pool MVP

Status: Planned

Goals:

* Allow multiple miners to contribute work.
* Prepare community mining.

Tasks:

* Pool server.
* Stratum-like protocol or simple IDR pool protocol.
* Worker registration.
* Share difficulty.
* Share validation.
* Payout accounting.
* Pool dashboard.
* Pool anti-cheat checks.

## Phase 4.0 - Public Testnet Multi-Host Deployment

Status: Completed

Goals:

* Document LAN, Tailscale, and VPS multi-host testnet deployment modes.
* Clarify bind address versus advertised P2P address.
* Add firewall checklist for Linux UFW, Tailscale, VPS, and Windows.
* Document peer diagnostics and block propagation validation across real hosts.
* Add production-like systemd seed-node environment example.

Tasks:

* Add `docs/MultiHostTestnet.md`.
* Update seed examples for LAN, Tailscale, and VPS.
* Add `examples/systemd/deskachain-testnet.env`.
* Link multi-host docs from release/deployment/preflight docs.

## Phase 4.1 - Public Testnet Long-Running Stability & Restart Recovery

Status: Completed

Goals:

* Validate long-running public-testnet operation across real hosts.
* Document graceful shutdown, restart recovery, datadir lock safety, and systemd behavior.
* Verify peer persistence, background reconnect/sync, and missed block recovery.
* Keep wallet/admin RPC private and preserve public RPC safety.

Tasks:

* Add `docs/Systemd.md`.
* Add `docs/LongRunTestnet.md`.
* Link restart recovery docs from deployment, multi-host, preflight, release, and README files.
* Include long-run and systemd docs in release packages.

## Phase 4.2 - Public Testnet Faucet + Service Node Multi-Host E2E

Status: Completed

Goals:

* Validate controlled faucet operation on a public-testnet seed/operator node.
* Prove faucet requests create normal transactions and do not mint silently.
* Document faucet-funded staking collateral and service-node simulation across two hosts.
* Keep wallet/admin RPC private while faucet/service RPC require explicit operator intent.
* Separate chain-backed stake visibility from node-local service simulation state.

Tasks:

* Update `docs/Faucet.md` with a multi-host faucet operator runbook.
* Update `docs/ServiceNode.md` and `docs/Staking.md` with multi-host collateral and local service-state notes.
* Link faucet/stake/service E2E from multi-host, long-run, deployment, preflight, release, and README docs.
* Include faucet, staking, and service-node docs in release packages.

## Phase 4.3 - Public Testnet Explorer API & Read-Only Indexer Prep

Status: Completed

Goals:

* Add read-only explorer endpoints for public-testnet chain, block, tx, address, stake, and local service simulation data.
* Keep explorer API safe in public RPC mode without enabling wallet/admin/write RPC.
* Document simple scan mode and future persistent indexer direction.
* Preserve consensus, genesis, block format, and tx format.

Tasks:

* Add `/explorer/*` read-only RPC endpoints.
* Add `docs/Explorer.md`.
* Link explorer docs from README, deployment, multi-host, and release docs.
* Include explorer docs in release packages.

## Phase 4.4 - Public Testnet Explorer Web UI MVP

Status: Completed

Goals:

* Add a lightweight embedded read-only explorer web UI backed by `/explorer/*`.
* Keep the UI safe in public RPC mode without wallet/admin/write actions.
* Cover dashboard, blocks, block detail, transaction detail, address history, stake records, service-node simulation, and search.
* Preserve consensus, genesis, block format, tx format, faucet behavior, staking collateral rules, and service-node simulation rules.

Tasks:

* Serve `/explorer-ui/` and `/explorer-ui/assets/*` from the node binary.
* Add plain HTML/CSS/JavaScript explorer pages with no npm, CDN, or tracking dependency.
* Add RPC static-serving and read-only safety tests.
* Document how to open the explorer UI locally and over LAN/Tailscale.

## Phase 4.5 - Explorer UX Polish + API Pagination/Search Hardening

Status: Completed

Goals:

* Add `/explorer/search` for height, block hash, transaction id, and active-network address lookup.
* Harden explorer pagination with default limit, max cap, offsets, and page metadata.
* Improve explorer UI search, pagination, empty states, copy feedback, and responsive behavior.
* Keep explorer UI/API read-only and safe in public RPC mode.

Tasks:

* Add search and pagination tests for explorer API.
* Add UI static smoke tests for search and copy helpers.
* Document search, pagination, and error format.

## Phase 4.6 - Public Testnet Release Candidate & Operator Checklist

Status: Completed

Goals:

* Prepare `v0.4.6-testnet-rc1` release candidate artifacts for external testers/operators.
* Add release notes, public quickstart, operator checklist, smoke scripts, and artifact verification guidance.
* Keep consensus, genesis, economics, and public RPC safety unchanged.

Tasks:

* Add `docs/ReleaseNotes-v0.4.6-testnet-rc1.md`.
* Add `docs/OperatorChecklist.md`.
* Add `docs/PublicTestnetQuickstart.md`.
* Add local release smoke scripts for PowerShell and Bash.
* Include RC docs in release archives and keep archive safety scans.

## Phase 4.7 - Public Testnet RC1 GitHub Release & External Tester Onboarding

Status: Completed

Goals:

* Prepare GitHub pre-release publishing materials for `v0.4.6-testnet-rc1`.
* Add external tester onboarding, bug report, feedback, announcement, and seed publishing docs.
* Keep release safety warnings clear and avoid auto-publishing real seed URLs.

Tasks:

* Add `docs/GitHubReleaseChecklist.md`.
* Add `docs/GitHubRelease-v0.4.6-testnet-rc1.md`.
* Add `docs/TesterOnboarding.md`.
* Add `.github/ISSUE_TEMPLATE/testnet-bug-report.md`.
* Add `docs/TestnetFeedbackChecklist.md`.
* Add `docs/Announcement-v0.4.6-testnet-rc1.md`.
* Add `docs/SeedOperatorPublish.md`.

## Phase 4.8 - Public Testnet RC1 Post-Release Monitoring & Feedback Loop

Status: Completed

Goals:

* Monitor RC1 seed/testnet health after release.
* Provide lightweight health-check scripts for RPC and explorer endpoints.
* Track known issues, triage severity, and prepare RC2 candidates.
* Keep public RPC safety checks visible for operators and testers.

Tasks:

* Add post-release monitoring and seed monitoring docs.
* Add known issues, issue triage, feedback summary, RC2 planning, and response snippets.
* Add PowerShell and Bash testnet health-check scripts.
* Link monitoring docs from README, quickstart, and operator checklist.
* Keep testnet IDR no-monetary-value language explicit.

## Phase 4.9 - Public Testnet Multi-Seed & Peer Discovery Hardening

Status: Completed

Goals:

* Harden public-testnet bootstrap when one seed is offline or slow.
* Support repeated seed flags, seed files, persisted peers, and discovered peers.
* Expose public read-only peer diagnostics without enabling wallet/admin RPC.
* Keep peer discovery bounded and always validate network/genesis before storing candidates.

Tasks:

* Add multi-seed bootstrap behavior and startup logging.
* Add P2P peer discovery hints and `/p2p/peers`.
* Extend peer metadata with success/failure/cooldown fields.
* Add `peer health`, `peer seeds`, and `peer discover` CLI/RPC flows.
* Upgrade health-check scripts with peer count and seed checks.
* Add `docs/TestnetTopology.md` and multi-seed docs/examples.

## Phase 4.10 - Public Testnet Mining Stability & Difficulty Observation

Status: completed.

Goals:

* Harden `idrminer` retry/backoff, stale job detection, submit timeout behavior, and stats output.
* Add read-only mining observation endpoints and CLI commands.
* Improve miner submit diagnostics for stale and duplicate blocks.
* Add health-check mining options and operator mining runbooks.
* Keep consensus, genesis, reward, block format, and monetary assumptions unchanged.

## Phase 4.11 - Wallet, Explorer, Faucet, and Public RPC

Status: Planned

Tasks:

* Wallet CLI production-ish.
* Flutter wallet mobile.
* Flutter wallet desktop.
* Flutter wallet web.
* Explorer API.
* Explorer web.
* Testnet faucet.
* Public RPC gateway.
* Public seed nodes.
* Docs for users and miners.

Requirements before this phase:

* Final IDR address format.
* Coinbase maturity.
* Difficulty adjustment.
* Stable node RPC.
* Stable send/balance APIs.
* Stable transaction format.

## Phase 5 — Public Testnet

Status: Planned

Requirements before launch:

* Address format finalized.
* Genesis testnet finalized.
* Dynamic difficulty active.
* Realistic cumulative work active.
* Coinbase maturity active.
* Reorg stable.
* Mempool reorg recovery stable.
* CPU miner CLI stable.
* Node public hardening done.
* Explorer running.
* Faucet running.
* Public RPC running.
* Wallet CLI stable.
* Basic Flutter wallet usable.
* Mining guide ready.
* Node install guide ready.
* Snapshot/reset policy ready.

Tasks:

* Release testnet node binary.
* Release miner binary.
* Publish docs.
* Open community mining.
* Add leaderboard.
* Add small bug bounty.
* Monitor chain health.

## Phase 5.1 — Testnet Mining Claim Program

## Phase 6 — Swap / Wrapped IDR Research

Status: Planned after stable testnet

Goals:

* Let users eventually swap IDR value without rushing exchange listing.
* Keep early swap testnet/simulation first.
* Avoid real-money promises.

Tasks:

* Swap simulation.
* Testnet mock assets.
* Wrapped IDR research.
* EVM contract research.
* Bridge relayer design.
* Reserve/liquidity model.
* Security notes.
* No real-money promise.

## Phase 7 — Mainnet Preparation

Status: Planned

Requirements:

* Mainnet genesis final.
* Tokenomics final.
* Reward schedule final.
* Address format final.
* Network profile final.
* Seed nodes ready.
* Explorer ready.
* Wallet ready.
* Miner ready.
* Public RPC ready.
* Docs ready.
* Reproducible builds.
* Backup/snapshot plan.
* Security review.

## Phase 8 — Mainnet Launch

Status: Future

Tasks:

* Fair launch.
* Mainnet seed nodes.
* Public binaries.
* Explorer.
* Wallet.
* Miner.
* Mining guide.
* Community onboarding.
* Monitoring.
* Post-launch patch plan.

