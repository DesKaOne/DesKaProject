# IndoChain Multi-Host Public Testnet Runbook

Phase 4.2 prepares public testnet rehearsal across multiple real hosts, with restart recovery, long-running checks, controlled faucet funding, staking collateral, and service-node simulation. This is not mainnet. Testnet dIDR has no monetary value, staking remains collateral-only, service points are simulation-only, and PoW remains the only block-production consensus.

## Deployment Modes

### LAN Mode

Use this when a Mini PC seed node and peer/miner machines are on the same trusted local network.

- Seed node advertises a LAN address such as `http://192.168.1.5:10311`.
- Peer nodes join with `--seed-peer http://192.168.1.5:10311/`.
- Open the P2P port on the seed host firewall for the LAN subnet.
- Keep wallet/admin RPC disabled on any RPC address reachable by other machines.

### Tailscale Mode

Use this for private public-testnet rehearsal without router port forwarding.

- Every peer using Tailscale IP or MagicDNS must be inside the same Tailnet.
- Seed node advertises a Tailscale IPv4 address or MagicDNS name.
- This behaves like a private overlay network, not open internet.
- It is a good rehearsal path before exposing a VPS or router-forwarded seed.

Examples:

- Mini PC advertise: `http://100.101.251.7:10311`
- MagicDNS advertise: `http://caca.tailxxxx.ts.net:10311`
- Windows peer seed: `http://100.101.251.7:10311/`
- VPS seed via Tailnet, if the VPS joins Tailnet: `http://100.101.251.7:10311/`

### Public VPS Mode

Use this only after LAN/Tailscale checks pass.

- VPS runs a public seed node.
- P2P port can be public.
- RPC should be private/firewalled, or public read-only with wallet/admin disabled.
- Wallet/admin RPC must never be public.
- Seed peers are not trusted authorities; network ID, chain ID, genesis hash, and protocol are still validated.

## Bind Address vs Advertise Address

Bind address is where the node listens locally:

- `--rpc :9311`
- `--p2p :10311`
- `--rpc 0.0.0.0:9311`
- `--p2p 0.0.0.0:10311`

Advertise address is what other peers use to reach this node:

- LAN: `--advertise-p2p http://192.168.1.5:10311`
- Tailscale: `--advertise-p2p http://100.101.251.7:10311`
- Tailscale MagicDNS: `--advertise-p2p http://caca.tailxxxx.ts.net:10311`
- VPS: `--advertise-p2p http://<VPS_PUBLIC_IP>:10311`

Do not advertise `127.0.0.1` to another host. It points back to the peer's own machine.

## Mini PC Seed Over Tailscale

```sh
./indochain --datadir /var/lib/indochain/testnet --network testnet init
./indochain --datadir /var/lib/indochain/testnet node start \
  --rpc 0.0.0.0:9311 \
  --p2p 0.0.0.0:10311 \
  --advertise-p2p http://100.101.251.7:10311 \
  --public-rpc \
  --enable-miner-rpc
```

## Windows Peer Joins

```powershell
.\indochain.exe --datadir .\data\testnet --network testnet init
.\indochain.exe --datadir .\data\testnet node start `
  --rpc :9312 `
  --p2p :10312 `
  --advertise-p2p http://100.83.159.107:10312 `
  --public-rpc `
  --enable-miner-rpc `
  --seed-peer http://100.101.251.7:10311/
```

## Public VPS Seed

Omit `--public-rpc` when RPC is bound to localhost only. Local/admin mode is acceptable when RPC is not reachable from the public internet.

```sh
./indochain --datadir /var/lib/indochain/testnet --network testnet init
./indochain --datadir /var/lib/indochain/testnet node start \
  --rpc 127.0.0.1:9311 \
  --p2p 0.0.0.0:10311 \
  --advertise-p2p http://<VPS_PUBLIC_IP>:10311
```

For public read-only RPC, bind RPC intentionally and use `--public-rpc`; wallet/admin remain disabled unless explicitly forced, and unsafe public binds are rejected.

## Firewall Checklist

Linux UFW LAN:

```sh
sudo ufw allow from 192.168.1.0/24 to any port 10311 proto tcp
```

Linux UFW Tailscale:

```sh
sudo ufw allow in on tailscale0 to any port 10311 proto tcp
sudo ufw allow in on tailscale0 to any port 9311 proto tcp
```

Safer Tailscale setup:

- expose P2P `10311` to Tailnet.
- expose RPC `9311` only if needed.
- never expose wallet/admin RPC.

Public VPS:

```sh
sudo ufw allow 10311/tcp
sudo ufw deny 9311/tcp
```

If public read-only RPC is intentionally enabled for a trusted operator address:

```sh
sudo ufw allow from <trusted-ip> to any port 9311 proto tcp
```

Windows firewall, PowerShell as Administrator:

```powershell
New-NetFirewallRule -DisplayName "IndoChain P2P 10312" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 10312 -Profile Private
New-NetFirewallRule -DisplayName "IndoChain RPC 9312 Private" -Direction Inbound -Action Allow -Protocol TCP -LocalPort 9312 -Profile Private
```

Allow RPC only on private/Tailnet networks when needed.

## Peer Diagnostics

Run from the peer node:

```sh
./indochain --rpc-url http://127.0.0.1:9312 peer list
./indochain --rpc-url http://127.0.0.1:9312 peer check http://100.101.251.7:10311
./indochain --rpc-url http://127.0.0.1:9312 chain info
./indochain --rpc-url http://127.0.0.1:9312 chain validate
```

`peer check` reports reachability, network ID, chain ID, height, and tip hash. Wrong network/genesis peers are rejected. Timeout/refused peers are marked offline in peer metadata.

## Seed Files

Copy and edit the example:

```sh
cp config/testnet-seeds.example.txt config/testnet-seeds.txt
./indochain --datadir ./data/node_b --network testnet node start \
  --rpc :9312 \
  --p2p :10312 \
  --advertise-p2p http://100.83.159.107:10312 \
  --public-rpc \
  --seed-file ./config/testnet-seeds.txt
```

Seed file parsing ignores blank lines and comments, normalizes and deduplicates URLs, rejects invalid URLs, and does not change network/genesis identity.

Direct seed options:

```sh
--seed-peer http://100.101.251.7:10311/
IND_SEED_PEERS=http://host1:10311,http://host2:10311
```

For public testnet RC sharing, publish only operator-controlled seed URLs and checksums. Keep placeholder seeds commented in `config/testnet-seeds.example.txt` until an operator intentionally publishes them.

## Systemd Seed Node

Install:

```sh
sudo useradd --system --home /var/lib/indochain --shell /usr/sbin/nologin indochain || true
sudo mkdir -p /opt/indochain /var/lib/indochain/testnet /etc/indochain
sudo chown -R indochain:indochain /var/lib/indochain
sudo install -m 0755 indochain indominer indoservice /opt/indochain/
sudo install -m 0644 examples/systemd/indochain-testnet.env /etc/indochain/testnet.env
sudo install -m 0644 examples/systemd/indochain-testnet.service /etc/systemd/system/indochain-testnet.service
sudo systemctl daemon-reload
sudo systemctl enable --now indochain-testnet
journalctl -u indochain-testnet -f
```

Edit `/etc/indochain/testnet.env` before starting. Set `IND_ADVERTISE_P2P` to a reachable LAN, Tailscale, or VPS address.

For restart recovery, journald diagnostics, duplicate-lock checks, and failure recovery, see `docs/Systemd.md`.

## Long-Running Recovery

Nodes persist their peer store, node ID, chain DB, mempool file, and service-node store in the datadir. On startup, seed peers and persisted peers are available to the periodic sync loop, so a node that missed block broadcasts while offline can catch up from healthy peers after restart.

Use `docs/LongRunTestnet.md` for the two-host soak checklist covering baseline sync, peer offline catch-up, seed restart, systemd restart, and duplicate datadir lock behavior.

## Faucet, Stake, And Service E2E

Phase 4.2 extends the multi-host rehearsal with a controlled faucet and service-node simulation. Host A may run as a faucet seed/operator node; Host B may join as the wallet, staker, and service owner.

Host A faucet node:

```sh
./indochain --datadir ./data/seed node start \
  --rpc 0.0.0.0:9311 \
  --p2p 0.0.0.0:10311 \
  --advertise-p2p http://<HOST_A_REACHABLE_IP>:10311 \
  --public-rpc \
  --enable-miner-rpc \
  --enable-faucet-rpc \
  --faucet-address <FAUCET_ADDR> \
  --faucet-amount 1000 \
  --faucet-min-interval 1h \
  --faucet-max-per-address 2000
```

Host B service owner flow:

```sh
./indochain --datadir ./data/node_b --network testnet init
./indochain --datadir ./data/node_b node start \
  --rpc 0.0.0.0:9312 \
  --p2p 0.0.0.0:10312 \
  --advertise-p2p http://<HOST_B_REACHABLE_IP>:10312 \
  --public-rpc \
  --enable-miner-rpc \
  --enable-service-rpc \
  --seed-peer http://<HOST_A_REACHABLE_IP>:10311/
```

Then create a testnet owner wallet, request faucet funds from Host A, mine a confirmation block, lock `1000 dIDR`, mine the stake transaction, register the service endpoint, and run `indoservice --once`. Full command sequences and safety warnings are in `docs/Faucet.md`, `docs/ServiceNode.md`, and `docs/Staking.md`.

Important architecture note: stake collateral is chain-backed and should be visible on both hosts after sync. Service registration, challenge samples, and simulated rewards are local service-node state on the RPC node that receives service requests.

## Read-Only Explorer UI/API

Phase 4.4 adds an embedded read-only explorer UI on top of the public-testnet explorer API. They are safe in public RPC mode and do not expose wallet/admin/write actions.

```sh
curl http://127.0.0.1:9311/explorer/status
curl "http://127.0.0.1:9311/explorer/blocks?limit=10"
curl "http://127.0.0.1:9311/explorer/search?q=0"
curl http://127.0.0.1:9311/explorer/address/<ADDRESS>
```

Open the UI locally or over the trusted LAN/Tailscale RPC address:

```text
http://127.0.0.1:9311/explorer-ui/
http://100.101.251.7:9311/explorer-ui/
```

The UI is read-only and uses `/explorer/*` for dashboard, blocks, transactions, addresses, stake records, and service-node summaries. Search is available through `/explorer/search?q=<height|hash|txid|address>`. List endpoints use `limit` and `offset`, default to `20`, and cap at `100`. Explorer service endpoints reflect the local RPC node's service simulation store. Service points are simulation-only and are not spendable dIDR. Testnet dIDR has no monetary value and mainnet is not available. See `docs/Explorer.md`.

## Multi-Host Validation Plan

Host A, Mini PC seed:

- LAN example: `192.168.1.5`
- Tailscale example: `100.101.251.7`
- RPC: `0.0.0.0:9311`
- P2P: `0.0.0.0:10311`
- Advertise: reachable host/IP
- Public RPC: enabled for read-only/miner-safe mode
- Miner RPC: enabled only if the seed accepts direct miners

Host B, Windows/Linux peer:

```sh
./indochain --datadir ./data/node_b --network testnet init
./indochain --datadir ./data/node_b node start \
  --rpc 0.0.0.0:9312 \
  --p2p 0.0.0.0:10312 \
  --advertise-p2p http://100.83.159.107:10312 \
  --public-rpc \
  --enable-miner-rpc \
  --seed-peer http://100.101.251.7:10311/
```

Host B checks:

```sh
./indochain --rpc-url http://127.0.0.1:9312 peer list
./indochain --rpc-url http://127.0.0.1:9312 peer check http://100.101.251.7:10311
```

Mine from Host B:

```sh
./indochain --datadir ./data/miner --network testnet init
ADDR="$(./indochain --datadir ./data/miner wallet new)"
./indominer --rpc-url http://127.0.0.1:9312 --address "$ADDR" --threads 2 --once
```

Check Host A:

```sh
./indochain --rpc-url http://127.0.0.1:9311 chain info
./indochain --rpc-url http://127.0.0.1:9311 chain validate
```

Expected:

- both nodes show the same height.
- both nodes show the same tip hash.
- both nodes are chain valid.
- peer list shows the seed as active after `peer check`.
- wallet/admin RPC remains disabled on public RPC.
- after restart, persisted seed/peer entries still work.
