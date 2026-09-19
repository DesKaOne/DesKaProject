# DesKaChain

DesKaChain is a small experimental CPU-mined blockchain written in Go.

Current phase: Phase 4.12 - Peer Discovery & Auto Bootstrap.

Warning: DesKaChain is experimental local blockchain software. Do not use Phase 1.5 wallets for real funds. Wallet private keys are stored locally for development convenience only.

Coin details:

- Name: DesKaChain
- Ticker: IDR
- Decimals: 8
- Smallest unit: 1 IDR = 100000000 units

## Scope

Included:

- Deterministic genesis block
- Account-based ledger with nonce tracking
- Local wallet and address generation
- Local proof-of-work mining
- bbolt chain storage
- JSON wallet and mempool files for local development
- CLI commands
- Basic JSON HTTP API
- Chain validation command
- Per-node data directories with `--datadir`
- Hardened local transfer and mempool flow
- Transaction lookup and wallet inspection commands
- Local HTTP P2P server for block sync, tx broadcast, and block broadcast
- Datadir runtime lock for running nodes
- Remote CLI control with `--rpc-url`
- Persistent `node_id` per datadir
- P2P handshake and network/genesis guard
- Header-first sync before full block import
- Bounded peer score, fork detection, P2P advertise URL, and two-way peer connect
- Runtime status cache, P2P latency debug, and guarded auto-sync loop
- Synchronous RPC mining job state, cancellable PoW, and bounded block broadcast timeout
- Canonical chain-info transaction stats, atomic mempool writes, duplicate mempool guard, and initial HTTP body/timeout safety
- Final `IDR...` Base58Check wallet address format, secp256k1 keys/signatures, network profiles, and initial protocol version metadata
- Dynamic localnet difficulty adjustment, timestamp sanity checks, `chain difficulty`, and cumulative work based on `16^difficulty`
- Coinbase maturity, mature/immature/spendable balance, and circulating supply based on mature rewards
- Standalone CPU miner CLI (`idrminer`) using node RPC block templates and submit validation
- Public RPC safety mode, basic rate limiting, CORS configuration, health/readiness endpoints, graceful shutdown, and JSON config foundation
- Service node / bandwidth contribution research layer with registration, heartbeat, local challenge simulation, scoring, and simulated service points
- Standalone `idrservice` agent for safe-mode service-node simulation cycles
- Staking collateral module for locking mature IDR and service-node eligibility
- Dev/testnet faucet RPC for normal signed funding transactions from a mature faucet wallet
- Public-testnet seed peer bootstrap through flags, config, environment, and seed files
- Public-testnet upstream peer backfill for miner-to-head-node block push without trusting the head node
- Testnet isolated mining and faucet/write guards with explicit local override flags
- Release build and packaging scripts for `deskachain`, `idrminer`, and `idrservice`
- GitHub Actions CI and release artifact workflows for public-testnet binaries
- Public-testnet genesis candidate, seed-node deployment profiles, and VPS deployment runbook
- Multi-host LAN, Tailscale, and VPS public-testnet deployment runbook
- Long-running public-testnet restart, offline catch-up, duplicate-lock, and systemd recovery runbooks
- Controlled public-testnet faucet, staking collateral, and service-node multi-host E2E runbooks
- Read-only public-testnet explorer UI/API for chain, block, transaction, address, stake, and local service simulation data
- Lightweight peer reputation, score-based outbound peer selection, cooldown recovery, and read-only `/p2p/peers` plus `/p2p/reputation` RPC diagnostics
- Automatic peer discovery/bootstrap maintenance with peer gossip, TTL pruning, retry backoff, subnet limits, and read-only `/p2p/discovery`, `/p2p/bootstrap`, plus `/p2p/known-peers` diagnostics

Not included yet:

- Smart contracts
- EVM
- Production wallet security
- Real service-node payout, public proxy/VPN relay, staking, PoS, GPU mining, or mining pool

## Protocol Basics

DesKaChain Phase 2.6.6 froze the address/key/protocol metadata foundation. Phase 2.7 added dynamic difficulty and more realistic cumulative work. Phase 2.8 added coinbase maturity and mature balance accounting. Phase 2.9 adds a standalone CPU miner that mines RPC block templates without wallet private keys.

Phase 2.10 hardens the node for public testnet preparation without changing consensus rules.

Phase 3.0 introduces service-node contribution research as simulation only. PoW still creates canonical blocks; service points are not IDR, are not spendable, and do not affect supply, difficulty, cumulative work, coinbase rewards, balances, or chain validation.

Phase 3.1 adds `idrservice`, a standalone safe-mode service node agent that registers to node RPC, sends heartbeats, creates/submits simulated challenges, fetches scores, and saves local agent state without opening a proxy, relay, or public listener.

Phase 3.2 adds staking as collateral only. It is not PoS, does not create validators, does not mint staking rewards, and does not affect PoW block production.

Phase 3.2.1 adds staking regression tests and consensus safety checks. Staking remains collateral only: no PoS, no validator set, no APY, no IDR staking reward, and no slashing in this phase.

Phase 3.2.2 separates localnet and testnet runtime profile plumbing before multi-node bootstrap. RPC, P2P, miner templates, staking info, and service collateral metadata now report the active network profile.

Phase 3.2.2.1 finalizes pre-testnet cleanup. Runtime state files are ignored, the Architecture doc path is fixed, safe reorg validation/apply paths are profile-aware, and the testnet profile is ready for Phase 3.3 bootstrap work.

Phase 3.3 adds a deterministic testnet genesis distinct from localnet, datadir profile auto-resolution, `--bootnode`/`--bootnodes` startup flags, and a controlled multi-node testnet runbook in `docs/Testnet.md`.

Phase 3.3.1 hardens `peer sync` so direct sync validates peer network ID, chain ID, genesis hash, and protocol compatibility before it can report that the local chain is already up to date.

Phase 3.3.2 hardens controlled multi-node sync by normalizing peer URLs, persisting deduplicated bootnodes across restarts, recording peer profile/status fields, marking offline/rejected peers with `last_error`, and expanding sync output with imported block counts, height, and tip hash.

Phase 3.4 adds a dev/testnet faucet flow. The faucet is disabled by default, testnet-only when enabled, uses a configured wallet as the source, creates normal signed transactions in the mempool, requires mining for confirmation, and does not mint supply directly.

Phase 3.4.1 adds an end-to-end faucet-funded service collateral scenario: request 1000 testnet IDR, mine the faucet transaction, lock 1000 IDR as collateral, mine the stake transaction, run the service simulation, and verify service score eligibility. See `docs/Faucet.md`, `docs/ServiceNode.md`, `docs/Staking.md`, and `docs/Testnet.md`.

Phase 3.5 adds public-testnet operator packaging: safe public RPC defaults, explicit miner RPC enablement, clearer startup/status summaries, operator docs, systemd and environment examples, and circulating supply regression tests. See `docs/Operator.md`.

Phase 3.6 adds public-testnet seed peer bootstrap. Nodes can load normalized seed peers from network profiles, `--seed-peer`, `--seed-peers`, `IDR_SEED_PEERS`, config `p2p.seed_peers`, or `--seed-file`; seeds are stored in the peer store with source `seed`, and normal peer validation still rejects wrong network or genesis peers. See `docs/Operator.md` and `docs/Testnet.md`.

Phase 3.7 adds public-testnet release build packaging: version metadata for all binaries, Windows/Linux build scripts, release archives, checksums, and quickstart docs. See `docs/Release.md`.

Phase 3.8 adds GitHub Actions CI and release artifact workflows. CI runs workspace sync and Go tests automatically; the release workflow builds Windows/Linux testnet binaries, runs Linux smoke tests, packages archives, verifies checksums, and uploads artifacts without publishing a GitHub Release automatically. See `docs/Release.md`.

Phase 3.9 documents the public-testnet genesis candidate and seed-node deployment preparation. It adds deployment profiles, a VPS runbook, a preflight checklist, an editable seed registry example, and operator notes for faucet/service nodes. See `docs/TestnetGenesis.md`, `docs/DeployTestnet.md`, and `docs/Preflight.md`.

Phase 4.0 documents multi-host public-testnet deployment across LAN, Tailscale, and VPS modes. It clarifies bind vs advertise addresses, firewall rules, peer diagnostics, systemd seed-node env files, and real-host block propagation validation. See `docs/MultiHostTestnet.md`.

Phase 4.1 documents long-running public-testnet operation and restart recovery. It covers graceful shutdown, datadir lock checks, peer persistence, offline catch-up, systemd restart behavior, health/status observability, and the two-host long-run checklist. See `docs/Systemd.md` and `docs/LongRunTestnet.md`.

Phase 4.2 documents the controlled public-testnet faucet, staking collateral, and service-node simulation flow across multiple hosts. Faucet transfers are normal transactions, stake collateral is chain-backed, service points remain simulation-only, and service registration/score samples are local simulation state on the RPC node used by the service agent. See `docs/Faucet.md`, `docs/Staking.md`, and `docs/ServiceNode.md`.

Phase 4.3 adds a read-only explorer API under `/explorer/*` for public-testnet chain summary, blocks, transactions, addresses, stake records, and local service simulation data.

Phase 4.4 adds an embedded read-only explorer web UI at `/explorer-ui/`. It provides dashboard, blocks, block detail, transaction detail, address history, stake records, service-node simulation summaries, and search without wallet/admin/write actions. Testnet IDR has no monetary value, mainnet is not available, and service points are simulation-only. See `docs/Explorer.md`.

Phase 4.5 hardens explorer search and pagination. The read-only API now includes `/explorer/search?q=<query>`, pagination metadata for list endpoints, capped limits, stable JSON explorer errors, and UI polish for search, copy buttons, empty states, and paging. See `docs/Explorer.md`.

Phase 4.6 prepares `v0.4.6-testnet-rc1` as a public testnet release candidate. It adds release notes, an operator checklist, a public quickstart, smoke scripts, artifact verification guidance, seed-node checklist updates, and final public safety copy. See `docs/ReleaseNotes-v0.4.6-testnet-rc1.md`, `docs/OperatorChecklist.md`, and `docs/PublicTestnetQuickstart.md`.

Phase 4.7 prepares RC1 for limited external testers and GitHub pre-release publishing. It adds a GitHub release checklist, copy-paste release body, tester onboarding guide, bug report template, feedback checklist, announcement draft, and seed operator publish guide. See `docs/GitHubReleaseChecklist.md`, `docs/GitHubRelease-v0.4.6-testnet-rc1.md`, `docs/TesterOnboarding.md`, `docs/TestnetFeedbackChecklist.md`, `docs/Announcement-v0.4.6-testnet-rc1.md`, and `docs/SeedOperatorPublish.md`.

Phase 4.8 adds post-release monitoring and RC2 planning workflows for RC1. It includes health-check scripts, seed monitoring, known issue tracking, issue triage, feedback summaries, tester response snippets, and suggested GitHub labels. See `docs/PostReleaseMonitoring.md`, `docs/SeedMonitoringChecklist.md`, `docs/KnownIssues.md`, `docs/IssueTriage.md`, `docs/FeedbackSummaryTemplate.md`, `docs/RC2Planning.md`, `docs/TesterResponseSnippets.md`, and `docs/GitHubLabels.md`.

Phase 4.9 hardens public-testnet peer connectivity with repeated `--seed-peer` support, multi-seed bootstrap, peer discovery hints, peer health metadata, read-only peer diagnostics, and upgraded health-check peer options. See `docs/TestnetTopology.md`, `docs/SeedMonitoringChecklist.md`, `docs/PostReleaseMonitoring.md`, and `docs/PublicTestnetQuickstart.md`.

Phase 4.10 hardens public-testnet mining runtime and observation without changing consensus. `idrminer` now has bounded retry/backoff, stale job detection, submit timeout controls, miner stats, and clearer submit handling; nodes expose read-only `/mining/*` metrics and CLI `mining status|difficulty|blocks`. See `docs/Mining.md`, `docs/MiningStability.md`, and `docs/DifficultyObservation.md`.

Phase 4.10.3 adds upstream seed backfill and isolated mining/write guards. Mining nodes can use `--upstream-peer` to push/backfill accepted blocks to a VPS head/explorer node while `--seed-peer` remains pull/discovery only. Testnet miner templates and faucet writes are blocked when isolated by default. PoW cumulative work remains consensus; no mainnet is available.

Phase 4.11 adds lightweight peer reputation persisted in `peers.json`. Successful syncs and low-latency checks raise peer scores, failures and high latency temporarily penalize peers, expired cooldowns recover automatically, and outbound peer selection prefers higher effective reputation without changing consensus.

Phase 4.12 adds automatic peer discovery and bootstrap maintenance. Nodes periodically gossip known peer lists, discover compatible peers, retry failed peers with bounded exponential backoff, prune expired non-seed peers by TTL, and enforce lightweight peer/IP limits while keeping `peers.json` backward compatible.

Address format:

- New wallet addresses use `IDR` + Base58Check payload.
- Payload is `version byte + HASH160(compressed secp256k1 public key)`.
- Checksum is the first 4 bytes of double SHA256 over the payload.
- Public key hash is `RIPEMD160(SHA256(compressed_public_key))`.
- The old dev address format is legacy localnet-only. New wallets always generate `IDR...` Base58Check addresses, and legacy dev addresses are not valid for public testnet/mainnet.

Keys and signatures:

- Private key export/import is raw 32-byte scalar hex, 64 lowercase hex characters.
- Public keys are compressed secp256k1 public keys, 33 bytes.
- Transaction signatures use secp256k1 ECDSA DER encoding.

Network profiles:

- `localnet`: chain id `777001`, network id `idr-local-1`, address version `0x1E`, RPC `8331`, P2P `9331`, legacy dev addresses allowed.
- `testnet`: chain id `777101`, network id `idr-testnet-1`, address version `0x1F`, RPC `18331`, P2P `19331`, legacy dev addresses disabled.
- `mainnet`: chain id `777000`, network id `idr-main-1`, address version `0x20`, RPC `8333`, P2P `9333`, legacy dev addresses disabled.

Protocol versions:

- protocol version: `1`
- block version: `1`
- tx version: `1`
- P2P protocol version: `idr-p2p/1`
- RPC API version: `v1`

Public network note: testnet IDR has no monetary value. No price, APY, or profit is promised. Any future mainnet claim process, if implemented, must be capped and time-limited.

## Difficulty

DesKaChain no longer relies on fixed difficulty. Phase 2.7 adds a conservative retarget rule for localnet/testnet experiments.

Localnet params:

- initial difficulty: `4`
- min difficulty: `1`
- max difficulty: `8`
- target block time: `10s`
- retarget window: `10` blocks
- max future timestamp drift: `900s`

Difficulty increases by at most 1 when the last retarget window is too fast, decreases by at most 1 when it is too slow, and otherwise stays unchanged. Cumulative work uses `16^difficulty`, so reorg decisions compare total work instead of height alone.

Useful commands:

```powershell
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain difficulty
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain info
```

## Coinbase Maturity

Mining rewards are confirmed immediately but are not spendable until they mature. Localnet uses `10` blocks of coinbase maturity; testnet and mainnet placeholders use `100`.

- `confirmed balance`: all confirmed funds in the canonical chain.
- `mature balance`: confirmed funds that may be spent.
- `immature balance`: confirmed mining rewards that are still locked by coinbase maturity.
- `spendable balance`: mature balance minus pending outgoing mempool transactions.

`send` uses spendable balance. A miner that has mined 3 localnet blocks has `150 IDR` confirmed, `0 IDR` mature, `150 IDR` immature, and `0 IDR` spendable. At height 11, the reward from height 1 matures, so the same miner has `550 IDR` confirmed, `50 IDR` mature, `500 IDR` immature, and `50 IDR` spendable.

`chain info` reports total supply as all confirmed coinbase rewards, while circulating supply is mature coinbase supply. Circulating supply includes mature coins locked as active/unlocking/released stake because they are owner-controlled collateral. Spendable balance is separate and excludes active/unlocking stake plus pending outgoing transactions.

## Standalone CPU Miner

Phase 2.9 adds `idrminer`, a separate CPU miner process. The miner only needs a node RPC URL and a reward address. It does not read wallet files, private keys, or node datadirs; the node builds block templates, validates submitted blocks, stores accepted blocks, clears confirmed mempool transactions, and broadcasts accepted blocks to peers.

Start a node:

```powershell
go run ./node/cmd/deskachain --datadir ./testdata/miner node start --rpc :8401 --p2p :9401 --advertise-p2p http://127.0.0.1:9401
```

Create a reward address through the local node:

```powershell
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8401 wallet new
```

Mine one block:

```powershell
go run ./node/cmd/idrminer --rpc-url http://127.0.0.1:8401 --address <IDR_ADDR> --threads 4 --once
```

Mine continuously or build the binary:

```powershell
go run ./node/cmd/idrminer --rpc-url http://127.0.0.1:8401 --address <IDR_ADDR> --threads 4
go build -o idrminer ./node/cmd/idrminer
```

`idrminer` reports new jobs, hashrate, found blocks, stale templates, retry attempts, and accepted submits. Coinbase maturity still applies to standalone miner rewards.

## Public RPC Hardening

Local/admin mode remains the default. Public RPC mode disables wallet management, admin/debug, miner, faucet, and service write endpoints unless explicitly re-enabled, while read-only chain endpoints remain available.

```powershell
go run ./node/cmd/deskachain --datadir ./testdata/node node start --rpc :8331 --p2p :9331 --advertise-p2p http://127.0.0.1:9331
go run ./node/cmd/deskachain --datadir ./testdata/node node start --rpc :8331 --p2p :9331 --public-rpc --enable-wallet-rpc=false
```

Health and readiness:

```powershell
curl http://127.0.0.1:8331/health
curl http://127.0.0.1:8331/ready
```

Optional JSON config:

```powershell
go run ./node/cmd/deskachain --datadir ./testdata/node node start --config ./docs/config.example.json
```

Do not expose wallet RPC to the internet. Public nodes should be read-only by default. Add `--enable-miner-rpc` only when the node should accept direct miner template/submit traffic, and keep the node behind a firewall or reverse proxy with production-grade rate limiting. Operator docs live in `docs/Operator.md`.

Seed peer bootstrap for public-testnet nodes:

```powershell
go run ./node/cmd/deskachain --datadir ./data/testnet-public --network testnet node start --rpc :8811 --p2p :9811 --advertise-p2p http://<PUBLIC_HOST>:9811 --public-rpc --seed-file ./examples/testnet/testnet-seeds.txt
```

## Release Build

Build public-testnet binaries:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.10-testnet-rc1
```

Package archives and checksums:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.4.10-testnet-rc1 -SkipTests
```

Binary names:

- `deskachain`
- `idrminer`
- `idrservice`

Linux/macOS equivalents:

```sh
sh ./scripts/build.sh v0.4.10-testnet-rc1
SKIP_TESTS=1 sh ./scripts/package.sh v0.4.10-testnet-rc1
```

GitHub Actions:

- `CI` runs on pull requests, pushes to `main`/`master`, and manual dispatch.
- `Release Artifacts` can be run manually with a version such as `v0.4.10-testnet-rc1`; tag pushes matching `v*` also build artifacts.
- Release artifacts include three archives plus `SHA256SUMS.txt`, and intentionally exclude runtime datadirs, wallets, chain DBs, peer/mempool stores, faucet/service state, private keys, `.git`, and intermediate build directories.

Release docs live in `docs/Release.md`; deployment docs live in `docs/DeployTestnet.md`; testnet topology docs live in `docs/TestnetTopology.md`; mining docs live in `docs/Mining.md`, `docs/MiningStability.md`, and `docs/DifficultyObservation.md`; multi-host docs live in `docs/MultiHostTestnet.md`; systemd docs live in `docs/Systemd.md`; long-run docs live in `docs/LongRunTestnet.md`; explorer docs live in `docs/Explorer.md`; faucet docs live in `docs/Faucet.md`; staking docs live in `docs/Staking.md`; service-node docs live in `docs/ServiceNode.md`; operator docs live in `docs/Operator.md`; post-release monitoring docs live in `docs/PostReleaseMonitoring.md`, `docs/SeedMonitoringChecklist.md`, `docs/KnownIssues.md`, `docs/IssueTriage.md`, `docs/FeedbackSummaryTemplate.md`, and `docs/RC2Planning.md`.

## Service Node Simulation

Phase 3.0 adds a research-only service-node layer. It records service node registration, heartbeat uptime, simulated verification challenges, score components, anti-abuse flags, and daily simulated service points. These points are for local/testnet research and leaderboards only; they are not IDR and are not spendable.

```powershell
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8431 service register --address <IDR_ADDR> --endpoint http://127.0.0.1:9501
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8431 service heartbeat --address <IDR_ADDR> --endpoint http://127.0.0.1:9501
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8431 service challenge create --address <IDR_ADDR>
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8431 service challenge submit --challenge-id <ID> --latency-ms 50 --bytes-up 10000000 --bytes-down 50000000 --success true
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8431 service score --address <IDR_ADDR>
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8431 service rewards --address <IDR_ADDR>
```

In public RPC mode, service write endpoints are disabled by default. Use `--enable-service-rpc=true` only for controlled verifier/test setups.

## Service Node Agent

`idrservice` automates the Phase 3.0 service simulation workflow. It needs only a node RPC URL and a IDR address. It does not read private keys, does not mine PoW blocks, does not create spendable IDR, and stays in safe simulation mode.

```powershell
go run ./node/cmd/deskachain --datadir ./testdata/service node start --rpc :8431 --p2p :9431 --advertise-p2p http://127.0.0.1:9431
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8431 wallet new
go run ./node/cmd/idrservice --rpc-url http://127.0.0.1:8431 --address <IDR_ADDR> --endpoint http://127.0.0.1:9501 --once
go run ./node/cmd/idrservice --rpc-url http://127.0.0.1:8431 --address <IDR_ADDR> --endpoint http://127.0.0.1:9501 --heartbeat-interval 30s --challenge-interval 60s
go build -o idrservice ./node/cmd/idrservice
```

Agent state defaults to `./idrservice-state.json`. Inspect it with:

```powershell
go run ./node/cmd/idrservice status --state ./idrservice-state.json
```

Do not enable service RPC publicly without rate limits and abuse protection. Phase 3.1 service RPC is for controlled testnet/verifier simulation.

## Staking Collateral

Phase 3.2 staking locks mature IDR as service-node collateral. Locked and unbonding stake reduce spendable balance, but confirmed balance, mature balance, total supply, coinbase rewards, difficulty, and PoW consensus are unchanged.

```powershell
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8471 stake info
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8471 stake lock --address <IDR_ADDR> --amount 10
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8471 stake list --address <IDR_ADDR>
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8471 stake unlock --address <IDR_ADDR> --stake-id <STAKE_ID>
```

Localnet requires 10 IDR minimum stake and 100 IDR active stake for service reward simulation eligibility. There is no slashing and no staking APY in Phase 3.2.

For testnet service-node collateral, the current threshold is 1000 IDR. A faucet-funded runbook is documented in `docs/Testnet.md`.

## Data Directories

All local state lives under `./data` by default. Use `--datadir` before the command to run separate local nodes:

```powershell
go run ./node/cmd/deskachain --datadir ./testdata/node1 init
go run ./node/cmd/deskachain --datadir ./testdata/node1 wallet new
go run ./node/cmd/deskachain --datadir ./testdata/node1 chain info
```

## Clean Run

```powershell
go run ./node/cmd/deskachain dev reset --yes
go run ./node/cmd/deskachain init
go run ./node/cmd/deskachain wallet new
go run ./node/cmd/deskachain mine --address <addr> --blocks 3
go run ./node/cmd/deskachain balance <addr>
go run ./node/cmd/deskachain chain info
go run ./node/cmd/deskachain chain validate
```

Total supply may be higher than one wallet balance because previous mined blocks may belong to another miner address.

## Transfer Example

Pending transactions do not change confirmed balances until they are mined into a block. Miners receive 50 IDR per block. Transaction fees are still 0 IDR in Phase 1.6.

```powershell
go run ./node/cmd/deskachain dev reset --yes
go run ./node/cmd/deskachain init
go run ./node/cmd/deskachain wallet new
go run ./node/cmd/deskachain wallet new
go run ./node/cmd/deskachain mine --address <walletA> --blocks 11
go run ./node/cmd/deskachain send --from <walletA> --to <walletB> --amount 10
go run ./node/cmd/deskachain mempool list
go run ./node/cmd/deskachain wallet inspect --address <walletA>
go run ./node/cmd/deskachain wallet inspect --address <walletB>
go run ./node/cmd/deskachain mine --address <walletA> --blocks 1
go run ./node/cmd/deskachain balance <walletA>
go run ./node/cmd/deskachain balance <walletB>
go run ./node/cmd/deskachain chain validate
go run ./node/cmd/deskachain tx get <txid>
```

In that flow, wallet A ends with 190 IDR, wallet B ends with 10 IDR, and total supply is 200 IDR.

## Local P2P Example

Phase 2 uses simple local HTTP P2P. Each node should use a different datadir, RPC port, and P2P port.

When `node start` is running, do not run local write commands against the same datadir from another process. Use `--rpc-url` to control the running node instead. DesKaChain writes `<datadir>/node.lock` while a node is running to reduce accidental concurrent writers.

Prepare two local nodes in Windows PowerShell:

```powershell
go run ./node/cmd/deskachain --datadir ./testdata/node1 dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/node2 dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/node1 init
go run ./node/cmd/deskachain --datadir ./testdata/node2 init
go run ./node/cmd/deskachain --datadir ./testdata/node1 wallet new
go run ./node/cmd/deskachain --datadir ./testdata/node2 wallet new
```

Start node 1 in terminal 1:

```powershell
go run ./node/cmd/deskachain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331 --advertise-p2p http://127.0.0.1:9331
```

Start node 2 in terminal 2:

```powershell
go run ./node/cmd/deskachain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --advertise-p2p http://127.0.0.1:9332 --peers http://127.0.0.1:9331
```

Mine on node 1 from terminal 3:

```powershell
go run ./node/cmd/deskachain --datadir ./testdata/node1 mine --address <walletNode1> --blocks 3
```

Connect peers and sync node 2:

```powershell
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer connect http://127.0.0.1:9331
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 chain info
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 chain validate
```

Expected: node 2 height and tip hash match node 1, and node 2 chain is valid.

Broadcast tx example:

```powershell
go run ./node/cmd/deskachain --datadir ./testdata/node1 send --from <walletNode1> --to <walletNode2> --amount 10
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 mempool list
go run ./node/cmd/deskachain --datadir ./testdata/node1 mine --address <walletNode1> --blocks 1
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 balance <walletNode2>
```

Preferred Phase 2.3 remote-control flow:

```powershell
# Terminal 1
go run ./node/cmd/deskachain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331 --advertise-p2p http://127.0.0.1:9331

# Terminal 2
go run ./node/cmd/deskachain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --advertise-p2p http://127.0.0.1:9332 --peers http://127.0.0.1:9331

# Terminal 3
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer connect http://127.0.0.1:9331
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 peer list
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer list
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node status
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 3 --timeout 5m
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 send --from <walletA> --to <walletB> --amount 10
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 mempool list
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 1
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 balance <walletB>
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node compare --peer http://127.0.0.1:8332
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain validate
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 chain validate
```

Expected: node 2 receives the tx broadcast, receives the block broadcast, wallet B balance becomes 10 IDR, both chains validate, and `node compare` reports `nodes in sync`.

Phase 2.3 broadcast debug command:

```powershell
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 p2p test-broadcast --from <walletA> --to <walletB> --amount 10 --miner <walletA> --peer-rpc http://127.0.0.1:8332
```

Expected output includes `tx broadcast: success=1 failed=0`, `block broadcast: success=1 failed=0`, `compare: nodes in sync`, and `p2p broadcast test passed`.

Phase 2.3.1 latency/debug commands:

```powershell
Invoke-RestMethod http://127.0.0.1:9331/p2p/status
Invoke-RestMethod http://127.0.0.1:9332/p2p/status

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 p2p ping http://127.0.0.1:9332
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 p2p ping http://127.0.0.1:9331
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 p2p debug
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 p2p debug
```

Expected: `/p2p/status` responds quickly, `p2p ping` prints `result: ok`, and `p2p debug` shows peer latency plus `last_error`.

Phase 2.3 remote peer commands while nodes are running:

```powershell
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer check http://127.0.0.1:9331
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer connect http://127.0.0.1:9331
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync --peer http://127.0.0.1:9331
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer status
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node compare --peer http://127.0.0.1:8332
```

If you accidentally run a local write command against a locked datadir, the CLI prints the matching `--rpc-url` command to use instead.

Phase 2 limitations:

- Local HTTP P2P only
- Forks are detected and rejected, but no automatic reorg yet
- No NAT traversal
- Public peer discovery was added later in Phase 4.9 through bounded `/p2p/peers` hints and manual `peer discover`
- Automatic background public peer discovery remains intentionally bounded/deferred beyond the manual Phase 4.9 flow
- Not production safe

Phase 2.3.1 notes:

- Each node stores a persistent `<datadir>/node_id`.
- `node start --advertise-p2p` stores the URL other nodes should use to call this node.
- `peer add` performs a handshake before saving a peer.
- `peer connect` adds a peer and tries to introduce this node back through `POST /p2p/peer`.
- Peers with different network ID, chain ID, genesis hash, or incompatible protocol versions are rejected.
- Sync is header-first: headers are checked for continuity before full blocks are fetched.
- Peer score is clamped to -100..100. Bad peers are scored and marked, but not permanently banned.
- Runtime RPC/P2P status uses an in-memory chain snapshot during `node start`; validation still reads storage as the source of truth.
- The auto-sync loop uses per-peer guards and throttles repeated timeout logs.
- Status, health, tip, and handshake endpoints should stay lightweight and should not perform outbound peer requests.

Troubleshooting P2P status timeout:

```powershell
Invoke-RestMethod http://127.0.0.1:9331/p2p/health
Invoke-RestMethod http://127.0.0.1:9331/p2p/status
Invoke-RestMethod http://127.0.0.1:9331/p2p/tip
Invoke-RestMethod http://127.0.0.1:9331/p2p/handshake

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 p2p ping http://127.0.0.1:9332
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 p2p debug
```

If `context deadline exceeded` appears for `/p2p/status`, first check the direct `Invoke-RestMethod` commands above. Broadcast tx/block and sync do not intentionally hold peer metadata locks while performing outbound requests.

Phase 2.3.2 mining diagnostics:

```powershell
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <addr> --blocks 3 --timeout 5m

# If mining appears stuck, run these from another terminal:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine status
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 debug locks
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 p2p debug
Invoke-RestMethod http://127.0.0.1:9331/p2p/status
```

Mining notes:

- RPC `/mine` remains synchronous for compatibility, but it is tracked as a mining job.
- Only one mining job can run per node in Phase 2.3.2.
- The PoW loop checks request cancellation and supports `--max-nonce`.
- Slow peer block broadcast is capped per peer and should not hang mining.
- Node logs show mining start, per-block found/committed, broadcast summary, and completion/failure.

Phase 2.3.3 runtime hygiene:

- All runtime files are scoped to the selected `--datadir`: `chain.db`, `wallets.json`, `mempool.json`, `peers.json`, `node_id`, and `node.lock`.
- `dev reset --yes` removes the full selected datadir, including stale `peers.json`.
- `dev inspect` reports runtime file state, chain height, peer count, and wallet count without taking the node lock.
- `node start` prints the peer store path and the peer source (`peers.json`, `flag`, or `peers.json + flag`) after deduplication.
- `peer clear --yes` removes peers only; it does not remove chain data or wallets.
- At genesis, tip difficulty is `0` and next difficulty is `4`. The first mined block and mining logs use next difficulty `4`.

Troubleshooting stale peers after reset:

```powershell
go run ./node/cmd/deskachain --datadir ./testdata/node1 dev inspect
go run ./node/cmd/deskachain --datadir ./testdata/node1 peer list --source
go run ./node/cmd/deskachain --datadir ./testdata/node1 peer clear --yes
```

Expected after `dev reset --yes` and `init`: `peers: 0`, with `peers.json: missing` or an empty peer file.

Phase 2.4 fork diagnostics:

```powershell
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain locator
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 fork check --peer http://127.0.0.1:9332
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain common-ancestor --peer http://127.0.0.1:9332
```

Current phase: Phase 4.12 - Peer Discovery & Auto Bootstrap.

Reorg is the explicit process of replacing the local canonical branch with a peer branch that has greater cumulative work. Reorg is experimental for localnet/testnet, is not automatic by default, and normal `peer sync` still rejects forks unless `--allow-reorg --yes` is supplied. DesKaChain uses cumulative work for reorg decisions, not height alone, and refuses equal-work or lower-work peer branches. Preview is dry-run only; apply requires an explicit `--yes`. Reorg preview/apply now validates handshake, chain ID, network ID, and consensus replay against the active network profile.

Phase 2.6 adds mempool recovery checks for reorgs with normal transactions:

- Blocks removed during reorg become orphaned/disconnected blocks.
- Normal transactions from orphaned blocks can be returned to the mempool only when they are still valid against the new canonical ledger.
- Coinbase transactions from orphaned blocks are never returned to the mempool.
- Transactions already confirmed by the newly connected peer branch are removed from pending state instead of being requeued.
- Conflicting, duplicate, or otherwise invalid transactions are dropped.
- Account balances, nonces, and total supply follow the new canonical branch only.

Example reorg transaction recovery scenario:

```powershell
go run ./node/cmd/deskachain dev reorg-tx-sim --datadir-a ./testdata/reorgTxA --datadir-b ./testdata/reorgTxB --scenario requeue-valid

go run ./node/cmd/deskachain --datadir ./testdata/reorgTxA node start --rpc :8351 --p2p :9351 --advertise-p2p http://127.0.0.1:9351
go run ./node/cmd/deskachain --datadir ./testdata/reorgTxB node start --rpc :8352 --p2p :9352 --advertise-p2p http://127.0.0.1:9352

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8351 reorg preview --peer http://127.0.0.1:9352
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8351 reorg apply --peer http://127.0.0.1:9352 --yes
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8351 mempool list --detail
```

Expected for `requeue-valid`:

```text
requeued transactions: 1
pending tx count: 1
chain valid: true
```

Phase 2.4.1 common ancestor troubleshooting:

If `chain common-ancestor --peer <p2p-url>` reports `common ancestor not found` while `fork check` says the nodes are in sync, inspect the exact locator being sent:

```powershell
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain locator
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain common-ancestor --peer http://127.0.0.1:9332 --debug
```

Expected for nodes in sync:

```text
common ancestor found
height: <tip height>
hash: <tip hash>
```

Phase 2.4.2 local fork simulation:

```powershell
go run ./node/cmd/deskachain dev fork-sim --datadir-a ./testdata/forkA --datadir-b ./testdata/forkB --blocks-a 3 --blocks-b 3
go run ./node/cmd/deskachain --datadir ./testdata/forkA fork inspect --other-datadir ./testdata/forkB
```

Expected:

```text
fork detected
common ancestor height: 0
reorg supported: false
```

Phase 2.4.2 only proves that real forks are detected and automatic sync rejects them safely. Automatic reorg is still disabled; safe reorg is planned for Phase 2.5.

## Commands

Initialize chain:

```powershell
go run ./node/cmd/deskachain init
```

Reset local development data:

```powershell
go run ./node/cmd/deskachain dev reset --yes
```

Inspect local runtime files:

```powershell
go run ./node/cmd/deskachain dev inspect
```

Create and list wallets:

```powershell
go run ./node/cmd/deskachain wallet new
go run ./node/cmd/deskachain wallet list
```

When a node is running and the datadir is locked, create wallets through the local/admin RPC instead:

```powershell
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 wallet new
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 wallet list
```

Wallet RPC creates private keys on the node. Do not expose wallet RPC on public RPC nodes. Public RPC nodes should disable wallet management in a future hardening phase.

Export a local development private key:

```powershell
go run ./node/cmd/deskachain wallet export --address <addr> --show-private-key
```

Inspect a wallet or address:

```powershell
go run ./node/cmd/deskachain wallet inspect --address <addr>
go run ./node/cmd/deskachain address validate <addr>
```

Check balance:

```powershell
go run ./node/cmd/deskachain balance <addr>
```

Create a pending transaction:

```powershell
go run ./node/cmd/deskachain send --from <fromAddress> --to <toAddress> --amount 1.25
```

Inspect or clear mempool:

```powershell
go run ./node/cmd/deskachain mempool list
go run ./node/cmd/deskachain mempool clear --yes
```

Lookup a transaction:

```powershell
go run ./node/cmd/deskachain tx get <txid>
```

Mine blocks:

```powershell
go run ./node/cmd/deskachain mine --address <addr> --blocks 3
```

Inspect and validate chain:

```powershell
go run ./node/cmd/deskachain chain info
go run ./node/cmd/deskachain chain locator
go run ./node/cmd/deskachain chain print
go run ./node/cmd/deskachain chain validate
go run ./node/cmd/deskachain chain common-ancestor --peer http://127.0.0.1:9332
```

Start the HTTP API:

```powershell
go run ./node/cmd/deskachain rpc --addr :8332
```

Start combined RPC + P2P node:

```powershell
go run ./node/cmd/deskachain node start --rpc :8332 --p2p :9332 --advertise-p2p http://127.0.0.1:9332 --peers http://127.0.0.1:9331
```

Show network identity:

```powershell
go run ./node/cmd/deskachain network info
```

Check node status:

```powershell
go run ./node/cmd/deskachain node status
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node status
```

Manage peers:

```powershell
go run ./node/cmd/deskachain peer list
go run ./node/cmd/deskachain peer list --source
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer check http://127.0.0.1:9331
go run ./node/cmd/deskachain peer add http://127.0.0.1:9331
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer connect http://127.0.0.1:9331
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer status
go run ./node/cmd/deskachain peer remove http://127.0.0.1:9331
go run ./node/cmd/deskachain peer clear --yes
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer clear --yes
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 fork check --peer http://127.0.0.1:9332
go run ./node/cmd/deskachain chain info
```

Running `peer sync` repeatedly should not duplicate blocks: height, total supply, and tip hash should stay unchanged once the node is up to date.

## RPC Endpoints

- `GET /health`
- `GET /ready`
- `GET /network/info`
- `GET /node/id`
- `GET /node/compare?peer=<rpc-url>`
- `GET /node/status`
- `GET /debug/p2p`
- `POST /debug/p2p/ping`
- `GET /debug/locks`
- `GET /peers`
- `GET /p2p/peers`
- `GET /p2p/reputation`
- `GET /p2p/discovery`
- `GET /p2p/bootstrap`
- `GET /p2p/known-peers`
- `POST /peers`
- `POST /peers/connect`
- `POST /peers/clear`
- `POST /peers/sync`
- `POST /peers/status`
- `GET /chain/info`
- `GET /chain/difficulty`
- `GET /chain/locator`
- `GET /chain/blocks`
- `GET /chain/validate`
- `POST /fork/check`
- `GET /faucet/info`
- `POST /faucet/request`
- `GET /miner/template?address=<IDR_ADDR>`
- `POST /miner/submit`
- `POST /service/register`
- `POST /service/heartbeat`
- `POST /service/challenge/create`
- `POST /service/challenge/submit`
- `GET /service/score?address=<IDR_ADDR>`
- `GET /service/rewards?address=<IDR_ADDR>`
- `GET /service/list`
- `GET /stake/info`
- `GET /stake/list`
- `GET /stake/status?id=<STAKE_ID>`
- `POST /stake/lock`
- `POST /stake/unlock`
- `GET /balance/{address}`
  - Returns `balance` for backward compatibility plus `confirmed_balance`, `mature_balance`, `immature_balance`, `spendable_balance`, `pending_outgoing`, `pending_incoming`, `coinbase_maturity`, and `current_height`.
- `GET /address/{address}`
- `GET /tx/{txid}`
- `GET /mempool`
- `POST /mempool/clear`
- `GET /wallets`
- `POST /wallet/new`
- `POST /send`
- `POST /mine`
- `GET /mine/status`

## P2P Endpoints

- `GET /p2p/handshake`
- `GET /p2p/status`
- `GET /p2p/headers`
- `GET /p2p/locator`
- `GET /p2p/block/{height}`
- `POST /p2p/tx`
- `POST /p2p/block`
- `POST /p2p/peer`
- `POST /p2p/common-ancestor`

P2P endpoints:

- `GET /p2p/health`
- `GET /p2p/handshake`
- `GET /p2p/status`
- `GET /p2p/tip`
- `GET /p2p/headers?from=<height>&limit=<n>`
- `GET /p2p/locator`
- `GET /p2p/block/{height}`
- `GET /p2p/blocks?from=<height>&limit=<n>`
- `POST /p2p/common-ancestor`
- `POST /p2p/tx`
- `POST /p2p/block`

Example send body:

```json
{
  "from": "<IDR_ADDR>",
  "to": "<IDR_ADDR>",
  "amount": "1.25"
}
```

Example mine body:

```json
{
  "address": "<IDR_ADDR>",
  "blocks": 1
}
```

## Validation

```powershell
go mod tidy
go test ./...
go run ./node/cmd/deskachain chain validate
```

## Roadmap

- Phase 2: P2P
- Phase 3: Difficulty + mining pool
- Phase 4: Explorer + faucet
- Phase 5: Flutter wallet
- Phase 6: Public testnet
- Phase 7: Mainnet

## Phase 2.4.3 — Same Height Fork Sync Fix & Chain Info Tx Count

Same height does not mean same chain. Nodes are only up to date when both the height and tip hash match. If two nodes have the same height but different tip hashes, DesKaChain treats that as a fork. Peer sync rejects forked chains until Phase 2.5 introduces safe automatic reorg support.

Example fork simulation and sync rejection:

```bash
go run ./node/cmd/deskachain dev fork-sim --datadir-a ./testdata/forkA --datadir-b ./testdata/forkB --blocks-a 3 --blocks-b 3

go run ./node/cmd/deskachain --datadir ./testdata/forkA node start --rpc :8341 --p2p :9341 --advertise-p2p http://127.0.0.1:9341

go run ./node/cmd/deskachain --datadir ./testdata/forkB node start --rpc :8342 --p2p :9342 --advertise-p2p http://127.0.0.1:9342

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 peer add http://127.0.0.1:9342
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 peer sync
```

Expected:

```text
sync failed: fork detected
common ancestor height: 0
automatic reorg: disabled
```


### Safe reorg experimental workflow

Create forked local chains:

```sh
go run ./node/cmd/deskachain dev fork-sim --datadir-a ./testdata/forkA --datadir-b ./testdata/forkB --blocks-a 3 --blocks-b 5
```

Start both nodes:

```sh
go run ./node/cmd/deskachain --datadir ./testdata/forkA node start --rpc :8341 --p2p :9341 --advertise-p2p http://127.0.0.1:9341
go run ./node/cmd/deskachain --datadir ./testdata/forkB node start --rpc :8342 --p2p :9342 --advertise-p2p http://127.0.0.1:9342
```

Preview and explicitly apply the safe reorg:

```sh
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 reorg preview --peer http://127.0.0.1:9342
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 reorg apply --peer http://127.0.0.1:9342 --yes
```

Verify:

```sh
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 node compare --peer http://127.0.0.1:8342
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 chain validate
```

Expected output includes `nodes in sync` and `chain valid`.
