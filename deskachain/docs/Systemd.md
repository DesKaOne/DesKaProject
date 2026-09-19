# DesKaChain systemd Testnet Node

This guide runs a public-testnet node under systemd with restart recovery. It does not launch mainnet. Testnet IDR has no monetary value, PoW remains the only block-production consensus, staking is collateral-only, and service points are simulation-only.

## Unit Behavior

The example unit in `examples/systemd/deskachain-testnet.service` is intended for long-running seed or miner-enabled testnet nodes.

- `Restart=always` restarts the node after a crash or non-zero exit.
- `RestartSec=5` avoids a tight restart loop.
- `KillSignal=SIGINT` lets the node run the same graceful shutdown path as Ctrl+C.
- `TimeoutStopSec=30` gives RPC and P2P servers time to stop and release `node.lock`.
- `LimitNOFILE=65535` raises the file descriptor limit for public peer traffic.
- `StandardOutput=journal` and `StandardError=journal` keep logs in journald.

The environment file must not contain private keys. Wallet files, faucet state, chain DBs, mempool, peer store, node ID, and locks stay in the datadir.

## Install

```sh
sudo useradd --system --home /var/lib/deskachain --shell /usr/sbin/nologin deskachain || true
sudo mkdir -p /opt/deskachain /var/lib/deskachain/testnet /etc/deskachain
sudo chown -R deskachain:deskachain /var/lib/deskachain
sudo install -m 0755 deskachain idrminer idrservice /opt/deskachain/
sudo install -m 0644 examples/systemd/deskachain-testnet.env /etc/deskachain/testnet.env
sudo install -m 0644 examples/systemd/deskachain-testnet.service /etc/systemd/system/deskachain-testnet.service
sudo editor /etc/deskachain/testnet.env
sudo systemctl daemon-reload
sudo systemctl enable --now deskachain-testnet
```

Set `IDR_ADVERTISE_P2P` to a LAN, Tailscale, or VPS address that other peers can reach. Keep wallet/admin RPC disabled on public RPC. Enable miner RPC only when the node should accept mining clients.

## Operate

```sh
sudo systemctl status deskachain-testnet
journalctl -u deskachain-testnet -f
journalctl -u deskachain-testnet --since "30 minutes ago"
sudo systemctl restart deskachain-testnet
sudo systemctl stop deskachain-testnet
```

Check the node after start or restart:

```sh
curl http://127.0.0.1:9311/health
./deskachain --rpc-url http://127.0.0.1:9311 chain info
./deskachain --rpc-url http://127.0.0.1:9311 chain validate
./deskachain --rpc-url http://127.0.0.1:9311 peer list --source
```

`/health` reports network, network ID, chain ID, genesis hash, height, tip hash, peer count, mempool count, RPC mode flags, service-node count, staking summary, and uptime seconds. It does not expose wallet private data.

## Restart Recovery Test

1. Start the service.
2. Mine at least one testnet block.
3. Record `chain info` height and tip.
4. Run `sudo systemctl restart deskachain-testnet`.
5. Re-run `chain info` and `chain validate`.

Expected result:

- height and tip persist across restart.
- `chain validate` passes.
- `node.lock` is released during graceful shutdown and recreated on start.
- persisted peers remain visible in `peer list`.

## Failure Recovery Test

```sh
pidof deskachain
sudo kill -9 <PID>
sudo systemctl status deskachain-testnet
journalctl -u deskachain-testnet --since "5 minutes ago"
./deskachain --rpc-url http://127.0.0.1:9311 chain validate
```

Expected result:

- systemd restarts the service.
- chain validation still passes after startup.
- if an unclean stop leaves `node.lock`, remove it only after confirming no `deskachain` process is using that datadir.

## Duplicate Datadir Lock Test

While the service is running, try starting a second node with the same datadir:

```sh
./deskachain --datadir /var/lib/deskachain/testnet --network testnet node start --rpc :9312 --p2p :10312
```

Expected result: startup fails clearly because the datadir is locked by the running node.

