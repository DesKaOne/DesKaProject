# DesKaChain Testnet Operator Guide

Phase 3.6 prepares public-testnet style operation with bootstrap seed peers. This is still experimental testnet software: testnet DKC has no monetary value, staking is collateral-only, service points are simulation-only, and PoW remains the only block-production consensus.

## Public Testnet Node

Initialize a testnet datadir:

```powershell
go run ./node/cmd/deskachain --datadir ./data/testnet-public --network testnet init
```

Start a public-safe read-only node:

```powershell
go run ./node/cmd/deskachain --datadir ./data/testnet-public node start --rpc :8811 --p2p :9811 --advertise-p2p http://<PUBLIC_HOST>:9811 --public-rpc
```

In `--public-rpc` mode, wallet, admin, miner, faucet, and service write RPC are disabled unless explicitly enabled. Do not expose wallet or admin RPC publicly. If public mode is bound to all interfaces and wallet/admin RPC are forced on, startup is rejected.

Health checks:

```powershell
go run ./node/cmd/deskachain --rpc-url http://<PUBLIC_HOST>:8811 chain info
go run ./node/cmd/deskachain --rpc-url http://<PUBLIC_HOST>:8811 chain validate
Invoke-RestMethod http://<PUBLIC_HOST>:8811/health
```

Open firewall ports:

- RPC: TCP `8811` or the port you choose.
- P2P: TCP `9811` or the port you choose.

## Miner-Enabled Testnet Node

Miner RPC is explicit in public mode:

```powershell
go run ./node/cmd/deskachain --datadir ./data/testnet-public node start --rpc :8811 --p2p :9811 --advertise-p2p http://<PUBLIC_HOST>:9811 --public-rpc --enable-miner-rpc
```

Mine with a testnet address:

```powershell
go run ./node/cmd/dkcminer --rpc-url http://<PUBLIC_HOST>:8811 --address <TESTNET_DKC_ADDR> --threads 4
```

HTTP RPC mining is solo/direct-node mining. Stratum and pool mining are not implemented.

## Private Wallet Node

Use a private/local node for wallet operations:

```powershell
go run ./node/cmd/deskachain --datadir ./data/testnet-wallet --network testnet init
go run ./node/cmd/deskachain --datadir ./data/testnet-wallet node start --rpc 127.0.0.1:8911 --p2p :9911 --advertise-p2p http://127.0.0.1:9911
```

Create wallets only on trusted local nodes:

```powershell
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 wallet new
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 wallet list
```

Back up `wallets.json` before deleting a datadir.

## Bootnode Workflow

Node A:

```powershell
go run ./node/cmd/deskachain --datadir ./data/node-a --network testnet init
go run ./node/cmd/deskachain --datadir ./data/node-a node start --rpc :8811 --p2p :9811 --advertise-p2p http://<A_HOST>:9811 --public-rpc --enable-miner-rpc
```

Node B:

```powershell
go run ./node/cmd/deskachain --datadir ./data/node-b --network testnet init
go run ./node/cmd/deskachain --datadir ./data/node-b node start --rpc :8812 --p2p :9812 --advertise-p2p http://<B_HOST>:9812 --bootnode http://<A_HOST>:9811 --public-rpc --enable-miner-rpc
```

Peer checks:

```powershell
go run ./node/cmd/deskachain --rpc-url http://<B_HOST>:8812 peer list
go run ./node/cmd/deskachain --rpc-url http://<B_HOST>:8812 peer check http://<A_HOST>:9811
go run ./node/cmd/deskachain --rpc-url http://<B_HOST>:8812 peer sync http://<A_HOST>:9811
go run ./node/cmd/deskachain --rpc-url http://<B_HOST>:8812 chain validate
```

Peers with wrong network ID, chain ID, genesis hash, or incompatible protocol are rejected.

## Seed Peer Bootstrap

Seed peers are startup hints for public-testnet discovery. They are normalized, deduplicated, skipped when they match the node's own advertised P2P URL, and stored in `peers.json` with source `seed`. A seed being offline does not fail node startup; later `peer check`, `peer status`, or `peer sync` records the failure in peer metadata.

Seed sources are merged in this order:

- Network profile seed peers.
- `DKC_SEED_PEERS` or config `p2p.seed_peers`.
- `--seed-file` or config `p2p.seed_file`.
- CLI `--seed-peer`, `--seed-peers`, `--bootnode`, and `--bootnodes`.

Create a seed file:

```text
# /etc/deskachain/testnet-seeds.txt
http://seed1.example.testnet:9811
http://seed2.example.testnet:9811 # inline comments are allowed
```

Start from a seed file:

```powershell
go run ./node/cmd/deskachain --datadir ./data/testnet-public --network testnet node start --rpc :8811 --p2p :9811 --advertise-p2p http://<PUBLIC_HOST>:9811 --public-rpc --seed-file ./examples/testnet/testnet-seeds.txt
```

Or pass seeds directly:

```powershell
go run ./node/cmd/deskachain --datadir ./data/testnet-public --network testnet node start --rpc :8811 --p2p :9811 --advertise-p2p http://<PUBLIC_HOST>:9811 --public-rpc --seed-peer http://seed1.example.testnet:9811 --seed-peers http://seed2.example.testnet:9811,http://seed3.example.testnet:9811
```

Use `--bootnode` for a specific operator-controlled peer and seed peers for a reusable bootstrap list. Both are persisted, but their source labels remain distinct.

For public deployment preparation, copy `config/testnet-seeds.example.txt` to `config/testnet-seeds.txt` and replace commented placeholders only when real operator-controlled seed nodes exist. Do not publish fake seed domains as active defaults.

## Faucet Operator Mode

The dev faucet is testnet-only and disabled by default. Run it only in controlled environments:

```powershell
go run ./node/cmd/deskachain --datadir ./data/testnet-faucet node start --rpc :8911 --p2p :9911 --advertise-p2p http://<FAUCET_HOST>:9911 --enable-faucet-rpc --faucet-address <FAUCET_ADDR> --faucet-amount 1000 --faucet-max-per-address 2000 --faucet-min-interval 1s
```

The faucet signs normal transactions from `<FAUCET_ADDR>` and requires mined blocks for confirmation. It does not mint directly.

## Service Node Operator Mode

Register and run the safe service simulation:

```powershell
go run ./node/cmd/deskachain --rpc-url http://<NODE_HOST>:8811 service register --address <OWNER_ADDR> --endpoint http://<SERVICE_HOST>:9971
go run ./node/cmd/dkcservice --rpc-url http://<NODE_HOST>:8811 --address <OWNER_ADDR> --endpoint http://<SERVICE_HOST>:9971 --once
go run ./node/cmd/deskachain --rpc-url http://<NODE_HOST>:8811 service score --address <OWNER_ADDR>
```

Service write RPC should be enabled only in controlled verifier/test setups. Service points are not DKC and are not spendable.

## Preflight Checklist

- Datadir initialized with `--network testnet`.
- `network.json` matches `dkc-testnet-1`, chain ID `777101`, and the testnet genesis hash.
- Public node uses `--public-rpc`.
- Wallet/admin RPC are not exposed publicly.
- Miner RPC is enabled only when intended.
- Faucet RPC is enabled only on controlled testnet faucet nodes.
- Advertised P2P URL is reachable by other nodes.
- Seed peers or bootnodes point to the same testnet profile.
- `chain validate` passes.
- `/health` reports expected network, height, peer count, and RPC mode toggles.

See `docs/Preflight.md` for the public-testnet deployment checklist and `docs/DeployTestnet.md` for VPS setup commands.

## Linux systemd

Example templates are available:

- `examples/testnet/public-node.env`
- `examples/testnet/miner-node.env`
- `examples/testnet/faucet-node.env`
- `examples/systemd/deskachain-testnet.service`
- `examples/systemd/dkcservice-testnet.service`

Typical commands:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now deskachain-testnet
journalctl -u deskachain-testnet -f
```

## Release Artifact Operations

Public-testnet release artifacts are built by `.github/workflows/release-artifacts.yml`. Run it manually from GitHub Actions with a version such as `v0.4.6-testnet-rc1`, or push a `v*` tag to build artifacts from the tag name.

The workflow uploads archives only. It does not publish a GitHub Release automatically, does not require secrets, and does not launch mainnet. Artifacts include `deskachain`, `dkcminer`, `dkcservice`, quickstart docs, README files, and example testnet/systemd templates.

Before deploying downloaded artifacts:

- Verify `SHA256SUMS.txt`.
- Confirm the binary `version` output matches the intended testnet version and commit.
- Keep wallet/admin RPC private.
- Use separate datadirs for public nodes, miner wallets, faucet nodes, and service agent state.

Artifacts intentionally exclude runtime datadirs, wallets, chain DBs, mempool and peer stores, faucet/service state, private keys, `.git`, and intermediate build output.

## Troubleshooting

- `endpoint disabled in public RPC mode`: the endpoint is intentionally unavailable in public mode.
- `service RPC disabled`: service write RPC is not enabled.
- `wrong network version`: the address belongs to another network profile.
- `peer rejected`: inspect network ID, chain ID, genesis hash, and protocol version.
- `invalid seed peer`: inspect the seed file line number or `DKC_SEED_PEERS` entry printed in the error.
- `circulating supply: 0 DKC` above maturity: run `chain validate` and check the active network profile; circulating supply should equal mature coinbase supply and includes active/unlocking stake.
