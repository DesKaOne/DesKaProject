# DesKaChain Public Testnet Preflight Checklist

Use this checklist before running a public testnet node on a VPS or external device.

- Binary version is correct: `deskachain version`.
- Release checksum has been verified.
- Network is `testnet`.
- Genesis hash is `db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4`.
- Datadir is clean or intentionally reused.
- `chain validate` passes.
- `/health` is reachable.
- `/health` reports the expected network, network ID, chain ID, genesis hash, height, tip hash, RPC mode flags, peer count, and uptime.
- `chain info` shows `network: testnet`, `network id: dkc-testnet-1`, and `chain id: 777101`.
- Wallet/admin RPC are not exposed publicly.
- Faucet RPC is disabled unless intentionally operating a controlled faucet.
- Service write RPC is disabled unless intentionally operating a controlled service verifier/test node.
- Enabling faucet/service RPC does not imply wallet/admin RPC should be enabled.
- Faucet wallet has mature spendable testnet balance before requests are opened.
- Service collateral tests use mined/confirmed funds and remember service state is node-local simulation state.
- Miner RPC is enabled only when the node should accept public mining traffic.
- Advertised P2P URL is set and reachable.
- Bind addresses and advertised P2P address are intentionally different when needed. Do not advertise `127.0.0.1` to another host.
- P2P port is open in the firewall.
- Seed peers are configured with `--seed-peer`, `--seed-file`, `DKC_SEED_PEERS`, or config.
- Seed peers are treated as hints, not trusted authorities.
- Wallet files are backed up when used.
- Testnet DKC has no monetary value.
- Mainnet is not available.
- If using systemd, `Restart=always`, `KillSignal=SIGINT`, `TimeoutStopSec=30`, and `LimitNOFILE=65535` are configured.

Common checks:

```sh
deskachain version
deskachain --datadir /var/lib/deskachain/testnet chain info
deskachain --datadir /var/lib/deskachain/testnet chain validate
curl http://127.0.0.1:9011/health
deskachain --rpc-url http://127.0.0.1:9011 peer list --source
```

Multi-host firewall, Tailscale, and bind-vs-advertise examples are in `docs/MultiHostTestnet.md`.
Long-running restart and offline catch-up checks are in `docs/LongRunTestnet.md`. systemd recovery details are in `docs/Systemd.md`.
