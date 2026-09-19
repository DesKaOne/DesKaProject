# DesKaChain Long-Running Public Testnet Checklist

Use this checklist for a two-host public-testnet soak and restart recovery run. It does not launch mainnet. Testnet IDR has no monetary value, PoW remains the only block-production consensus, staking is collateral-only, and service points are simulation-only.

## Hosts

- Host A: Mini PC seed node.
- Host B: Windows or Linux peer/miner node.
- Network: `testnet`.
- Network ID: `idr-testnet-1`.
- Chain ID: `777101`.
- Genesis hash: `db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4`.

## Start Host A

```sh
./deskachain --datadir ./data/seed --network testnet init
./deskachain --datadir ./data/seed node start \
  --rpc 0.0.0.0:9311 \
  --p2p 0.0.0.0:10311 \
  --advertise-p2p http://<HOST_A_REACHABLE_IP>:10311 \
  --public-rpc \
  --enable-miner-rpc
```

## Start Host B

```sh
./deskachain --datadir ./data/node_b --network testnet init
./deskachain --datadir ./data/node_b node start \
  --rpc 0.0.0.0:9312 \
  --p2p 0.0.0.0:10312 \
  --advertise-p2p http://<HOST_B_REACHABLE_IP>:10312 \
  --public-rpc \
  --enable-miner-rpc \
  --seed-peer http://<HOST_A_REACHABLE_IP>:10311/
```

## Baseline

1. Start Host A.
2. Start Host B with Host A as seed peer.
3. Check peers:

```sh
./deskachain --rpc-url http://127.0.0.1:9311 peer list --source
./deskachain --rpc-url http://127.0.0.1:9312 peer list --source
./deskachain --rpc-url http://127.0.0.1:9312 peer check http://<HOST_A_REACHABLE_IP>:10311
```

4. Mine one block from either host:

```sh
./idrminer --rpc-url http://127.0.0.1:9311 --address <ADDR_A> --threads 2 --max-blocks 1
```

5. Confirm both nodes match:

```sh
./deskachain --rpc-url http://127.0.0.1:9311 chain info
./deskachain --rpc-url http://127.0.0.1:9312 chain info
./deskachain --rpc-url http://127.0.0.1:9311 chain validate
./deskachain --rpc-url http://127.0.0.1:9312 chain validate
```

Expected result: same height, same tip hash, both chains valid.

## Peer Offline Catch-Up

1. Stop Host B with Ctrl+C or systemd stop.
2. Mine 3 to 10 blocks on Host A.
3. Restart Host B with the same datadir and seed peer.
4. Wait for the periodic peer sync loop.
5. Confirm Host B catches up:

```sh
./deskachain --rpc-url http://127.0.0.1:9312 peer sync http://<HOST_A_REACHABLE_IP>:10311
./deskachain --rpc-url http://127.0.0.1:9311 chain info
./deskachain --rpc-url http://127.0.0.1:9312 chain info
./deskachain --rpc-url http://127.0.0.1:9312 chain validate
```

Expected result: Host B imports missed blocks and ends with the same height and tip hash as Host A.

## Seed Restart

1. Stop Host A.
2. Keep Host B running.
3. Restart Host A with the same datadir.
4. Run `peer check` from Host B.
5. Mine one block on either host.
6. Confirm both nodes sync to the same height and tip.

Expected result: peer entries persist, reconnect works, and wrong network/genesis peers remain rejected.

## Systemd Restart

Run Host A under `examples/systemd/deskachain-testnet.service`, then:

```sh
sudo systemctl restart deskachain-testnet
sudo systemctl status deskachain-testnet
journalctl -u deskachain-testnet --since "10 minutes ago"
./deskachain --rpc-url http://127.0.0.1:9311 chain info
./deskachain --rpc-url http://127.0.0.1:9311 chain validate
```

Expected result: the node stops through SIGINT, releases `node.lock`, restarts, keeps height/tip, and validates the chain.

## Duplicate Datadir Lock

1. Start Host A normally.
2. Attempt a second node with the same datadir.

Expected result: the second node fails with a datadir lock error. Do not delete `node.lock` while a node process is still running.

## Recovery Commands

Peer state:

```sh
./deskachain --rpc-url http://127.0.0.1:9311 peer list --source
./deskachain --rpc-url http://127.0.0.1:9311 peer check http://<peer>:<p2p>
./deskachain --rpc-url http://127.0.0.1:9311 peer sync http://<peer>:<p2p>
```

Chain state:

```sh
./deskachain --rpc-url http://127.0.0.1:9311 chain info
./deskachain --rpc-url http://127.0.0.1:9311 chain validate
curl http://127.0.0.1:9311/health
```

Logs:

```sh
journalctl -u deskachain-testnet -f
journalctl -u deskachain-testnet --since "30 minutes ago"
```

Expected recoverable errors:

- `connection refused`: peer process is not listening or firewall blocks the port.
- `timeout`: peer is slow or unreachable; the node should keep running.
- `network id mismatch`: peer is on a different network profile and must not sync.
- `genesis hash mismatch`: peer is on a different chain and must not sync.
- `stale template`: miner should request a fresh block template.
- `datadir is locked`: another node process owns the datadir.

## Long-Run Notes

- Leave both nodes running for the planned soak period.
- Periodically record `/health`, `chain info`, `peer list`, and `chain validate` from both hosts.
- Keep wallet/admin RPC disabled on any public RPC bind.
- Use seed peers as bootstrap hints only; they are not trusted authorities.
- Do not treat mined testnet IDR as having monetary value.

## Faucet + Service E2E Add-On

After the baseline sync checks pass, run the controlled Phase 4.2 flow:

1. Start Host A with `--enable-faucet-rpc`, a mature operator faucet wallet, and wallet/admin RPC still disabled.
2. Start Host B with `--seed-peer` pointing at Host A and `--enable-service-rpc` only if Host B is the controlled service test node.
3. Create `OWNER_ADDR` in a Host B wallet datadir.
4. Request `1000 IDR` from Host A faucet RPC.
5. Mine one block and confirm Host B sees the faucet transfer after sync.
6. Lock `1000 IDR` as stake from Host B.
7. Mine one block and confirm `stake list` reports active stake.
8. Register Host B service endpoint and run `idrservice --once`.
9. Confirm `service score` reports required stake `1000 IDR`, active stake `1000 IDR`, and `stake eligible: true`.
10. Confirm Host A and Host B have the same height/tip and both pass `chain validate`.
11. Confirm `chain info` supply changed only due to mined coinbase rewards, not faucet or service scoring.

Service registration and score samples are local simulation state on the RPC node used by `idrservice`. Chain-backed stake and supply should sync across hosts.
