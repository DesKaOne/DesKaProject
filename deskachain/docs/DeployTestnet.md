# IndoChain Public Testnet Deployment Runbook

This runbook prepares a public testnet seed or miner-enabled node from release artifacts. It does not launch mainnet. Testnet dIDR has no monetary value.

For LAN, Tailscale, and public VPS multi-host setup, see `docs/MultiHostTestnet.md`. For systemd restart recovery, see `docs/Systemd.md`. For long-running two-host validation, see `docs/LongRunTestnet.md`. For read-only explorer UI/API usage, see `docs/Explorer.md`.

## Verify Release Artifacts

Download artifacts only from the official repository workflow run or release process. Checksum verification confirms download integrity; it does not replace source review.

Linux:

```sh
cd dist/releases
sha256sum -c SHA256SUMS.txt
```

Windows:

```powershell
Get-FileHash .\indochain-v0.4.6-testnet-rc1-windows-amd64.zip -Algorithm SHA256
Get-Content .\SHA256SUMS.txt
```

Checksum verification confirms download integrity. It does not replace source review. Download artifacts only from the official repository workflow or release process, and do not run random binaries from unknown sources. For RC notes and operator checklist, see `docs/ReleaseNotes-v0.4.6-testnet-rc1.md` and `docs/OperatorChecklist.md`.

## Linux VPS Deployment

Install binaries:

```sh
sudo mkdir -p /opt/indochain /etc/indochain /var/lib/indochain/testnet
sudo tar -xzf indochain-v0.4.6-testnet-rc1-linux-amd64.tar.gz -C /opt/indochain
sudo install -m 0755 /opt/indochain/indochain /usr/local/bin/indochain
sudo install -m 0755 /opt/indochain/indominer /usr/local/bin/indominer
sudo install -m 0755 /opt/indochain/indoservice /usr/local/bin/indoservice
indochain version
```

Create an optional service user:

```sh
sudo useradd --system --home /var/lib/indochain --shell /usr/sbin/nologin indochain || true
sudo chown -R indochain:indochain /var/lib/indochain
```

Initialize the datadir:

```sh
sudo -u indochain indochain --datadir /var/lib/indochain/testnet --network testnet init
sudo -u indochain indochain --datadir /var/lib/indochain/testnet chain info
sudo -u indochain indochain --datadir /var/lib/indochain/testnet chain validate
```

Create `/etc/indochain/testnet.env` from `examples/testnet/seed-node.env` or `examples/testnet/public-node.env`, then edit hostnames and ports:

```sh
sudo install -m 0644 examples/testnet/seed-node.env /etc/indochain/testnet.env
sudo editor /etc/indochain/testnet.env
```

For a production-like seed unit, `examples/systemd/indochain-testnet.env` uses `0.0.0.0:9311` RPC, `0.0.0.0:10311` P2P, and an editable advertised P2P address. Use a LAN/Tailscale/VPS address other peers can actually reach.

Install systemd:

```sh
sudo install -m 0644 examples/systemd/indochain-testnet.service /etc/systemd/system/indochain-testnet.service
sudo systemctl daemon-reload
sudo systemctl enable --now indochain-testnet
sudo journalctl -u indochain-testnet -f
```

The example systemd unit uses `Restart=always`, `KillSignal=SIGINT`, `TimeoutStopSec=30`, `LimitNOFILE=65535`, and journald logging. Do not put private keys in `/etc/indochain/testnet.env`.

Check the node:

```sh
curl http://127.0.0.1:9311/health
indochain --rpc-url http://127.0.0.1:9311 chain info
indochain --rpc-url http://127.0.0.1:9311 chain validate
```

After a restart, run the same checks and confirm height/tip persist. If a crash leaves `node.lock`, remove it only after confirming no `indochain` process is using the datadir.

Open the P2P port in the VPS firewall. Expose RPC only when using public-safe read-only mode and only with the intended flags.

Read-only explorer endpoints are available under `/explorer/*` in public RPC mode without enabling wallet/admin RPC:

```sh
curl http://127.0.0.1:9311/explorer/status
curl "http://127.0.0.1:9311/explorer/blocks?limit=10"
curl "http://127.0.0.1:9311/explorer/search?q=0"
```

The embedded explorer web UI is available from the same node:

```text
http://127.0.0.1:9311/explorer-ui/
http://100.101.251.7:9311/explorer-ui/
```

The UI is read-only. It does not expose wallet, admin, faucet, staking, mining, or service write actions. Testnet dIDR has no monetary value, mainnet is not available, and service points are simulation-only.

Explorer list endpoints use `limit` and `offset`; the default limit is `20` and the maximum is `100`. Invalid pagination returns JSON errors with `ok: false`, `error`, and `message` fields.

## Mining From A Separate Wallet

Use a separate wallet datadir:

```sh
indochain --datadir /var/lib/indochain/miner-wallet --network testnet init
ADDR="$(indochain --datadir /var/lib/indochain/miner-wallet wallet new)"
indominer --rpc-url http://127.0.0.1:9011 --address "$ADDR" --threads 2 --once
indochain --rpc-url http://127.0.0.1:9011 chain validate
```

Miner RPC should be enabled only on nodes intended to accept direct miner traffic.

## Add A Seed Peer To Another Node

Use one of:

```sh
indochain --datadir /var/lib/indochain/node-b --network testnet init
indochain --datadir /var/lib/indochain/node-b node start --rpc :9012 --p2p :10012 --advertise-p2p http://<PUBLIC_HOST_B>:10012 --public-rpc --seed-peer http://<SEED_HOST>:10011
indochain --datadir /var/lib/indochain/node-b node start --rpc :9012 --p2p :10012 --advertise-p2p http://<PUBLIC_HOST_B>:10012 --public-rpc --seed-file config/testnet-seeds.txt
IND_SEED_PEERS=http://host1:10011,http://host2:10011 indochain --datadir /var/lib/indochain/node-b node start --rpc :9012 --p2p :10012 --advertise-p2p http://<PUBLIC_HOST_B>:10012 --public-rpc
```

Seed peers are not trusted authorities. Offline seeds should not crash startup, and wrong network/genesis peers are rejected by normal peer validation.

If a peer was offline during mining or block broadcast failed, wait for periodic sync or run:

```sh
indochain --rpc-url http://127.0.0.1:9012 peer sync http://<SEED_HOST>:10011
indochain --rpc-url http://127.0.0.1:9012 chain validate
```

## Optional Faucet Node

Faucet RPC is disabled by default. Enable it only deliberately on controlled testnet nodes. The faucet source wallet must have mature spendable balance, and faucet requests create normal transactions that require mining.

Recommended controlled service-collateral testing amount:

- `IND_FAUCET_AMOUNT=1000`
- `IND_FAUCET_MAX_PER_ADDRESS=2000`
- `IND_FAUCET_MIN_INTERVAL=1m` or stricter for public tests

Back up `wallets.json`. Back up `faucet_state.json` if you want rate-limit continuity.

For the multi-host faucet request and service-collateral flow, see `docs/Faucet.md`, `docs/ServiceNode.md`, and `docs/LongRunTestnet.md`.

## Optional Service Node Agent

Service-node eligibility on testnet currently requires `1000 dIDR` active collateral. Staking is collateral-only, has no APY, no slashing, no validators, and does not change PoW block production.

Run the service agent with its own state file:

```sh
indoservice --rpc-url http://127.0.0.1:9011 --address <OWNER_ADDR> --endpoint http://<SERVICE_HOST>:9971 --state /var/lib/indochain/service-agent-state.json
```

Service points are simulation-only and are not spendable dIDR. Back up the owner wallet.

Service registration and challenge data are local simulation state on the RPC node used by `indoservice`; stake collateral itself is chain-backed and syncs between peers.

## Windows Notes

Extract `indochain-v0.4.6-testnet-rc1-windows-amd64.zip`, then:

```powershell
.\indochain.exe version
.\indochain.exe --datadir .\data\testnet --network testnet init
.\indochain.exe --datadir .\data\testnet node start --rpc :9011 --p2p :10011 --advertise-p2p http://127.0.0.1:10011 --public-rpc --enable-miner-rpc
```

Create a separate miner wallet:

```powershell
.\indochain.exe --datadir .\data\miner --network testnet init
$addr = .\indochain.exe --datadir .\data\miner wallet new
.\indominer.exe --rpc-url http://127.0.0.1:9011 --address $addr --threads 2 --once
.\indochain.exe --rpc-url http://127.0.0.1:9011 chain validate
```
