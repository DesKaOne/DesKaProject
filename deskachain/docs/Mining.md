# IndoChain Mining

IndoChain public testnet mining is for testing only. Testnet dIDR has no monetary value, mainnet is not available, and no mining profit is promised.

PoW is the only block-production consensus. Staking is collateral-only and does not produce blocks. Service points are simulation-only and are not spendable.

## Run Miner

Windows:

```powershell
.\indominer.exe --rpc-url http://127.0.0.1:9311 --address <IND_ADDRESS> --threads 2
```

Linux:

```sh
./indominer --rpc-url http://127.0.0.1:9311 --address <IND_ADDRESS> --threads 2
```

Mine one accepted block:

```sh
./indominer --rpc-url http://127.0.0.1:9311 --address <IND_ADDRESS> --threads 2 --once
```

Stop after a bounded run:

```sh
./indominer --rpc-url http://127.0.0.1:9311 --address <IND_ADDRESS> --threads 2 --max-blocks 5
```

The miner retries temporary RPC failures in continuous mode. Use `--duration`, `--retry-delay`, `--max-retry-delay`, `--submit-timeout`, `--job-refresh-interval`, and `--log-interval` to bound runtime and logging.

## Safety

Do not expose wallet or admin RPC publicly. Public RPC may expose read-only mining observation endpoints, but mining template/submit RPC must be explicitly enabled when needed.

On testnet, isolated mining is disabled by default. A miner template requires at least the configured active peer count or one reachable `--upstream-peer`; override only for controlled testing with `--allow-isolated-mining`.

Coinbase maturity is 100 blocks on testnet. A mined reward can be visible as confirmed/immature before it becomes spendable.

## Upstream Backfill

Use `--seed-peer` for pull/sync/discovery and `--upstream-peer` for push/backfill. A mining node can seed from the VPS and also push accepted blocks back to it:

```sh
./indochain --datadir ./data/testnet --network testnet node start \
  --public-rpc --enable-miner-rpc \
  --seed-peer http://100.86.152.39:10311 \
  --upstream-peer http://100.86.152.39:10311 \
  --min-mining-peers 1
```

The VPS remains a normal validating node, not a consensus authority. PoW cumulative work still decides the canonical chain.
