# IndoChain Public Testnet Topology

Testnet dIDR has no monetary value. Mainnet is not available. Seeds are bootstrap helpers only; nodes still validate network ID, chain ID, genesis hash, headers, and blocks.

## Recommended Topology

```text
                 public / Tailscale

  VPS seed 100.86.152.39:10311
          |
          +---- Mini PC seed 100.101.251.7:10311
          |
          +---- Windows peer/miner 100.83.159.107:10312
          |
          +---- optional community seeds
```

- VPS public seed: stable public bootstrap peer.
- Mini PC Tailscale/LAN seed: secondary operator seed.
- Windows peer/miner: tester node that can bootstrap from both seeds.
- Optional community seeds: added only when operators publish them deliberately.

`--seed-peer` is for pulling blocks and discovering peers. `--upstream-peer` is for pushing accepted blocks and backfilling missing parents to a head/explorer node. The VPS can run without seed peers and still catch up when mining nodes configure it as upstream.

## Port Plan

- P2P: `10311` for seed nodes.
- Windows peer P2P example: `10312`.
- RPC: `9311` for seed/public RPC examples.
- Explorer UI: `/explorer-ui/`.

## Safe Exposure

- P2P can be public.
- RPC should be local/trusted unless `--public-rpc` is enabled intentionally.
- Wallet/admin RPC must not be public.
- Public RPC must report `wallet_rpc=false` and `admin_rpc=false`.
- Public seed operators should publish P2P URLs, not wallet files, datadirs, private keys, or service/faucet state.

## Seed Node

```sh
./indochain --datadir ./data/testnet --network testnet node start \
  --rpc 0.0.0.0:9311 \
  --p2p 0.0.0.0:10311 \
  --advertise-p2p http://100.86.152.39:10311 \
  --public-rpc \
  --max-reorg-depth 128
```

## Public Peer Node

```powershell
.\indochain.exe --datadir .\data\testnet --network testnet node start `
  --rpc 127.0.0.1:9312 `
  --p2p 0.0.0.0:10312 `
  --advertise-p2p http://100.83.159.107:10312 `
  --public-rpc `
  --enable-miner-rpc `
  --seed-peer http://100.86.152.39:10311 `
  --seed-peer http://100.101.251.7:10311 `
  --upstream-peer http://100.86.152.39:10311 `
  --min-mining-peers 1 `
  --max-reorg-depth 128
```

## Miner Node

```powershell
.\indominer.exe --rpc-url http://127.0.0.1:9312 --address <IND_ADDRESS> --threads 2
```

## Service Node Simulation

```sh
./indoservice --rpc-url http://127.0.0.1:9311 --address <IND_ADDRESS> --once
```

Service points are simulation-only and are not spendable dIDR.

## Diagnostics

```powershell
.\indochain.exe --rpc-url http://127.0.0.1:9312 peer list
.\indochain.exe --rpc-url http://127.0.0.1:9312 peer health
.\indochain.exe --rpc-url http://127.0.0.1:9312 peer seeds
.\indochain.exe --rpc-url http://127.0.0.1:9312 peer discover
.\indochain.exe --datadir .\data\testnet --network testnet upstream status --upstream-peer http://100.86.152.39:10311
.\indochain.exe --datadir .\data\testnet --network testnet upstream push http://100.86.152.39:10311
.\indochain.exe --rpc-url http://127.0.0.1:9312 chain validate
.\indochain.exe --rpc-url http://127.0.0.1:9312 mining status
.\indochain.exe --rpc-url http://127.0.0.1:9312 mining difficulty
```

Health check:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\testnet-health.ps1 -RpcUrl http://127.0.0.1:9312 -ExpectedNetwork testnet -ExpectedNetworkID ind-testnet-1 -ExpectedChainID 777101 -CheckPeerList
```

## Troubleshooting

- Wrong network: confirm `network_id=ind-testnet-1`.
- Genesis mismatch: confirm genesis hash `db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4`.
- Peer offline: check firewall and P2P port `10311`.
- Firewall blocked: test from another host, not only localhost.
- `peer_count` zero: add multiple seeds and run `peer discover`.
- Local ahead: the local node has more blocks than the checked peer; this is not automatically an error.
- Stale peer: run `peer health`, wait for cooldown, or remove stale peers deliberately.
- Stale mining job: another miner or peer advanced the tip first; refresh the job and confirm `chain validate`.
