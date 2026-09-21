# Mining Stability Runbook

This runbook covers public testnet mining stability. It does not change consensus, rewards, block format, or genesis.

## Miner Runtime

`indominer` survives temporary RPC failures in continuous mode. It logs RPC unreachable, retry delay, new jobs, found blocks, accepted submits, rejected submits, stale jobs, and clean shutdown.

Useful flags:

```sh
./indominer --rpc-url http://127.0.0.1:9311 --address <IND_ADDRESS> \
  --threads 2 \
  --retry \
  --retry-delay 3s \
  --max-retry-delay 30s \
  --submit-timeout 20s \
  --job-refresh-interval 5s \
  --log-interval 10s
```

## Stale Jobs

A job is stale when the node tip changes, the job height is no longer the next height, submit returns stale, or the chain height has already advanced. The miner discards stale jobs and fetches a fresh template.

Stale submit or duplicate submit is expected when two miners race. The node should reject the losing block clearly while the chain remains valid.

## Multi-Miner Examples

One miner to a local Windows node:

```powershell
.\indominer.exe --rpc-url http://127.0.0.1:9312 --address <IND_ADDRESS> --threads 2 --max-blocks 3
```

One miner to the VPS seed:

```powershell
.\indominer.exe --rpc-url http://100.86.152.39:9311 --address <IND_ADDRESS_2> --threads 2 --max-blocks 3
```

After racing miners, check:

```powershell
.\indochain.exe --rpc-url http://127.0.0.1:9312 chain validate
.\indochain.exe --rpc-url http://127.0.0.1:9312 mining status
.\indochain.exe --rpc-url http://127.0.0.1:9312 peer health
```

## Isolated Mining Guard

Testnet nodes reject miner templates and submits when isolated by default. Configure miners with at least one seed or a reachable upstream:

```powershell
.\indochain.exe --datadir .\data\testnet --network testnet node start `
  --public-rpc `
  --enable-miner-rpc `
  --seed-peer http://100.86.152.39:10311 `
  --upstream-peer http://100.86.152.39:10311 `
  --min-mining-peers 1
```

`mining status` reports `active_peer_count`, `upstream_peer_count`, `upstream_reachable_count`, `min_mining_peers`, and whether templates are currently enabled.
