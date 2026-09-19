# Seed Operator Publish Guide

This guide helps operators publish a public testnet seed peer safely.

Testnet DKC has no monetary value. Mainnet is not available.

## Choose A Seed URL

Publish the P2P URL that peers can reach:

```text
http://<host>:10311
```

Examples:

- LAN: `http://192.168.1.5:10311`
- Tailscale: `http://100.101.251.7:10311`
- VPS: `http://<VPS_PUBLIC_IP>:10311`
- RC1 seed: `http://100.86.152.39:10311`

Do not publish `127.0.0.1` as a seed for other hosts.

Seeds are not trusted authorities. They only help nodes find peers; nodes validate network ID, chain ID, genesis hash, headers, and blocks.

## Firewall

- P2P port can be public for seed operation.
- RPC should preferably stay local or trusted.
- If public RPC is enabled, wallet/admin RPC must remain disabled.

## Advertise Address

Start the seed node with a reachable advertised P2P address:

```sh
deskachain --datadir /var/lib/deskachain/testnet --network testnet node start \
  --rpc 127.0.0.1:9311 \
  --p2p 0.0.0.0:10311 \
  --advertise-p2p http://<host>:10311 \
  --public-rpc
```

## Publish Seed Peer

Publish:

```text
http://<host>:10311
```

Add it to operator docs or a seed file only after external reachability is verified.
Publish more than one operator seed when available:

```text
http://100.86.152.39:10311
http://100.101.251.7:10311
```

## Verify From Another Host

From another node:

```sh
deskachain --rpc-url http://127.0.0.1:9312 peer check http://<host>:10311
deskachain --rpc-url http://127.0.0.1:9312 peer sync http://<host>:10311
deskachain --rpc-url http://127.0.0.1:9312 chain info
```

If seed RPC is intentionally reachable from a trusted network:

```sh
curl http://<host>:9311/explorer/status
```

## Rotate A Seed URL

1. Publish the new seed URL.
2. Keep the old seed online long enough for operators to update.
3. Remove the old seed from published seed files.
4. Document the change.

## Remove A Bad Seed

1. Remove it from published seed docs/files.
2. Announce the removal.
3. Ask operators to remove or ignore the bad peer.
4. Verify remaining seeds still match network ID, chain ID, genesis hash, and protocol.
