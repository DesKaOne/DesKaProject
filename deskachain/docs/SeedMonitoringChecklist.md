# IndoChain Seed Monitoring Checklist

Testnet dIDR has no monetary value. Mainnet is not available. Do not expose wallet/admin RPC publicly.

Use this checklist for RC1 seed nodes and any operator-published bootstrap peer.

## Core Checks

- Seed process is running.
- P2P port is reachable from outside the host.
- Peer check works from another node.
- Seed advertises the correct URL.
- Multiple seeds are published when available:
  - `http://100.86.152.39:10311`
  - `http://100.101.251.7:10311`
- Seeds are not trusted authorities; network/genesis/block validation remains mandatory.
- Network ID is `ind-testnet-1`.
- Chain ID is `777101`.
- Genesis hash is `db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4`.
- Public RPC safety is correct:
  - `wallet_rpc` is `false`.
  - `admin_rpc` is `false`.
- Explorer UI works if RPC is intentionally public.
- Logs have no repeated panic/errors.
- Disk space is OK.
- Restart recovery is OK.

## Commands

Health check:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\testnet-health.ps1 -RpcUrl http://<host>:9311 -ExpectedNetwork testnet -ExpectedNetworkID ind-testnet-1 -ExpectedChainID 777101
```

Peer check from another node:

```powershell
.\indochain.exe --rpc-url http://127.0.0.1:9311 peer check http://<seed-host>:10311/
```

Peer health and discovery:

```powershell
.\indochain.exe --rpc-url http://127.0.0.1:9311 peer health
.\indochain.exe --rpc-url http://127.0.0.1:9311 peer seeds
.\indochain.exe --rpc-url http://127.0.0.1:9311 peer discover
```

Chain validation on the seed host:

```powershell
.\indochain.exe --rpc-url http://127.0.0.1:9311 chain validate
```

Systemd logs:

```sh
sudo systemctl status indochain-testnet --no-pager
journalctl -u indochain-testnet -n 200 --no-pager
```

## LAN Example

- Bind P2P to the LAN interface or `0.0.0.0`.
- Advertise a LAN URL such as `http://192.168.1.20:10311/`.
- Keep RPC on localhost unless the LAN tester group needs read-only explorer access.
- Confirm another LAN machine can run `peer check` against the advertised P2P URL.

## Tailscale Example

- Bind P2P to the Tailscale host or `0.0.0.0`.
- Advertise a Tailscale URL such as `http://100.x.y.z:10311/`.
- Confirm ACLs allow tester nodes to reach the seed.
- Keep wallet/admin RPC disabled even on a private tailnet.

## VPS/Public Example

- Open only the intended P2P port and optional public read-only RPC port.
- Publish the advertised P2P URL, network ID, chain ID, genesis hash, and checksums.
- Run health checks from a machine outside the VPS provider network.
- Use systemd restart recovery checks after updates.
- Never publish wallet files, datadirs, private keys, faucet state, or service state.
