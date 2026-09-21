# IndoChain RC1 Post-Release Monitoring

Testnet dIDR has no monetary value. Mainnet is not available. Do not expose wallet/admin RPC on public nodes.

## Purpose

Use this checklist after publishing `v0.4.6-testnet-rc1` to monitor public-testnet health, catch operational issues early, collect tester feedback, and prepare RC2 fixes.

The goal is to answer:

- Is the seed node reachable?
- Are peers syncing?
- Is the explorer alive?
- Are public RPC safety flags correct?
- Are testers reporting common issues?
- Which issues block RC2?

## What To Monitor

- Seed P2P reachability from a different network.
- Multi-seed bootstrap health; one failed seed should not prevent startup.
- RPC `/health`.
- Explorer `/explorer/status`.
- Explorer UI `/explorer-ui/`.
- Current height.
- Tip hash.
- Peer count.
- Known peer count and active peer count from `peer health`.
- Seed list from `peer seeds`.
- Mining current difficulty, next difficulty, last block age, average interval, and blocks until retarget from `mining status`.
- Mempool count.
- Public RPC flags:
  - `wallet_rpc` must be `false` on public nodes.
  - `admin_rpc` must be `false` on public nodes.
  - `public_rpc` should be `true` when intentionally public.
- `chain validate` result.
- Disk usage for the node datadir and host.
- Process uptime or systemd service uptime.
- Recent error logs, especially repeated panics, validation errors, peer rejection loops, and disk write failures.

PowerShell health check:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\testnet-health.ps1 -RpcUrl http://127.0.0.1:9311 -ExpectedNetwork testnet -ExpectedNetworkID ind-testnet-1 -ExpectedChainID 777101
```

Bash health check:

```sh
bash ./scripts/testnet-health.sh http://127.0.0.1:9311 --expected-network testnet --expected-network-id ind-testnet-1 --expected-chain-id 777101 --check-peer-list --check-mining
```

Peer diagnostics:

```powershell
.\indochain.exe --rpc-url http://127.0.0.1:9311 peer health
.\indochain.exe --rpc-url http://127.0.0.1:9311 peer seeds
.\indochain.exe --rpc-url http://127.0.0.1:9311 peer discover
.\indochain.exe --rpc-url http://127.0.0.1:9311 mining status
.\indochain.exe --rpc-url http://127.0.0.1:9311 mining blocks --limit 10
```

## What Is Not Monitored

- Financial value.
- Mining profitability.
- Any claim that testnet dIDR has monetary value.
- Staking yield.
- Service point value.

Any report framed around price, profit, APY, or spendable service points is outside RC1 scope.

## Suggested Cadence

- First 24 hours: every 1-2 hours.
- First week: at least daily.
- After stabilization: daily or as needed.

Record snapshots in `docs/FeedbackSummaryTemplate.md` or an issue/discussion thread.

## Escalation Levels

- Info: cosmetic UI issue, docs typo, wording confusion, non-blocking question.
- Warning: one tester report, recoverable sync issue, slow node, explorer pagination/search confusion, faucet operator misconfiguration.
- Critical: seed unreachable, chain validation fails, consensus mismatch, public wallet/admin exposure, reproducible panic, package checksum mismatch, or wrong network/genesis in a published artifact.

Critical issues should be labeled `testnet-rc1`, assigned an owner, and considered for `rc2-candidate`.
