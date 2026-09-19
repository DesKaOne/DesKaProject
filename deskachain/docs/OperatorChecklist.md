# DesKaChain Public Testnet Operator Checklist

Testnet IDR has no monetary value. Mainnet is not available. Do not expose wallet/admin RPC publicly.

## Before Running

- Download the release artifact only from the official repository workflow or release.
- Verify SHA256 checksums.
- Confirm binary version is `v0.4.6-testnet-rc1` or the intended RC version.
- Choose mode: private local test, LAN, Tailscale, or public VPS.
- Choose a dedicated datadir.
- Confirm testnet genesis and network identity.
- Confirm firewall plan.

## Network Setup

- P2P port is reachable by intended peers.
- RPC binding is intentional.
- Advertised P2P address is reachable by peers.
- Seed peers are configured deliberately.
- Multiple seed peers are configured where possible.
- Mining nodes configure the public head/explorer node as `--upstream-peer` for block backfill.
- Seeds are treated as discovery helpers only, not trusted consensus authorities.
- Wallet/admin RPC are not public.

## Node Startup

- Initialize testnet datadir.
- Start the node.
- Check `/health`.
- Check `/explorer/status`.
- Run `scripts/testnet-health.ps1` or `scripts/testnet-health.sh`.
- Run `peer health`.
- Run `peer discover` when peer count is zero or stale.
- Run `chain info`.
- Run `chain validate`.
- Check peer list.

## Mining

- Create a separate miner wallet datadir.
- Mine once with `idrminer --once`.
- Confirm testnet miner RPC is not isolated: active peers meet `--min-mining-peers` or an upstream peer is reachable.
- Confirm chain height increases.
- Confirm Explorer block list updates.
- Run `mining status`, `mining difficulty`, and `mining blocks --limit 10`.
- Watch for stale job and duplicate submit logs when multiple miners race.

## Faucet Operator

- Create a faucet wallet in a controlled datadir.
- Mine mature faucet funds.
- Start faucet RPC only with explicit operator intent.
- Confirm faucet/write RPC is not isolated unless `--allow-isolated-writes` is deliberately set for a private test.
- Configure amount, minimum interval, and max per address.
- Confirm faucet state path.
- Confirm faucet does not expose wallet/admin RPC.

## Service Node

- Confirm owner has `1000 IDR` testnet collateral.
- Lock stake.
- Confirm active stake.
- Enable service RPC deliberately.
- Run `idrservice --once`.
- Confirm service eligibility.
- Confirm service points are simulation-only and are not spendable IDR.

## Explorer

- Open `/explorer-ui/`.
- Search by height, block hash, txid, and address.
- Confirm no write controls are visible.
- Confirm public safety banner is visible.

## Restart And Recovery

- Restart node or systemd service.
- Confirm chain height and tip persist.
- Confirm peers reconnect or can sync.
- Run `chain validate`.

## Before Sharing With Testers

- Publish seed peer URL.
- Publish checksums.
- Publish safety warnings.
- Publish known limitations.
- Link `docs/PostReleaseMonitoring.md`, `docs/SeedMonitoringChecklist.md`, `docs/KnownIssues.md`, `docs/IssueTriage.md`, `docs/FeedbackSummaryTemplate.md`, and `docs/RC2Planning.md`.
- Link `docs/TestnetTopology.md`.
- Link `docs/Mining.md`, `docs/MiningStability.md`, and `docs/DifficultyObservation.md`.
- Ask testers to report issues with the testnet bug report template.
- Do not publish private wallets, datadirs, faucet state, service state, or secrets.

## After Installing RC1

- Run the health check against the intended public RPC URL.
- Verify `/explorer-ui/` in a browser.
- Confirm `wallet_rpc=false` and `admin_rpc=false` on public nodes.
- Confirm network ID `idr-testnet-1` and chain ID `777101`.
- Confirm multi-seed failover is configured with at least two operator seeds where available.
- Report or triage issues using `docs/IssueTriage.md`.
- Track common problems in `docs/KnownIssues.md`.
- Remember testnet IDR has no monetary value.

## Release Safety Audit

- Archive has no runtime datadir.
- Archive has no wallet files.
- Archive has no `faucet_state.json`.
- Archive has no `idrservice-state.json`.
- Archive has no private keys.
- Archive has no `.env` files outside safe examples.
- Example env files contain placeholders only.
- Public examples do not enable wallet/admin RPC.
